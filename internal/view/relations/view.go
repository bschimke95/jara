// Package relations implements the split-pane relations view.
// Left pane: searchable relation list. Right pane: application + unit databags.
package relations

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/confirmodal"
)

// RelationDataMsg carries fetched databag contents for a relation.
type RelationDataMsg struct {
	RelationID int
	Data       *model.RelationData
}

// FetchRelationDataMsg requests the app to fetch relation data for a relation ID.
type FetchRelationDataMsg struct {
	RelationID int
}

// New creates a new relations view.
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

func (r *View) SetSize(width, height int) {
	r.width = width
	r.height = height
	leftWidth, _ := r.splitWidths()
	leftInner := leftWidth - 2
	r.table.SetWidth(leftInner)
	r.table.SetHeight(height - 3) // -2 for border, -1 for header row
	r.table.SetColumns(ui.ScaleColumns(Columns(), leftInner))
}

// SetStatus implements view.StatusReceiver.
func (r *View) SetStatus(status *model.FullStatus) {
	r.status = status
	if status != nil {
		r.applyFilter()
	}
}

// SetRelationData stores the fetched relation databag.
func (r *View) SetRelationData(data *model.RelationData) {
	r.relationData = data
	r.appScroll = 0
	r.unitScroll = 0
}

// KeyHints returns the view-specific key hints for the header.
func (r *View) KeyHints() []view.KeyHint {
	hints := []view.KeyHint{
		{Key: view.BindingKey(r.keys.DeleteRelation), Desc: "delete"},
		{Key: view.BindingKey(r.keys.LogsJump), Desc: "logs"},
	}
	if r.focus != focusTable {
		hints = append(hints, view.KeyHint{Key: "tab", Desc: "switch pane"})
	}
	return hints
}

// CopySelection implements view.Copyable.
func (r *View) CopySelection() string {
	return view.CopySelectedRow(r.table)
}

func (r *View) Init() tea.Cmd { return nil }

func (r *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── Confirm modal takes priority ──
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

	// ── Relation data response ──
	if rdMsg, ok := msg.(RelationDataMsg); ok {
		sel := r.selectedRelation()
		if sel != nil && sel.ID == rdMsg.RelationID {
			r.relationData = rdMsg.Data
			r.appScroll = 0
			r.unitScroll = 0
		}
		return r, nil
	}

	if kp, ok := msg.(tea.KeyPressMsg); ok {
		return r.handleKeyPress(kp)
	}

	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)
	return r, cmd
}

