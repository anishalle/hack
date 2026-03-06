package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anishalle/hack/server/internal/handler"
	"github.com/anishalle/hack/server/internal/middleware"
	"github.com/anishalle/hack/server/internal/provider/secrets"
	"github.com/anishalle/hack/server/internal/store"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject == "" {
		slog.Warn("GCP_PROJECT not set, Firestore will use default project")
	}

	ctx := context.Background()

	firestore, err := store.NewFirestoreClient(ctx, gcpProject)
	if err != nil {
		slog.Error("failed to initialize Firestore", "error", err)
		os.Exit(1)
	}
	defer firestore.Close()

	userStore := store.NewUserStore(firestore)
	projectStore := store.NewProjectStore(firestore)
	auditStore := store.NewAuditStore(firestore)

	gsm, err := secrets.NewGSMProvider(ctx)
	if err != nil {
		slog.Warn("failed to initialize Secret Manager (env commands will not work)", "error", err)
	}
	if gsm != nil {
		defer gsm.Close()
	}

	authMiddleware := middleware.NewAuthMiddleware(userStore)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.StructuredLogger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"dev"}`)
	})

	authHandler := handler.NewAuthHandler(userStore)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/device-code", authHandler.DeviceCode)
		r.Get("/callback", authHandler.Callback)
		r.Post("/token", authHandler.Token)
		r.Post("/refresh", authHandler.Refresh)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)

		r.Get("/me", authHandler.Me)

		envHandler := handler.NewEnvHandler(projectStore, auditStore, gsm)
		r.Route("/projects/{project}/env/{environment}", func(r chi.Router) {
			r.Use(middleware.RequirePermission("env:read"))
			r.Get("/", envHandler.Pull)
			r.Get("/list", envHandler.List)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("env:write"))
				r.Put("/", envHandler.Push)
				r.Post("/set", envHandler.Set)
				r.Post("/unset", envHandler.Unset)
			})
		})
		r.Get("/projects/{project}/env/diff/{env1}/{env2}", envHandler.Diff)

		projectHandler := handler.NewProjectHandler(projectStore, userStore)
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)
			r.Get("/{project}", projectHandler.Get)
		})

		deployHandler := handler.NewDeployHandler(projectStore, auditStore)
		r.Route("/projects/{project}/deploy/{environment}", func(r chi.Router) {
			r.Use(middleware.RequirePermission("deploy:read"))
			r.Get("/status", deployHandler.Status)
			r.Get("/logs", deployHandler.Logs)

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("deploy:write"))
				r.Post("/up", deployHandler.Up)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("deploy:rollback"))
				r.Post("/rollback", deployHandler.Rollback)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("deploy:restart"))
				r.Post("/restart", deployHandler.Restart)
			})
		})

		adminHandler := handler.NewAdminHandler(userStore, auditStore)
		r.Route("/projects/{project}/admin", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("admin:users"))
				r.Get("/users", adminHandler.ListUsers)
				r.Post("/users", adminHandler.AddUser)
				r.Delete("/users/{email}", adminHandler.RemoveUser)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("admin:roles"))
				r.Get("/roles", adminHandler.ListRoles)
				r.Post("/roles/assign", adminHandler.AssignRole)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission("admin:audit"))
				r.Get("/audit", adminHandler.AuditLog)
			})
		})
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	slog.Info("server stopped")
}
