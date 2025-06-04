package navigation

import (
	tea "github.com/charmbracelet/bubbletea"
)

// GoBackMsg is a message to indicate we should go back to the previous view
type GoBackMsg struct{}

func GoBack() tea.Cmd {
	return func() tea.Msg {
		return GoBackMsg{}
	}
}

// GoToOpts contains options for the GoTo navigation
type GoToOpts struct {
	// SkipHistory prevents adding the current page to navigation history
	SkipHistory bool
}

type GoToMsg struct {
	Page tea.Model
	Opts GoToOpts
}

// GoTo creates a command to navigate to a new page
// Options can be provided to control the navigation behavior
func GoTo(page tea.Model, opts ...GoToOpts) tea.Cmd {
	var opt GoToOpts
	if len(opts) > 0 {
		opt = opts[0]
	}
	return func() tea.Msg {
		return GoToMsg{
			Page: page,
			Opts: opt,
		}
	}
}
