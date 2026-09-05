package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"arca-invoice-proxy/internal/application/authentication"
	"arca-invoice-proxy/internal/domain/errors"
)

type contextKey string

const (
	UserIDKey     contextKey = "user_id"
	APIKeyIDKey   contextKey = "api_key_id"
	RequestIDKey  contextKey = "request_id"
)

func Authentication(authService *authentication.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, errors.New(errors.CodeInvalidAPIKey, "authorization header required"), http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, errors.New(errors.CodeInvalidAPIKey, "invalid authorization format"), http.StatusUnauthorized)
				return
			}

			rawKey := parts[1]
			key, err := authService.ValidateAPIKey(r.Context(), rawKey)
			if err != nil {
				status := http.StatusUnauthorized
				if errors.IsCode(err, errors.CodeAPIKeyRevoked) || errors.IsCode(err, errors.CodeAPIKeyExpired) {
					status = http.StatusForbidden
				}
				writeError(w, err, status)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, key.CustomerID)
			ctx = context.WithValue(ctx, APIKeyIDKey, key.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func GetAPIKeyID(ctx context.Context) string {
	if v := ctx.Value(APIKeyIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	code := errors.GetCode(err)
	details := make(map[string]string)
	if appErr, ok := err.(*errors.AppError); ok {
		details = appErr.Details
	}

	response := struct {
		Error   string            `json:"error"`
		Code    string            `json:"code"`
		Details map[string]string `json:"details,omitempty"`
	}{
		Error:   err.Error(),
		Code:    string(code),
		Details: details,
	}

	writeJSON(w, response)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	_ = json.NewEncoder(w).Encode(data)
}

func generateRequestID() string {
	return "req_" + time.Now().UTC().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}