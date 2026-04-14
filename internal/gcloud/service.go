package gcloud

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

var ErrNotInstalled = errors.New("gcloud is not installed or not on PATH")

// Request describes a gcloud command invocation.
type Request struct {
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Runner executes gcloud commands.
type Runner interface {
	Run(ctx context.Context, req Request) error
	Output(ctx context.Context, args ...string) ([]byte, error)
}

// CommandError wraps an execution failure with captured stderr.
type CommandError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *CommandError) Error() string {
	stderr := strings.TrimSpace(e.Stderr)
	if stderr == "" {
		return fmt.Sprintf("gcloud %s: %v", strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("gcloud %s: %s", strings.Join(e.Args, " "), stderr)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// ExecRunner runs gcloud through os/exec.
type ExecRunner struct {
	binary string
}

func NewExecRunner(binary string) *ExecRunner {
	if binary == "" {
		binary = "gcloud"
	}

	return &ExecRunner{binary: binary}
}

func (r *ExecRunner) Run(ctx context.Context, req Request) error {
	path, err := exec.LookPath(r.binary)
	if err != nil {
		return ErrNotInstalled
	}

	cmd := exec.CommandContext(ctx, path, req.Args...)
	cmd.Stdin = req.Stdin
	cmd.Stdout = req.Stdout

	var stderr bytes.Buffer
	if req.Stderr != nil {
		cmd.Stderr = io.MultiWriter(req.Stderr, &stderr)
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		return &CommandError{
			Args:   req.Args,
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	return nil
}

func (r *ExecRunner) Output(ctx context.Context, args ...string) ([]byte, error) {
	path, err := exec.LookPath(r.binary)
	if err != nil {
		return nil, ErrNotInstalled
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, &CommandError{
			Args:   args,
			Stderr: stderr.String(),
			Err:    err,
		}
	}

	return output, nil
}

// Service provides a small wrapper over gcloud's Secret Manager commands.
type Service struct {
	runner Runner
}

func NewService(runner Runner) *Service {
	return &Service{runner: runner}
}

func (s *Service) CheckInstalled(ctx context.Context) error {
	_, err := s.runner.Output(ctx, "--version")
	return err
}

func (s *Service) Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	return s.runner.Run(ctx, Request{
		Args:   []string{"auth", "login"},
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
}

func (s *Service) ActiveAccount(ctx context.Context) (string, error) {
	output, err := s.runner.Output(ctx, "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (s *Service) CurrentProject(ctx context.Context) (string, error) {
	output, err := s.runner.Output(ctx, "config", "get-value", "project")
	if err != nil {
		return "", err
	}

	project := strings.TrimSpace(string(output))
	if project == "" || project == "(unset)" {
		return "", nil
	}

	return project, nil
}

func (s *Service) SetProject(ctx context.Context, project string) error {
	return s.runner.Run(ctx, Request{
		Args: []string{"config", "set", "project", project},
	})
}

func (s *Service) ListSecrets(ctx context.Context, project string) ([]string, error) {
	output, err := s.runner.Output(ctx, "secrets", "list", "--project", project, "--format=value(name)")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	var names []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names, nil
}

func (s *Service) SecretExists(ctx context.Context, project, secretID string) (bool, error) {
	_, err := s.runner.Output(ctx, "secrets", "describe", secretID, "--project", project, "--format=value(name)")
	if err == nil {
		return true, nil
	}
	if IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (s *Service) CreateSecret(ctx context.Context, project, secretID string) error {
	return s.runner.Run(ctx, Request{
		Args: []string{"secrets", "create", secretID, "--project", project, "--replication-policy=automatic"},
	})
}

func (s *Service) AccessSecret(ctx context.Context, project, secretID, version string) ([]byte, error) {
	return s.runner.Output(ctx, "secrets", "versions", "access", version, "--secret", secretID, "--project", project)
}

func (s *Service) AddSecretVersion(ctx context.Context, project, secretID string, payload []byte) error {
	return s.runner.Run(ctx, Request{
		Args:  []string{"secrets", "versions", "add", secretID, "--project", project, "--data-file=-"},
		Stdin: bytes.NewReader(payload),
	})
}

func IsNotFound(err error) bool {
	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		return false
	}

	message := strings.ToLower(commandErr.Stderr)
	return strings.Contains(message, "not found") || strings.Contains(message, "not exist")
}
