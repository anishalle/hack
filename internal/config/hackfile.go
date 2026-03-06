package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const hackfileName = "hackfile.yaml"

type Hackfile struct {
	Project      string                  `yaml:"project"`
	Version      string                  `yaml:"version"`
	Environments map[string]*Environment `yaml:"environments"`
	Secrets      SecretsConfig           `yaml:"secrets"`
}

type Environment struct {
	Deploy *DeployConfig `yaml:"deploy,omitempty"`
	DB     *DBConfig     `yaml:"db,omitempty"`
	Auth   *AuthProviderConfig `yaml:"auth,omitempty"`
}

type DeployConfig struct {
	Provider   string            `yaml:"provider"`
	Project    string            `yaml:"project,omitempty"`
	Region     string            `yaml:"region,omitempty"`
	Service    string            `yaml:"service,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Extra      map[string]string `yaml:"extra,omitempty"`
}

type DBConfig struct {
	Provider   string            `yaml:"provider"`
	ProjectID  string            `yaml:"project_id,omitempty"`
	ProjectRef string            `yaml:"project_ref,omitempty"`
	Branch     string            `yaml:"branch,omitempty"`
	Database   string            `yaml:"database,omitempty"`
	Extra      map[string]string `yaml:"extra,omitempty"`
}

type AuthProviderConfig struct {
	Provider      string            `yaml:"provider"`
	Project       string            `yaml:"project,omitempty"`
	ConnectionURI string            `yaml:"connection_uri,omitempty"`
	Extra         map[string]string `yaml:"extra,omitempty"`
}

type SecretsConfig struct {
	Provider string `yaml:"provider"`
	Project  string `yaml:"project"`
	Prefix   string `yaml:"prefix,omitempty"`
}

func LoadHackfile(dir string) (*Hackfile, error) {
	path := filepath.Join(dir, hackfileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("hackfile.yaml not found in %s (run 'hack project init')", dir)
		}
		return nil, fmt.Errorf("failed to read hackfile: %w", err)
	}

	var hf Hackfile
	if err := yaml.Unmarshal(data, &hf); err != nil {
		return nil, fmt.Errorf("failed to parse hackfile.yaml: %w", err)
	}

	if err := hf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid hackfile.yaml: %w", err)
	}

	return &hf, nil
}

func (hf *Hackfile) Validate() error {
	if hf.Project == "" {
		return fmt.Errorf("'project' field is required")
	}
	if hf.Version == "" {
		return fmt.Errorf("'version' field is required")
	}
	if len(hf.Environments) == 0 {
		return fmt.Errorf("at least one environment must be defined")
	}
	return nil
}

func (hf *Hackfile) GetEnvironment(name string) (*Environment, error) {
	env, ok := hf.Environments[name]
	if !ok {
		available := make([]string, 0, len(hf.Environments))
		for k := range hf.Environments {
			available = append(available, k)
		}
		return nil, fmt.Errorf("environment %q not found (available: %v)", name, available)
	}
	return env, nil
}

func (hf *Hackfile) EnvironmentNames() []string {
	names := make([]string, 0, len(hf.Environments))
	for k := range hf.Environments {
		names = append(names, k)
	}
	return names
}

func (hf *Hackfile) Save(dir string) error {
	path := filepath.Join(dir, hackfileName)

	data, err := yaml.Marshal(hf)
	if err != nil {
		return fmt.Errorf("failed to serialize hackfile: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write hackfile.yaml: %w", err)
	}

	return nil
}
