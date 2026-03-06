package secrets

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GSMProvider struct {
	client *secretmanager.Client
}

func NewGSMProvider(ctx context.Context) (*GSMProvider, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Secret Manager client: %w", err)
	}
	return &GSMProvider{client: client}, nil
}

func (p *GSMProvider) Close() error {
	return p.client.Close()
}

func (p *GSMProvider) secretName(gcpProject, prefix, env, key string) string {
	return fmt.Sprintf("projects/%s/secrets/%s_%s_%s", gcpProject, prefix, env, key)
}

func (p *GSMProvider) secretID(prefix, env, key string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, env, key)
}

func (p *GSMProvider) GetAll(ctx context.Context, gcpProject, prefix, env string) (map[string]string, error) {
	parent := fmt.Sprintf("projects/%s", gcpProject)
	filterPrefix := fmt.Sprintf("%s_%s_", prefix, env)

	it := p.client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	})

	vars := make(map[string]string)

	for {
		secret, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		name := lastSegment(secret.Name)
		if !strings.HasPrefix(name, filterPrefix) {
			continue
		}

		key := strings.TrimPrefix(name, filterPrefix)

		versionName := secret.Name + "/versions/latest"
		result, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
			Name: versionName,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return nil, fmt.Errorf("failed to access secret %s: %w", key, err)
		}

		vars[key] = string(result.Payload.Data)
	}

	return vars, nil
}

func (p *GSMProvider) GetKeys(ctx context.Context, gcpProject, prefix, env string) ([]string, error) {
	parent := fmt.Sprintf("projects/%s", gcpProject)
	filterPrefix := fmt.Sprintf("%s_%s_", prefix, env)

	it := p.client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		Parent: parent,
	})

	var keys []string

	for {
		secret, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		name := lastSegment(secret.Name)
		if !strings.HasPrefix(name, filterPrefix) {
			continue
		}

		key := strings.TrimPrefix(name, filterPrefix)
		keys = append(keys, key)
	}

	return keys, nil
}

func (p *GSMProvider) Set(ctx context.Context, gcpProject, prefix, env, key, value string) error {
	secretID := p.secretID(prefix, env, key)
	secretName := p.secretName(gcpProject, prefix, env, key)

	_, err := p.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretName,
	})

	if status.Code(err) == codes.NotFound {
		_, err = p.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", gcpProject),
			SecretId: secretID,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create secret %s: %w", key, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check secret %s: %w", key, err)
	}

	_, err = p.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set secret %s: %w", key, err)
	}

	return nil
}

func (p *GSMProvider) SetBulk(ctx context.Context, gcpProject, prefix, env string, vars map[string]string) error {
	for key, value := range vars {
		if err := p.Set(ctx, gcpProject, prefix, env, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (p *GSMProvider) Delete(ctx context.Context, gcpProject, prefix, env, key string) error {
	secretName := p.secretName(gcpProject, prefix, env, key)

	err := p.client.DeleteSecret(ctx, &secretmanagerpb.DeleteSecretRequest{
		Name: secretName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("failed to delete secret %s: %w", key, err)
	}

	return nil
}

func (p *GSMProvider) Diff(ctx context.Context, gcpProject, prefix, env1, env2 string) (added, removed, changed map[string]string, changedPairs map[string][2]string, err error) {
	vars1, err := p.GetAll(ctx, gcpProject, prefix, env1)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get %s env: %w", env1, err)
	}

	vars2, err := p.GetAll(ctx, gcpProject, prefix, env2)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get %s env: %w", env2, err)
	}

	added = make(map[string]string)
	removed = make(map[string]string)
	changedPairs = make(map[string][2]string)

	for k, v := range vars2 {
		if _, ok := vars1[k]; !ok {
			added[k] = v
		}
	}

	for k, v := range vars1 {
		if _, ok := vars2[k]; !ok {
			removed[k] = v
		}
	}

	for k, v1 := range vars1 {
		if v2, ok := vars2[k]; ok && v1 != v2 {
			changedPairs[k] = [2]string{v1, v2}
		}
	}

	return added, removed, nil, changedPairs, nil
}

func lastSegment(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}
