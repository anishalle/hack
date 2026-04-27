package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/anishalle/hack/internal/cloud"
	"github.com/anishalle/hack/internal/envmanager"
	"github.com/spf13/cobra"
)

func newAuthCmd(deps *dependencies) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Hack's Google Cloud auth context",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return newUsageError(
					"`hack auth` does not take arguments",
					"Run `hack auth` for the guided login/project flow, or use one of the explicit subcommands.",
					"hack auth",
					"hack auth login",
					"hack auth projects",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthFlow(context.Background(), deps)
		},
	}

	authCmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Log into Google Cloud through gcloud",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deps.auth.Login(context.Background(), deps.stdin, deps.stdout, deps.stderr)
			if err != nil {
				return err
			}

			printAuthStatus(deps.stdout, status)
			return nil
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the active Hack and gcloud auth context",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deps.auth.Status(context.Background())
			if err != nil {
				return err
			}

			printAuthStatus(deps.stdout, status)
			return nil
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "projects",
		Short: "Pick the active Google Cloud project for Hack",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := selectProject(context.Background(), deps)
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.stdout, renderSuccess("Active project is now %s.", status.StoredProject))
			return nil
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Log out of the active gcloud account and clear Hack auth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deps.auth.Logout(context.Background())
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.stdout, renderSuccess("Logged out and cleared Hack auth context."))
			printAuthStatus(deps.stdout, status)
			return nil
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "use <project-id>",
		Short: "Set the active Google Cloud project for Hack",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(
					"Project id required",
					"`hack auth use` saves one Google Cloud project id into Hack's local config. It does not change global gcloud config.",
					"hack auth use hackutd-prod",
					"hack auth projects",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deps.auth.UseProject(context.Background(), args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.stdout, renderSuccess("Active project is now %s.", status.StoredProject))
			printAuthStatus(deps.stdout, status)
			return nil
		},
	})

	return authCmd
}

func runAuthFlow(ctx context.Context, deps *dependencies) error {
	status, err := deps.auth.Status(ctx)
	if err != nil {
		return err
	}

	if status.DetectedAccount == "" {
		ok, err := confirmAuthSetup(deps)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(deps.stdout, mutedStyle.Render("Auth setup cancelled."))
			return nil
		}

		status, err = deps.auth.Login(ctx, deps.stdin, deps.stdout, deps.stderr)
		if err != nil {
			return err
		}
	}

	status, err = selectProject(ctx, deps)
	if err != nil {
		return err
	}

	fmt.Fprintln(deps.stdout, renderSuccess("Active project is now %s.", status.StoredProject))
	printAuthStatus(deps.stdout, status)
	return nil
}

func confirmAuthSetup(deps *dependencies) (bool, error) {
	if !deps.interactive {
		return false, newUsageError(
			"Not logged in",
			"Hack could not find an active gcloud account. Run the guided auth flow in an interactive terminal.",
			"hack auth",
			"hack auth login",
		)
	}

	fmt.Fprint(deps.stderr, titleStyle.Render("Not logged in, set auth?")+" "+mutedStyle.Render("(Y, n)")+" ")
	reader := bufio.NewReader(deps.stdin)
	response, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes", nil
}

func selectProject(ctx context.Context, deps *dependencies) (envmanager.AuthStatus, error) {
	projects, err := deps.auth.ListProjects(ctx)
	if err != nil {
		return envmanager.AuthStatus{}, err
	}

	picker := deps.pickProject
	if picker == nil {
		picker = runProjectPicker
	}

	project, err := picker(ctx, projects, deps)
	if err != nil {
		return envmanager.AuthStatus{}, err
	}
	if project.ID == "" {
		return envmanager.AuthStatus{}, fmt.Errorf("selected project did not include a project id")
	}

	return deps.auth.UseProject(ctx, project.ID)
}

func projectLabel(project cloud.Project) string {
	if project.Name == "" || project.Name == project.ID {
		return project.ID
	}
	return fmt.Sprintf("%s (%s)", project.ID, project.Name)
}

func printAuthStatus(w io.Writer, status envmanager.AuthStatus) {
	gcloudStatus := "missing"
	if status.GCloudAvailable {
		gcloudStatus = "available"
	}

	fmt.Fprintf(w, "gcloud: %s\n", gcloudStatus)
	if status.DetectedAccount != "" {
		fmt.Fprintf(w, "account: %s\n", status.DetectedAccount)
	} else if status.StoredAccount != "" {
		fmt.Fprintf(w, "account: %s (stored)\n", status.StoredAccount)
	} else {
		fmt.Fprintln(w, "account: not set")
	}

	if status.StoredProject != "" {
		fmt.Fprintf(w, "project: %s\n", status.StoredProject)
	} else {
		fmt.Fprintln(w, "project: not set")
	}

	if status.DetectedProject != "" && status.DetectedProject != status.StoredProject {
		fmt.Fprintf(w, "gcloud project: %s\n", status.DetectedProject)
	}

	fmt.Fprintf(w, "config: %s\n", status.ConfigPath)
}
