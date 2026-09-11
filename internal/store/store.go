// Package store provides a thread-safe in-memory persistence layer.
//
// The assignment explicitly allows in-memory storage ("No complex database
// schema is required. Use in-memory storage, SQLite, PostgreSQL, or any
// simple persistent store."). In-memory storage is used here to keep the
// service simple, dependency-free, and trivially deployable on any
// free-tier host without provisioning a database. The Store is defined
// behind method calls only, so swapping in a real database later (e.g.
// SQLite via database/sql) would not require changing any handler code.
package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"ticket-system/internal/models"
	"ticket-system/internal/utils"
)

var (
	ErrUserExists     = errors.New("store: user with this email already exists")
	ErrUserNotFound   = errors.New("store: user not found")
	ErrTicketNotFound = errors.New("store: ticket not found")
)

// Store holds all application state in memory, guarded by a single mutex.
// Traffic volumes for this assignment do not warrant finer-grained locking.
type Store struct {
	mu sync.RWMutex

	usersByID    map[string]*models.User
	usersByEmail map[string]string // normalized email -> user ID

	tickets map[string]*models.Ticket
}

// New creates an empty, ready-to-use Store.
func New() *Store {
	return &Store{
		usersByID:    make(map[string]*models.User),
		usersByEmail: make(map[string]string),
		tickets:      make(map[string]*models.Ticket),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// CreateUser creates and stores a new user with the given email and
// pre-hashed password. Returns ErrUserExists if the email is already taken.
func (s *Store) CreateUser(email, passwordHash string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	norm := normalizeEmail(email)
	if _, exists := s.usersByEmail[norm]; exists {
		return nil, ErrUserExists
	}

	user := &models.User{
		ID:           utils.NewID(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	s.usersByID[user.ID] = user
	s.usersByEmail[norm] = user.ID

	return user, nil
}

// GetUserByEmail looks up a user by email (case-insensitive).
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.usersByEmail[normalizeEmail(email)]
	if !ok {
		return nil, ErrUserNotFound
	}
	return s.usersByID[id], nil
}

// GetUserByID looks up a user by their ID.
func (s *Store) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// CreateTicket creates a new ticket owned by userID, starting in the "open" status.
func (s *Store) CreateTicket(userID, title, description string) *models.Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ticket := &models.Ticket{
		ID:          utils.NewID(),
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tickets[ticket.ID] = ticket
	return ticket
}

// GetTicket fetches a ticket by ID regardless of owner. Callers are
// responsible for enforcing ownership checks against the caller's user ID.
func (s *Store) GetTicket(id string) (*models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ticket, ok := s.tickets[id]
	if !ok {
		return nil, ErrTicketNotFound
	}
	return ticket, nil
}

// ListTicketsByUser returns all tickets owned by userID, most recently
// created first. Always returns a non-nil (possibly empty) slice.
func (s *Store) ListTicketsByUser(userID string) []*models.Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Ticket, 0)
	for _, t := range s.tickets {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	sortTicketsByCreatedAtDesc(result)
	return result
}

// UpdateTicketStatus sets a ticket's status. It does not perform ownership
// or transition-legality checks — those are the handler's responsibility so
// that the correct HTTP status code (403 vs 400) can be chosen by the caller.
func (s *Store) UpdateTicketStatus(id string, newStatus models.TicketStatus) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, ok := s.tickets[id]
	if !ok {
		return nil, ErrTicketNotFound
	}

	ticket.Status = newStatus
	ticket.UpdatedAt = time.Now().UTC()
	return ticket, nil
}

func sortTicketsByCreatedAtDesc(tickets []*models.Ticket) {
	// Simple insertion sort: ticket counts per user are expected to be
	// small for this assignment, so this avoids importing "sort" for a
	// one-line comparison — but using sort.Slice is equally fine.
	for i := 1; i < len(tickets); i++ {
		j := i
		for j > 0 && tickets[j-1].CreatedAt.Before(tickets[j].CreatedAt) {
			tickets[j-1], tickets[j] = tickets[j], tickets[j-1]
			j--
		}
	}
}
