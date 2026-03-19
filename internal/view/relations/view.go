// Package relations implements the self-contained relations table view.
package relations

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/confirmodal"
)

// New creates a new relations view.
func New(keys ui.KeyMap) *View {
	cols := Columns()
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(ui.StyledTable())
	return &View{table: t, keys: keys}
}

func (r *View) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.table.SetWidth(width)
	r.table.SetHeight(height)
	r.table.SetColumns(ui.ScaleColumns(Columns(), width))
}

// SetStatus implements view.StatusReceiver.
func (r *View) SetStatus(status *model.FullStatus) {
	r.status = status
	if status != nil {
		r.table.SetRows(Rows(status.Relations))
	}
}

// KeyHints returns the view-specific key hints for the header.
func (r *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(r.keys.DeleteRelation), Desc: "delete"},
		{Key: bk(r.keys.LogsJump), Desc: "logs"},
		{Key: bk(r.keys.LogsView), Desc: "logs"},
	}
}

func (r *View) Init() tea.Cmd { return nil }

func (r *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if r.confirmOpen {
		switch msg.(type) {
		case confirmodal.ConfirmedMsg:
			r.confirmOpen = false
			a, b := r.deletingA, r.deletingB
			return r, func() tea.Msg {
				return view.DestroyRelationRequestMsg{EndpointA: a, EndpointB: b}
			}
		case confirmodal.CancelledMsg:
			r.confirmOpen = false
			return r, nil
		default:
			updated, cmd := r.confirmModal.Update(msg)
			if cm, ok := updated.(*confirmodal.Modal); ok {
				r.confirmModal = *cm
			}
			return r, cmd
		}
	}

	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, r.keys.DeleteRelation):
			if r.status != nil && len(r.status.Relations) > 0 {
				idx := r.table.Cursor()
				if idx >= 0 && idx < len(r.status.Relations) {
					rel := r.status.Relations[idx]
					if len(rel.Endpoints) >= 2 {
						a := fmt.Sprintf("%s:%s", rel.Endpoints[0].ApplicationName, rel.Endpoints[0].Name)
						b := fmt.Sprintf("%s:%s", rel.Endpoints[1].ApplicationName, rel.Endpoints[1].Name)
						r.deletingA = a
						r.deletingB = b
						r.confirmModal = confirmodal.New(
							r.keys,
							"Delete Relation",
							fmt.Sprintf("Remove relation %s ↔ %s?", a, b),
						)
						r.confirmModal.SetSize(r.width, r.height)
						r.confirmOpen = true
						return r, nil
					}
				}
			}
			return r, nil
		case key.Matches(kp, r.keys.LogsJump):
			return r, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
			}
		case key.Matches(kp, r.keys.LogsView):
			return r, func() tea.Msg {
				return view.NavigateMsg{Target: nav.DebugLogView}
			}
		}
	}
	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)
	return r, cmd
}

func (r *View) View() tea.View {
	background := r.table.View()
	if r.confirmOpen {
		return tea.NewView(r.confirmModal.Render(background))
	}
	return tea.NewView(background)
}

func (r *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return nil, nil }
func (r *View) Leave() tea.Cmd                                { return nil }
