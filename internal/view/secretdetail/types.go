package secretdetail

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/revealmodal"
)

// View is the Bubble Tea model for the secret detail view.
type View struct {
	revTable    table.Model
	accessTable table.Model
	keys        ui.KeyMap
	styles      *color.Styles
	width       int
	height      int
	status      *model.FullStatus
	secretURI   string
	secret      *model.Secret // cached reference from status
	focusAccess bool          // true when the access table has focus

	revealOpen  bool // true when the reveal modal is visible
	revealModal revealmodal.Modal
}
