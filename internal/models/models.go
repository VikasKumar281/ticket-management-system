package models

import "time"

// User represents a registered account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialized in API responses
	CreatedAt    time.Time `json:"created_at"`
}

// TicketStatus is the lifecycle state of a ticket.
type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusClosed     TicketStatus = "closed"
)

// IsValid reports whether s is one of the supported ticket statuses.
func (s TicketStatus) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	default:
		return false
	}
}

// allowedTransitions defines the only legal forward moves in the ticket
// status lifecycle: open -> in_progress -> closed. A closed ticket can
// never move back to open or in_progress, and no status may be skipped.
var allowedTransitions = map[TicketStatus]TicketStatus{
	StatusOpen:       StatusInProgress,
	StatusInProgress: StatusClosed,
}

// CanTransition reports whether moving from "from" to "to" is a legal
// status transition.
func CanTransition(from, to TicketStatus) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false // closed (or unknown) has no legal next state
	}
	return next == to
}

// Ticket represents a single support ticket owned by a user.
type Ticket struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
