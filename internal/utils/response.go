package utils

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent; just log-worthy, nothing
		// more we can do. We avoid importing a logger here to keep this
		// package dependency-free.
		return
	}
}

// ErrorResponse is the standard JSON shape returned for all error cases.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError writes a standard {"error": "..."} JSON body with the given status.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// DecodeJSON decodes the request body into dst. It rejects unknown fields
// so that typos in client payloads are caught early with a clear error
// instead of being silently ignored.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
