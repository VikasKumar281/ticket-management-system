package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticket-system/internal/handlers"
	"ticket-system/internal/middleware"
	"ticket-system/internal/store"
)

// newTestServer builds a fully wired httptest server backed by a fresh
// in-memory store, mirroring the routing set up in main().
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	const testSecret = "test-secret"
	s := store.New()
	authHandler := handlers.NewAuthHandler(s, testSecret)
	ticketHandler := handlers.NewTicketHandler(s)
	requireAuth := middleware.Auth(testSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.Handle("POST /tickets", requireAuth(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /tickets", requireAuth(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /tickets/{id}", requireAuth(http.HandlerFunc(ticketHandler.Get)))
	mux.Handle("PATCH /tickets/{id}/status", requireAuth(http.HandlerFunc(ticketHandler.UpdateStatus)))

	return httptest.NewServer(mux)
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := doJSON(t, http.MethodGet, srv.URL+"/health", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	decodeBody(t, resp, &body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body)
	}
}

func TestFullTicketLifecycle(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// Register user A.
	resp := doJSON(t, http.MethodPost, srv.URL+"/auth/register", "", map[string]string{
		"email": "alice@example.com", "password": "hunter22",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Duplicate registration should conflict.
	resp = doJSON(t, http.MethodPost, srv.URL+"/auth/register", "", map[string]string{
		"email": "alice@example.com", "password": "hunter22",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Login as alice.
	resp = doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{
		"email": "alice@example.com", "password": "hunter22",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", resp.StatusCode)
	}
	var loginBody struct {
		Token string `json:"token"`
	}
	decodeBody(t, resp, &loginBody)
	if loginBody.Token == "" {
		t.Fatal("expected non-empty token")
	}
	aliceToken := loginBody.Token

	// Wrong password should be rejected.
	resp = doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{
		"email": "alice@example.com", "password": "wrong-password",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Unauthenticated ticket creation should be rejected.
	resp = doJSON(t, http.MethodPost, srv.URL+"/tickets", "", map[string]string{
		"title": "No auth", "description": "should fail",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create: expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Create a ticket as alice.
	resp = doJSON(t, http.MethodPost, srv.URL+"/tickets", aliceToken, map[string]string{
		"title": "Printer on fire", "description": "Send help",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create ticket: expected 201, got %d", resp.StatusCode)
	}
	var ticket struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, resp, &ticket)
	if ticket.Status != "open" {
		t.Fatalf("expected new ticket status open, got %s", ticket.Status)
	}

	// List tickets as alice: should contain exactly one.
	resp = doJSON(t, http.MethodGet, srv.URL+"/tickets", aliceToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tickets: expected 200, got %d", resp.StatusCode)
	}
	var list []map[string]any
	decodeBody(t, resp, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(list))
	}

	// Register + login user B (bob), who should NOT see alice's ticket.
	resp = doJSON(t, http.MethodPost, srv.URL+"/auth/register", "", map[string]string{
		"email": "bob@example.com", "password": "supersecret",
	})
	resp.Body.Close()
	resp = doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{
		"email": "bob@example.com", "password": "supersecret",
	})
	decodeBody(t, resp, &loginBody)
	bobToken := loginBody.Token

	resp = doJSON(t, http.MethodGet, srv.URL+"/tickets/"+ticket.ID, bobToken, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob accessing alice's ticket: expected 403, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", bobToken, map[string]string{
		"status": "in_progress",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob updating alice's ticket: expected 403, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Alice moves open -> in_progress.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", aliceToken, map[string]string{
		"status": "in_progress",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open->in_progress: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Alice cannot skip back to open.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", aliceToken, map[string]string{
		"status": "open",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("in_progress->open: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Alice moves in_progress -> closed.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", aliceToken, map[string]string{
		"status": "closed",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("in_progress->closed: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Closed ticket must not be reopened.
	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", aliceToken, map[string]string{
		"status": "open",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("closed->open: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = doJSON(t, http.MethodPatch, srv.URL+"/tickets/"+ticket.ID+"/status", aliceToken, map[string]string{
		"status": "in_progress",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("closed->in_progress: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Fetching a nonexistent ticket returns 404.
	resp = doJSON(t, http.MethodGet, srv.URL+"/tickets/does-not-exist", aliceToken, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("nonexistent ticket: expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
