package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anishalle/hack/internal/cloud"
	"github.com/anishalle/hack/internal/envmanager"
)

func TestEnvImportWritesDotEnv(t *testing.T) {
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
	root.SetArgs([]string{"env", "import", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute import: %v", err)
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

func TestEnvImportRefusesOverwriteWithoutPrompt(t *testing.T) {
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
	root.SetArgs([]string{"env", "import", "prod"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvExportMergesDotEnv(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(target, []byte("OPENAI_API_KEY=\"new\"\nNEW_KEY=value\n"), 0o600); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	stub := &stubEnvService{environment: testEnvironment()}
	deps := &dependencies{
		auth:        &stubAuthService{},
		env:         stub,
		stdin:       bytes.NewBuffer(nil),
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		workingDir:  func() (string, error) { return tempDir, nil },
		interactive: false,
	}

	root := newRootCmd(deps)
	root.SetArgs([]string{"env", "export", ".env", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute export: %v", err)
	}

	if got := stub.environment.Values["OPENAI_API_KEY"]; got != "new" {
		t.Fatalf("expected overwritten value, got %q", got)
	}
	if got := stub.environment.Values["DATABASE_URL"]; got != "postgres://localhost/db" {
		t.Fatalf("expected untouched remote value, got %q", got)
	}
	if got := stub.environment.Values["NEW_KEY"]; got != "value" {
		t.Fatalf("expected added value, got %q", got)
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
	root.SetArgs([]string{"env", "load", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute load: %v", err)
	}

	expected := "export DATABASE_URL='postgres://localhost/db'\nexport OPENAI_API_KEY='secret'\n"
	if stdout.String() != expected {
		t.Fatalf("expected %q, got %q", expected, stdout.String())
	}
}

func TestEnvShowPrintsDotenv(t *testing.T) {
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
	root.SetArgs([]string{"env", "show", "prod"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute show: %v", err)
	}

	expected := "DATABASE_URL=\"postgres://localhost/db\"\nOPENAI_API_KEY=\"secret\"\n"
	if stdout.String() != expected {
		t.Fatalf("expected %q, got %q", expected, stdout.String())
	}
}

func TestEnvShowKeyPrintsValue(t *testing.T) {
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
	root.SetArgs([]string{"env", "show", "prod", "OPENAI_API_KEY"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute show key: %v", err)
	}

	if stdout.String() != "secret\n" {
		t.Fatalf("expected secret value, got %q", stdout.String())
	}
}

func TestEnvShowMissingVersionIsVerbose(t *testing.T) {
	t.Parallel()

	deps := &dependencies{
		auth:        &stubAuthService{},
		env:         &stubEnvService{environment: testEnvironment()},
		stdin:       bytes.NewBuffer(nil),
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		workingDir:  func() (string, error) { return t.TempDir(), nil },
		interactive: false,
	}

	root := newRootCmd(deps)
	root.SetArgs([]string{"env", "show"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Environment/version required") || !strings.Contains(err.Error(), "hack env show prod") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type stubAuthService struct{}

func (s *stubAuthService) ListProjects(ctx context.Context) ([]cloud.Project, error) {
	return []cloud.Project{{ID: "hack-project", Name: "Hack Project"}}, nil
}

func (s *stubAuthService) Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) (envmanager.AuthStatus, error) {
	return envmanager.AuthStatus{}, nil
}

func (s *stubAuthService) Logout(ctx context.Context) (envmanager.AuthStatus, error) {
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

func (s *stubEnvService) MergeValues(ctx context.Context, app, environment string, values map[string]string) (envmanager.Environment, error) {
	for key, value := range values {
		s.environment.Values[key] = value
	}
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
