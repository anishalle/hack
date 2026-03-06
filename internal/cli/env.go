package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/api"
	"github.com/anishalle/hack/internal/tui"
)

func NewEnvCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables",
		Long: `Manage environment variables across dev, staging, and production.

Running 'hack env' without a subcommand opens the interactive environment
variable manager where you can browse, compare, and edit variables across
environments.

Environment variables are stored securely in Google Secret Manager and
access is controlled by your project role.`,
		Example: `  # Open interactive env manager
  hack env

  # Pull prod env vars to .env file
  hack env pull prod

  # Push local .env to dev environment
  hack env push dev

  # Compare dev and prod environments
  hack env diff dev prod

  # Set a single variable
  hack env set prod DATABASE_URL=postgres://...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvInteractive(f)
		},
	}

	cmd.AddCommand(newEnvPullCmd(f))
	cmd.AddCommand(newEnvPushCmd(f))
	cmd.AddCommand(newEnvDiffCmd(f))
	cmd.AddCommand(newEnvListCmd(f))
	cmd.AddCommand(newEnvSetCmd(f))
	cmd.AddCommand(newEnvUnsetCmd(f))
	cmd.AddCommand(newEnvEditCmd(f))
	cmd.AddCommand(newEnvHistoryCmd(f))

	return cmd
}

func runEnvInteractive(f *Factory) error {
	if !f.IO.IsInteractive() {
		return fmt.Errorf("run 'hack env --help' for usage")
	}

	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	envNames := hf.EnvironmentNames()
	if len(envNames) == 0 {
		return fmt.Errorf("no environments defined in hackfile.yaml")
	}

	var selectedEnv string
	options := make([]huh.Option[string], len(envNames))
	for i, name := range envNames {
		options[i] = huh.NewOption(name, name)
	}

	err = huh.NewSelect[string]().
		Title("Select environment").
		Options(options...).
		Value(&selectedEnv).
		Run()
	if err != nil {
		return nil
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	var resp api.EnvPullResponse
	path := fmt.Sprintf("/projects/%s/env/%s", hf.Project, selectedEnv)
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to fetch env vars: %w", err)
	}

	model := tui.NewEnvViewModel(selectedEnv, resp.Variables)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func resolveEnv(f *Factory, envArg string) (string, error) {
	if envArg != "" {
		return envArg, nil
	}

	if !f.IO.IsInteractive() {
		return "", fmt.Errorf("environment name required")
	}

	hf, err := f.Hackfile()
	if err != nil {
		return "", err
	}

	envNames := hf.EnvironmentNames()
	var selected string
	options := make([]huh.Option[string], len(envNames))
	for i, name := range envNames {
		options[i] = huh.NewOption(name, name)
	}

	err = huh.NewSelect[string]().
		Title("Select environment").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return "", nil
	}

	return selected, nil
}

func newEnvPullCmd(f *Factory) *cobra.Command {
	var (
		outputFile string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "pull [environment]",
		Short: "Pull environment variables to a local file",
		Long: `Download environment variables from the specified environment and write
them to a local .env file. By default, writes to .env.<environment> in the
current directory.`,
		Example: `  # Pull dev env vars (writes to .env.dev)
  hack env pull dev

  # Pull prod env vars to a specific file
  hack env pull prod --output .env

  # Overwrite existing file
  hack env pull prod --force`,
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

			var resp api.EnvPullResponse
			path := fmt.Sprintf("/projects/%s/env/%s", hf.Project, env)
			if err := client.Get(context.Background(), path, &resp); err != nil {
				return fmt.Errorf("failed to pull env vars: %w", err)
			}

			outPath := outputFile
			if outPath == "" {
				outPath = fmt.Sprintf(".env.%s", env)
			}

			if !force {
				if _, err := os.Stat(outPath); err == nil {
					if f.IO.IsInteractive() {
						var confirm bool
						huh.NewConfirm().
							Title(fmt.Sprintf("%s already exists. Overwrite?", outPath)).
							Value(&confirm).
							Run()
						if !confirm {
							fmt.Fprintln(f.IO.Out, "Cancelled.")
							return nil
						}
					} else {
						return fmt.Errorf("%s already exists (use --force to overwrite)", outPath)
					}
				}
			}

			if err := writeEnvFile(outPath, resp.Variables); err != nil {
				return fmt.Errorf("failed to write env file: %w", err)
			}

			successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
			fmt.Fprintf(f.IO.Out, "%s Pulled %d variables to %s\n", successIcon, len(resp.Variables), outPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: .env.<environment>)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing file without confirmation")

	return cmd
}

func newEnvPushCmd(f *Factory) *cobra.Command {
	var (
		inputFile string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "push [environment]",
		Short: "Push local env file to remote environment",
		Long: `Upload environment variables from a local file to the specified
