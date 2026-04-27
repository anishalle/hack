package cmd

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	accentColor = lipgloss.Color("#00A878")
	warnColor   = lipgloss.Color("#FFB000")
	errorColor  = lipgloss.Color("#FF4D6D")
	mutedColor  = lipgloss.Color("#7D8590")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	errorTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	errorBoxStyle = boxStyle.
			BorderForeground(errorColor)

	exampleStyle = lipgloss.NewStyle().
			Foreground(warnColor)
)

type usageError struct {
	title    string
	message  string
	examples []string
}

func (e usageError) Error() string {
	var builder strings.Builder
	builder.WriteString(errorTitleStyle.Render(e.title))
	if e.message != "" {
		builder.WriteString("\n\n")
		builder.WriteString(e.message)
	}
	if len(e.examples) > 0 {
		builder.WriteString("\n\n")
		builder.WriteString(mutedStyle.Render("Examples"))
		for _, example := range e.examples {
			builder.WriteString("\n  ")
			builder.WriteString(exampleStyle.Render(example))
		}
	}
	return errorBoxStyle.Render(builder.String())
}

func newUsageError(title, message string, examples ...string) error {
	return usageError{
		title:    title,
		message:  message,
		examples: examples,
	}
}

func renderSuccess(format string, args ...any) string {
	return titleStyle.Render(fmt.Sprintf(format, args...))
}
