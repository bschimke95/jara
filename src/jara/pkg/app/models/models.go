package models

import (
	"fmt"

	"github.com/bschimke95/jara/pkg/app/model"
	"github.com/bschimke95/jara/pkg/app/navigation"
	"github.com/bschimke95/jara/pkg/env"
	"github.com/bschimke95/jara/pkg/types/juju"
	"github.com/bschimke95/jara/pkg/ui"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	provider   env.Provider
	models     []juju.Model
	controller juju.Controller
	table      *ui.Table
}

func New(provider env.Provider) *Model {
	return &Model{
		provider: provider,
		table: ui.NewTable(ui.TableConfig{
			Theme: provider.Config().Theme,
			Columns: []ui.TableColumn{
				{Title: "Name", Width: 20},
				{Title: "Status", Width: 20},
				{Title: "UUID", Width: 36},
			},
			Rows:   []ui.TableRow{},
			Height: 10,
		}),
	}
}

func (m Model) Init() tea.Cmd {
	return m.refresh()
}

func (m *Model) View() string {
	layout := ui.NewLayout(ui.WithHeader(ui.HeaderInfo{
		Controller: m.controller,
		KeyHints:   env.DefaultKeyMap,
	}))

	// Calculate column widths
	colWidth := layout.Width / 3
	if colWidth < 10 { // Ensure minimum width
		colWidth = 10
	}

	// Create table config
	config := ui.TableConfig{
		Columns: []ui.TableColumn{
			{Title: "Name", Width: colWidth},
			{Title: "Status", Width: colWidth},
			{Title: "UUID", Width: colWidth},
		},
		Rows:   make([]ui.TableRow, 0, len(m.models)),
		Height: layout.Height,
	}

	// Convert models to table rows
	for _, model := range m.models {
		config.Rows = append(config.Rows, ui.TableRow{
			model.Name,
			model.Status,
			model.ModelUUID,
		})
	}

	// Only update table if we have models
	if len(m.models) > 0 {
		// Convert columns to table columns
		columns := make([]table.Column, len(config.Columns))
		for i, col := range config.Columns {
			columns[i] = table.Column{
				Title: col.Title,
				Width: col.Width,
			}
		}

		// Convert rows to table rows
		rows := make([]table.Row, len(config.Rows))
		for i, row := range config.Rows {
			rows[i] = table.Row(row)
		}

		// Set dimensions before content to avoid viewport issues
		// Use the full available width minus 2 for padding
		availableWidth := layout.Width - 2
		m.table.SetWidth(availableWidth)
		m.table.SetHeight(config.Height)
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
	}

	// Create a centered title
	title := fmt.Sprintf("\n  Models @ %s\n\n", m.controller.Name)

	// Get the table view
	tableView := m.table.View()

	// Calculate padding to center the table
	padding := (layout.Width - len(tableView)) / 2
	if padding < 0 {
		padding = 0
	}

	// Create a padded table view
	paddedTableView := lipgloss.NewStyle().
		PaddingLeft(padding).
		Render(tableView)

	// Combine title and padded table
	content := title + paddedTableView

	return layout.Render(content)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Get the selected model and switch to its view
			if m.table != nil && len(m.models) > 0 {
				// Get the selected row from the table
				selectedRow := m.table.SelectedRow()
				if len(selectedRow) >= 3 { // Ensure we have all columns
					// Find the model with matching ModelUUID
					for _, modelItem := range m.models {
						if modelItem.ModelUUID == selectedRow[2] { // ModelUUID is in the third column
							// Create a new model view with the selected model
							modelView := model.New(m.provider)
							// Set the model UUID to be loaded
							modelView.SetModelUUID(modelItem.ModelUUID)
							// Initialize the model view
							// Initialize the model view and return it
							// The app will handle adding the current page to history
							return nil, navigation.GoTo(modelView)
						}
					}
				}
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		// Update table width on window resize
		if m.table != nil {
			m.table.SetWidth(msg.Width)
		}
	case jujuMsg:
		m.models = msg.models
		m.controller = msg.controller
		return m, nil
	}

	// Update table if it exists
	if m.table != nil {
		var tableCmd tea.Cmd
		m.table, tableCmd = m.table.Update(msg)
		if tableCmd != nil {
			cmd = tea.Batch(cmd, tableCmd)
		}
	}

	return m, cmd
}

// TODO(ben): We probably don't need to refresh the controller here
// Maybe depends on if we cache in the JujuClient
type jujuMsg struct {
	controller juju.Controller
	models     []juju.Model
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		// Get the Juju client from the provider
		jujuClient := m.provider.JujuClient()
		if jujuClient == nil {
			// Return an empty model if we can't get the client
			return jujuMsg{
				controller: juju.Controller{},
				models:     []juju.Model{},
			}
		}

		models, err := jujuClient.Models(m.provider.Context())
		if err != nil {
			// Return an empty model if there's an error
			// TODO(ben): Handle error
			models = []juju.Model{}
		}

		controller, err := jujuClient.CurrentController(m.provider.Context())
		if err != nil {
			// Return an empty model if there's an error
			// TODO(ben): Handle error
			controller = juju.Controller{}
		}

		return jujuMsg{
			controller: controller,
			models:     models,
		}
	}
}
