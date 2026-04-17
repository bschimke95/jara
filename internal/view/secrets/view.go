// Package secrets implements the self-contained secrets table view.
package secrets

import (
	"fmt"
	"sort"
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
	t.SetStyles(ui.StyledTableHighlightOnly(styles))
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
	s.rebuildRows()
}

// SetFilter implements view.Filterable.
func (s *View) SetFilter(filter string) {
	s.filterStr = filter
	s.rebuildRows()
}

func (s *View) rebuildRows() {
	if s.status == nil {
		return
	}
	var filtered []model.Secret
	for _, sec := range s.status.Secrets {
		if s.appName != "" && secretOwnerApp(sec.Owner) != s.appName {
			continue
		}
		filtered = append(filtered, sec)
	}
	allRows := Rows(filtered)
	s.table.SetRows(view.FilterRows(allRows, 0, s.filterStr, s.styles.SearchHighlight))
}

// KeyHints returns the view-specific key hints for the header.
func (s *View) KeyHints() []view.KeyHint {
	return []view.KeyHint{
		{Key: view.BindingKey(s.keys.Enter), Desc: "detail"},
		{Key: view.BindingKey(s.keys.Inspect), Desc: "info"},
		{Key: view.BindingKey(s.keys.LogsView), Desc: "logs"},
		{Key: view.BindingKey(s.keys.EntitySwitch), Desc: "switch app"},
	}
}

// CopySelection implements view.Copyable.
func (s *View) CopySelection() string {
	return view.CopySelectedRow(s.table)
}

func (s *View) Init() tea.Cmd { return nil }

func (s *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, s.keys.Enter):
			if row := s.table.SelectedRow(); row != nil {
				uri := row[0]
				return s, func() tea.Msg {
					return view.NavigateMsg{Target: nav.SecretDetailView, Context: uri}
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

func (s *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	s.appName = ctx.Context
	s.rebuildRows()
	return nil, nil
}

func (s *View) Leave() tea.Cmd { return nil }

// SwitchTitle implements view.EntitySwitchable.
func (s *View) SwitchTitle() string { return "Switch Application" }

// SwitchableEntities implements view.EntitySwitchable.
func (s *View) SwitchableEntities() ([]string, string) {
	if s.status == nil {
		return nil, s.appName
	}
	seen := make(map[string]bool)
	for _, sec := range s.status.Secrets {
		app := secretOwnerApp(sec.Owner)
		if app != "" {
			seen[app] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, s.appName
}

// secretOwnerApp extracts the application name from a secret owner string.
// Owner is like "application-postgresql", "unit-mysql-0", or "model".
func secretOwnerApp(owner string) string {
	if strings.HasPrefix(owner, "application-") {
		return owner[len("application-"):]
	}
	if strings.HasPrefix(owner, "unit-") {
		rest := owner[5:]
		if idx := strings.LastIndex(rest, "-"); idx > 0 {
			return rest[:idx]
		}
	}
	return owner
}

// InspectSelection implements view.Inspectable.
func (s *View) InspectSelection() *view.InspectData {
	if s.status == nil || len(s.status.Secrets) == 0 {
		return nil
	}
	row := s.table.SelectedRow()
	if row == nil {
		return nil
	}
	uri := row[0]
	var sec model.Secret
	found := false
	for _, sc := range s.status.Secrets {
		if sc.URI == uri {
			sec = sc
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	expire := ""
	if sec.ExpireTime != nil {
		expire = sec.ExpireTime.Format("2006-01-02 15:04:05")
	}
	return &view.InspectData{
		Title: sec.Label,
		Fields: []view.InspectField{
			{Label: "URI", Value: sec.URI},
			{Label: "Label", Value: sec.Label},
			{Label: "Description", Value: sec.Description},
			{Label: "Owner", Value: sec.Owner},
			{Label: "Rotate Policy", Value: sec.RotatePolicy},
			{Label: "Revision", Value: fmt.Sprintf("%d", sec.Revision)},
			{Label: "Backend", Value: sec.Backend},
			{Label: "Auto Prune", Value: fmt.Sprintf("%v", sec.AutoPrune)},
			{Label: "Created", Value: sec.CreateTime.Format("2006-01-02 15:04:05")},
			{Label: "Updated", Value: sec.UpdateTime.Format("2006-01-02 15:04:05")},
			{Label: "Expires", Value: expire},
		},
	}
}
