package cli

import (
	"io"
	"os"

	"github.com/anishalle/hack/internal/api"
	"github.com/anishalle/hack/internal/config"
	"golang.org/x/term"
)

type Factory struct {
	Config    func() (*config.UserConfig, error)
	Hackfile  func() (*config.Hackfile, error)
	APIClient func() (*api.Client, error)
	IO        *IOStreams
}

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func (s *IOStreams) IsInteractive() bool {
	if f, ok := s.In.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func NewFactory() *Factory {
	io := &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	var cachedConfig *config.UserConfig
	configFunc := func() (*config.UserConfig, error) {
		if cachedConfig != nil {
			return cachedConfig, nil
		}
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		cachedConfig = cfg
		return cfg, nil
	}

	hackfileFunc := func() (*config.Hackfile, error) {
		return config.LoadHackfile(".")
	}

	apiClientFunc := func() (*api.Client, error) {
		cfg, err := configFunc()
		if err != nil {
			return nil, err
		}
		return api.NewClient(cfg.API.BaseURL, cfg.Auth.AccessToken), nil
	}

	return &Factory{
		Config:    configFunc,
		Hackfile:  hackfileFunc,
		APIClient: apiClientFunc,
		IO:        io,
	}
}
