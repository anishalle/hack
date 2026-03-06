package tui

import "github.com/charmbracelet/lipgloss"

var (
	AppName = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	Bold = lipgloss.NewStyle().
		Bold(true)

	Key = lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	Value = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	Added = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	Removed = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))

	Changed = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	EnvBadge = func(env string) string {
		var color lipgloss.Color
		switch env {
		case "prod", "production":
			color = "9"
		case "staging":
			color = "11"
		default:
			color = "10"
		}
		return lipgloss.NewStyle().
			Foreground(color).
			Bold(true).
			Render(env)
	}

	HelpKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Bold(true)

	HelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	SelectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	ActiveDot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Render("●")

	InactiveDot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("○")
)
