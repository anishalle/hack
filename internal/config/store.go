package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config stores Hack's local auth context.
type Config struct {
	ActiveProject string `json:"active_project,omitempty"`
	ActiveAccount string `json:"active_account,omitempty"`
}

// Store persists local config state.
type Store interface {
	Load() (Config, error)
	Save(Config) error
	Path() string
}

// JSONStore saves config as a JSON file on disk.
type JSONStore struct {
	path string
}

func NewJSONStore(path string) *JSONStore {
	if path == "" {
		path = defaultPath()
	}

	return &JSONStore{path: path}
}

func (s *JSONStore) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (s *JSONStore) Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tempPath, s.path)
}

func (s *JSONStore) Path() string {
	return s.path
}

func defaultPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".hack", "config.json")
	}

	return filepath.Join(configDir, "hack", "config.json")
}
