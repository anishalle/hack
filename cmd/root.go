package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const hiddenApp = "default"

// Execute builds the command tree and runs the CLI.
func Execute() {
	deps := defaultDependencies()
	rootCmd := newRootCmd(deps)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(deps.stderr, err)
		os.Exit(1)
	}
}

func newRootCmd(deps *dependencies) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "hack",
		Short:         "Manage HackUTD environments and deployments",
		Long:          "hack wraps Google Cloud tooling into a cleaner CLI for HackUTD environment management.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.SetIn(deps.stdin)
	rootCmd.SetOut(deps.stdout)
	rootCmd.SetErr(deps.stderr)

	rootCmd.AddCommand(newAuthCmd(deps))
	rootCmd.AddCommand(newEnvCmd(deps))
	rootCmd.AddCommand(newListCmd(deps))

	return rootCmd
}

func newListCmd(deps *dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List environment versions",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return newUsageError(
					"`hack list` does not take arguments",
					"List shows every environment/version Hack can see in the active project.",
					"hack list",
					"hack env list",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return printEnvironmentList(context.Background(), deps)
		},
	}
}
