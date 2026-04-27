package cmd

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/anishalle/hack/internal/cloud"
)

type projectItem struct {
	project cloud.Project
}

func (i projectItem) FilterValue() string {
	return i.project.ID + " " + i.project.Name
}

func (i projectItem) Title() string {
	return i.project.ID
}

func (i projectItem) Description() string {
	if i.project.Name == "" {
		return "Google Cloud project"
	}
	return i.project.Name
}

type projectPickerModel struct {
	list      list.Model
	selected  cloud.Project
	cancelled bool
}

func newProjectPickerModel(projects []cloud.Project) projectPickerModel {
	items := make([]list.Item, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectItem{project: project})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(accentColor).
		BorderLeftForeground(accentColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(accentColor)
	delegate.SetSpacing(0)

	model := list.New(items, delegate, 76, 18)
	model.Title = "Select a Google Cloud project"
	model.SetShowStatusBar(false)
	model.SetShowPagination(true)
	model.SetShowHelp(true)
	model.SetFilteringEnabled(true)
	model.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Padding(0, 1)

	return projectPickerModel{list: model}
}

func (m projectPickerModel) Init() tea.Cmd {
	return nil
}

func (m projectPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(projectItem)
			if ok {
				m.selected = item.project
				return m, tea.Quit
			}
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	next, cmd := m.list.Update(msg)
	m.list = next
	return m, cmd
}

func (m projectPickerModel) View() tea.View {
	content := titleStyle.Render("Hack auth") + "\n" +
		mutedStyle.Render("Type to search. Enter selects. Esc cancels.") + "\n\n" +
		m.list.View()
	return tea.NewView(boxStyle.Render(content))
}

func runProjectPicker(ctx context.Context, projects []cloud.Project, deps *dependencies) (cloud.Project, error) {
	if len(projects) == 0 {
		return cloud.Project{}, newUsageError(
			"No projects found",
			"gcloud did not return any active projects for the current account.",
			"hack auth login",
			"gcloud projects list",
		)
	}

	if !deps.interactive {
		return cloud.Project{}, newUsageError(
			"Project selection needs a terminal",
			"Run this command interactively, or set a project directly.",
			"hack auth use hackutd-prod",
			"hack auth projects",
		)
	}

	program := tea.NewProgram(
		newProjectPickerModel(projects),
		tea.WithContext(ctx),
		tea.WithInput(deps.stdin),
		tea.WithOutput(deps.stdout),
	)
	finalModel, err := program.Run()
	if err != nil {
		return cloud.Project{}, err
	}

	model, ok := finalModel.(projectPickerModel)
	if !ok {
		return cloud.Project{}, fmt.Errorf("project picker returned unexpected model")
	}
	if model.cancelled {
		return cloud.Project{}, fmt.Errorf("project selection cancelled")
	}
	if model.selected.ID == "" {
		return cloud.Project{}, fmt.Errorf("no project selected")
	}

	return model.selected, nil
}
