package config

import "fmt"

type Context struct {
	UserConfig *UserConfig
	Hackfile   *Hackfile
}

func NewContext() (*Context, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	hf, _ := LoadHackfile(".")

	return &Context{
		UserConfig: cfg,
		Hackfile:   hf,
	}, nil
}

func (c *Context) RequireAuth() error {
	if !c.UserConfig.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}
	if c.UserConfig.IsTokenExpired() {
		return fmt.Errorf("session expired. Run 'hack login' to re-authenticate")
	}
	return nil
}

func (c *Context) RequireProject() error {
	if c.UserConfig.ActiveProject == "" {
		return fmt.Errorf("no active project. Run 'hack project switch' to select a project")
	}
	return nil
}

func (c *Context) RequireHackfile() error {
	if c.Hackfile == nil {
		return fmt.Errorf("hackfile.yaml not found. Run 'hack project init' to create one")
	}
	return nil
}

func (c *Context) RequireEnvironment(name string) (*Environment, error) {
	if err := c.RequireHackfile(); err != nil {
		return nil, err
	}
	return c.Hackfile.GetEnvironment(name)
}
