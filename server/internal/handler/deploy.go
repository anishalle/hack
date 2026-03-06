package handler

import (
	"net/http"

	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
)

type DeployHandler struct {
	projectStore *store.ProjectStore
	auditStore   *store.AuditStore
}

func NewDeployHandler(projectStore *store.ProjectStore, auditStore *store.AuditStore) *DeployHandler {
	return &DeployHandler{projectStore: projectStore, auditStore: auditStore}
}

func (h *DeployHandler) Up(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	// TODO: Deploy via provider (Cloud Run, Compute Engine, etc.)
	// 1. Load hackfile.yaml deploy config for this env
	// 2. Build image if needed
	// 3. Push to Artifact Registry
	// 4. Deploy to target service

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "deploy:write",
		Resource: env,
		Details:  "deployed",
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"status":      "deploying",
	})
}

func (h *DeployHandler) Status(w http.ResponseWriter, r *http.Request) {
	env := chi.URLParam(r, "environment")

	// TODO: Query deploy provider for current status

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"status":      "running",
		"revision":    "unknown",
	})
}

func (h *DeployHandler) Logs(w http.ResponseWriter, r *http.Request) {
	env := chi.URLParam(r, "environment")

	// TODO: Stream logs from deploy provider
	// For Cloud Run, use Cloud Logging API

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"logs":        []string{},
	})
}

func (h *DeployHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	// TODO: Rollback via provider

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "deploy:rollback",
		Resource: env,
		Details:  "rolled back",
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"status":      "rolling_back",
	})
}

func (h *DeployHandler) Restart(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	// TODO: Restart via provider

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "deploy:restart",
		Resource: env,
		Details:  "restarted",
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"status":      "restarting",
	})
}
