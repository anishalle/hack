package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anishalle/hack/internal/envmanager"
)

func TestEnvExportWritesDotEnv(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &dependencies{
		auth:        &stubAuthService{},
		env:         &stubEnvService{environment: testEnvironment()},
		stdin:       bytes.NewBuffer(nil),
		stdout:      stdout,
		stderr:      stderr,
		workingDir:  func() (string, error) { return tempDir, nil },
		interactive: false,
	}

	root := newRootCmd(deps)
	root.SetArgs([]string{"env", "export", "api", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute export: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	expected := "DATABASE_URL=\"postgres://localhost/db\"\nOPENAI_API_KEY=\"secret\"\n"
	if string(content) != expected {
		t.Fatalf("expected %q, got %q", expected, string(content))
	}
}

func TestEnvExportRefusesOverwriteWithoutPrompt(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(target, []byte("EXISTING=1\n"), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	deps := &dependencies{
		auth:        &stubAuthService{},
		env:         &stubEnvService{environment: testEnvironment()},
		stdin:       bytes.NewBuffer(nil),
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		workingDir:  func() (string, error) { return tempDir, nil },
		interactive: false,
	}

	root := newRootCmd(deps)
	root.SetArgs([]string{"env", "export", "api", "prod"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvLoadPrintsShellExportsOnly(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	deps := &dependencies{
		auth:        &stubAuthService{},
		env:         &stubEnvService{environment: testEnvironment()},
		stdin:       bytes.NewBuffer(nil),
		stdout:      stdout,
		stderr:      &bytes.Buffer{},
		workingDir:  func() (string, error) { return t.TempDir(), nil },
		interactive: false,
	}

	root := newRootCmd(deps)
	root.SetArgs([]string{"env", "load", "api", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute load: %v", err)
	}

	expected := "export DATABASE_URL='postgres://localhost/db'\nexport OPENAI_API_KEY='secret'\n"
	if stdout.String() != expected {
		t.Fatalf("expected %q, got %q", expected, stdout.String())
	}
}

type stubAuthService struct{}

func (s *stubAuthService) Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) (envmanager.AuthStatus, error) {
	return envmanager.AuthStatus{}, nil
}

func (s *stubAuthService) Status(ctx context.Context) (envmanager.AuthStatus, error) {
	return envmanager.AuthStatus{}, nil
}

func (s *stubAuthService) UseProject(ctx context.Context, project string) (envmanager.AuthStatus, error) {
	return envmanager.AuthStatus{StoredProject: project}, nil
}

type stubEnvService struct {
	environment envmanager.Environment
}

func (s *stubEnvService) ListEnvironments(ctx context.Context, app string) ([]string, error) {
	return []string{"prod"}, nil
}

func (s *stubEnvService) GetEnvironment(ctx context.Context, app, environment string) (envmanager.Environment, error) {
	return s.environment, nil
}

func (s *stubEnvService) SetValue(ctx context.Context, app, environment, key, value string) (envmanager.Environment, error) {
	s.environment.Values[key] = value
	return s.environment, nil
}

func testEnvironment() envmanager.Environment {
	return envmanager.Environment{
		App:      "api",
		Name:     "prod",
		Project:  "hack-project",
		SecretID: "hackutd--api--prod",
		Values: map[string]string{
			"DATABASE_URL":   "postgres://localhost/db",
			"OPENAI_API_KEY": "secret",
		},
	}
}
