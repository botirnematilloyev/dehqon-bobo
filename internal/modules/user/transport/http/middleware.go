package http

import (
	"context"
	"dehqon-bobo/internal/modules/user/domain"
	"log/slog"
	"net/http"
	"strings"
)

type TokenParser interface {
	ParseAccessToken(token string) (userID int64, sessionID int64, err error)
}

type SessionChecker interface {
	IsSessionRevoked(ctx context.Context, sessionID int64) (bool, error)
}

type contextKey struct {
	name string
}

type RoleReader interface {
	Role(ctx context.Context, userID int64) (domain.Role, error)
}

type Permission func(domain.Role) bool

var (
	userIDKey    = contextKey{"userID"}
	sessionIDKey = contextKey{"sessionID"}
)

func AuthMiddleware(parser TokenParser, sessions SessionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")

			if !ok || token == "" {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			userID, sessionID, err := parser.ParseAccessToken(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			revoked, err := sessions.IsSessionRevoked(r.Context(), sessionID)
			if err != nil {
				slog.Error("auth: session check failed", "error", err)
				writeError(w, http.StatusServiceUnavailable, "service unavailable")
				return
			}
			if revoked {
				writeError(w, http.StatusUnauthorized, "session is no longer valid")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, sessionIDKey, sessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware — sahifa hammaga ochiq, lekin kirgan
// foydalanuvchini taniydi.
//
// Yo'l xaritasi shunday sahifa: mehmon ham ko'radi, kirgan odam esa
// ustiga o'z progressini oladi.
//
// Sarlavha YO'Q bo'lsa — mehmon, xato emas. Sarlavha BOR, lekin token
// yaroqsiz bo'lsa — 401, garchi "e'tiborsiz qoldirib mehmon deb hisobla"
// ham ishlagan bo'lardi. Sabab mijoz tomonida: 401 ni ko'rgan mijoz
// tokenni yangilab qayta so'raydi, jimgina mehmon holatida esa u
// kirgan bo'la turib bo'sh progressni ko'rsatardi va buni hech kim
// xato deb tushunmasdi.
func OptionalAuthMiddleware(parser TokenParser, sessions SessionChecker) func(http.Handler) http.Handler {
	required := AuthMiddleware(parser, sessions)

	return func(next http.Handler) http.Handler {
		authenticated := required(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				next.ServeHTTP(w, r)
				return
			}
			authenticated.ServeHTTP(w, r)
		})
	}
}

func RequirePermission(roles RoleReader, allowed Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			role, err := roles.Role(r.Context(), userID)
			if err != nil {
				slog.Error("authz: role lookup failed", "error", err, "user_id", userID)
				writeError(w, http.StatusServiceUnavailable, "service unavailable")
				return
			}

			if !allowed(role) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

func SessionIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(sessionIDKey).(int64)
	return id, ok
}
