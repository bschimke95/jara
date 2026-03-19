package relations

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

// Columns defines the columns for the relation table.
func Columns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 5},
		{Title: "ENDPOINT 1", Width: 28},
		{Title: "ENDPOINT 2", Width: 28},
		{Title: "INTERFACE", Width: 22},
		{Title: "TYPE", Width: 8},
		{Title: "STATUS", Width: 10},
	}
}

// Rows converts a slice of relations to table rows.
func Rows(rels []model.Relation) []table.Row {
	rows := make([]table.Row, 0, len(rels))
	for _, r := range rels {
		ep1, ep2 := "", ""
		if len(r.Endpoints) > 0 {
			ep1 = formatEndpoint(r.Endpoints[0])
		}
		if len(r.Endpoints) > 1 {
			ep2 = formatEndpoint(r.Endpoints[1])
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.ID), ep1, ep2, r.Interface, r.Scope, r.Status,
		})
	}
	return rows
}

// RowsForApp returns relation rows involving a specific application.
func RowsForApp(rels []model.Relation, appName string) []table.Row {
	var rows []table.Row
	for _, r := range rels {
		involved := false
		for _, ep := range r.Endpoints {
			if ep.ApplicationName == appName {
				involved = true
				break
			}
		}
		if !involved {
			continue
		}
		ep1, ep2 := "", ""
		if len(r.Endpoints) > 0 {
			ep1 = formatEndpoint(r.Endpoints[0])
		}
		if len(r.Endpoints) > 1 {
			ep2 = formatEndpoint(r.Endpoints[1])
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.ID), ep1, ep2, r.Interface, r.Scope, r.Status,
		})
	}
	return rows
}

// CompactColumn defines the single-column layout for the relations pane inside ModelView.
func CompactColumn() []table.Column {
	return []table.Column{
		{Title: "RELATION", Width: 60},
	}
}

// CompactRowsForApp returns compact relation rows for the relations pane
// in ModelView.
func CompactRowsForApp(rels []model.Relation, appName string) []table.Row {
	var rows []table.Row
	for _, r := range rels {
		var local, remote *model.Endpoint
		for i := range r.Endpoints {
			ep := &r.Endpoints[i]
			if ep.ApplicationName == appName {
				local = ep
			} else {
				remote = ep
			}
		}
		if local == nil {
			continue
		}

		lhs := appName + ":" + local.Name

		ifaceSeg := ""
		if remote != nil && local.Name != remote.Name {
			ifaceSeg = " --" + r.Interface
		}

		if remote == nil || remote.ApplicationName == appName {
			rows = append(rows, table.Row{lhs + " ↩"})
			continue
		}

		crossModel := isCrossModelRelation(r.Key, remote.ApplicationName)
		var modelPrefix string
		if crossModel {
			modelPrefix = extractModelPrefix(r.Key, remote.ApplicationName) + ":"
		}

		rhs := modelPrefix + remote.ApplicationName + ":" + remote.Name
		rows = append(rows, table.Row{lhs + ifaceSeg + " -> " + rhs})
	}
	return rows
}

// formatEndpoint formats an endpoint as "app:name" or just "app" when name is empty.
func formatEndpoint(ep model.Endpoint) string {
	if ep.Name == "" {
		return ep.ApplicationName
	}
	return ep.ApplicationName + ":" + ep.Name
}

// isCrossModelRelation reports whether the relation key indicates a cross-model
// relation involving the given remote application name.
func isCrossModelRelation(key, remoteApp string) bool {
	for _, segment := range strings.Fields(key) {
		colon := strings.Index(segment, ":")
		if colon < 0 {
			continue
		}
		appPart := segment[:colon]
		if strings.Contains(appPart, ".") {
			bare := appPart[strings.LastIndex(appPart, ".")+1:]
			if bare == remoteApp || appPart == remoteApp {
				return true
			}
		}
	}
	return false
}

// extractModelPrefix returns the "[user/]model" part from a CMR key segment
// that matches the given remote application name.
func extractModelPrefix(key, remoteApp string) string {
	for _, segment := range strings.Fields(key) {
		colon := strings.Index(segment, ":")
		if colon < 0 {
			continue
		}
		appPart := segment[:colon]
		if dot := strings.LastIndex(appPart, "."); dot >= 0 {
			bare := appPart[dot+1:]
			if bare == remoteApp || appPart == remoteApp {
				return appPart[:dot]
			}
		}
	}
	return ""
}
