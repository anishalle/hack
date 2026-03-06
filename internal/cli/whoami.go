package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func NewWhoamiCmd(f *Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated user",
		Long: `Display details about the currently authenticated user, including
email, roles, and active project context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhoami(f)
		},
	}
}

func runWhoami(f *Factory) error {
	cfg, err := f.Config()
	if err != nil {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	if !cfg.IsLoggedIn() {
		return fmt.Errorf("not logged in. Run 'hack login' first")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(16)
	valueStyle := lipgloss.NewStyle().Bold(true)

	fmt.Fprintf(f.IO.Out, "\n%s\n\n", headerStyle.Render("  Authenticated User"))
	fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Email:"), valueStyle.Render(cfg.Auth.Email))

	if cfg.ActiveProject != "" {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Active project:"), valueStyle.Render(cfg.ActiveProject))
	} else {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Active project:"), dimStyle.Render("none (run 'hack project switch')"))
	}

	if !cfg.IsTokenExpired() {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Session:"), successStyle.Render("active"))
	} else {
		fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Session:"), errorStyle.Render("expired (run 'hack login')"))
	}

	client, err := f.APIClient()
	if err == nil {
		var meResp struct {
			Email    string            `json:"email"`
			Name     string            `json:"name"`
			Projects map[string]string `json:"projects"`
		}
		if err := client.Get(context.Background(), "/me", &meResp); err == nil {
			if meResp.Name != "" {
				fmt.Fprintf(f.IO.Out, "  %s %s\n", labelStyle.Render("Name:"), valueStyle.Render(meResp.Name))
			}
			if len(meResp.Projects) > 0 {
				fmt.Fprintf(f.IO.Out, "\n  %s\n", headerStyle.Render("  Projects"))
				for name, role := range meResp.Projects {
					roleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
					fmt.Fprintf(f.IO.Out, "    %s %s\n", valueStyle.Render(name), roleStyle.Render("("+role+")"))
				}
			}
		}
	}

	fmt.Fprintln(f.IO.Out)
	return nil
}
