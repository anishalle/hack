package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/anishalle/hack/internal/api"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
	codeStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8"))
)

func NewLoginCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with HackUTD",
		Long: `Authenticate with your HackUTD Google account using OAuth.

This opens a browser for authentication. In environments without a browser
(SSH, containers), a device code is displayed to enter at the verification URL.`,
		Example: `  # Interactive login
  hack login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(f)
		},
	}

	return cmd
}

func runLogin(f *Factory) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	if cfg.IsLoggedIn() && !cfg.IsTokenExpired() {
		fmt.Fprintf(f.IO.Out, "%s Already logged in as %s\n",
			successStyle.Render("✓"),
			boldStyle.Render(cfg.Auth.Email))
		return nil
	}

	client := api.NewClient(cfg.API.BaseURL, "")

	ctx := context.Background()
	var deviceResp api.DeviceCodeResponse
	if err := client.Post(ctx, "/auth/device-code", nil, &deviceResp); err != nil {
		return fmt.Errorf("failed to initiate login: %w\n\n%s",
			err,
			dimStyle.Render("Make sure the hack server is running. Check 'hack server' or contact your admin."))
	}

	fmt.Fprintf(f.IO.Out, "\n%s Opening browser for authentication...\n\n",
		boldStyle.Render("→"))

	fmt.Fprintf(f.IO.Out, "  Your code: %s\n\n",
		codeStyle.Render(deviceResp.UserCode))

	if err := browser.OpenURL(deviceResp.VerificationURL); err != nil {
		fmt.Fprintf(f.IO.Out, "  %s Could not open browser automatically.\n",
			dimStyle.Render("!"))
		fmt.Fprintf(f.IO.Out, "  Open this URL manually:\n")
		fmt.Fprintf(f.IO.Out, "  %s\n\n", deviceResp.VerificationURL)
	}

	var tokenResp api.TokenResponse
	pollErr := make(chan error, 1)

	go func() {
		interval := 5 * time.Second
		if deviceResp.Interval > 0 {
			interval = time.Duration(deviceResp.Interval) * time.Second
		}

		deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

		for time.Now().Before(deadline) {
			time.Sleep(interval)

			err := client.Post(ctx, "/auth/token", map[string]string{
				"device_code": deviceResp.DeviceCode,
			}, &tokenResp)

			if err == nil && tokenResp.AccessToken != "" {
				pollErr <- nil
				return
			}

			if apiErr, ok := err.(*api.APIError); ok {
				if apiErr.Code == 410 {
					pollErr <- fmt.Errorf("login expired, please try again")
					return
				}
			}
		}

		pollErr <- fmt.Errorf("login timed out, please try again")
	}()

	spinnerErr := spinner.New().
		Title(" Waiting for authentication...").
		Action(func() {
			err = <-pollErr
		}).
		Run()

	if spinnerErr != nil {
		return spinnerErr
	}

	if err != nil {
		fmt.Fprintf(f.IO.Out, "\n%s %s\n", errorStyle.Render("✗"), err)
		return err
	}

	cfg.Auth.AccessToken = tokenResp.AccessToken
	cfg.Auth.RefreshToken = tokenResp.RefreshToken
	cfg.Auth.Email = tokenResp.Email
	cfg.Auth.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("logged in but failed to save credentials: %w", err)
	}

	fmt.Fprintf(f.IO.Out, "\n%s Logged in as %s\n",
		successStyle.Render("✓"),
		boldStyle.Render(tokenResp.Email))

	return nil
}
