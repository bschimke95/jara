package view

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
)

// Overview is the Bubble Tea model for the tree overview.
type Overview struct {
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}

// NewOverview creates a new overview view.
func NewOverview() *Overview {
	return &Overview{keys: ui.DefaultKeyMap()}
}

func (o *Overview) SetSize(width, height int) {
	o.width = width
	o.height = height
}

func (o *Overview) SetStatus(status *model.FullStatus) {
	o.status = status
}

func (o *Overview) Init() tea.Cmd { return nil }

func (o *Overview) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(msg, o.keys.Enter) {
			return o, func() tea.Msg {
				return NavigateMsg{Target: nav.ApplicationsView}
			}
		}
	}
	return o, nil
}

func (o *Overview) View() tea.View {
	if o.status == nil {
		return tea.NewView("Loading...")
	}

	var b strings.Builder

	b.WriteString(o.renderTree())
	b.WriteString("\n\n")

	summaryStyle := lipgloss.NewStyle().Foreground(color.Muted)
	b.WriteString(summaryStyle.Render(fmt.Sprintf(
		"  %d applications  |  %d machines  |  %d relations",
		len(o.status.Applications),
		len(o.status.Machines),
		len(o.status.Relations),
	)))

	return tea.NewView(b.String())
}

func (o *Overview) renderTree() string {
	s := o.status
	modelLabel := fmt.Sprintf("%s [%s/%s] (%s)",
		s.Model.Name, s.Model.Cloud, s.Model.Region, s.Model.Version)

	appStyle := lipgloss.NewStyle().Foreground(color.Title)
	unitStyle := lipgloss.NewStyle().Foreground(color.Muted)

	appNames := make([]string, 0, len(s.Applications))
	for name := range s.Applications {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)

	var appTrees []any
	for _, name := range appNames {
		app := s.Applications[name]
		statusColor := color.StatusColor(app.Status)
		statusStr := lipgloss.NewStyle().Foreground(statusColor).Render(app.Status)
		appLabel := appStyle.Render(name) + " " + statusStr

		unitNodes := make([]any, 0, len(app.Units))
		for _, u := range app.Units {
			uColor := color.StatusColor(u.WorkloadStatus)
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
		EnumeratorStyle(lipgloss.NewStyle().Foreground(color.Subtle)).
		ItemStyle(lipgloss.NewStyle().Foreground(color.Title))

	return t.String()
}
