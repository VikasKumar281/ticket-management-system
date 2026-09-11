package handlers

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ticket-system/internal/auth"
	"ticket-system/internal/store"
	"ticket-system/internal/utils"
)

// tokenTTL controls how long an issued JWT remains valid.
const tokenTTL = 24 * time.Hour

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// AuthHandler groups the register/login endpoints and their dependencies.
type AuthHandler struct {
	store     *store.Store
	jwtSecret string
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(s *store.Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: s, jwtSecret: jwtSecret}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) {
			utils.WriteError(w, http.StatusBadRequest, "request body is required")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || !emailPattern.MatchString(req.Email) {
		utils.WriteError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(req.Password) < 6 {
		utils.WriteError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user, err := h.store.CreateUser(req.Email, hash)
	if err != nil {
		if errors.Is(err, store.ErrUserExists) {
			utils.WriteError(w, http.StatusConflict, "a user with this email already exists")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, registerResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		if errors.Is(err, io.EOF) {
			utils.WriteError(w, http.StatusBadRequest, "request body is required")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		// Same message for "no such user" and "wrong password" to avoid
		// leaking which emails are registered.
		utils.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	ok, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		utils.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, h.jwtSecret, tokenTTL)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	utils.WriteJSON(w, http.StatusOK, loginResponse{Token: token})
}
