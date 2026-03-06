package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/tui"
)

func NewAuthCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication providers",
		Long: `Manage your project's authentication provider (SuperTokens, Firebase, Supabase Auth).

Running 'hack auth' without a subcommand opens the interactive auth
dashboard showing user counts, recent signups, and provider configuration.`,
		Example: `  # Open auth dashboard
  hack auth

  # List auth users in prod
  hack auth users prod

  # View auth provider configuration
  hack auth config dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runAuthInteractive(f)
		},
	}

	cmd.AddCommand(newAuthUsersCmd(f))
	cmd.AddCommand(newAuthConfigCmd(f))
	cmd.AddCommand(newAuthSessionsCmd(f))

	return cmd
}

func runAuthInteractive(f *Factory) error {
	var action string
	err := huh.NewSelect[string]().
		Title("Auth provider actions").
		Options(
			huh.NewOption("List users", "users"),
			huh.NewOption("View configuration", "config"),
			huh.NewOption("Active sessions", "sessions"),
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
	case "users":
		return authUsers(f, env, 25, "")
	case "config":
		return authConfig(f, env)
	case "sessions":
		return authSessions(f, env)
	}

	return nil
}

func newAuthUsersCmd(f *Factory) *cobra.Command {
	var (
		limit  int
		search string
	)

	cmd := &cobra.Command{
		Use:   "users [environment]",
		Short: "List authentication users",
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

			return authUsers(f, env, limit, search)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum users to display")
	cmd.Flags().StringVar(&search, "search", "", "Search by email or user ID")

	return cmd
}

func authUsers(f *Factory, env string, limit int, search string) error {
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

	path := fmt.Sprintf("/projects/%s/auth/%s/users?limit=%d", hf.Project, env, limit)
	if search != "" {
		path += "&search=" + search
	}

	var resp struct {
		Users []struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			CreatedAt string `json:"created_at"`
		} `json:"users"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s auth users (%d)\n\n", tui.EnvBadge(env), len(resp.Users))

	for _, u := range resp.Users {
		fmt.Fprintf(f.IO.Out, "  %s  %s  %s\n",
			tui.Dim.Render(u.ID[:8]+"..."),
			tui.Bold.Render(u.Email),
			tui.Dim.Render(u.CreatedAt),
		)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newAuthConfigCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [environment]",
		Short: "View auth provider configuration",
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

			return authConfig(f, env)
		},
	}

	return cmd
}

func authConfig(f *Factory, env string) error {
	hf, err := f.Hackfile()
	if err != nil {
		return err
	}

	envCfg, err := hf.GetEnvironment(env)
	if err != nil {
		return err
	}

	fmt.Fprintf(f.IO.Out, "\n  %s auth configuration\n\n", tui.EnvBadge(env))

	if envCfg.Auth == nil {
		fmt.Fprintln(f.IO.Out, tui.Dim.Render("  No auth provider configured."))
		return nil
	}

	fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Provider:"), tui.Bold.Render(envCfg.Auth.Provider))
	if envCfg.Auth.ConnectionURI != "" {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Endpoint:"), envCfg.Auth.ConnectionURI)
	}
	if envCfg.Auth.Project != "" {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Project:"), envCfg.Auth.Project)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newAuthSessionsCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions [environment]",
		Short: "View active sessions (SuperTokens)",
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

			return authSessions(f, env)
		},
	}

	return cmd
}

func authSessions(f *Factory, env string) error {
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

	path := fmt.Sprintf("/projects/%s/auth/%s/sessions", hf.Project, env)
	var resp map[string]any
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s active sessions\n\n", tui.EnvBadge(env))

	if count, ok := resp["active_sessions"]; ok {
		fmt.Fprintf(f.IO.Out, "  Active sessions: %v\n", count)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}
