package http

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"dehqon-bobo/internal/modules/user/service"
	"dehqon-bobo/internal/platform/i18n"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type ProfileService interface {
	Get(ctx context.Context, userID int64) (*domain.User, error)
	Update(ctx context.Context, userID int64, in service.UpdateProfileInput) (*domain.User, error)
}

type ProfileHandler struct {
	profiles ProfileService
}

func NewProfileHandler(profiles ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	user, err := h.profiles.Get(r.Context(), userID)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("get profile failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toProfileResponse(user))
}

func (h *ProfileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, i18n.ErrUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, i18n.ErrInvalidJSON)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, validationKey(err))
		return
	}

	in := service.UpdateProfileInput{
		// `Username` xom holda o'tadi: normalizatsiya (kichik harf,
		// kesish) va tekshiruv domenda, chunki ular API shakli emas,
		// mahsulot qoidasi — va boshqa transport (masalan bot) ham
		// aynan shu qoidaga bo'ysunishi kerak.
		Username:   req.Username,
		Locale:     req.Locale,
		RegionID:   req.RegionID,
		DistrictID: req.DistrictID,
	}
	if req.FullName != nil {
		trimmed := strings.TrimSpace(*req.FullName)
		in.FullName = &trimmed
	}

	user, err := h.profiles.Update(r.Context(), userID, in)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("update profile failed", "error", err)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toProfileResponse(user))
}
