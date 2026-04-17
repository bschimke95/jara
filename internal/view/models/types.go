package models

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/confirmodal"
	"github.com/bschimke95/jara/internal/view/newmodelmodal"
)

// UpdatedMsg is sent when the model list for a controller arrives.
type UpdatedMsg struct {
	Models []model.ModelSummary
}

// View is the Bubble Tea model for the models list view.
type View struct {
	table              table.Model
	keys               ui.KeyMap
	styles             *color.Styles
	width              int
	height             int
	models             []model.ModelSummary
	pollFn             func(controller string) tea.Cmd
	selectControllerFn func(name string) error
	controllerNameFn   func() string
	filterStr          string

	// New model modal state.
	newModelModal newmodelmodal.Modal
	newModelOpen  bool

	// Remove model modal state.
	confirmModal confirmodal.Modal
	confirmOpen  bool
	removingName string
	removeForce  bool
}
