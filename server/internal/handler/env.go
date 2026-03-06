package handler

import (
	"encoding/json"
	"net/http"

	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/provider/secrets"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
)

type EnvHandler struct {
	projectStore *store.ProjectStore
	auditStore   *store.AuditStore
	gsm          *secrets.GSMProvider
}

func NewEnvHandler(projectStore *store.ProjectStore, auditStore *store.AuditStore, gsm *secrets.GSMProvider) *EnvHandler {
	return &EnvHandler{projectStore: projectStore, auditStore: auditStore, gsm: gsm}
}

func (h *EnvHandler) Pull(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	vars, err := h.gsm.GetAll(r.Context(), proj.GCPProject, project, env)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch environment variables: "+err.Error())
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "env:read",
		Resource: env,
		Details:  "pulled environment variables",
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"variables":   vars,
		"count":       len(vars),
	})
}

func (h *EnvHandler) List(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	keys, err := h.gsm.GetKeys(r.Context(), proj.GCPProject, project, env)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list keys: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"keys":        keys,
		"count":       len(keys),
	})
}

func (h *EnvHandler) Push(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Variables map[string]string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := h.gsm.SetBulk(r.Context(), proj.GCPProject, project, env, req.Variables); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to push variables: "+err.Error())
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "env:write",
		Resource: env,
		Details:  "pushed environment variables",
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"updated":     len(req.Variables),
	})
}

func (h *EnvHandler) Set(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := h.gsm.Set(r.Context(), proj.GCPProject, project, env, req.Key, req.Value); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to set variable: "+err.Error())
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "env:write",
		Resource: env,
		Details:  "set " + req.Key,
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"key":         req.Key,
		"environment": env,
	})
}

func (h *EnvHandler) Unset(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env := chi.URLParam(r, "environment")
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := h.gsm.Delete(r.Context(), proj.GCPProject, project, env, req.Key); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete variable: "+err.Error())
		return
	}

	h.auditStore.Log(r.Context(), &store.AuditEntry{
		Project:  project,
		User:     user.Email,
		Action:   "env:write",
		Resource: env,
		Details:  "unset " + req.Key,
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"key":         req.Key,
		"environment": env,
		"removed":     true,
	})
}

func (h *EnvHandler) Diff(w http.ResponseWriter, r *http.Request) {
	project := chi.URLParam(r, "project")
	env1 := chi.URLParam(r, "env1")
	env2 := chi.URLParam(r, "env2")

	proj, err := h.projectStore.Get(r.Context(), project)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	added, removed, _, changed, err := h.gsm.Diff(r.Context(), proj.GCPProject, project, env1, env2)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to diff environments: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"env1":    env1,
		"env2":    env2,
		"added":   added,
		"removed": removed,
		"changed": changed,
	})
}
