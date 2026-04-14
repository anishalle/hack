package envmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/anishalle/hack/internal/config"
)

const secretPrefix = "hackutd"

var (
	scopePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]*$`)
	envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// SecretClient captures the gcloud operations the env manager needs.
type SecretClient interface {
	CheckInstalled(ctx context.Context) error
	Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error
	ActiveAccount(ctx context.Context) (string, error)
	CurrentProject(ctx context.Context) (string, error)
	SetProject(ctx context.Context, project string) error
	ListSecrets(ctx context.Context, project string) ([]string, error)
	SecretExists(ctx context.Context, project, secretID string) (bool, error)
	CreateSecret(ctx context.Context, project, secretID string) error
	AccessSecret(ctx context.Context, project, secretID, version string) ([]byte, error)
	AddSecretVersion(ctx context.Context, project, secretID string, payload []byte) error
}

// Service handles Hack's auth context and environment secrets.
type Service struct {
	store  config.Store
	client SecretClient
}

// AuthStatus is a combined view of Hack config and gcloud state.
type AuthStatus struct {
	ConfigPath      string
	StoredProject   string
	StoredAccount   string
	DetectedProject string
	DetectedAccount string
	GCloudAvailable bool
}

// Environment is a resolved app/environment secret payload.
type Environment struct {
	App      string
	Name     string
	Project  string
	SecretID string
	Values   map[string]string
}

func NewService(store config.Store, client SecretClient) *Service {
	return &Service{
		store:  store,
		client: client,
	}
}

func (e Environment) Keys() []string {
	keys := make([]string, 0, len(e.Values))
	for key := range e.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *Service) Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) (AuthStatus, error) {
	if err := s.client.CheckInstalled(ctx); err != nil {
		return AuthStatus{}, err
	}

	if err := s.client.Login(ctx, stdin, stdout, stderr); err != nil {
		return AuthStatus{}, err
	}

	cfg, err := s.store.Load()
	if err != nil {
		return AuthStatus{}, err
	}

	account, err := s.client.ActiveAccount(ctx)
	if err == nil {
		cfg.ActiveAccount = account
	}

	project, err := s.client.CurrentProject(ctx)
	if err == nil && project != "" {
		cfg.ActiveProject = project
	}

	if err := s.store.Save(cfg); err != nil {
		return AuthStatus{}, err
	}

	return s.Status(ctx)
}

func (s *Service) Status(ctx context.Context) (AuthStatus, error) {
	cfg, err := s.store.Load()
	if err != nil {
		return AuthStatus{}, err
	}

	status := AuthStatus{
		ConfigPath:    s.store.Path(),
		StoredProject: cfg.ActiveProject,
		StoredAccount: cfg.ActiveAccount,
	}

	if err := s.client.CheckInstalled(ctx); err != nil {
		return status, nil
	}

	status.GCloudAvailable = true

	account, err := s.client.ActiveAccount(ctx)
	if err == nil {
		status.DetectedAccount = account
	}

	project, err := s.client.CurrentProject(ctx)
	if err == nil {
		status.DetectedProject = project
	}

	return status, nil
}

func (s *Service) UseProject(ctx context.Context, project string) (AuthStatus, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return AuthStatus{}, fmt.Errorf("project id cannot be empty")
	}

	if err := s.client.CheckInstalled(ctx); err != nil {
		return AuthStatus{}, err
	}

	if err := s.client.SetProject(ctx, project); err != nil {
		return AuthStatus{}, err
	}

	cfg, err := s.store.Load()
	if err != nil {
		return AuthStatus{}, err
	}

	cfg.ActiveProject = project
	account, err := s.client.ActiveAccount(ctx)
	if err == nil && account != "" {
		cfg.ActiveAccount = account
	}

	if err := s.store.Save(cfg); err != nil {
		return AuthStatus{}, err
	}

	return s.Status(ctx)
}

func (s *Service) ListEnvironments(ctx context.Context, app string) ([]string, error) {
	app, err := normalizeScope("app", app)
	if err != nil {
		return nil, err
	}

	project, err := s.resolveProject(ctx)
	if err != nil {
		return nil, err
	}

	secretNames, err := s.client.ListSecrets(ctx, project)
	if err != nil {
		return nil, err
	}

	return environmentsFromSecretNames(app, secretNames), nil
}

func (s *Service) GetEnvironment(ctx context.Context, app, environment string) (Environment, error) {
	app, environment, project, secretID, err := s.resolveEnvironment(ctx, app, environment)
	if err != nil {
		return Environment{}, err
	}

	payload, err := s.client.AccessSecret(ctx, project, secretID, "latest")
	if err != nil {
		return Environment{}, err
	}

	values, err := ParseSecretPayload(payload)
	if err != nil {
		return Environment{}, err
	}

	return Environment{
		App:      app,
		Name:     environment,
		Project:  project,
		SecretID: secretID,
		Values:   values,
	}, nil
}

func (s *Service) SetValue(ctx context.Context, app, environment, key, value string) (Environment, error) {
	app, environment, project, secretID, err := s.resolveEnvironment(ctx, app, environment)
	if err != nil {
		return Environment{}, err
	}

	if err := ValidateKey(key); err != nil {
		return Environment{}, err
	}

	exists, err := s.client.SecretExists(ctx, project, secretID)
	if err != nil {
		return Environment{}, err
	}

	values := map[string]string{}
	if exists {
		payload, err := s.client.AccessSecret(ctx, project, secretID, "latest")
		if err != nil {
			return Environment{}, err
		}

		values, err = ParseSecretPayload(payload)
		if err != nil {
			return Environment{}, err
		}
	} else {
		if err := s.client.CreateSecret(ctx, project, secretID); err != nil {
			return Environment{}, err
		}
	}

	values[key] = value

	payload, err := MarshalPayload(values)
	if err != nil {
		return Environment{}, err
	}

	if err := s.client.AddSecretVersion(ctx, project, secretID, payload); err != nil {
		return Environment{}, err
	}

	return Environment{
		App:      app,
		Name:     environment,
		Project:  project,
		SecretID: secretID,
		Values:   values,
	}, nil
}

func (s *Service) resolveEnvironment(ctx context.Context, app, environment string) (string, string, string, string, error) {
	app, err := normalizeScope("app", app)
	if err != nil {
		return "", "", "", "", err
	}

	environment, err = normalizeScope("environment", environment)
	if err != nil {
		return "", "", "", "", err
	}

	project, err := s.resolveProject(ctx)
	if err != nil {
		return "", "", "", "", err
	}

	return app, environment, project, SecretID(app, environment), nil
}

func (s *Service) resolveProject(ctx context.Context) (string, error) {
	cfg, err := s.store.Load()
	if err != nil {
		return "", err
	}

	if cfg.ActiveProject != "" {
		return cfg.ActiveProject, nil
	}

	detectedProject := ""
	if err := s.client.CheckInstalled(ctx); err == nil {
		project, err := s.client.CurrentProject(ctx)
		if err == nil {
			detectedProject = project
		}
	}

	if detectedProject != "" {
		return "", fmt.Errorf("no active Hack project set; run `hack auth use %s`", detectedProject)
	}

	return "", fmt.Errorf("no active Hack project set; run `hack auth login` or `hack auth use <project-id>`")
}

func SecretID(app, environment string) string {
	return fmt.Sprintf("%s--%s--%s", secretPrefix, app, environment)
}

func NormalizeScope(label, value string) (string, error) {
	return normalizeScope(label, value)
}

func normalizeScope(label, value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "", fmt.Errorf("%s cannot be empty", label)
	}
	if !scopePattern.MatchString(normalized) {
		return "", fmt.Errorf("%s must match %s", label, scopePattern.String())
	}
	return normalized, nil
}

func ValidateKey(key string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("env key %q must match %s", key, envKeyPattern.String())
	}
	return nil
}

func ParseSecretPayload(payload []byte) (map[string]string, error) {
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(string(payload))))
	decoder.UseNumber()

	rawValues := map[string]any{}
	if err := decoder.Decode(&rawValues); err != nil {
		return nil, fmt.Errorf("secret payload must be valid JSON: %w", err)
	}

	values := make(map[string]string, len(rawValues))
	for key, raw := range rawValues {
		if err := ValidateKey(key); err != nil {
			return nil, err
		}

		switch typed := raw.(type) {
		case string:
			values[key] = typed
		case json.Number:
			values[key] = typed.String()
		case bool:
			if typed {
				values[key] = "true"
			} else {
				values[key] = "false"
			}
		case nil:
			values[key] = ""
		default:
			return nil, fmt.Errorf("env key %q must be a string, number, bool, or null", key)
		}
	}

	return values, nil
}

func MarshalPayload(values map[string]string) ([]byte, error) {
	return json.MarshalIndent(values, "", "  ")
}

func environmentsFromSecretNames(app string, names []string) []string {
	prefix := SecretID(app, "")
	seen := map[string]struct{}{}

	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		if slash := strings.LastIndex(name, "/"); slash >= 0 {
			name = name[slash+1:]
		}

		if !strings.HasPrefix(name, prefix) {
			continue
		}

		environment := strings.TrimPrefix(name, prefix)
		if environment != "" {
			seen[environment] = struct{}{}
		}
	}

	environments := make([]string, 0, len(seen))
	for environment := range seen {
		environments = append(environments, environment)
	}
	sort.Strings(environments)

	return environments
}
