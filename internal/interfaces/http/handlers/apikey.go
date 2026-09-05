package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"arca-invoice-proxy/internal/application/authentication"
	"arca-invoice-proxy/internal/interfaces/http/dto"
	"arca-invoice-proxy/internal/interfaces/http/middleware"
)

type APIKeyHandler struct {
	service *authentication.Service
}

func NewAPIKeyHandler(service *authentication.Service) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req dto.APIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var expiresAt *int64
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}

	key, rawKey, err := h.service.CreateAPIKey(r.Context(), userID, req.Name, req.Prefix, req.Scopes, expiresAt)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := dto.APIKeyResponse{
		ID:        key.ID,
		Name:      key.Name,
		Prefix:    key.Prefix,
		LastFour:  key.LastFour,
		Scopes:    key.Scopes,
		ExpiresAt: key.ExpiresAt,
		RevokedAt: key.RevokedAt,
		CreatedAt: key.CreatedAt.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"api_key": rawKey,
		"details": resp,
	})
}

func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := h.service.ListAPIKeys(r.Context(), userID)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]dto.APIKeyResponse, len(keys))
	for i, key := range keys {
		var expiresAt, revokedAt *int64
		if key.ExpiresAt != nil {
			t := key.ExpiresAt.Unix()
			expiresAt = &t
		}
		if key.RevokedAt != nil {
			t := key.RevokedAt.Unix()
			revokedAt = &t
		}
		resp[i] = dto.APIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Prefix:    key.Prefix,
			LastFour:  key.LastFour,
			Scopes:    key.Scopes,
			ExpiresAt: expiresAt,
			RevokedAt: revokedAt,
			CreatedAt: key.CreatedAt.Unix(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keyID := r.PathValue("id")
	if keyID == "" {
		writeError(w, "api key id required", http.StatusBadRequest)
		return
	}

	if err := h.service.RevokeAPIKey(r.Context(), keyID); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: message,
		Code:  "error",
	})
}

func parseTime(t *int64) *time.Time {
	if t == nil {
		return nil
	}
	parsed := time.Unix(*t, 0).UTC()
	return &parsed
}