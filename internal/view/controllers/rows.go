package controllers

import (
	"fmt"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

func columns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 18},
		{Title: "CLOUD", Width: 14},
		{Title: "REGION", Width: 16},
		{Title: "VERSION", Width: 10},
		{Title: "STATUS", Width: 12},
		{Title: "HA", Width: 5},
		{Title: "MODELS", Width: 7},
		{Title: "MACHINES", Width: 9},
		{Title: "ACCESS", Width: 12},
		{Title: "ADDRESS", Width: 22},
	}
}

func controllerRows(ctrls []model.Controller) []table.Row {
	rows := make([]table.Row, 0, len(ctrls))
	for _, c := range ctrls {
		rows = append(rows, table.Row{
			c.Name,
			c.Cloud,
			c.Region,
			c.Version,
			c.Status,
			c.HA,
			fmt.Sprintf("%d", c.Models),
			fmt.Sprintf("%d", c.Machines),
			c.Access,
			c.Addr,
		})
	}
	return rows
}