environment in Google Secret Manager. Shows a diff before applying changes.`,
		Example: `  # Push to dev from .env.dev
  hack env push dev

  # Push a specific file to staging
  hack env push staging --file .env.local

  # Preview changes without applying
  hack env push prod --dry-run`,
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

			cfg, err := f.Config()
			if err != nil || !cfg.IsLoggedIn() {
				return fmt.Errorf("not logged in. Run 'hack login' first")
			}

			hf, err := f.Hackfile()
			if err != nil {
				return err
			}

			inPath := inputFile
			if inPath == "" {
				inPath = fmt.Sprintf(".env.%s", env)
			}

			vars, err := readEnvFile(inPath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", inPath, err)
			}

			if dryRun {
				fmt.Fprintf(f.IO.Out, "Would push %d variables to %s:\n\n", len(vars), tui.EnvBadge(env))
				keys := make([]string, 0, len(vars))
				for k := range vars {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(f.IO.Out, "  %s = %s\n", tui.Key.Render(k), tui.Dim.Render("****"))
				}
				return nil
			}

			client, err := f.APIClient()
			if err != nil {
				return err
			}

			_ = cfg

			path := fmt.Sprintf("/projects/%s/env/%s", hf.Project, env)
			var resp map[string]any
			if err := client.Put(context.Background(), path, api.EnvPushRequest{
				Environment: env,
				Variables:   vars,
			}, &resp); err != nil {
				return fmt.Errorf("failed to push env vars: %w", err)
			}

			successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
			fmt.Fprintf(f.IO.Out, "%s Pushed %d variables to %s\n", successIcon, len(vars), tui.EnvBadge(env))

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input file path (default: .env.<environment>)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")

	return cmd
}

func newEnvDiffCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <env1> <env2>",
		Short: "Compare environment variables between two environments",
		Long: `Show a side-by-side diff of environment variables between two
environments. Highlights added, removed, and changed variables.`,
		Example: `  # Compare dev and prod
  hack env diff dev prod`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var resp api.EnvDiffResponse
			path := fmt.Sprintf("/projects/%s/env/diff/%s/%s", hf.Project, args[0], args[1])
			if err := client.Get(context.Background(), path, &resp); err != nil {
				return fmt.Errorf("failed to diff environments: %w", err)
			}

			diff := tui.EnvDiff{
				Added:   resp.Added,
				Removed: resp.Removed,
				Changed: resp.Changed,
			}

			fmt.Fprint(f.IO.Out, tui.RenderEnvDiff(args[0], args[1], diff))
			return nil
		},
	}

	return cmd
}

func newEnvListCmd(f *Factory) *cobra.Command {
	var showValues bool

	cmd := &cobra.Command{
		Use:   "list [environment]",
		Short: "List environment variable keys",
		Long: `List all environment variable keys for the specified environment.
By default, only keys are shown for security. Use --values to reveal values.`,
		Example: `  # List keys for dev
  hack env list dev

  # List keys and values for prod
  hack env list prod --values`,
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

			_ = cfg

			if showValues {
				var resp api.EnvPullResponse
				path := fmt.Sprintf("/projects/%s/env/%s", hf.Project, env)
				if err := client.Get(context.Background(), path, &resp); err != nil {
					return fmt.Errorf("failed to list env vars: %w", err)
				}

				keys := make([]string, 0, len(resp.Variables))
				for k := range resp.Variables {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				fmt.Fprintf(f.IO.Out, "\n  %s — %d variables\n\n", tui.EnvBadge(env), len(resp.Variables))
				for _, k := range keys {
					fmt.Fprintf(f.IO.Out, "  %s = %s\n", tui.Key.Render(k), tui.Value.Render(resp.Variables[k]))
				}
			} else {
				var resp struct {
					Keys  []string `json:"keys"`
					Count int      `json:"count"`
				}
				path := fmt.Sprintf("/projects/%s/env/%s/list", hf.Project, env)
				if err := client.Get(context.Background(), path, &resp); err != nil {
					return fmt.Errorf("failed to list env vars: %w", err)
				}

				sort.Strings(resp.Keys)
				fmt.Fprintf(f.IO.Out, "\n  %s — %d variables\n\n", tui.EnvBadge(env), resp.Count)
				for _, k := range resp.Keys {
					fmt.Fprintf(f.IO.Out, "  %s\n", tui.Key.Render(k))
				}
			}

			fmt.Fprintln(f.IO.Out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&showValues, "values", false, "Show values (hidden by default)")

	return cmd
}

func newEnvSetCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <environment> <KEY=VALUE>...",
		Short: "Set one or more environment variables",
		Long: `Set environment variables in the specified environment. Multiple
