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
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List environment versions",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return newUsageError(
					"`hack env list` does not take arguments",
					"Environment versions are discovered from Secret Manager names in the active project.",
					"hack env list",
					"hack list",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return printEnvironmentList(context.Background(), deps)
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "show <environment> [key]",
		Short: "Print an environment or one key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return newUsageError(
					"Environment/version required",
					"`hack env show` needs the environment/version name to fetch, such as prod, test, or dev.",
					"hack env show prod",
					"hack env show prod OPENAI_API_KEY",
				)
			}
			if len(args) > 2 {
				return newUsageError(
					"Too many arguments",
					"`hack env show` accepts an environment/version and optionally one key.",
					"hack env show prod",
					"hack env show prod DATABASE_URL",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), hiddenApp, args[0])
			if err != nil {
				return err
			}

			if len(args) == 2 {
				value, ok := environment.Values[args[1]]
				if !ok {
					return newUsageError(
						"Key not found",
						fmt.Sprintf("%s does not exist in %s.", args[1], args[0]),
						"hack env show "+args[0],
						"hack env set "+args[0]+" "+args[1]+" value",
					)
				}
				fmt.Fprintln(deps.stdout, value)
				return nil
			}

			rendered, err := envmanager.RenderDotenv(environment.Values)
			if err != nil {
				return err
			}
			_, err = io.WriteString(deps.stdout, rendered)
			return err
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "import <environment>",
		Short: "Pull an environment into .env",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(
					"Environment/version required",
					"`hack env import` pulls one remote environment/version into a local .env file.",
					"hack env import prod",
					"hack env import test",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), hiddenApp, args[0])
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

			fmt.Fprintln(deps.stdout, renderSuccess("Imported %d values from %s into %s.", len(environment.Values), environment.Name, target))
			return nil
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "export <file> <environment>",
		Short: "Push an env file into an environment",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return newUsageError(
					"File and environment/version required",
					"`hack env export` reads a local env file and pushes those keys into one remote environment/version.",
					"hack env export .env prod",
					"hack env export .env.local test",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			if !filepath.IsAbs(target) {
				cwd, err := deps.workingDir()
				if err != nil {
					return err
				}
				target = filepath.Join(cwd, target)
			}

			data, err := os.ReadFile(target)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return newUsageError(
						"Env file not found",
						"`hack env export` pushes keys from a local env file. Create the file first or import an existing environment.",
						"hack env import prod",
						"hack env export .env prod",
					)
				}
				return err
			}

			values, err := envmanager.ParseDotenv(data)
			if err != nil {
				return err
			}

			environment, err := deps.env.MergeValues(context.Background(), hiddenApp, args[1], values)
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.stdout, renderSuccess("Exported %d values from %s into %s.", len(values), target, environment.Name))
			return nil
		},
	})

	envCmd.AddCommand(&cobra.Command{
		Use:   "load <environment>",
		Short: "Print shell exports for eval",
		Long:  "Print shell exports for eval. Example: eval \"$(hack env load prod)\"",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(
					"Environment/version required",
					"`hack env load` prints shell export commands for one environment/version.",
					"eval \"$(hack env load prod)\"",
					"hack env load test",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			environment, err := deps.env.GetEnvironment(context.Background(), hiddenApp, args[0])
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
		Use:   "set <environment> <key> [value]",
		Short: "Set one key in an environment secret",
		Long:  "Set one key in an environment secret. If value is omitted, Hack reads it from stdin or prompts for it interactively.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return newUsageError(
					"Environment and key required",
					"`hack env set` needs an environment/version name and an env var key. Value can be passed as an argument, piped through stdin, or typed interactively.",
					"hack env set prod OPENAI_API_KEY sk-...",
					"printf %s \"$DATABASE_URL\" | hack env set prod DATABASE_URL",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := readSecretValue(args[2:], deps)
			if err != nil {
				return err
			}

			environment, err := deps.env.SetValue(context.Background(), hiddenApp, args[0], args[1], value)
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.stdout, renderSuccess("Updated %s in %s (%d total keys).", args[1], environment.Name, len(environment.Values)))
			return nil
		},
	})

	return envCmd
}

func printEnvironmentList(ctx context.Context, deps *dependencies) error {
	environments, err := deps.env.ListEnvironments(ctx, hiddenApp)
	if err != nil {
		return err
	}

	if len(environments) == 0 {
		fmt.Fprintln(deps.stdout, mutedStyle.Render("No environments found."))
		return nil
	}

	for _, environment := range environments {
		fmt.Fprintln(deps.stdout, environment)
	}
	return nil
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
