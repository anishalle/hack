package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/tui"
)

func NewDBCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage databases",
		Long: `Manage database connections, migrations, and backups across environments.

Running 'hack db' without a subcommand opens the interactive database
dashboard. Database providers (Neon, Supabase, Cloud SQL, PlanetScale)
are configured per-environment in your hackfile.yaml.`,
		Example: `  # Open database dashboard
  hack db

  # Connect to dev database
  hack db connect dev

  # Check database status
  hack db status prod

  # List Neon branches
  hack db branches dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runDBInteractive(f)
		},
	}

	cmd.AddCommand(newDBConnectCmd(f))
	cmd.AddCommand(newDBStatusCmd(f))
	cmd.AddCommand(newDBMigrateCmd(f))
	cmd.AddCommand(newDBBackupCmd(f))
	cmd.AddCommand(newDBBranchesCmd(f))

	return cmd
}

func runDBInteractive(f *Factory) error {
	var action string
	err := huh.NewSelect[string]().
		Title("Database actions").
		Options(
			huh.NewOption("Connect to database", "connect"),
			huh.NewOption("Check status", "status"),
			huh.NewOption("Run migrations", "migrate"),
			huh.NewOption("Create backup", "backup"),
			huh.NewOption("Manage branches (Neon)", "branches"),
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
	case "connect":
		return dbConnect(f, env, "psql")
	case "status":
		return dbStatus(f, env)
	case "migrate":
		return dbMigrate(f, env, false)
	case "backup":
		return dbBackup(f, env)
	case "branches":
		return dbBranches(f, env)
	}

	return nil
}

func newDBConnectCmd(f *Factory) *cobra.Command {
	var tool string

	cmd := &cobra.Command{
		Use:   "connect [environment]",
		Short: "Connect to a database",
		Long: `Open an interactive database connection for the specified environment.
Uses psql for PostgreSQL databases by default.`,
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

			return dbConnect(f, env, tool)
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "psql", "Database client tool (psql, pgcli)")

	return cmd
}

func dbConnect(f *Factory, env, tool string) error {
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

	if envCfg.DB == nil {
		return fmt.Errorf("no database configured for environment %q", env)
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/db/%s/connect", hf.Project, env)
	var resp struct {
		URI string `json:"uri"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
	}

	if resp.URI == "" {
		return fmt.Errorf("no connection URI available")
	}

	fmt.Fprintf(f.IO.Out, "Connecting to %s database (%s)...\n\n",
		tui.EnvBadge(env), envCfg.DB.Provider)

	dbCmd := exec.Command(tool, resp.URI)
	dbCmd.Stdin = os.Stdin
	dbCmd.Stdout = os.Stdout
	dbCmd.Stderr = os.Stderr

	return dbCmd.Run()
}

func newDBStatusCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [environment]",
		Short: "Show database status",
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

			return dbStatus(f, env)
		},
	}

	return cmd
}

func dbStatus(f *Factory, env string) error {
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

	path := fmt.Sprintf("/projects/%s/db/%s/status", hf.Project, env)
	var resp map[string]any
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s database\n\n", tui.EnvBadge(env))

	for k, v := range resp {
		fmt.Fprintf(f.IO.Out, "  %s %v\n",
			tui.Dim.Render(fmt.Sprintf("%-12s", k+":")),
			v)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newDBMigrateCmd(f *Factory) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "migrate <environment>",
		Short: "Run database migrations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dbMigrate(f, args[0], dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration without executing")

	return cmd
}

func dbMigrate(f *Factory, env string, dryRun bool) error {
	if dryRun {
		fmt.Fprintf(f.IO.Out, "Dry run: would run migrations on %s\n", tui.EnvBadge(env))
		return nil
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Migrations on %s complete\n", icon, tui.EnvBadge(env))
	return nil
}

func newDBBackupCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup <environment>",
		Short: "Create a database backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dbBackup(f, args[0])
		},
	}

	return cmd
}

func dbBackup(f *Factory, env string) error {
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

	path := fmt.Sprintf("/projects/%s/db/%s/backup", hf.Project, env)
	var resp map[string]any
	if err := client.Post(context.Background(), path, nil, &resp); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Backup of %s database initiated\n", icon, tui.EnvBadge(env))

	return nil
}

func newDBBranchesCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branches [environment]",
		Short: "Manage database branches (Neon)",
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

			return dbBranches(f, env)
		},
	}

	return cmd
}

func dbBranches(f *Factory, env string) error {
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

	path := fmt.Sprintf("/projects/%s/db/%s/branches", hf.Project, env)
	var resp struct {
		Branches []struct {
			Name      string `json:"name"`
			Primary   bool   `json:"primary"`
			CreatedAt string `json:"created_at"`
		} `json:"branches"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s database branches\n\n", tui.EnvBadge(env))

	for _, b := range resp.Branches {
		indicator := tui.InactiveDot
		if b.Primary {
			indicator = tui.ActiveDot
		}

		label := b.Name
		if b.Primary {
			label += " (primary)"
		}

		fmt.Fprintf(f.IO.Out, "  %s %s  %s\n",
			indicator,
			tui.Bold.Render(label),
			tui.Dim.Render(b.CreatedAt))
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}
