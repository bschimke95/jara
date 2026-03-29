package secretdetail

import (
	"fmt"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

// RevisionColumns defines the columns for the revisions table.
func RevisionColumns() []table.Column {
	return []table.Column{
		{Title: "REV", Width: 6},
		{Title: "CREATED", Width: 22},
		{Title: "EXPIRED", Width: 22},
		{Title: "BACKEND", Width: 16},
	}
}

// RevisionRows converts secret revisions to table rows, newest first.
func RevisionRows(revisions []model.SecretRevision) []table.Row {
	rows := make([]table.Row, 0, len(revisions))
	for i := len(revisions) - 1; i >= 0; i-- {
		r := revisions[i]
		expired := "-"
		if r.ExpiredAt != nil {
			expired = r.ExpiredAt.Format("2006-01-02 15:04:05")
		}
		backend := r.Backend
		if backend == "" {
			backend = "-"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.Revision),
			r.CreatedAt.Format("2006-01-02 15:04:05"),
			expired,
			backend,
		})
	}
	return rows
}

// AccessColumns defines the columns for the access table.
func AccessColumns() []table.Column {
	return []table.Column{
		{Title: "TARGET", Width: 28},
		{Title: "SCOPE", Width: 20},
		{Title: "ROLE", Width: 12},
	}
}

// AccessRows converts secret access info to table rows.
func AccessRows(access []model.SecretAccessInfo) []table.Row {
	rows := make([]table.Row, 0, len(access))
	for _, a := range access {
		rows = append(rows, table.Row{
			a.Target,
			a.Scope,
			a.Role,
		})
	}
	return rows
}
