package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/tui"
)

func NewDeployCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Manage deployments",
		Long: `Deploy, monitor, and manage your project's cloud infrastructure.

Running 'hack deploy' without a subcommand opens the interactive deployment
dashboard showing service status, recent deploys, and quick actions.

Deployment targets are configured per-environment in your hackfile.yaml and
support Cloud Run, Compute Engine, and App Engine.`,
		Example: `  # Open deployment dashboard
  hack deploy

  # Deploy to production
  hack deploy up prod

  # Check deployment status
  hack deploy status prod

  # Stream logs from staging
  hack deploy logs staging --follow`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runDeployInteractive(f)
		},
	}

	cmd.AddCommand(newDeployUpCmd(f))
	cmd.AddCommand(newDeployStatusCmd(f))
	cmd.AddCommand(newDeployLogsCmd(f))
	cmd.AddCommand(newDeployRollbackCmd(f))
	cmd.AddCommand(newDeployRestartCmd(f))

	return cmd
}

func runDeployInteractive(f *Factory) error {
	var action string
	err := huh.NewSelect[string]().
		Title("Deployment actions").
		Options(
			huh.NewOption("Deploy to environment", "up"),
			huh.NewOption("Check status", "status"),
			huh.NewOption("View logs", "logs"),
			huh.NewOption("Rollback", "rollback"),
			huh.NewOption("Restart services", "restart"),
		).
		Value(&action).
		Run()
	if err != nil {
		return nil
	}

	env, err := resolveEnv(f, "")
	if err != nil {
		return err
	}
	if env == "" {
		return nil
	}

	switch action {
	case "up":
		return deployUp(f, env, "", false)
	case "status":
		return deployStatus(f, env)
	case "logs":
		return deployLogs(f, env, false, 50)
	case "rollback":
		return deployRollback(f, env, "")
	case "restart":
		return deployRestart(f, env)
	}

	return nil
}

func newDeployUpCmd(f *Factory) *cobra.Command {
	var (
		tag     string
		confirm bool
	)

	cmd := &cobra.Command{
		Use:   "up [environment]",
		Short: "Deploy to an environment",
		Long: `Build and deploy your project to the specified environment. For
containerized deployments, this builds the Docker image, pushes it to
Artifact Registry, and deploys to the configured service.`,
		Example: `  # Deploy to dev
  hack deploy up dev

  # Deploy to prod with a specific tag
  hack deploy up prod --tag v1.2.3

  # Deploy to prod without confirmation prompt
  hack deploy up prod --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envArg := ""
			if len(args) > 0 {
				envArg = args[0]
			}

			env, err := resolveEnv(f, envArg)
			if err != nil {
				return err
			}
			if env == "" {
				return nil
			}

			return deployUp(f, env, tag, confirm)
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Image tag (default: git SHA)")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func deployUp(f *Factory, env, tag string, skipConfirm bool) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	envCfg, err := hf.GetEnvironment(env)
	if err != nil {
		return err
	}

	if envCfg.Deploy == nil {
		return fmt.Errorf("no deploy config for environment %q", env)
	}

	if !skipConfirm && f.IO.IsInteractive() && (env == "prod" || env == "production") {
		var confirmed bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Deploy to %s?", tui.EnvBadge(env))).
			Description("This will affect production services.").
			Value(&confirmed).
			Run()
		if !confirmed {
			fmt.Fprintln(f.IO.Out, "Deploy cancelled.")
			return nil
		}
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	body := map[string]any{}
	if tag != "" {
		body["tag"] = tag
	}

	path := fmt.Sprintf("/projects/%s/deploy/%s/up", hf.Project, env)
	var resp map[string]any
	if err := client.Post(context.Background(), path, body, &resp); err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Deployment to %s initiated\n", icon, tui.EnvBadge(env))
	fmt.Fprintf(f.IO.Out, "  Provider: %s\n", envCfg.Deploy.Provider)
	fmt.Fprintf(f.IO.Out, "  Service: %s\n", envCfg.Deploy.Service)

	return nil
}

func newDeployStatusCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [environment]",
		Short: "Show deployment status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envArg := ""
			if len(args) > 0 {
				envArg = args[0]
			}

			env, err := resolveEnv(f, envArg)
			if err != nil {
				return err
			}
			if env == "" {
				return nil
			}

			return deployStatus(f, env)
		},
	}

	return cmd
}

func deployStatus(f *Factory, env string) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/deploy/%s/status", hf.Project, env)
	var resp map[string]any
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s deployment\n\n", tui.EnvBadge(env))

	status, _ := resp["status"].(string)
	statusStyle := tui.Success
	if status != "running" && status != "ready" {
		statusStyle = tui.Warning
	}

	fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Status:"), statusStyle.Render(status))

	if rev, ok := resp["revision"].(string); ok {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Revision:"), rev)
	}
	if url, ok := resp["url"].(string); ok && url != "" {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("URL:"), url)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newDeployLogsCmd(f *Factory) *cobra.Command {
	var (
		follow bool
		tail   int
	)

	cmd := &cobra.Command{
		Use:   "logs <environment>",
		Short: "Stream deployment logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployLogs(f, args[0], follow, tail)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream logs in real-time")
	cmd.Flags().IntVar(&tail, "tail", 50, "Number of recent lines to show")

	return cmd
}

func deployLogs(f *Factory, env string, follow bool, tail int) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	_ = follow
	_ = tail

	path := fmt.Sprintf("/projects/%s/deploy/%s/logs", hf.Project, env)
	var resp struct {
		Logs []struct {
			Timestamp string `json:"timestamp"`
			Severity  string `json:"severity"`
			Message   string `json:"message"`
		} `json:"logs"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	for _, entry := range resp.Logs {
		sevStyle := tui.Dim
		switch entry.Severity {
		case "ERROR":
			sevStyle = tui.Error
		case "WARNING":
			sevStyle = tui.Warning
		}

		fmt.Fprintf(f.IO.Out, "%s %s %s\n",
			tui.Dim.Render(entry.Timestamp),
			sevStyle.Render(entry.Severity),
			entry.Message,
		)
	}

	return nil
}

func newDeployRollbackCmd(f *Factory) *cobra.Command {
	var revision string

	cmd := &cobra.Command{
		Use:   "rollback <environment>",
		Short: "Rollback to a previous deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployRollback(f, args[0], revision)
		},
	}

	cmd.Flags().StringVar(&revision, "revision", "", "Target revision (default: previous)")

	return cmd
}

func deployRollback(f *Factory, env, revision string) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	body := map[string]any{}
	if revision != "" {
		body["revision"] = revision
	}

	path := fmt.Sprintf("/projects/%s/deploy/%s/rollback", hf.Project, env)
	var resp map[string]any
	if err := client.Post(context.Background(), path, body, &resp); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Rolling back %s\n", icon, tui.EnvBadge(env))

	return nil
}

func newDeployRestartCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <environment>",
		Short: "Restart running services",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployRestart(f, args[0])
		},
	}

	return cmd
}

func deployRestart(f *Factory, env string) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/deploy/%s/restart", hf.Project, env)
	var resp map[string]any
	if err := client.Post(context.Background(), path, nil, &resp); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Restarting %s\n", icon, tui.EnvBadge(env))

	return nil
}
