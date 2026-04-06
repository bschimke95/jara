// Package secrets implements the self-contained secrets table view.
package secrets

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new secrets view.
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

func (s *View) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.table.SetWidth(width)
	s.table.SetHeight(height)
	s.table.SetColumns(ui.ScaleColumns(Columns(), width))
}

// SetStatus implements view.StatusReceiver.
func (s *View) SetStatus(status *model.FullStatus) {
	s.status = status
	if status != nil {
		s.table.SetRows(Rows(status.Secrets))
	}
}

// KeyHints returns the view-specific key hints for the header.
func (s *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(s.keys.Enter), Desc: "detail"},
		{Key: bk(s.keys.LogsView), Desc: "logs"},
	}
}

// CopySelection implements view.Copyable.
func (s *View) CopySelection() string {
	if row := s.table.SelectedRow(); row != nil {
		return strings.Join(row, "\t")
	}
	return ""
}

func (s *View) Init() tea.Cmd { return nil }

func (s *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, s.keys.Enter):
			if s.status != nil && len(s.status.Secrets) > 0 {
				idx := s.table.Cursor()
				if idx >= 0 && idx < len(s.status.Secrets) {
					uri := s.status.Secrets[idx].URI
					return s, func() tea.Msg {
						return view.NavigateMsg{Target: nav.SecretDetailView, Context: uri}
					}
				}
			}
		case key.Matches(kp, s.keys.LogsView):
			return s, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
			}
		}
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return s, cmd
}

func (s *View) View() tea.View {
	return tea.NewView(s.table.View())
}

func (s *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return nil, nil }
func (s *View) Leave() tea.Cmd                                { return nil }
