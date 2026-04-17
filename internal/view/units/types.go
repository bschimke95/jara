package units

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/actionmodal"
	"github.com/bschimke95/jara/internal/view/confirmodal"
)

// View is the Bubble Tea model for the units table view.
type View struct {
	table        table.Model
	keys         ui.KeyMap
	styles       *color.Styles
	width        int
	height       int
	status       *model.FullStatus
	appName      string
	pendingScale map[string]int // net pending unit delta per app
	filterStr    string

	actionModal     *actionmodal.Modal
	actionModalOpen bool

	confirmModal confirmodal.Modal
	confirmOpen  bool
	removingUnit string // unit name pending removal
	removeForce  bool   // force-remove toggle
}
