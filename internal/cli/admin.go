package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/tui"
)

func NewAdminCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin commands for managing users and roles",
		Long: `Administrative commands for managing hack users, roles, and viewing
audit logs. Requires admin or owner role on the current project.

Running 'hack admin' without a subcommand opens the interactive admin panel.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runAdminInteractive(f)
		},
	}

	cmd.AddCommand(newAdminUsersCmd(f))
	cmd.AddCommand(newAdminRolesCmd(f))
	cmd.AddCommand(newAdminAuditCmd(f))

	return cmd
}

func runAdminInteractive(f *Factory) error {
	var action string
	err := huh.NewSelect[string]().
		Title("Admin actions").
		Options(
			huh.NewOption("Manage users", "users"),
			huh.NewOption("Manage roles", "roles"),
			huh.NewOption("View audit log", "audit"),
		).
		Value(&action).
		Run()
	if err != nil {
		return nil
	}

	switch action {
	case "users":
		return adminListUsers(f)
	case "roles":
		return adminListRoles(f)
	case "audit":
		return adminAudit(f, 20, "", "")
	}

	return nil
}

func newAdminUsersCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage hack users",
		RunE: func(cmd *cobra.Command, args []string) error {
			return adminListUsers(f)
		},
	}

	cmd.AddCommand(newAdminUsersAddCmd(f))
	cmd.AddCommand(newAdminUsersRemoveCmd(f))

	return cmd
}

func adminListUsers(f *Factory) error {
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

	path := fmt.Sprintf("/projects/%s/admin/users", hf.Project)
	var resp struct {
		Users []struct {
			Email string `json:"email"`
			Name  string `json:"name"`
			Role  string `json:"role"`
		} `json:"users"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Project Users"))

	roleColors := map[string]lipgloss.Color{
		"owner":     "9",
		"admin":     "11",
		"deployer":  "14",
		"developer": "10",
		"viewer":    "8",
	}

	for _, u := range resp.Users {
		color, ok := roleColors[u.Role]
		if !ok {
			color = "8"
		}
		roleStyle := lipgloss.NewStyle().Foreground(color)

		name := u.Email
		if u.Name != "" {
			name = u.Name + " <" + u.Email + ">"
		}

		fmt.Fprintf(f.IO.Out, "  %s  %s\n",
			tui.Bold.Render(name),
			roleStyle.Render("("+u.Role+")"),
		)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newAdminUsersAddCmd(f *Factory) *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Invite a user to the project",
		Example: `  # Invite with default role (developer)
  hack admin users add colleague@hackutd.co

  # Invite as deployer
  hack admin users add ops@hackutd.co --role deployer`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			return adminAddUser(f, email, role)
		},
	}

	cmd.Flags().StringVarP(&role, "role", "r", "developer", "Role to assign (viewer, developer, deployer, admin)")

	return cmd
}

func adminAddUser(f *Factory, email, role string) error {
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

	path := fmt.Sprintf("/projects/%s/admin/users", hf.Project)
	var resp map[string]any
	if err := client.Post(context.Background(), path, map[string]string{
		"email": email,
		"role":  role,
	}, &resp); err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Added %s as %s\n", icon, tui.Bold.Render(email), role)

	return nil
}

func newAdminUsersRemoveCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove a user from the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return adminRemoveUser(f, args[0])
		},
	}

	return cmd
}

func adminRemoveUser(f *Factory, email string) error {
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

	if f.IO.IsInteractive() {
		var confirmed bool
		huh.NewConfirm().
			Title(fmt.Sprintf("Remove %s from the project?", email)).
			Value(&confirmed).
			Run()
		if !confirmed {
			fmt.Fprintln(f.IO.Out, "Cancelled.")
			return nil
		}
	}

	path := fmt.Sprintf("/projects/%s/admin/users/%s", hf.Project, email)
	var resp map[string]any
	if err := client.Delete(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Removed %s from project\n", icon, tui.Bold.Render(email))

	return nil
}

func newAdminRolesCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "Manage project roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return adminListRoles(f)
		},
	}

	cmd.AddCommand(newAdminRolesAssignCmd(f))

	return cmd
}

func adminListRoles(f *Factory) error {
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

	path := fmt.Sprintf("/projects/%s/admin/roles", hf.Project)
	var resp struct {
		Roles []struct {
			Name        string   `json:"name"`
			Permissions []string `json:"permissions"`
			BuiltIn     bool     `json:"built_in"`
		} `json:"roles"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Available Roles"))

	for _, r := range resp.Roles {
		fmt.Fprintf(f.IO.Out, "  %s\n", tui.Bold.Render(r.Name))
		for _, p := range r.Permissions {
			fmt.Fprintf(f.IO.Out, "    %s %s\n", tui.Dim.Render("•"), tui.Key.Render(p))
		}
		fmt.Fprintln(f.IO.Out)
	}

	return nil
}

func newAdminRolesAssignCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign <user> <role>",
		Short: "Assign a role to a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return adminAssignRole(f, args[0], args[1])
		},
	}

	return cmd
}

func adminAssignRole(f *Factory, email, role string) error {
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

	path := fmt.Sprintf("/projects/%s/admin/roles/assign", hf.Project)
	var resp map[string]any
	if err := client.Post(context.Background(), path, map[string]string{
		"email": email,
		"role":  role,
	}, &resp); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Assigned %s role to %s\n", icon, role, tui.Bold.Render(email))

	return nil
}

func newAdminAuditCmd(f *Factory) *cobra.Command {
	var (
		limit  int
		action string
		user   string
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			return adminAudit(f, limit, action, user)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum entries to show")
	cmd.Flags().StringVar(&action, "action", "", "Filter by action")
	cmd.Flags().StringVar(&user, "user", "", "Filter by user email")

	return cmd
}

func adminAudit(f *Factory, limit int, action, user string) error {
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

	_ = limit

	path := fmt.Sprintf("/projects/%s/admin/audit", hf.Project)
	var resp struct {
		Entries []struct {
			Timestamp string `json:"timestamp"`
			User      string `json:"user"`
			Action    string `json:"action"`
			Resource  string `json:"resource"`
			Details   string `json:"details"`
		} `json:"entries"`
	}
	if err := client.Get(context.Background(), path, &resp); err != nil {
		return fmt.Errorf("failed to fetch audit log: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Audit Log"))

	if len(resp.Entries) == 0 {
		fmt.Fprintln(f.IO.Out, tui.Dim.Render("  No entries found."))
		return nil
	}

	for _, e := range resp.Entries {
		if action != "" && e.Action != action {
			continue
		}
		if user != "" && e.User != user {
			continue
		}

		fmt.Fprintf(f.IO.Out, "  %s  %-20s  %s  %s  %s\n",
			tui.Dim.Render(e.Timestamp),
			tui.Bold.Render(e.User),
			tui.Key.Render(e.Action),
			e.Resource,
			tui.Dim.Render(e.Details),
		)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}
