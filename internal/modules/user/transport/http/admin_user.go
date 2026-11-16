package http

import (
	"context"
	"dehqon-bobo/internal/platform/i18n"
	"log/slog"
	"net/http"
	"strconv"

	"dehqon-bobo/internal/modules/user/domain"

	"github.com/go-chi/chi/v5"
)

type AdminUserService interface {
	List(ctx context.Context, f domain.UserListFilter) (*domain.UserListPage, error)
	Get(ctx context.Context, id int64) (*domain.UserDetail, error)
}

// AdminUserHandler — `/admin/users`. Faqat `RequireUserManager` ortida.
type AdminUserHandler struct {
	users AdminUserService
}

func NewAdminUserHandler(users AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{users: users}
}

const (
	adminUsersDefaultPerPage = 50
	adminUsersMaxPerPage     = 200
)

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := min(queryInt(r, "per_page", adminUsersDefaultPerPage), adminUsersMaxPerPage)

	res, err := h.users.List(r.Context(), domain.UserListFilter{
		Query:  r.URL.Query().Get("q"),
		Limit:  int32(perPage),
		Offset: int32((page - 1) * perPage),
	})
	if err != nil {
		slog.Error("admin: list users failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, i18n.ErrInternal)
		return
	}

	items := make([]AdminUserSummaryResponse, 0, len(res.Items))
	for _, u := range res.Items {
		items = append(items, toAdminUserSummaryResponse(u))
	}

	writeJSON(w, http.StatusOK, AdminUserListResponse{
		Users:   items,
		Total:   res.Total,
		Page:    page,
		PerPage: perPage,
	})
}

func (h *AdminUserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, http.StatusNotFound, msgUserNotFound)
		return
	}

	detail, err := h.users.Get(r.Context(), id)
	if err != nil {
		status, msg := statusFromError(err)
		if status == http.StatusInternalServerError {
			slog.Error("admin: get user failed", "error", err, "user_id", id)
		}
		writeError(w, r, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, toAdminUserDetailResponse(detail))
}

// queryInt — musbat butun son; yo'q yoki yaroqsiz bo'lsa `fallback`.
func queryInt(r *http.Request, key string, fallback int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v < 1 {
		return fallback
	}
	return v
}

type AdminUserListResponse struct {
	Users   []AdminUserSummaryResponse `json:"users"`
	Total   int                        `json:"total"`
	Page    int                        `json:"page"`
	PerPage int                        `json:"per_page"`
}

type AdminUserSummaryResponse struct {
	ID        int64                   `json:"id"`
	Username  string                  `json:"username"`
	FullName  *string                 `json:"full_name"`
	Role      string                  `json:"role"`
	Region    *LocationNameResponse   `json:"region"`
	Accounts  []LinkedAccountResponse `json:"accounts"`
	CreatedAt int64                   `json:"created_at"`

	Farms        int    `json:"farms"`
	LastActiveAt *int64 `json:"last_active_at"`
}

type AdminUserDetailResponse struct {
	ID        int64                   `json:"id"`
	Username  string                  `json:"username"`
	FullName  *string                 `json:"full_name"`
	Role      string                  `json:"role"`
	Region    *LocationNameResponse   `json:"region"`
	District  *LocationNameResponse   `json:"district"`
	Accounts  []LinkedAccountResponse `json:"accounts"`
	Sessions  []SessionResponse       `json:"sessions"`
	CreatedAt int64                   `json:"created_at"`
	UpdatedAt int64                   `json:"updated_at"`
}

type LocationNameResponse struct {
	ID   int32             `json:"id"`
	Name map[string]string `json:"name"`
}

type LinkedAccountResponse struct {
	ID         int64          `json:"id"`
	Provider   string         `json:"provider"`
	Identifier string         `json:"identifier"`
	IsVerified bool           `json:"is_verified"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  int64          `json:"created_at"`
}

func toAdminUserSummaryResponse(u domain.UserSummary) AdminUserSummaryResponse {
	res := AdminUserSummaryResponse{
		ID:           u.ID,
		Username:     u.Username,
		FullName:     u.FullName,
		Role:         roleName(u.Role),
		Accounts:     toLinkedAccountResponses(u.Accounts),
		CreatedAt:    u.CreatedAt,
		Farms:        u.Farms,
		LastActiveAt: u.LastActiveAt,
	}
	if u.Region != nil {
		res.Region = &LocationNameResponse{ID: u.Region.ID, Name: u.Region.Name}
	}
	return res
}

func toAdminUserDetailResponse(d *domain.UserDetail) AdminUserDetailResponse {
	res := AdminUserDetailResponse{
		ID:        d.ID,
		Username:  d.Username,
		FullName:  d.FullName,
		Role:      roleName(d.Role),
		Accounts:  toLinkedAccountResponses(d.Accounts),
		Sessions:  make([]SessionResponse, 0, len(d.Sessions)),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if d.Region != nil {
		res.Region = &LocationNameResponse{ID: d.Region.ID, Name: d.Region.Name}
	}
	if d.District != nil {
		res.District = &LocationNameResponse{ID: d.District.ID, Name: d.District.Name}
	}
	// Admin boshqa odamning sessiyasini ko'radi — "joriy" sessiya yo'q.
	for _, s := range d.Sessions {
		res.Sessions = append(res.Sessions, toSessionResponse(s, 0))
	}
	return res
}

func toLinkedAccountResponses(accounts []domain.LinkedAccount) []LinkedAccountResponse {
	out := make([]LinkedAccountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, LinkedAccountResponse{
			ID:         a.ID,
			Provider:   providerName(a.Provider),
			Identifier: a.Identifier,
			IsVerified: a.IsVerified,
			Metadata:   a.Metadata,
			CreatedAt:  a.CreatedAt,
		})
	}
	return out
}

func providerName(p domain.AuthProvider) string {
	switch p {
	case domain.ProviderPhone:
		return "phone"
	case domain.ProviderEmail:
		return "email"
	case domain.ProviderTelegram:
		return "telegram"
	default:
		return "unknown"
	}
}
