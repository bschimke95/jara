package modelview

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view/deploymodal"
	"github.com/bschimke95/jara/internal/view/relatemodal"
)

// View is the split-pane model overview.
type View struct {
	appTable      table.Model
	unitTable     table.Model
	relationTable table.Model

	keys   ui.KeyMap
	status *model.FullStatus

	width        int
	height       int
	selectedApp  string
	pendingScale map[string]int

	deployModalOpen bool
	deployModal     deploymodal.Modal

	relateModalOpen bool
	relateModal     relatemodal.Modal

	charmhubSuggestions []string
	charmEndpoints      map[string]map[string]model.CharmEndpoint

	selectModelFn func(name string) error
}
