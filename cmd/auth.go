package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/anishalle/hack/internal/envmanager"
	"github.com/spf13/cobra"
)

func newAuthCmd(deps *dependencies) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Hack's Google Cloud auth context",
	}

	authCmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Log into Google Cloud through gcloud",
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
		Use:   "use <project-id>",
		Short: "Set the active Google Cloud project for Hack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deps.auth.UseProject(context.Background(), args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(deps.stdout, "Active project is now %s.\n", status.StoredProject)
			printAuthStatus(deps.stdout, status)
			return nil
		},
	})

	return authCmd
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
