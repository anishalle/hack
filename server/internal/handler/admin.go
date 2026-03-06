package handler

import (
	"encoding/json"
	"net/http"

	"github.com/anishalle/hack/internal/models"
	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	userStore  *store.UserStore
	auditStore *store.AuditStore
}

func NewAdminHandler(userStore *store.UserStore, auditStore *store.AuditStore) *AdminHandler {
	return &AdminHandler{userStore: userStore, auditStore: auditStore}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")

	users, err := h.userStore.ListByProject(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	type userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}

	result := make([]userInfo, 0, len(users))
	for _, u := range users {
		result = append(result, userInfo{
			Email: u.Email,
			Name:  u.Name,
			Role:  u.RoleIn(project),
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"users": result})
}

func (h *AdminHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	actor := middleware.UserFromContext(r.Context())

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role == "" {
		req.Role = "developer"
	}

	if _, ok := models.DefaultRoles[req.Role]; !ok {
		respondError(w, http.StatusBadRequest, "invalid role: "+req.Role)
		return
	}

	user, err := h.userStore.GetByEmail(r.Context(), req.Email)
	if err != nil {
		user = &models.User{
			Email:    req.Email,
			Projects: map[string]string{project: req.Role},
		}
		if err := h.userStore.Create(r.Context(), user); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create user")
			return
		}
	} else {
		user.Projects[project] = req.Role
		if err := h.userStore.Update(r.Context(), user); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update user")
			return
		}
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     actor.Email,
		Action:   "admin:users",
		Resource: req.Email,
		Details:  "added user with role " + req.Role,
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"email": req.Email,
		"role":  req.Role,
	})
}

func (h *AdminHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	email := chi.URLParam(r, "email")
	actor := middleware.UserFromContext(r.Context())

	user, err := h.userStore.GetByEmail(r.Context(), email)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	delete(user.Projects, project)
	if err := h.userStore.Update(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     actor.Email,
		Action:   "admin:users",
		Resource: email,
		Details:  "removed user",
	})

	respondJSON(w, http.StatusOK, map[string]any{"removed": email})
}

func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	type roleInfo struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		BuiltIn     bool     `json:"built_in"`
	}

	roles := make([]roleInfo, 0, len(models.DefaultRoles))
	for _, role := range models.DefaultRoles {
		perms := make([]string, len(role.Permissions))
		for i, p := range role.Permissions {
			perms[i] = string(p)
		}
		roles = append(roles, roleInfo{
			Name:        role.Name,
			Permissions: perms,
			BuiltIn:     role.BuiltIn,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

func (h *AdminHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	actor := middleware.UserFromContext(r.Context())

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, ok := models.DefaultRoles[req.Role]; !ok {
		respondError(w, http.StatusBadRequest, "invalid role: "+req.Role)
		return
	}

	user, err := h.userStore.GetByEmail(r.Context(), req.Email)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	user.Projects[project] = req.Role
	if err := h.userStore.Update(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update user role")
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     actor.Email,
		Action:   "admin:roles",
		Resource: req.Email,
		Details:  "assigned role " + req.Role,
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"email": req.Email,
		"role":  req.Role,
	})
}

func (h *AdminHandler) AuditLog(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")

	entries, err := h.auditStore.Query(r.Context(), project, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch audit log")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
