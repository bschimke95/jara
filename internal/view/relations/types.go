package relations

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/confirmodal"
)

// View is the Bubble Tea model for the relations table view.
type View struct {
	table  table.Model
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int
	status *model.FullStatus

	confirmOpen  bool
	confirmModal confirmodal.Modal
	deletingA    string // endpoint A of the relation being deleted
	deletingB    string // endpoint B of the relation being deleted
}
