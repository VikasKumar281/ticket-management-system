package handlers

import (
	"net/http"

	"ticket-system/internal/utils"
)

// Health handles GET /health.
func Health(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
