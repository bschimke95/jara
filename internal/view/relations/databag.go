package relations

import (
	"fmt"
	icolor "image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
	"github.com/bschimke95/jara/internal/view"
)

// appColors is a palette of border colours assigned to each application in a
// relation. The index wraps around when there are more endpoints than colours.
var appColors = []icolor.Color{
	lipgloss.Color("#00bfff"), // cyan / primary
	lipgloss.Color("#f0c674"), // gold
	lipgloss.Color("#b48ead"), // purple
	lipgloss.Color("#a3be8c"), // green
	lipgloss.Color("#d08770"), // orange
}

// renderDatabagPane renders the outer "Databags" box containing two equal-height
// sub-boxes: "Application Data" (top) and "Unit Data" (bottom).
func renderDatabagPane(rd *model.RelationData, rel *model.Relation, width, height int, s *color.Styles) string {
	if rd == nil || rel == nil {
		placeholder := lipgloss.NewStyle().Foreground(s.Muted).Render("Select a relation to view databag contents")
		return ui.BorderBox(view.PadToHeight(placeholder, height-2), "Databags", width, s)
	}

	innerWidth := width - 2 // outer border
	contentHeight := height - 2

	topH := contentHeight / 2
	botH := contentHeight - topH

	appBox := renderAppDataBox(rd, rel, innerWidth, topH, s)
	unitBox := renderUnitDataBox(rd, rel, innerWidth, botH, s)

	combined := strings.Split(appBox+"\n"+unitBox, "\n")

	for len(combined) < contentHeight {
		combined = append(combined, "")
	}
	if len(combined) > contentHeight {
		combined = combined[:contentHeight]
	}

	content := strings.Join(combined, "\n")
	return ui.BorderBox(content, "Databags", width, s)
}

// renderAppDataBox renders the "Application Data" box with one coloured
// sub-box per endpoint application.
func renderAppDataBox(rd *model.RelationData, rel *model.Relation, width, height int, s *color.Styles) string {
	boxInner := width - 2
	keyStyle := lipgloss.NewStyle().Foreground(s.InfoLabelColor)
	valStyle := lipgloss.NewStyle().Foreground(s.InfoValueColor)

	var lines []string
	for i, ep := range rel.Endpoints {
		appName := ep.ApplicationName
		appColor := appColors[i%len(appColors)]

		data, ok := rd.ApplicationData[appName]
		var content string
		if !ok || len(data) == 0 {
			content = lipgloss.NewStyle().Foreground(s.Muted).Render("(empty)")
		} else {
			var kvLines []string
			for _, kv := range sortedKV(data, boxInner-6) {
				kvLines = append(kvLines, keyStyle.Render(kv.key)+" "+valStyle.Render(kv.val))
			}
			content = strings.Join(kvLines, "\n")
		}

		title := fmt.Sprintf("%s (%s)", appName, ep.Role)
		box := coloredBorderBox(content, title, boxInner, appColor)
		lines = append(lines, strings.Split(box, "\n")...)
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(s.Muted).Render("(no application data)"))
	}

	innerH := height - 2
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	if len(lines) > innerH {
		lines = lines[:innerH]
	}

	return ui.BorderBox(strings.Join(lines, "\n"), "Application Data", width, s)
}

// renderUnitDataBox renders the "Unit Data" box with one coloured sub-box per unit.
func renderUnitDataBox(rd *model.RelationData, rel *model.Relation, width, height int, s *color.Styles) string {
	boxInner := width - 2
	keyStyle := lipgloss.NewStyle().Foreground(s.InfoLabelColor)
	valStyle := lipgloss.NewStyle().Foreground(s.InfoValueColor)
	leaderStyle := lipgloss.NewStyle().Foreground(s.CheckGreenColor).Bold(true)

	var lines []string
	for i, ep := range rel.Endpoints {
		appName := ep.ApplicationName
		appColor := appColors[i%len(appColors)]

		var unitNames []string
		for uName := range rd.UnitData {
			if strings.HasPrefix(uName, appName+"/") {
				unitNames = append(unitNames, uName)
			}
		}
		sort.Strings(unitNames)

		for _, uName := range unitNames {
			data := rd.UnitData[uName]

			title := uName
			if v, ok := data["leader"]; ok && v == "true" {
				title = uName + " " + leaderStyle.Render("★")
			}

			filtered := make(map[string]string, len(data))
			for k, v := range data {
				if k == "leader" {
					continue
				}
				filtered[k] = v
			}

			var content string
			if len(filtered) == 0 {
				content = lipgloss.NewStyle().Foreground(s.Muted).Render("(empty)")
			} else {
				var kvLines []string
				for _, kv := range sortedKV(filtered, boxInner-6) {
					kvLines = append(kvLines, keyStyle.Render(kv.key)+" "+valStyle.Render(kv.val))
				}
				content = strings.Join(kvLines, "\n")
			}

			box := coloredBorderBox(content, title, boxInner, appColor)
			lines = append(lines, strings.Split(box, "\n")...)
		}
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(s.Muted).Render("(no unit data)"))
	}

	innerH := height - 2
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	if len(lines) > innerH {
		lines = lines[:innerH]
	}

	return ui.BorderBox(strings.Join(lines, "\n"), "Unit Data", width, s)
}

// coloredBorderBox is like ui.BorderBox but uses a custom border colour.
func coloredBorderBox(content, title string, width int, borderColor icolor.Color) string {
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	var top string
	if title != "" {
		titleRendered := titleStyle.Render(" " + title + " ")
		titleLen := lipgloss.Width(titleRendered)
		totalPad := innerWidth - titleLen
		if totalPad < 0 {
			totalPad = 0
		}
		leftPad := totalPad / 2
		rightPad := totalPad - leftPad
		top = borderStyle.Render(borderChars.TopLeft+strings.Repeat(borderChars.Top, leftPad)) +
			titleRendered +
			borderStyle.Render(strings.Repeat(borderChars.Top, rightPad)+borderChars.TopRight)
	} else {
		top = borderStyle.Render(borderChars.TopLeft + strings.Repeat(borderChars.Top, innerWidth) + borderChars.TopRight)
	}

	bot := borderStyle.Render(borderChars.BottomLeft + strings.Repeat(borderChars.Top, innerWidth) + borderChars.BottomRight)

	lines := strings.Split(content, "\n")
	var body strings.Builder
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		pad := innerWidth - lineWidth
		if pad < 0 {
			pad = 0
		}
		body.WriteString(
			borderStyle.Render(borderChars.Left) +
				line + strings.Repeat(" ", pad) +
				borderStyle.Render(borderChars.Right) + "\n",
		)
	}

	return top + "\n" + body.String() + bot
}

// borderChars mirrors the rounded border runes from the ui package.
var borderChars = lipgloss.RoundedBorder()

type kvPair struct {
	key string
	val string
}

// sortedKV returns key = value pairs sorted by key, with keys right-padded
// to align the values.
func sortedKV(data map[string]string, maxWidth int) []kvPair {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	pairs := make([]kvPair, 0, len(keys))
	for _, k := range keys {
		padded := k + strings.Repeat(" ", maxKeyLen-len(k))
		val := "= " + data[k]
		if len(padded)+len(val)+1 > maxWidth {
			avail := maxWidth - len(padded) - 4
			if avail > 0 && len(data[k]) > avail {
				val = "= " + data[k][:avail] + "…"
			}
		}
		pairs = append(pairs, kvPair{key: padded, val: val})
	}
	return pairs
}
