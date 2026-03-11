package view

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/nav"
)

// StatusUpdatedMsg is sent when fresh status data arrives from the API.
type StatusUpdatedMsg struct {
	Status *model.FullStatus
}

// NavigateMsg requests navigation to a different view.
type NavigateMsg struct {
	Target  nav.ViewID
	Context string
}

// GoBackMsg requests navigation back to the previous view.
type GoBackMsg struct{}

// View is the interface all resource views must implement.
type View interface {
	tea.Model
	SetSize(width, height int)
	SetStatus(status *model.FullStatus)
}
