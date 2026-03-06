package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".hack"
	configFile = "config.yaml"
)

type UserConfig struct {
	ActiveProject string                    `yaml:"active_project,omitempty"`
	Auth          AuthConfig                `yaml:"auth,omitempty"`
	API           APIConfig                 `yaml:"api"`
	Projects      map[string]*ProjectConfig `yaml:"projects,omitempty"`

	path string `yaml:"-"`
}

type AuthConfig struct {
	AccessToken  string    `yaml:"access_token,omitempty"`
	RefreshToken string    `yaml:"refresh_token,omitempty"`
	ExpiresAt    time.Time `yaml:"expires_at,omitempty"`
	Email        string    `yaml:"email,omitempty"`
}

type APIConfig struct {
	BaseURL string `yaml:"base_url"`
}

type ProjectConfig struct {
	Path       string `yaml:"path"`
	DefaultEnv string `yaml:"default_env,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

func Load() (*UserConfig, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &UserConfig{
				API:      APIConfig{BaseURL: "http://localhost:8080"},
				Projects: make(map[string]*ProjectConfig),
				path:     path,
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Projects == nil {
		cfg.Projects = make(map[string]*ProjectConfig)
	}

	cfg.path = path
	return &cfg, nil
}

func (c *UserConfig) Save() error {
	if c.path == "" {
		dir, err := Dir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, configFile)
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *UserConfig) IsLoggedIn() bool {
	return c.Auth.AccessToken != "" && c.Auth.Email != ""
}

func (c *UserConfig) IsTokenExpired() bool {
	if c.Auth.ExpiresAt.IsZero() {
		return true
	}
	return time.Now().After(c.Auth.ExpiresAt)
}

func (c *UserConfig) ActiveProjectConfig() *ProjectConfig {
	if c.ActiveProject == "" {
		return nil
	}
	return c.Projects[c.ActiveProject]
}
