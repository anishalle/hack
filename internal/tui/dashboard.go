package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardItem struct {
	Title       string
	Description string
	Command     string
}

type DashboardModel struct {
	Project  string
	Items    []DashboardItem
	cursor   int
	width    int
	height   int
	selected string
	quitting bool
}

type dashKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Quit   key.Binding
}

var dashKeys = dashKeyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Quit:  key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
}

func NewDashboardModel(project string) DashboardModel {
	items := []DashboardItem{
		{Title: "Environment Variables", Description: "Manage env vars across environments", Command: "env"},
		{Title: "Deployments", Description: "Deploy, monitor, and manage services", Command: "deploy"},
		{Title: "Database", Description: "Connect, migrate, and manage databases", Command: "db"},
		{Title: "Authentication", Description: "Manage auth provider and users", Command: "auth"},
		{Title: "Project Settings", Description: "View project config and team", Command: "project info"},
		{Title: "Admin Panel", Description: "Manage users, roles, and audit log", Command: "admin"},
	}

	return DashboardModel{
		Project: project,
		Items:   items,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, dashKeys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, dashKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, dashKeys.Down):
			if m.cursor < len(m.Items)-1 {
				m.cursor++
			}
		case key.Matches(msg, dashKeys.Enter):
			m.selected = m.Items[m.cursor].Command
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m DashboardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Render("  hack")
	version := Dim.Render(" — HackUTD CLI")

	b.WriteString("\n" + logo + version + "\n")

	if m.Project != "" {
		b.WriteString(fmt.Sprintf("  Project: %s\n", Bold.Render(m.Project)))
	}
	b.WriteString("\n")

	for i, item := range m.Items {
		cursor := "  "
		titleStyle := NormalItem
		descStyle := Dim

		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("▸ ")
			titleStyle = SelectedItem
			descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		}

		b.WriteString(fmt.Sprintf("%s%s\n", cursor, titleStyle.Render(item.Title)))
		b.WriteString(fmt.Sprintf("    %s\n\n", descStyle.Render(item.Description)))
	}

	help := fmt.Sprintf("  %s %s  %s %s  %s %s",
		HelpKey.Render("↑↓"), HelpDesc.Render("navigate"),
		HelpKey.Render("enter"), HelpDesc.Render("select"),
		HelpKey.Render("q"), HelpDesc.Render("quit"))
	b.WriteString(help + "\n")

	return b.String()
}

func (m DashboardModel) Selected() string {
	return m.selected
}
