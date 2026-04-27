package envmanager

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/anishalle/hack/internal/cloud"
	"github.com/anishalle/hack/internal/config"
	"github.com/anishalle/hack/internal/gcloud"
)

func TestListEnvironmentsFiltersSecretNames(t *testing.T) {
	t.Parallel()

	store := config.NewJSONStore(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{ActiveProject: "hack-project"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	client := &fakeSecretClient{
		secrets: []string{
			"hackutd--api--dev",
			"projects/hack-project/secrets/hackutd--api--prod",
			"hackutd--site--prod",
		},
	}

	service := NewService(store, client)
	environments, err := service.ListEnvironments(context.Background(), "api")
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}

	expected := []string{"dev", "prod"}
	if strings.Join(environments, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, got %v", expected, environments)
	}
}

func TestResolveProjectSuggestsDetectedProject(t *testing.T) {
	t.Parallel()

	store := config.NewJSONStore(t.TempDir() + "/config.json")
	client := &fakeSecretClient{
		project: "detected-project",
	}

	service := NewService(store, client)
	_, err := service.ListEnvironments(context.Background(), "api")
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "hack auth use detected-project") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetValueCreatesSecretWhenMissing(t *testing.T) {
	t.Parallel()

	store := config.NewJSONStore(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{ActiveProject: "hack-project"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	client := &fakeSecretClient{}
	service := NewService(store, client)

	environment, err := service.SetValue(context.Background(), "api", "prod", "OPENAI_API_KEY", "secret")
	if err != nil {
		t.Fatalf("set value: %v", err)
	}

	if !client.created["hackutd--api--prod"] {
		t.Fatalf("expected secret creation")
	}
	if got := client.latest["hackutd--api--prod"]["OPENAI_API_KEY"]; got != "secret" {
		t.Fatalf("expected stored secret value, got %q", got)
	}
	if environment.Project != "hack-project" {
		t.Fatalf("expected project hack-project, got %q", environment.Project)
	}
}

func TestParseSecretPayloadSupportsScalars(t *testing.T) {
	t.Parallel()

	values, err := ParseSecretPayload([]byte(`{"STRING":"value","NUMBER":7,"BOOL":true,"EMPTY":null}`))
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}

	if values["STRING"] != "value" || values["NUMBER"] != "7" || values["BOOL"] != "true" || values["EMPTY"] != "" {
		t.Fatalf("unexpected parsed values: %#v", values)
	}
}

type fakeSecretClient struct {
	checkInstalledErr error
	account           string
	project           string
	secrets           []string
	latest            map[string]map[string]string
	created           map[string]bool
}

func (f *fakeSecretClient) CheckInstalled(ctx context.Context) error {
	return f.checkInstalledErr
}

func (f *fakeSecretClient) Login(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	return nil
}

func (f *fakeSecretClient) Revoke(ctx context.Context, account string) error {
	return nil
}

func (f *fakeSecretClient) ActiveAccount(ctx context.Context) (string, error) {
	return f.account, nil
}

func (f *fakeSecretClient) CurrentProject(ctx context.Context) (string, error) {
	return f.project, nil
}

func (f *fakeSecretClient) SetProject(ctx context.Context, project string) error {
	f.project = project
	return nil
}

func (f *fakeSecretClient) ListProjects(ctx context.Context) ([]cloud.Project, error) {
	return []cloud.Project{{ID: "hack-project", Name: "Hack Project"}}, nil
}

func (f *fakeSecretClient) ListSecrets(ctx context.Context, project string) ([]string, error) {
	return append([]string(nil), f.secrets...), nil
}

func (f *fakeSecretClient) SecretExists(ctx context.Context, project, secretID string) (bool, error) {
	if f.latest == nil {
		return false, nil
	}
	_, ok := f.latest[secretID]
	return ok, nil
}

func (f *fakeSecretClient) CreateSecret(ctx context.Context, project, secretID string) error {
	if f.created == nil {
		f.created = map[string]bool{}
	}
	f.created[secretID] = true
	return nil
}

func (f *fakeSecretClient) AccessSecret(ctx context.Context, project, secretID, version string) ([]byte, error) {
	if f.latest == nil {
		return nil, &gcloud.CommandError{Err: errors.New("missing"), Stderr: "not found"}
	}

	values, ok := f.latest[secretID]
	if !ok {
		return nil, &gcloud.CommandError{Err: errors.New("missing"), Stderr: "not found"}
	}

	return MarshalPayload(values)
}

func (f *fakeSecretClient) AddSecretVersion(ctx context.Context, project, secretID string, payload []byte) error {
	if f.latest == nil {
		f.latest = map[string]map[string]string{}
	}

	values, err := ParseSecretPayload(payload)
	if err != nil {
		return err
	}

	f.latest[secretID] = values
	return nil
}
