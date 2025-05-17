package views

import (
	"fmt"
	"strings"

	"github.com/canonical/k8s/pkg/ui"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FrontpageModel represents the model for the frontpage view
type FrontpageModel struct {
	models        []JujuModel
	selectedModel int
	expanded      bool
}

// Init initializes the frontpage model
func NewFrontpageModel() FrontpageModel {
	return FrontpageModel{
		models:        []JujuModel{},
		selectedModel: -1,
		expanded:      false,
	}
}

// Update handles frontpage events
func (m *FrontpageModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ui.DefaultKeyMap.Quit):
			return tea.Quit
		case key.Matches(msg, ui.DefaultKeyMap.Refresh):
			// TODO: Refresh models
		case key.Matches(msg, ui.DefaultKeyMap.Select):
			if m.selectedModel >= 0 {
				m.expanded = !m.expanded
			}
		case key.Matches(msg, ui.DefaultKeyMap.Back):
			m.expanded = false
		}
	}
	return nil
}

// View renders the frontpage UI
func (m FrontpageModel) View() string {
	var sb strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Render("JARA - Juju Application Runner and Analyzer")

	// Status line
	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("Status: %d model(s) found", len(m.models)))

	// Keybindings help
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("%s, %s", ui.DefaultKeyMap.Quit.Help().Desc, ui.DefaultKeyMap.Refresh.Help()))

	// Models list
	models := ""
	for i, model := range m.models {
		style := lipgloss.NewStyle()
		if i == m.selectedModel {
			style = style.Foreground(lipgloss.Color("208")).Bold(true)
		}

		if m.expanded && i == m.selectedModel {
			// Expanded view showing applications
			modelContent := fmt.Sprintf("%s\n", style.Render(model.Name))
			for _, app := range model.Applications {
				modelContent += fmt.Sprintf("  - %s (%d units)\n", app.Name, app.Units)
			}
			models += modelContent
		} else {
			// Compact view
			models += fmt.Sprintf("%s\n", style.Render(model.Name))
		}
	}

	if models == "" {
		models = "No Juju models found"
	}

	sb.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s", header, status, models, help))
	return sb.String()
}
