package view

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/render"
	"github.com/bschimke95/jara/internal/ui"
)

// ControllersUpdatedMsg is sent when fresh controller data arrives.
type ControllersUpdatedMsg struct {
	Controllers []model.Controller
}

// Controllers is the Bubble Tea model for the controllers table view.
type Controllers struct {
	table       table.Model
	keys        ui.KeyMap
	width       int
	height      int
	controllers []model.Controller
}

// NewControllers creates a new controllers view.
func NewControllers() *Controllers {
	cols := render.ControllerColumns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(styledTable())
	return &Controllers{table: t, keys: ui.DefaultKeyMap()}
}

func (c *Controllers) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.table.SetWidth(width)
	c.table.SetHeight(height)
	c.table.SetColumns(render.ScaleColumns(render.ControllerColumns(), width))
}

// SetStatus is a no-op for the controllers view — it uses SetControllers instead.
func (c *Controllers) SetStatus(_ *model.FullStatus) {}

// SetControllers updates the controller list.
func (c *Controllers) SetControllers(controllers []model.Controller) {
	c.controllers = controllers
	c.table.SetRows(render.ControllerRows(controllers))
}

func (c *Controllers) Init() tea.Cmd { return nil }

func (c *Controllers) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(ControllersUpdatedMsg); ok {
		c.SetControllers(msg.Controllers)
		return c, nil
	}
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(msg, c.keys.Enter) {
			if row := c.table.SelectedRow(); row != nil {
				controllerName := row[0]
				return c, func() tea.Msg {
					return NavigateMsg{Target: nav.ModelsView, Context: controllerName}
				}
			}
		}
	}
	var cmd tea.Cmd
	c.table, cmd = c.table.Update(msg)
	return c, cmd
}

func (c *Controllers) View() tea.View {
	return tea.NewView(c.table.View())
}
