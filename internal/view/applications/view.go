// Package applications implements the self-contained applications table view.
package applications

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/deploymodal"
)

// New creates a new applications view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	cols := columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTableHighlightOnly(styles))
	return &View{table: t, keys: keys, styles: styles}
}

func (a *View) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.table.SetWidth(width)
	a.table.SetHeight(height)
	a.table.SetColumns(ui.ScaleColumns(columns(), width))
}

// SetStatus implements view.StatusReceiver.
func (a *View) SetStatus(status *model.FullStatus) {
	a.status = status
	if status != nil {
		a.table.SetRows(rows(status.Applications, a.styles))
	}
}

// SetCharmSuggestions stores external charm suggestions for deploy modal.
func (a *View) SetCharmSuggestions(names []string) {
	a.charmhubSuggestions = append([]string(nil), names...)
}

// KeyHints returns the view-specific key hints for the header.
func (a *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(a.keys.Enter), Desc: "units"},
		{Key: bk(a.keys.ConfigNav), Desc: "config"},
		{Key: bk(a.keys.Deploy), Desc: "deploy"},
		{Key: bk(a.keys.LogsJump), Desc: "logs (app)"},
		{Key: bk(a.keys.LogsView), Desc: "logs"},
	}
}

// CopySelection implements view.Copyable.
func (a *View) CopySelection() string {
	if row := a.table.SelectedRow(); row != nil {
		return strings.Join(row, "\t")
	}
	return ""
}

func (a *View) Init() tea.Cmd { return nil }

func (a *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.deployModalOpen {
		switch msg := msg.(type) {
		case deploymodal.AppliedMsg:
			a.deployModalOpen = false
			return a, func() tea.Msg {
				return view.DeployRequestMsg{ModelName: msg.ModelName, Options: msg.Options}
			}
		case deploymodal.ClosedMsg:
			a.deployModalOpen = false
			return a, nil
		default:
			updated, cmd := a.deployModal.Update(msg)
			if dm, ok := updated.(*deploymodal.Modal); ok {
				a.deployModal = *dm
			}
			return a, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, a.keys.Deploy) {
			a.deployModal = deploymodal.New("", a.keys, a.styles, a.charmSuggestions(), a.applicationSuggestions())
			a.deployModal.SetSize(a.width, a.height)
			a.deployModalOpen = true
			return a, a.deployModal.BeginCharmEdit()
		}
		if key.Matches(msg, a.keys.Enter) {
			if row := a.table.SelectedRow(); row != nil {
				return a, func() tea.Msg {
					return view.NavigateMsg{Target: nav.UnitsView, Context: row[0]}
				}
			}
		}
		if key.Matches(msg, a.keys.LogsJump) {
			var filter *model.DebugLogFilter
			if row := a.table.SelectedRow(); row != nil {
				f := model.DebugLogFilter{Applications: []string{row[0]}}
				filter = &f
			}
			return a, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView, Filter: filter}
			}
		}
		if key.Matches(msg, a.keys.LogsView) {
			return a, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
			}
		}
		if key.Matches(msg, a.keys.ConfigNav) {
			if row := a.table.SelectedRow(); row != nil {
				return a, func() tea.Msg {
					return view.NavigateMsg{Target: nav.AppConfigView, Context: row[0]}
				}
			}
		}
	}
	var cmd tea.Cmd
	a.table, cmd = a.table.Update(msg)
	return a, cmd
}

func (a *View) View() tea.View {
	background := a.tableView()
	if a.deployModalOpen {
		return tea.NewView(a.deployModal.Render(background))
	}
	return tea.NewView(background)
}

func (a *View) tableView() string {
	cursor := a.table.Cursor()
	rows := a.table.Rows()
	if cursor >= 0 && cursor < len(rows) {
		original := rows[cursor]
		stripped := make(table.Row, len(original))
		for i, cell := range original {
			stripped[i] = ansi.Strip(cell)
		}
		if len(stripped) > 1 {
			stripped[1] = a.styles.StatusText(stripped[1])
		}
		rows[cursor] = stripped
		a.table.SetRows(rows)
		defer func() {
			rows[cursor] = original
			a.table.SetRows(rows)
		}()
	}
	return a.table.View()
}

func (a *View) charmSuggestions() []string {
	out := append([]string(nil), a.charmhubSuggestions...)
	if a.status == nil {
		return out
	}
	for _, app := range a.status.Applications {
		if app.Charm != "" {
			out = append(out, app.Charm)
		}
	}
	return out
}

func (a *View) applicationSuggestions() []string {
	if a.status == nil {
		return nil
	}
	out := make([]string, 0, len(a.status.Applications))
	for name := range a.status.Applications {
		out = append(out, name)
	}
	return out
}

func (a *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return nil, nil }
func (a *View) Leave() tea.Cmd                                { return nil }
