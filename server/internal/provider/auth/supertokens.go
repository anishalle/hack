package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anishalle/hack/server/internal/provider"
)

type SuperTokensProvider struct {
	connectionURI string
	apiKey        string
	client        *http.Client
}

func NewSuperTokensProvider(connectionURI, apiKey string) *SuperTokensProvider {
	return &SuperTokensProvider{
		connectionURI: connectionURI,
		apiKey:        apiKey,
		client:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *SuperTokensProvider) ListUsers(ctx context.Context, opts provider.AuthListOptions) ([]provider.AuthUser, error) {
	path := "/recipe/users"
	params := fmt.Sprintf("?limit=%d", opts.Limit)
	if opts.Search != "" {
		params += "&email=" + opts.Search
	}

	var resp struct {
		Users []struct {
			User struct {
				ID        string `json:"id"`
				Email     string `json:"email"`
				TimeJoined int64 `json:"timeJoined"`
			} `json:"user"`
		} `json:"users"`
	}

	if err := p.get(ctx, path+params, &resp); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]provider.AuthUser, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = provider.AuthUser{
			ID:        u.User.ID,
			Email:     u.User.Email,
			CreatedAt: time.Unix(u.User.TimeJoined/1000, 0).Format(time.RFC3339),
		}
	}

	return users, nil
}

func (p *SuperTokensProvider) GetConfig(ctx context.Context) (map[string]any, error) {
	path := "/recipe/dashboard/user/sessions/count"

	var sessionCount struct {
		Count int `json:"count"`
	}
	p.get(ctx, path, &sessionCount)

	return map[string]any{
		"provider":       "supertokens",
		"connection_uri": p.connectionURI,
		"active_sessions": sessionCount.Count,
	}, nil
}

func (p *SuperTokensProvider) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.connectionURI+path, nil)
	if err != nil {
		return err
	}
	if p.apiKey != "" {
		req.Header.Set("api-key", p.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SuperTokens API error: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
