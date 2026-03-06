package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/tui"
)

func NewRootCmd(f *Factory, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hack",
		Short: "HackUTD's internal CLI for envs, deploys, and infrastructure",
		Long: `hack is HackUTD's internal CLI tool for managing environment variables,
deployments, databases, and authentication across all your projects.

Run hack without arguments to open the interactive dashboard.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runDashboard(f, cmd)
		},
	}

	cmd.SetIn(f.IO.In)
	cmd.SetOut(f.IO.Out)
	cmd.SetErr(f.IO.ErrOut)

	cmd.PersistentFlags().Bool("no-interactive", false, "Disable interactive prompts")

	cmd.AddCommand(NewLoginCmd(f))
	cmd.AddCommand(NewLogoutCmd(f))
	cmd.AddCommand(NewWhoamiCmd(f))
	cmd.AddCommand(NewEnvCmd(f))
	cmd.AddCommand(NewDeployCmd(f))
	cmd.AddCommand(NewDBCmd(f))
	cmd.AddCommand(NewAuthCmd(f))
	cmd.AddCommand(NewProjectCmd(f))
	cmd.AddCommand(NewAdminCmd(f))
	cmd.AddCommand(NewVersionCmd(version))

	return cmd
}

func runDashboard(f *Factory, rootCmd *cobra.Command) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		fmt.Fprintln(f.IO.Out, "Welcome to hack! Run 'hack login' to get started.")
		return nil
	}

	model := tui.NewDashboardModel(cfg.ActiveProject)
	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return err
	}

	if dash, ok := result.(tui.DashboardModel); ok {
		selected := dash.Selected()
		if selected != "" {
			switch selected {
			case "env":
				return NewEnvCmd(f).RunE(rootCmd, nil)
			case "deploy":
				fmt.Fprintln(f.IO.Out, "Run: hack deploy")
			case "db":
				fmt.Fprintln(f.IO.Out, "Run: hack db")
			case "auth":
				fmt.Fprintln(f.IO.Out, "Run: hack auth")
			case "project info":
				return runProjectInfo(f)
			case "admin":
				fmt.Fprintln(f.IO.Out, "Run: hack admin")
			}
		}
	}

	return nil
}

func Execute(version string) int {
	f := NewFactory()
	rootCmd := NewRootCmd(f, version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}
	return 0
}
