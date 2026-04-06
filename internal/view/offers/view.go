// Package offers implements the self-contained offers table view.
package offers

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new offers view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	cols := Columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTable(styles))
	return &View{table: t, keys: keys, styles: styles}
}

func (v *View) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetWidth(width)
	v.table.SetHeight(height)
	v.table.SetColumns(ui.ScaleColumns(Columns(), width))
}

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	return nil
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if dataMsg, ok := msg.(OffersDataMsg); ok {
		if dataMsg.Err != nil {
			v.err = dataMsg.Err
			return v, nil
		}
		v.err = nil
		v.hasData = true
		v.table.SetRows(Rows(dataMsg.Offers))
		return v, nil
	}

	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *View) View() tea.View {
	if v.err != nil {
		return tea.NewView(v.styles.ErrorStyle.Render("Error: " + v.err.Error()))
	}
	if !v.hasData {
		return tea.NewView(v.styles.MutedText.Render("Loading offers..."))
	}
	if len(v.table.Rows()) == 0 {
		return tea.NewView(v.styles.MutedText.Render("No application offers found."))
	}
	return tea.NewView(v.table.View())
}

func (v *View) Enter(_ view.NavigateContext) (tea.Cmd, error) {
	return func() tea.Msg { return FetchOffersMsg{} }, nil
}

func (v *View) Leave() tea.Cmd { return nil }
