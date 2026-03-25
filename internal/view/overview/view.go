// Package overview implements the tree overview of model status.
package overview

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// New creates a new overview view.
func New(keys ui.KeyMap, styles *color.Styles) *View {
	return &View{keys: keys, styles: styles}
}

func (o *View) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// SetStatus implements view.StatusReceiver.
func (o *View) SetStatus(status *model.FullStatus) {
	o.status = status
}

// KeyHints returns the view-specific key hints for the header.
func (o *View) KeyHints() []view.KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }
	return []view.KeyHint{
		{Key: bk(o.keys.Enter), Desc: "applications"},
	}
}

func (o *View) Init() tea.Cmd { return nil }

func (o *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(msg, o.keys.Enter) {
			return o, func() tea.Msg {
				return view.NavigateMsg{Target: nav.ApplicationsView}
			}
		}
	}
	return o, nil
}

func (o *View) View() tea.View {
	if o.status == nil {
		return tea.NewView("Loading...")
	}

	var b strings.Builder
	b.WriteString(o.renderTree())
	b.WriteString("\n\n")

	summaryStyle := lipgloss.NewStyle().Foreground(o.styles.Muted)
	b.WriteString(summaryStyle.Render(fmt.Sprintf(
		"  %d applications  |  %d machines  |  %d relations",
		len(o.status.Applications),
		len(o.status.Machines),
		len(o.status.Relations),
	)))

	return tea.NewView(b.String())
}

func (o *View) renderTree() string {
	s := o.status
	modelLabel := fmt.Sprintf("%s [%s/%s] (%s)",
		s.Model.Name, s.Model.Cloud, s.Model.Region, s.Model.Version)

	appStyle := lipgloss.NewStyle().Foreground(o.styles.Title)
	unitStyle := lipgloss.NewStyle().Foreground(o.styles.Muted)

	appNames := make([]string, 0, len(s.Applications))
	for name := range s.Applications {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)

	var appTrees []any
	for _, name := range appNames {
		app := s.Applications[name]
		statusColor := o.styles.StatusColor(app.Status)
		statusStr := lipgloss.NewStyle().Foreground(statusColor).Render(app.Status)
		appLabel := appStyle.Render(name) + " " + statusStr

		unitNodes := make([]any, 0, len(app.Units))
		for _, u := range app.Units {
			uColor := o.styles.StatusColor(u.WorkloadStatus)
			leader := ""
			if u.Leader {
				leader = " *"
			}
			uLabel := unitStyle.Render(u.Name) + " " +
				lipgloss.NewStyle().Foreground(uColor).Render(u.WorkloadStatus) + leader
			unitNodes = append(unitNodes, uLabel)
		}

		appTree := tree.Root(appLabel).Child(unitNodes...)
		appTrees = append(appTrees, appTree)
	}

	t := tree.Root(modelLabel).
		Child(appTrees...).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(lipgloss.NewStyle().Foreground(o.styles.Subtle)).
		ItemStyle(lipgloss.NewStyle().Foreground(o.styles.Title))

	return t.String()
}

func (o *View) Enter(_ view.NavigateContext) (tea.Cmd, error) { return nil, nil }
func (o *View) Leave() tea.Cmd                                { return nil }
