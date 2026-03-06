package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type EnvVar struct {
	Key   string
	Value string
}

type EnvDiff struct {
	Added   map[string]string
	Removed map[string]string
	Changed map[string][2]string
}

type EnvViewModel struct {
	Environment string
	Variables   []EnvVar
	cursor      int
	showValues  bool
	width       int
	height      int
	quitting    bool
}

type envKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Quit   key.Binding
}

var envKeys = envKeyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Toggle: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle values")),
	Quit:   key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
}

func NewEnvViewModel(env string, vars map[string]string) EnvViewModel {
	sorted := make([]EnvVar, 0, len(vars))
	for k, v := range vars {
		sorted = append(sorted, EnvVar{Key: k, Value: v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})

	return EnvViewModel{
		Environment: env,
		Variables:   sorted,
	}
}

func (m EnvViewModel) Init() tea.Cmd {
	return nil
}

func (m EnvViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, envKeys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, envKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, envKeys.Down):
			if m.cursor < len(m.Variables)-1 {
				m.cursor++
			}
		case key.Matches(msg, envKeys.Toggle):
			m.showValues = !m.showValues
		}
	}

	return m, nil
}

func (m EnvViewModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	header := fmt.Sprintf("  Environment Variables — %s", EnvBadge(m.Environment))
	b.WriteString(Title.Render(header))
	b.WriteString("\n")
	b.WriteString(Dim.Render(fmt.Sprintf("  %d variables", len(m.Variables))))
	b.WriteString("\n\n")

	if len(m.Variables) == 0 {
		b.WriteString(Dim.Render("  No environment variables found.\n"))
		b.WriteString(Dim.Render("  Use 'hack env set' to add variables.\n"))
	}

	maxKeyLen := 0
	for _, v := range m.Variables {
		if len(v.Key) > maxKeyLen {
			maxKeyLen = len(v.Key)
		}
	}

	visibleStart := 0
	visibleCount := m.height - 8
	if visibleCount < 5 {
		visibleCount = 20
	}
	if m.cursor >= visibleStart+visibleCount {
		visibleStart = m.cursor - visibleCount + 1
	}
	if m.cursor < visibleStart {
		visibleStart = m.cursor
	}

	for i := visibleStart; i < len(m.Variables) && i < visibleStart+visibleCount; i++ {
		v := m.Variables[i]

		cursor := "  "
		keyStyle := Key
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("▸ ")
			keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
		}

		paddedKey := v.Key + strings.Repeat(" ", maxKeyLen-len(v.Key))

		if m.showValues {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", cursor, keyStyle.Render(paddedKey), Value.Render(v.Value)))
		} else {
			masked := strings.Repeat("•", min(len(v.Value), 20))
			b.WriteString(fmt.Sprintf("%s%s = %s\n", cursor, keyStyle.Render(paddedKey), Dim.Render(masked)))
		}
	}

	b.WriteString("\n")

	help := fmt.Sprintf("  %s %s  %s %s  %s %s",
		HelpKey.Render("↑↓"), HelpDesc.Render("navigate"),
		HelpKey.Render("v"), HelpDesc.Render("toggle values"),
		HelpKey.Render("q"), HelpDesc.Render("quit"))
	b.WriteString(help)
	b.WriteString("\n")

	return b.String()
}

func RenderEnvDiff(env1, env2 string, diff EnvDiff) string {
	var b strings.Builder

	header := fmt.Sprintf("  Diff: %s ↔ %s", EnvBadge(env1), EnvBadge(env2))
	b.WriteString(Title.Render(header))
	b.WriteString("\n\n")

	if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Changed) == 0 {
		b.WriteString(Dim.Render("  No differences found.\n"))
		return b.String()
	}

	if len(diff.Added) > 0 {
		b.WriteString(Added.Render(fmt.Sprintf("  + %d added (in %s only)\n", len(diff.Added), env2)))
		keys := sortedKeys(diff.Added)
		for _, k := range keys {
			b.WriteString(Added.Render(fmt.Sprintf("    + %s\n", k)))
		}
		b.WriteString("\n")
	}

	if len(diff.Removed) > 0 {
		b.WriteString(Removed.Render(fmt.Sprintf("  - %d removed (in %s only)\n", len(diff.Removed), env1)))
		keys := sortedKeys(diff.Removed)
		for _, k := range keys {
			b.WriteString(Removed.Render(fmt.Sprintf("    - %s\n", k)))
		}
		b.WriteString("\n")
	}

	if len(diff.Changed) > 0 {
		b.WriteString(Changed.Render(fmt.Sprintf("  ~ %d changed\n", len(diff.Changed))))
		for k, vals := range diff.Changed {
			b.WriteString(Changed.Render(fmt.Sprintf("    ~ %s\n", k)))
			b.WriteString(Removed.Render(fmt.Sprintf("      - %s\n", vals[0])))
			b.WriteString(Added.Render(fmt.Sprintf("      + %s\n", vals[1])))
		}
	}

	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
