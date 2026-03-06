package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anishalle/hack/internal/models"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
)

type contextKey string

const (
	UserContextKey    contextKey = "user"
	ProjectContextKey contextKey = "project"
)

type AuthMiddleware struct {
	userStore *store.UserStore
}

func NewAuthMiddleware(userStore *store.UserStore) *AuthMiddleware {
	return &AuthMiddleware{userStore: userStore}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		claims, err := ValidateToken(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		user, err := m.userStore.GetByEmail(r.Context(), claims.Email)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "user not found")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			project := chi.URLParam(r, "project")
			if project == "" {
				writeError(w, http.StatusBadRequest, "project not specified")
				return
			}

			role := user.RoleIn(project)
			if role == "" {
				writeError(w, http.StatusForbidden, "you don't have access to this project")
				return
			}

			if !models.HasPermission(role, models.Permission(perm)) {
				writeError(w, http.StatusForbidden, "insufficient permissions: requires "+perm)
				return
			}

			ctx := context.WithValue(r.Context(), ProjectContextKey, project)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(UserContextKey).(*models.User)
	return user
}

func ProjectFromContext(ctx context.Context) string {
	project, _ := ctx.Value(ProjectContextKey).(string)
	return project
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
