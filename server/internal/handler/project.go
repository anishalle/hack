package handler

import (
	"encoding/json"
	"net/http"

	"github.com/anishalle/hack/internal/models"
	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
)

type ProjectHandler struct {
	projectStore *store.ProjectStore
	userStore    *store.UserStore
}

func NewProjectHandler(projectStore *store.ProjectStore, userStore *store.UserStore) *ProjectHandler {
	return &ProjectHandler{projectStore: projectStore, userStore: userStore}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	type projectInfo struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}

	projects := make([]projectInfo, 0)
	for name, role := range user.Projects {
		projects = append(projects, projectInfo{Name: name, Role: role})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
	})
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	user := middleware.UserFromContext(r.Context())

	if !user.HasProject(name) {
		respondError(w, http.StatusForbidden, "you don't have access to this project")
		return
	}

	project, err := h.projectStore.Get(r.Context(), name)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	respondJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		GCPProject   string   `json:"gcp_project"`
		Environments []string `json:"environments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "project name is required")
		return
	}

	project := &models.Project{
		Name:         req.Name,
		Description:  req.Description,
		GCPProject:   req.GCPProject,
		Environments: req.Environments,
		CreatedBy:    user.Email,
	}

	if err := h.projectStore.Create(r.Context(), project); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	user.Projects[req.Name] = "owner"
	if err := h.userStore.Update(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "project created but failed to set ownership")
		return
	}

	respondJSON(w, http.StatusCreated, project)
}
