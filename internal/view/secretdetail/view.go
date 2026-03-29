// Package secretdetail implements the drill-down detail view for a single secret.
package secretdetail

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
	"github.com/bschimke95/jara/internal/view/revealmodal"
)

// New creates a new secret detail view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	revCols := RevisionColumns()
	rt := table.New(
		table.WithColumns(revCols),
		table.WithFocused(true),
		table.WithHeight(5),
	)
	rt.SetStyles(ui.StyledTable(styles))

	accCols := AccessColumns()
	at := table.New(
		table.WithColumns(accCols),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	at.SetStyles(ui.UnfocusedTableStyles(styles))

	return &View{
		revTable:    rt,
		accessTable: at,
		keys:        keys,
		styles:      styles,
	}
}

func (v *View) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.recalcLayout()
}

// SetStatus implements view.StatusReceiver.
func (v *View) SetStatus(status *model.FullStatus) {
	v.status = status
	v.refreshSecret()
}

// KeyHints returns the view-specific key hints for the header.
func (v *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(v.keys.Decode), Desc: "decode"},
		{Key: bk(v.keys.Tab), Desc: "switch"},
		{Key: bk(v.keys.Back), Desc: "back"},
	}
}

func (v *View) Init() tea.Cmd { return nil }

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle reveal modal messages when open.
	if v.revealOpen {
		switch msg.(type) {
		case revealmodal.ClosedMsg:
			v.revealOpen = false
			return v, nil
		}
		_, cmd := v.revealModal.Update(msg)
		return v, cmd
	}

	// Handle the reveal response from the app layer.
	if resp, ok := msg.(view.RevealSecretResponseMsg); ok {
		modal := revealmodal.New(v.keys, v.styles, "Secret Content", resp.Values)
		modal.SetSize(v.width, v.height)
		v.revealModal = modal
		v.revealOpen = true
		return v, nil
	}

	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, v.keys.Decode):
			if v.secretURI != "" {
				rev := v.selectedRevisionNumber()
				return v, func() tea.Msg {
					return view.RevealSecretRequestMsg{URI: v.secretURI, Revision: rev}
				}
			}
			return v, nil
		case key.Matches(kp, v.keys.Tab):
			v.focusAccess = !v.focusAccess
			if v.focusAccess {
				v.revTable.SetStyles(ui.UnfocusedTableStyles(v.styles))
				v.accessTable.SetStyles(ui.StyledTable(v.styles))
				v.revTable.Blur()
				v.accessTable.Focus()
			} else {
				v.revTable.SetStyles(ui.StyledTable(v.styles))
				v.accessTable.SetStyles(ui.UnfocusedTableStyles(v.styles))
				v.accessTable.Blur()
				v.revTable.Focus()
			}
			return v, nil
		}
	}
	var cmd tea.Cmd
	if v.focusAccess {
		v.accessTable, cmd = v.accessTable.Update(msg)
	} else {
		v.revTable, cmd = v.revTable.Update(msg)
	}
	return v, cmd
}

