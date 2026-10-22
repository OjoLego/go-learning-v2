package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"dime-api/internal/auth"
	"dime-api/internal/config"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

type tokenRequest struct {
	UserID string `json:"user_id"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		RespondError(w, r, errors.New("user_id is required"))
		return
	}

	token, err := auth.GenerateToken(req.UserID, h.cfg.JWTSecret, time.Duration(h.cfg.JWTExpiryMins)*time.Minute)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	respondJSON(w, http.StatusOK, tokenResponse{Token: token})
}
