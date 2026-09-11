// Command ticket-system runs the Backend Intern Ticket System REST API.
//
// See README.md for the full API contract, run instructions, and
// deployment notes.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"ticket-system/internal/handlers"
	"ticket-system/internal/middleware"
	"ticket-system/internal/store"
)

const defaultJWTSecret = "dev-only-insecure-secret-change-me"

func main() {
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Println("WARNING: JWT_SECRET is not set; using an insecure default. " +
			"Set JWT_SECRET in the environment for any non-local deployment.")
		jwtSecret = defaultJWTSecret
	}

	s := store.New()

	authHandler := handlers.NewAuthHandler(s, jwtSecret)
	ticketHandler := handlers.NewTicketHandler(s)
	requireAuth := middleware.Auth(jwtSecret)

	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// Protected routes (require Authorization: Bearer <token>).
	mux.Handle("POST /tickets", requireAuth(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /tickets", requireAuth(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /tickets/{id}", requireAuth(http.HandlerFunc(ticketHandler.Get)))
	mux.Handle("PATCH /tickets/{id}/status", requireAuth(http.HandlerFunc(ticketHandler.UpdateStatus)))

	var rootHandler http.Handler = mux
	rootHandler = middleware.Logger(rootHandler)
	rootHandler = middleware.Recover(rootHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      rootHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("ticket-system listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
