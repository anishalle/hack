package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/api"
	"github.com/anishalle/hack/internal/config"
	"github.com/anishalle/hack/internal/tui"
)

func NewProjectCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long: `Manage hack projects. Each project has its own hackfile.yaml, environments,
and team permissions.

Running 'hack project' without a subcommand opens the interactive project
switcher where you can browse and select your active project context.`,
		Example: `  # Open project switcher
  hack project

  # Initialize a new project
  hack project init

  # List all projects
  hack project list

  # Switch active project
  hack project switch

  # Show current project info
  hack project info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.IO.IsInteractive() {
				return cmd.Help()
			}
			return runProjectSwitch(f)
		},
	}

	cmd.AddCommand(newProjectInitCmd(f))
	cmd.AddCommand(newProjectListCmd(f))
	cmd.AddCommand(newProjectSwitchCmd(f))
	cmd.AddCommand(newProjectInfoCmd(f))
	cmd.AddCommand(newProjectRegisterCmd(f))

	return cmd
}

func newProjectInitCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new hack project",
		Long: `Create a hackfile.yaml in the current directory. If run interactively,
walks you through configuring environments, providers, and GCP project settings.`,
		Example: `  # Interactive initialization
  hack project init`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectInit(f)
		},
	}

	return cmd
}

func runProjectInit(f *Factory) error {
	if _, err := os.Stat("hackfile.yaml"); err == nil {
		return fmt.Errorf("hackfile.yaml already exists in this directory")
	}

	var (
		projectName string
		gcpProject  string
		envChoices  []string
	)

	if f.IO.IsInteractive() {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Project name").
					Description("A short identifier for your project").
					Placeholder("hackutd").
					Value(&projectName),
				huh.NewInput().
					Title("GCP project ID").
					Description("The Google Cloud project that holds your secrets").
					Placeholder("my-gcp-project").
					Value(&gcpProject),
				huh.NewMultiSelect[string]().
					Title("Environments").
					Options(
						huh.NewOption("dev", "dev"),
						huh.NewOption("staging", "staging"),
						huh.NewOption("prod", "prod"),
					).
					Value(&envChoices),
			),
		)

		if err := form.Run(); err != nil {
			return nil
		}
	} else {
		return fmt.Errorf("interactive mode required for project init")
	}

	if projectName == "" {
		return fmt.Errorf("project name is required")
	}
	if len(envChoices) == 0 {
		envChoices = []string{"dev", "prod"}
	}

	environments := make(map[string]*config.Environment)
	for _, env := range envChoices {
		environments[env] = &config.Environment{
			Deploy: &config.DeployConfig{
				Provider: "cloud-run",
				Project:  gcpProject,
				Region:   "us-central1",
			},
			DB: &config.DBConfig{
				Provider: "neon",
			},
			Auth: &config.AuthProviderConfig{
				Provider: "supertokens",
			},
		}
	}

	hf := &config.Hackfile{
		Project:      projectName,
		Version:      "1",
		Environments: environments,
		Secrets: config.SecretsConfig{
			Provider: "google-secret-manager",
			Project:  gcpProject,
			Prefix:   projectName,
		},
	}

	if err := hf.Save("."); err != nil {
		return err
	}

	cfg, err := f.Config()
	if err == nil {
		cwd, _ := os.Getwd()
		if cfg.Projects == nil {
			cfg.Projects = make(map[string]*config.ProjectConfig)
		}
		cfg.Projects[projectName] = &config.ProjectConfig{
			Path:       cwd,
			DefaultEnv: "dev",
		}
		cfg.ActiveProject = projectName
		cfg.Save()
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "\n%s Created hackfile.yaml for %s\n",
		icon, tui.Bold.Render(projectName))
	fmt.Fprintf(f.IO.Out, "  Environments: %v\n", envChoices)
	fmt.Fprintf(f.IO.Out, "  GCP Project: %s\n\n", gcpProject)
	fmt.Fprintf(f.IO.Out, "  Next steps:\n")
	fmt.Fprintf(f.IO.Out, "    %s Register with backend\n", tui.Dim.Render("1."))
	fmt.Fprintf(f.IO.Out, "       %s\n", tui.Dim.Render("hack project register"))
	fmt.Fprintf(f.IO.Out, "    %s Set your first env variable\n", tui.Dim.Render("2."))
	fmt.Fprintf(f.IO.Out, "       %s\n\n", tui.Dim.Render("hack env set dev MY_VAR=hello"))

	return nil
}

func newProjectListCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all accessible projects",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectList(f)
		},
	}

	return cmd
}

func runProjectList(f *Factory) error {
	cfg, err := f.Config()
	if err != nil || !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	var resp struct {
		Projects []api.ProjectInfo `json:"projects"`
	}
	if err := client.Get(context.Background(), "/projects", &resp); err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(resp.Projects) == 0 {
		fmt.Fprintln(f.IO.Out, tui.Dim.Render("  No projects found. Run 'hack project init' to create one."))
		return nil
	}

	fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Your Projects"))

	for _, p := range resp.Projects {
		indicator := tui.InactiveDot
		if p.Name == cfg.ActiveProject {
			indicator = tui.ActiveDot
		}

		roleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Fprintf(f.IO.Out, "  %s %s  %s\n",
			indicator,
			tui.Bold.Render(p.Name),
			roleStyle.Render("("+p.Role+")"),
		)
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newProjectSwitchCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "switch [project]",
		Short:   "Switch active project context",
		Aliases: []string{"use"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return switchProjectDirect(f, args[0])
			}
			return runProjectSwitch(f)
		},
	}

	return cmd
}

func switchProjectDirect(f *Factory, project string) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	cfg.ActiveProject = project
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to switch project: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Switched to project: %s\n", icon, tui.Bold.Render(project))
	return nil
}

func runProjectSwitch(f *Factory) error {
	if !f.IO.IsInteractive() {
		return fmt.Errorf("project name required (e.g., 'hack project switch hackutd')")
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	if !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	client, err := f.APIClient()
	if err != nil {
		return err
	}

	var resp struct {
		Projects []api.ProjectInfo `json:"projects"`
	}
	if err := client.Get(context.Background(), "/projects", &resp); err != nil {
		if len(cfg.Projects) > 0 {
			options := make([]huh.Option[string], 0)
			for name := range cfg.Projects {
				label := name
				if name == cfg.ActiveProject {
					label += " (active)"
				}
				options = append(options, huh.NewOption(label, name))
			}

			var selected string
			if err := huh.NewSelect[string]().
				Title("Switch project").
				Options(options...).
				Value(&selected).
				Run(); err != nil {
				return nil
			}

			return switchProjectDirect(f, selected)
		}
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(resp.Projects) == 0 {
		fmt.Fprintln(f.IO.Out, tui.Dim.Render("No projects found. Run 'hack project init' to create one."))
		return nil
	}

	options := make([]huh.Option[string], len(resp.Projects))
	for i, p := range resp.Projects {
		label := fmt.Sprintf("%s (%s)", p.Name, p.Role)
		if p.Name == cfg.ActiveProject {
			label += " ← active"
		}
		options[i] = huh.NewOption(label, p.Name)
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Switch project").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil
	}

	return switchProjectDirect(f, selected)
}

func newProjectInfoCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show current project details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectInfo(f)
		},
	}

	return cmd
}

func runProjectInfo(f *Factory) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	if cfg.ActiveProject == "" {
		return fmt.Errorf("no active project. Run 'hack project switch' first")
	}

	fmt.Fprintf(f.IO.Out, "\n  %s\n\n", tui.Title.Render("Project Info"))
	fmt.Fprintf(f.IO.Out, "  %s %s\n",
		tui.Dim.Render("Project:"),
		tui.Bold.Render(cfg.ActiveProject))

	hf, err := f.Hackfile()
	if err != nil {
		fmt.Fprintf(f.IO.Out, "  %s\n\n", tui.Dim.Render("(hackfile.yaml not found in current directory)"))
		return nil
	}

	fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Version:"), hf.Version)
	fmt.Fprintf(f.IO.Out, "  %s %s\n", tui.Dim.Render("Secrets:"), hf.Secrets.Provider)

	fmt.Fprintf(f.IO.Out, "\n  %s\n", tui.Bold.Render("Environments"))
	for name, env := range hf.Environments {
		fmt.Fprintf(f.IO.Out, "\n    %s\n", tui.EnvBadge(name))
		if env.Deploy != nil {
			fmt.Fprintf(f.IO.Out, "      %s %s (%s)\n",
				tui.Dim.Render("Deploy:"),
				env.Deploy.Provider,
				env.Deploy.Region)
		}
		if env.DB != nil {
			fmt.Fprintf(f.IO.Out, "      %s %s\n",
				tui.Dim.Render("DB:"),
				env.DB.Provider)
		}
		if env.Auth != nil {
			fmt.Fprintf(f.IO.Out, "      %s %s\n",
				tui.Dim.Render("Auth:"),
				env.Auth.Provider)
		}
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}

func newProjectRegisterCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a project with the hack backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectRegister(f)
		},
	}

	return cmd
}

func runProjectRegister(f *Factory) error {
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

	envNames := hf.EnvironmentNames()

	var resp api.ProjectInfo
	if err := client.Post(context.Background(), "/projects", map[string]any{
		"name":         hf.Project,
		"gcp_project":  hf.Secrets.Project,
		"environments": envNames,
	}, &resp); err != nil {
		return fmt.Errorf("failed to register project: %w", err)
	}

	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("✓")
	fmt.Fprintf(f.IO.Out, "%s Registered project %s with the hack backend\n",
		icon, tui.Bold.Render(hf.Project))

	return nil
}
