package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"ticket-system/internal/middleware"
	"ticket-system/internal/models"
	"ticket-system/internal/store"
	"ticket-system/internal/utils"
)

// TicketHandler groups the ticket endpoints and their dependencies.
type TicketHandler struct {
	store *store.Store
}

// NewTicketHandler builds a TicketHandler.
func NewTicketHandler(s *store.Store) *TicketHandler {
	return &TicketHandler{store: s}
}

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Create handles POST /tickets.
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTicketRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) {
			utils.WriteError(w, http.StatusBadRequest, "request body is required")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		utils.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}

	ticket := h.store.CreateTicket(userID, req.Title, req.Description)
	utils.WriteJSON(w, http.StatusCreated, ticket)
}

// List handles GET /tickets — returns only tickets owned by the caller.
func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tickets := h.store.ListTicketsByUser(userID)
	utils.WriteJSON(w, http.StatusOK, tickets)
}

// Get handles GET /tickets/{id} — only the owner may view their ticket.
func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	ticket, err := h.store.GetTicket(id)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	if ticket.UserID != userID {
		// 403 is used consistently across all ticket endpoints for
		// ownership violations, rather than masking as 404.
		utils.WriteError(w, http.StatusForbidden, "you do not have access to this ticket")
		return
	}

	utils.WriteJSON(w, http.StatusOK, ticket)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /tickets/{id}/status — only the owner may
// update their ticket, and only along the allowed open -> in_progress ->
// closed lifecycle. Closed tickets can never be reopened.
func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	ticket, err := h.store.GetTicket(id)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	if ticket.UserID != userID {
		utils.WriteError(w, http.StatusForbidden, "you do not have access to this ticket")
		return
	}

	var req updateStatusRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) {
			utils.WriteError(w, http.StatusBadRequest, "request body is required")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	newStatus := models.TicketStatus(strings.TrimSpace(req.Status))
	if !newStatus.IsValid() {
		utils.WriteError(w, http.StatusBadRequest, "status must be one of: open, in_progress, closed")
		return
	}

	if ticket.Status == newStatus {
		utils.WriteError(w, http.StatusBadRequest, "ticket is already in status "+string(newStatus))
		return
	}

	if !models.CanTransition(ticket.Status, newStatus) {
		utils.WriteError(w, http.StatusBadRequest,
			"invalid status transition from "+string(ticket.Status)+" to "+string(newStatus))
		return
	}

	updated, err := h.store.UpdateTicketStatus(id, newStatus)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "ticket not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, updated)
}
