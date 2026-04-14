package cmd

import (
	"context"
	"io"
	"os"

	"github.com/anishalle/hack/internal/config"
	"github.com/anishalle/hack/internal/envmanager"
	"github.com/anishalle/hack/internal/gcloud"
)

type authService interface {
	Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) (envmanager.AuthStatus, error)
	Status(ctx context.Context) (envmanager.AuthStatus, error)
	UseProject(ctx context.Context, project string) (envmanager.AuthStatus, error)
}

type envService interface {
	ListEnvironments(ctx context.Context, app string) ([]string, error)
	GetEnvironment(ctx context.Context, app, environment string) (envmanager.Environment, error)
	SetValue(ctx context.Context, app, environment, key, value string) (envmanager.Environment, error)
}

type dependencies struct {
	auth        authService
	env         envService
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	workingDir  func() (string, error)
	interactive bool
}

func defaultDependencies() *dependencies {
	store := config.NewJSONStore("")
	client := gcloud.NewService(gcloud.NewExecRunner("gcloud"))
	service := envmanager.NewService(store, client)

	return &dependencies{
		auth:        service,
		env:         service,
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		workingDir:  os.Getwd,
		interactive: isInteractive(os.Stdin),
	}
}

func isInteractive(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}