func (v *View) View() tea.View {
	if v.secret == nil {
		return tea.NewView("No secret selected")
	}

	leftW, rightW := v.splitWidths()

	// ── Left pane: metadata details ──
	leftContent := v.renderMetadata()
	titleStyle := lipgloss.NewStyle().Foreground(v.styles.BorderTitleColor).Bold(true)
	leftBox := ui.BorderBoxRawTitle(
		padToHeight(leftContent, v.height-2),
		titleStyle.Render(" Details "),
		leftW,
		v.styles,
	)

	// ── Right pane: revisions (top) + access (bottom) ──
	halfH := (v.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	revBox := ui.BorderBoxRawTitle(
		padToHeight(v.revTable.View(), halfH),
		titleStyle.Render(" Revisions "),
		rightW,
		v.styles,
	)
	accBox := ui.BorderBoxRawTitle(
		padToHeight(v.accessTable.View(), halfH),
		titleStyle.Render(" Access "),
		rightW,
		v.styles,
	)
	rightBox := lipgloss.JoinVertical(lipgloss.Left, revBox, accBox)

	bg := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	if v.revealOpen {
		return tea.NewView(v.revealModal.Render(bg))
	}

	return tea.NewView(bg)
}

// renderMetadata builds the metadata field lines for the left pane.
// The revision-specific fields reflect the currently selected revision row.
func (v *View) renderMetadata() string {
	labelStyle := lipgloss.NewStyle().Foreground(v.styles.Muted)
	valueStyle := lipgloss.NewStyle().Foreground(v.styles.Title)
	field := func(label, value string) string {
		return labelStyle.Render(label+": ") + valueStyle.Render(value)
	}

	s := v.secret
	var lines []string
	lines = append(lines,
		field("URI", s.URI),
		field("Label", s.Label),
		field("Description", s.Description),
		field("Owner", s.Owner),
		field("Rotation", s.RotatePolicy),
		field("Auto-Prune", fmt.Sprintf("%v", s.AutoPrune)),
	)
	if s.NextRotateTime != nil {
		lines = append(lines, field("Next Rotation", s.NextRotateTime.Format("2006-01-02 15:04:05")))
	}

	// Revision-specific section.
	rev := v.selectedRevision()
	if rev != nil {
		lines = append(lines, field("Revision", fmt.Sprintf("%d", rev.Revision)))
		lines = append(lines, field("Created", rev.CreatedAt.Format("2006-01-02 15:04:05")))
		if rev.ExpiredAt != nil {
			lines = append(lines, field("Expired", rev.ExpiredAt.Format("2006-01-02 15:04:05")))
		}
		if rev.Backend != "" {
			lines = append(lines, field("Backend", rev.Backend))
		}
	} else {
		lines = append(lines, field("Revision", fmt.Sprintf("%d (latest)", s.Revision)))
		lines = append(lines, field("Created", s.CreateTime.Format("2006-01-02 15:04:05")))
		lines = append(lines, field("Updated", s.UpdateTime.Format("2006-01-02 15:04:05")))
	}

	return strings.Join(lines, "\n")
}

// selectedRevision returns the revision matching the current table cursor,
// or nil when the revisions list is empty.
func (v *View) selectedRevision() *model.SecretRevision {
	if v.secret == nil || len(v.secret.Revisions) == 0 {
		return nil
	}
	idx := v.revTable.Cursor()
	// The table cursor is -1 before the first render; treat as first row.
	if idx < 0 {
		idx = 0
	}
	if idx >= len(v.secret.Revisions) {
		return nil
	}
	// Rows are rendered newest-first, so map cursor back to the original slice.
	reversed := len(v.secret.Revisions) - 1 - idx
	return &v.secret.Revisions[reversed]
}

// selectedRevisionNumber returns the revision number under the cursor,
// or 0 to indicate "latest".
func (v *View) selectedRevisionNumber() int {
	if rev := v.selectedRevision(); rev != nil {
		return rev.Revision
	}
	return 0
}

func (v *View) Enter(ctx view.NavigateContext) (tea.Cmd, error) {
	v.secretURI = ctx.Context
	v.focusAccess = false
	v.revTable.SetStyles(ui.StyledTable(v.styles))
	v.accessTable.SetStyles(ui.UnfocusedTableStyles(v.styles))
	v.revTable.Focus()
	v.accessTable.Blur()
	// Reset cursors so a fresh navigation always starts at the first row.
	v.revTable.SetCursor(0)
	v.accessTable.SetCursor(0)
	v.refreshSecret()
	return nil, nil
}

func (v *View) Leave() tea.Cmd {
	v.revealOpen = false
	return nil
}

// refreshSecret finds the matching secret from status and populates tables.
func (v *View) refreshSecret() {
	v.secret = nil
	if v.status == nil || v.secretURI == "" {
		v.revTable.SetRows(nil)
		v.accessTable.SetRows(nil)
		return
	}
	for i := range v.status.Secrets {
		if v.status.Secrets[i].URI == v.secretURI {
			v.secret = &v.status.Secrets[i]
			break
		}
	}
	if v.secret == nil {
		v.revTable.SetRows(nil)
		v.accessTable.SetRows(nil)
		return
	}
	// Preserve the current cursor positions across status refreshes so the
	// user's selection is not lost when the status stream fires.
	revCursor := v.revTable.Cursor()
	accCursor := v.accessTable.Cursor()

	v.revTable.SetRows(RevisionRows(v.secret.Revisions))
	v.accessTable.SetRows(AccessRows(v.secret.Access))

	// SetRows clamps the cursor if it exceeds the new row count; restore
	// the previous position (or 0 for a fresh view where cursor is -1).
	if revCursor < 0 {
		revCursor = 0
	}
	v.revTable.SetCursor(revCursor)
	if accCursor < 0 {
		accCursor = 0
	}
	v.accessTable.SetCursor(accCursor)
}

func (v *View) recalcLayout() {
	_, rightW := v.splitWidths()
	rightInner := rightW - 2
	if rightInner < 10 {
		rightInner = 10
	}

	halfH := (v.height - 4) / 2
	if halfH < 2 {
		halfH = 2
	}

	v.revTable.SetWidth(rightInner)
	v.revTable.SetHeight(halfH)
	v.revTable.SetColumns(ui.ScaleColumns(RevisionColumns(), rightInner))

	v.accessTable.SetWidth(rightInner)
	v.accessTable.SetHeight(halfH)
	v.accessTable.SetColumns(ui.ScaleColumns(AccessColumns(), rightInner))
}

func (v *View) splitWidths() (int, int) {
	left := v.width * 40 / 100
	if left < 30 {
		left = 30
	}
	right := v.width - left
	if right < 30 {
		right = 30
	}
	return left, right
}

// padToHeight pads or truncates content to exactly the given number of lines.
func padToHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