func (r *View) handleKeyPress(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab toggles focus between table, app data, and unit data.
	if key.Matches(kp, r.keys.Tab) {
		switch r.focus {
		case focusTable:
			r.focus = focusAppData
			r.table.SetStyles(ui.UnfocusedTableStyles(r.styles))
		case focusAppData:
			r.focus = focusUnitData
		case focusUnitData:
			r.focus = focusTable
			r.table.SetStyles(ui.StyledTableHighlightOnly(r.styles))
		}
		return r, nil
	}

	// ── Right pane focused: scroll or enter edit ──
	if r.focus == focusAppData || r.focus == focusUnitData {
		return r.handleRightPaneKey(kp)
	}

	// ── Left pane (table) focused ──
	prevCursor := r.table.Cursor()

	switch {
	case key.Matches(kp, r.keys.DeleteRelation):
		if sel := r.selectedRelation(); sel != nil && len(sel.Endpoints) >= 2 {
			a := formatEndpoint(sel.Endpoints[0])
			b := formatEndpoint(sel.Endpoints[1])
			r.deletingA = a
			r.deletingB = b
			r.confirmModal = confirmodal.New(
				r.keys,
				r.styles,
				"Delete Relation",
				fmt.Sprintf("Remove relation %s ↔ %s?", a, b),
			)
			r.confirmModal.SetSize(r.width, r.height)
			r.confirmOpen = true
			return r, nil
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

	var cmd tea.Cmd
	r.table, cmd = r.table.Update(kp)

	if r.table.Cursor() != prevCursor {
		if fetchCmd := r.requestRelationData(); fetchCmd != nil {
			return r, tea.Batch(cmd, fetchCmd)
		}
	}
	return r, cmd
}

func (r *View) handleRightPaneKey(kp tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(kp, r.keys.Down):
		r.scrollActive(1)
		return r, nil
	case key.Matches(kp, r.keys.Up):
		r.scrollActive(-1)
		return r, nil
	case key.Matches(kp, r.keys.PageDown):
		r.scrollActive(5)
		return r, nil
	case key.Matches(kp, r.keys.PageUp):
		r.scrollActive(-5)
		return r, nil
	case key.Matches(kp, r.keys.Back):
		r.focus = focusTable
		r.table.SetStyles(ui.StyledTableHighlightOnly(r.styles))
		return r, consumedKeyCmd()
	}
	return r, nil
}

func (r *View) scrollActive(delta int) {
	sel := r.selectedRelation()
	_, rightWidth := r.splitWidths()
	contentH := r.height - 2
	boxH := contentH/2 - 2 // inner height of each sub-box

	if r.focus == focusAppData {
		maxScroll := appDataContentLines(r.relationData, sel, rightWidth) - boxH
		if maxScroll < 0 {
			maxScroll = 0
		}
		r.appScroll += delta
		if r.appScroll < 0 {
			r.appScroll = 0
		}
		if r.appScroll > maxScroll {
			r.appScroll = maxScroll
		}
	} else {
		maxScroll := unitDataContentLines(r.relationData, sel) - boxH
		if maxScroll < 0 {
			maxScroll = 0
		}
		r.unitScroll += delta
		if r.unitScroll < 0 {
			r.unitScroll = 0
		}
		if r.unitScroll > maxScroll {
			r.unitScroll = maxScroll
		}
	}
}

func (r *View) View() tea.View {
	background := r.renderSplitPane()
	if r.confirmOpen {
		return tea.NewView(r.confirmModal.Render(background))
	}
	return tea.NewView(background)
}

func (r *View) Enter(_ view.NavigateContext) (tea.Cmd, error) {
	return r.requestRelationData(), nil
}

func (r *View) Leave() tea.Cmd {
	r.relationData = nil
	r.appScroll = 0
	r.unitScroll = 0
	r.focus = focusTable
	r.table.SetStyles(ui.StyledTableHighlightOnly(r.styles))
	return nil
}

// FilterStr returns the current filter string.
func (r *View) FilterStr() string {
	return r.filterStr
}

// SetFilter applies a new filter string and refreshes the table.
// It also clears any cached relation data and resets scroll positions so the
// right pane does not show stale content after the selection changes.
func (r *View) SetFilter(s string) {
	r.filterStr = s
	r.relationData = nil
	r.appScroll = 0
	r.unitScroll = 0
	r.applyFilter()
}

// ── Internal helpers ──

func (r *View) renderSplitPane() string {
	leftWidth, rightWidth := r.splitWidths()

	leftContent := view.PadToHeight(r.table.View(), r.height-2)
	leftBox := ui.BorderBox(leftContent, r.leftPaneTitle(), leftWidth, r.styles)

	sel := r.selectedRelation()
	rightBox := renderDatabagPane(r.relationData, sel, rightWidth, r.height, r.focus, r.appScroll, r.unitScroll, r.styles)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (r *View) splitWidths() (int, int) {
	left := r.width * 45 / 100
	if left < 30 {
		left = 30
	}
	right := r.width - left
	if right < 20 {
		right = 20
	}
	return left, right
}

func (r *View) leftPaneTitle() string {
	title := "Relations"
	if r.filterStr != "" {
		title += " [/" + r.filterStr + "]"
	}
	return title
}

func (r *View) selectedRelation() *model.Relation {
	idx := r.table.Cursor()
	rels := r.filteredRels
	if r.status == nil || idx < 0 || idx >= len(rels) {
		return nil
	}
	return &rels[idx]
}

func (r *View) applyFilter() {
	if r.status == nil {
		r.filteredRels = nil
		r.table.SetRows(nil)
		return
	}

	var rels []model.Relation
	if r.filterStr == "" {
		rels = append(rels, r.status.Relations...)
	} else {
		lower := strings.ToLower(r.filterStr)
		for _, rel := range r.status.Relations {
			if matchesFilter(rel, lower) {
				rels = append(rels, rel)
			}
		}
	}

	sort.Slice(rels, func(i, j int) bool { return rels[i].ID < rels[j].ID })

	r.filteredRels = rels
	r.table.SetRows(Rows(rels))
}

func matchesFilter(rel model.Relation, lower string) bool {
	for _, ep := range rel.Endpoints {
		if strings.Contains(strings.ToLower(ep.ApplicationName), lower) ||
			strings.Contains(strings.ToLower(ep.Name), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(rel.Interface), lower) ||
		strings.Contains(strings.ToLower(rel.Scope), lower) ||
		strings.Contains(strings.ToLower(rel.Status), lower) {
		return true
	}
	return false
}

func (r *View) requestRelationData() tea.Cmd {
	sel := r.selectedRelation()
	if sel == nil {
		r.relationData = nil
		return nil
	}
	id := sel.ID
	return func() tea.Msg {
		return FetchRelationDataMsg{RelationID: id}
	}
}

// consumedKeyCmd returns a no-op command that signals to the parent model
// that the key press was handled. This prevents the app's global-key handler
// from re-processing keys the view already consumed (e.g. Escape).
func consumedKeyCmd() tea.Cmd {
	return func() tea.Msg { return nil }
}
