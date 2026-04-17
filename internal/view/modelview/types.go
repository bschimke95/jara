package modelview

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/actionmodal"
	"github.com/bschimke95/jara/internal/view/deploymodal"
	"github.com/bschimke95/jara/internal/view/relatemodal"
)

// View is the split-pane model overview.
type View struct {
	appTable      table.Model
	unitTable     table.Model
	relationTable table.Model

	keys   ui.KeyMap
	styles *color.Styles
	status *model.FullStatus

	width        int
	height       int
	selectedApp  string
	pendingScale map[string]int
	filterStr    string

	deployModalOpen bool
	deployModal     deploymodal.Modal

	relateModalOpen bool
	relateModal     relatemodal.Modal

	actionModalOpen bool
	actionModal     *actionmodal.Modal

	charmhubSuggestions []string
	charmEndpoints      map[string]map[string]model.CharmEndpoint

	selectModelFn func(name string) error
}
