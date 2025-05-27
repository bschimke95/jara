package ui

import (
	"github.com/bschimke95/jara/pkg/env"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableColumn represents a column in the table
type TableColumn struct {
	Title string
	Width int
}

// TableRow represents a row in the table
type TableRow []string

// TableConfig holds configuration for creating a new table
type TableConfig struct {
	Theme   env.Theme
	Columns []TableColumn
	Rows    []TableRow
	Height  int
}

// Table is a wrapper around bubbles/table with VIM keybindings
type Table struct {
	table table.Model
}

// NewTable creates a new table with the given configuration
func NewTable(config TableConfig) *Table {
	// Convert columns to bubbles/table columns
	columns := make([]table.Column, len(config.Columns))
	for i, col := range config.Columns {
		columns[i] = table.Column{
			Title: col.Title,
			Width: col.Width,
		}
	}

	// Convert rows to bubbles/table rows
	rows := make([]table.Row, len(config.Rows))
	for i, row := range config.Rows {
		rows[i] = table.Row(row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(config.Height),
	)

	// Apply VIM keybindings
	// TODO(ben): Make this configurable and move to config
	t.KeyMap = table.KeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "u"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "d"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("home", "g"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("end", "G"),
		),
	}

	// Apply custom styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		Background(lipgloss.Color(config.Theme.HeaderBg)).
		Foreground(lipgloss.Color(config.Theme.HeaderText)).
		Bold(true)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color(config.Theme.SelectedText)).
		Background(lipgloss.Color(config.Theme.SelectedBg)).
		Bold(true)

	t.SetStyles(s)

	return &Table{
		table: t,
	}
}

// Update handles table update events and returns the updated table and a command if any
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	var cmd tea.Cmd

	// Update the underlying table
	var tableCmd tea.Cmd
	t.table, tableCmd = t.table.Update(msg)
	if tableCmd != nil {
		cmd = tea.Batch(cmd, tableCmd)
	}

	return t, cmd
}

// View renders the table
func (t *Table) View() string {
	return t.table.View()
}

// SelectedRow returns the currently selected row
func (t *Table) SelectedRow() table.Row {
	return t.table.SelectedRow()
}

// SelectedIndex returns the index of the currently selected row
func (t *Table) SelectedIndex() int {
	return t.table.Cursor()
}

// SetWidth sets the width of the table
func (t *Table) SetWidth(width int) {
	t.table.SetWidth(width)
}

// SetHeight sets the height of the table
func (t *Table) SetHeight(height int) {
	t.table.SetHeight(height)
}

// SetColumns sets the columns of the table
func (t *Table) SetColumns(columns []table.Column) {
	t.table.SetColumns(columns)
}

// SetRows sets the rows of the table
func (t *Table) SetRows(rows []table.Row) {
	t.table.SetRows(rows)
}