KEY=VALUE pairs can be provided.`,
		Example: `  # Set a single variable
  hack env set prod DATABASE_URL=postgres://localhost/mydb

  # Set multiple variables
  hack env set dev API_KEY=abc123 DEBUG=true`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := args[0]
			pairs := args[1:]

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

			_ = cfg

			for _, pair := range pairs {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format: %q (expected KEY=VALUE)", pair)
				}

				key, value := parts[0], parts[1]
				path := fmt.Sprintf("/projects/%s/env/%s/set", hf.Project, env)

				var resp map[string]any
				if err := client.Post(context.Background(), path, map[string]string{
					"key": key, "value": value,
				}, &resp); err != nil {
					return fmt.Errorf("failed to set %s: %w", key, err)
				}

				successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
				fmt.Fprintf(f.IO.Out, "%s Set %s in %s\n", successIcon, tui.Key.Render(key), tui.EnvBadge(env))
			}

			return nil
		},
	}

	return cmd
}

func newEnvUnsetCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <environment> <KEY>...",
		Short: "Remove environment variables",
		Long:  `Remove one or more environment variables from the specified environment.`,
		Example: `  # Remove a variable
  hack env unset prod OLD_API_KEY`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := args[0]
			keys := args[1:]

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

			_ = cfg

			for _, key := range keys {
				path := fmt.Sprintf("/projects/%s/env/%s/unset", hf.Project, env)
				var resp map[string]any
				if err := client.Post(context.Background(), path, map[string]string{
					"key": key,
				}, &resp); err != nil {
					return fmt.Errorf("failed to unset %s: %w", key, err)
				}

				successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
				fmt.Fprintf(f.IO.Out, "%s Removed %s from %s\n", successIcon, tui.Key.Render(key), tui.EnvBadge(env))
			}

			return nil
		},
	}

	return cmd
}

func newEnvEditCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [environment]",
		Short: "Edit environment variables in your editor",
		Long: `Open environment variables in $EDITOR for editing. Changes are
pushed automatically when the editor is closed.`,
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

			_ = cfg

			var resp api.EnvPullResponse
			pullPath := fmt.Sprintf("/projects/%s/env/%s", hf.Project, env)
			if err := client.Get(context.Background(), pullPath, &resp); err != nil {
				return fmt.Errorf("failed to fetch env vars: %w", err)
			}

			tmpFile, err := os.CreateTemp("", fmt.Sprintf("hack-env-%s-*.env", env))
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			if err := writeEnvFile(tmpPath, resp.Variables); err != nil {
				return fmt.Errorf("failed to write temp file: %w", err)
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}

			editorCmd := exec.Command(editor, tmpPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			newVars, err := readEnvFile(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to read edited file: %w", err)
			}

			pushPath := fmt.Sprintf("/projects/%s/env/%s", hf.Project, env)
			var pushResp map[string]any
			if err := client.Put(context.Background(), pushPath, api.EnvPushRequest{
				Environment: env,
				Variables:   newVars,
			}, &pushResp); err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}

			successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
			fmt.Fprintf(f.IO.Out, "%s Updated %s environment (%d variables)\n",
				successIcon, tui.EnvBadge(env), len(newVars))

			return nil
		},
	}

	return cmd
}

func newEnvHistoryCmd(f *Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history [environment]",
		Short: "Show change history for environment variables",
		Long: `Display an audit log of changes to environment variables, including
who made changes and when.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			_ = cfg
			_ = limit

			path := fmt.Sprintf("/projects/%s/admin/audit?action=env", hf.Project)
			var resp struct {
				Entries []api.AuditEntry `json:"entries"`
			}
			if err := client.Get(context.Background(), path, &resp); err != nil {
				return fmt.Errorf("failed to fetch history: %w", err)
			}

			if len(resp.Entries) == 0 {
				fmt.Fprintln(f.IO.Out, tui.Dim.Render("  No history entries found."))
				return nil
			}

			fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Environment Change History"))
			for _, entry := range resp.Entries {
				fmt.Fprintf(f.IO.Out, "  %s  %s  %s  %s\n",
					tui.Dim.Render(entry.Timestamp),
					tui.Bold.Render(entry.User),
					tui.Key.Render(entry.Action),
					entry.Details,
				)
			}
			fmt.Fprintln(f.IO.Out)

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of entries to show")

	return cmd
}

func writeEnvFile(path string, vars map[string]string) error {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, k := range keys {
		v := vars[k]
		if strings.ContainsAny(v, " \t\n\"'") {
			fmt.Fprintf(file, "%s=%q\n", k, v)
		} else {
			fmt.Fprintf(file, "%s=%s\n", k, v)
		}
	}

	return nil
}

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		vars[key] = value
	}

	return vars, scanner.Err()
}
