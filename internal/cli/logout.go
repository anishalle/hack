package cli

import (
	"fmt"

	"github.com/anishalle/hack/internal/config"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of HackUTD",
		Long:  `Remove stored authentication credentials from ~/.hack/config.yaml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return fmt.Errorf("not currently logged in")
			}

			cfg.Auth = config.AuthConfig{}
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to clear credentials: %w", err)
			}

			fmt.Fprintln(f.IO.Out, "Logged out successfully.")
			return nil
		},
	}
}
