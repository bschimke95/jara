// Package controllers implements the self-contained controllers table view.
package controllers

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new controllers view.
// pollFn is called from Enter to fetch data; it must return a tea.Cmd.
func New(keys ui.KeyMap, pollFn func() tea.Cmd) *View {
	cols := columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTable())
	return &View{table: t, keys: keys, pollFn: pollFn}
}

func (c *View) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.table.SetWidth(width)
	c.table.SetHeight(height)
	c.table.SetColumns(ui.ScaleColumns(columns(), width))
}

// SetControllers updates the controller list.
func (c *View) SetControllers(ctrls []model.Controller) {
	c.controllers = ctrls
	c.table.SetRows(controllerRows(ctrls))
}

// KeyHints returns the view-specific key hints for the header.
func (c *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(c.keys.Enter), Desc: "models"},
	}
}

func (c *View) Init() tea.Cmd { return nil }

func (c *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(UpdatedMsg); ok {
		c.SetControllers(msg.Controllers)
		return c, nil
	}
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(msg, c.keys.Enter) {
			if row := c.table.SelectedRow(); row != nil {
				controllerName := row[0]
				return c, func() tea.Msg {
					return view.NavigateMsg{Target: nav.ModelsView, Context: controllerName}
				}
			}
		}
	}
	var cmd tea.Cmd
	c.table, cmd = c.table.Update(msg)
	return c, cmd
}

func (c *View) View() tea.View {
	return tea.NewView(c.table.View())
}

func (c *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return c.pollFn(), nil }
func (c *View) Leave() tea.Cmd                                { return nil }
