package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anishalle/hack/internal/envmanager"
	"github.com/spf13/cobra"
)

func newEnvCmd(deps *dependencies) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment secrets",
	}

	envCmd.AddCommand(&cobra.Command{
		Use:     "list <app>",
		Aliases: []string{"ls"},
		Short:   "List environments available for an app",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			environments, err := deps.env.ListEnvironments(context.Background(), args[0])
			if err != nil {
				return err
			}

			if len(environments) == 0 {
				fmt.Fprintln(deps.stdout, "No environments found.")
				return nil
			}

			for _, environment := range environments {
				fmt.Fprintln(deps.stdout, environment)
			}
			return nil
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "show <app> <environment>",
		Short: "Show the keys stored in an environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}

			keys := environment.Keys()
			fmt.Fprintf(deps.stdout, "app: %s\n", environment.App)
			fmt.Fprintf(deps.stdout, "environment: %s\n", environment.Name)
			fmt.Fprintf(deps.stdout, "project: %s\n", environment.Project)
			fmt.Fprintf(deps.stdout, "secret: %s\n", environment.SecretID)
			fmt.Fprintf(deps.stdout, "keys: %d\n", len(keys))
			for _, key := range keys {
				fmt.Fprintf(deps.stdout, "  %s\n", key)
			}

			return nil
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "export <app> <environment>",
		Short: "Write an environment into .env in the current directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}

			rendered, err := envmanager.RenderDotenv(environment.Values)
			if err != nil {
				return err
			}

			cwd, err := deps.workingDir()
			if err != nil {
				return err
			}

			target := filepath.Join(cwd, ".env")
			if err := confirmOverwrite(target, deps); err != nil {
				return err
			}

			if err := os.WriteFile(target, []byte(rendered), 0o600); err != nil {
				return err
			}

			fmt.Fprintf(deps.stdout, "Wrote %d values to %s.\n", len(environment.Values), target)
			return nil
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "load <app> <environment>",
		Short: "Print shell exports for eval",
		Long:  "Print shell exports for eval. Example: eval \"$(hack env load api prod)\"",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}

			rendered, err := envmanager.RenderExports(environment.Values)
			if err != nil {
				return err
			}

			_, err = io.WriteString(deps.stdout, rendered)
			return err
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "set <app> <environment> <key> [value]",
		Short: "Set one key in an environment secret",
		Long:  "Set one key in an environment secret. If value is omitted, Hack reads it from stdin or prompts for it interactively.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("expected at least <app> <environment> <key>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := readSecretValue(args[3:], deps)
			if err != nil {
				return err
			}

			environment, err := deps.env.SetValue(context.Background(), args[0], args[1], args[2], value)
			if err != nil {
				return err
			}

			fmt.Fprintf(deps.stdout, "Updated %s in %s/%s (%d total keys).\n", args[2], environment.App, environment.Name, len(environment.Values))
			return nil
		},
	})

	return envCmd
}

func confirmOverwrite(target string, deps *dependencies) error {
	_, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	if !deps.interactive {
		return fmt.Errorf("%s already exists; rerun interactively to confirm overwrite", target)
	}

	fmt.Fprintf(deps.stderr, "%s already exists. Overwrite? [y/N] ", target)

	reader := bufio.NewReader(deps.stdin)
	response, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		return nil
	}

	return fmt.Errorf("refused to overwrite %s", target)
}

func readSecretValue(raw []string, deps *dependencies) (string, error) {
	if len(raw) > 0 {
		return strings.Join(raw, " "), nil
	}

	if !deps.interactive {
		value, err := io.ReadAll(deps.stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(value), "\r\n"), nil
	}

	fmt.Fprint(deps.stderr, "value: ")
	reader := bufio.NewReader(deps.stdin)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimRight(value, "\r\n"), nil
}
