package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anishalle/hack/server/internal/provider"
)

type NeonProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewNeonProvider(apiKey string) *NeonProvider {
	return &NeonProvider{
		apiKey:  apiKey,
		baseURL: "https://console.neon.tech/api/v2",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *NeonProvider) Connect(ctx context.Context, opts provider.DBConnectOptions) (*provider.DBConnection, error) {
	path := fmt.Sprintf("/projects/%s/connection_uri", opts.ProjectID)

	params := "?database_name=" + opts.Database
	if opts.Branch != "" {
		params += "&branch_name=" + opts.Branch
	}

	var resp struct {
		URI string `json:"uri"`
	}
	if err := p.get(ctx, path+params, &resp); err != nil {
		return nil, fmt.Errorf("failed to get connection URI: %w", err)
	}

	return &provider.DBConnection{
		URI: resp.URI,
	}, nil
}

func (p *NeonProvider) Status(ctx context.Context, opts provider.DBStatusOptions) (*provider.DBStatus, error) {
	path := fmt.Sprintf("/projects/%s", opts.ProjectID)

	var resp struct {
		Project struct {
			ID         string `json:"id"`
			RegionID   string `json:"region_id"`
			StoreUsage int64  `json:"store_usage"`
		} `json:"project"`
	}

	if err := p.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get project status: %w", err)
	}

	return &provider.DBStatus{
		Provider: "neon",
		Status:   "active",
		Region:   resp.Project.RegionID,
		Size:     fmt.Sprintf("%d MB", resp.Project.StoreUsage/1024/1024),
	}, nil
}

func (p *NeonProvider) Branches(ctx context.Context, opts provider.DBBranchOptions) ([]provider.DBBranch, error) {
	path := fmt.Sprintf("/projects/%s/branches", opts.ProjectID)

	var resp struct {
		Branches []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Primary   bool   `json:"primary"`
			CreatedAt string `json:"created_at"`
		} `json:"branches"`
	}

	if err := p.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	branches := make([]provider.DBBranch, len(resp.Branches))
	for i, b := range resp.Branches {
		branches[i] = provider.DBBranch{
			Name:      b.Name,
			Primary:   b.Primary,
			CreatedAt: b.CreatedAt,
		}
	}

	return branches, nil
}

func (p *NeonProvider) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Neon API error: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
