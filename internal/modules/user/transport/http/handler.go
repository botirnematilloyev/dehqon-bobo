package http

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"
	"dehqon-bobo/internal/platform/i18n"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type AuthService interface {
	RequestOTP(ctx context.Context, provider domain.AuthProvider, identifier string) error
	VerifyOTP(ctx context.Context, in service.VerifyOTPInput) (*service.AuthResult, error)
	RefreshToken(ctx context.Context, rawRefreshToken string) (*service.AuthResult, error)
	Logout(ctx context.Context, sessionID int64) error
	VerifyTelegramLogin(ctx context.Context, in service.VerifyTelegramLoginInput) (*service.AuthResult, error)
	VerifyTelegramWebApp(ctx context.Context, in service.VerifyTelegramWebAppInput) (*service.AuthResult, error)

	ListSessions(ctx context.Context, userID int64) ([]domain.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID int64) error
	LogoutAll(ctx context.Context, userID int64) error
}

type Handler struct {
	authService AuthService
}

func NewHandler(authService AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req RequestOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, validationKey(err))
		return
	}

	provider, err := parseProvider(req.Provider)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, msgUnsupportedProvider)
		return
	}

	if err := h.authService.RequestOTP(r.Context(), provider, req.Identifier); err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("request otp failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, validationKey(err))
		return
	}

	provider, err := parseProvider(req.Provider)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, msgUnsupportedProvider)
		return
	}

	ip := clientIP(r)

	in := service.VerifyOTPInput{
		Provider:   provider,
		Identifier: req.Identifier,
		Code:       req.Code,
		DeviceInfo: service.DeviceInfo{
			FCMToken:   req.FCMToken,
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			IPAddress:  &ip,
			Platform:   parsePlatform(req.Platform),
			OS:         req.OS,
		},
	}

	result, err := h.authService.VerifyOTP(r.Context(), in)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("verify otp failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, validationKey(err))
		return
	}

	result, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("refresh token failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := SessionIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	if err := h.authService.Logout(r.Context(), sessionID); err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("logout failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *Handler) VerifyTelegramLogin(w http.ResponseWriter, r *http.Request) {
	var req TelegramVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, validationKey(err))
		return
	}

	ip := clientIP(r)

	in := service.VerifyTelegramLoginInput{
		Code: req.Code,
		DeviceInfo: service.DeviceInfo{
			FCMToken:   req.FCMToken,
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			IPAddress:  &ip,
			Platform:   parsePlatform(req.Platform),
			OS:         req.OS,
		},
	}

	result, err := h.authService.VerifyTelegramLogin(r.Context(), in)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("verify telegram login failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

// VerifyTelegramWebApp — POST /auth/telegram/webapp: Mini App initData bilan
// kirish (Telegram.WebApp.initData, o'zgartirmasdan).
func (h *Handler) VerifyTelegramWebApp(w http.ResponseWriter, r *http.Request) {
	var req TelegramWebAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}
	if strings.TrimSpace(req.InitData) == "" {
		writeError(w, r, http.StatusBadRequest, msgInitDataRequired)
		return
	}

	ip := clientIP(r)
	result, err := h.authService.VerifyTelegramWebApp(r.Context(), service.VerifyTelegramWebAppInput{
		InitData: req.InitData,
		DeviceInfo: service.DeviceInfo{
			FCMToken:   req.FCMToken,
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			IPAddress:  &ip,
			Platform:   parsePlatform(req.Platform),
			OS:         req.OS,
		},
	})
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("verify telegram webapp failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}
	currentID, _ := SessionIDFromContext(r.Context())

	sessions, err := h.authService.ListSessions(r.Context(), userID)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("list sessions failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	out := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toSessionResponse(s, currentID))
	}

	writeJSON(w, http.StatusOK, SessionsResponse{Sessions: out})
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	sessionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || sessionID <= 0 {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidID)
		return
	}

	if err := h.authService.RevokeSession(r.Context(), userID, sessionID); err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("revoke session failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	if err := h.authService.LogoutAll(r.Context(), userID); err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("logout all failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out everywhere"})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
