// Package ui provides the chrome layer for jara: the header, footer, border
// boxes, key-hint rendering, and the shared key-binding map used across all
// views.
package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/bschimke95/jara/internal/color"
)

const logo = `
      ██╗ █████╗ ██████╗  █████╗
      ██║██╔══██╗██╔══██╗██╔══██╗
      ██║███████║██████╔╝███████║
 ██   ██║██╔══██║██╔══██╗██╔══██║
 ╚█████╔╝██║  ██║██║  ██║██║  ██║
  ╚════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝`

// borderChars holds the rounded border runes.
var borderChars = lipgloss.RoundedBorder()

// BorderBox wraps content in a rounded border with an optional title in the top border.
// The title is embedded inline and centered: ╭────── Title ──────╮
func BorderBox(content, title string, width int) string {
	borderStyle := lipgloss.NewStyle().Foreground(color.Border)
	titleStyle := lipgloss.NewStyle().Foreground(color.BorderTitle).Bold(true)

	innerWidth := width - 2 // left + right border chars
	if innerWidth < 0 {
		innerWidth = 0
	}

	// Build top border line with embedded title.
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

	// Build bottom border.
	bot := borderStyle.Render(borderChars.BottomLeft + strings.Repeat(borderChars.Top, innerWidth) + borderChars.BottomRight)

	// Pad each content line to innerWidth and add side borders.
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

// BorderBoxRawTitle is like BorderBox but accepts a pre-rendered title string.
// The caller is responsible for styling; lipgloss.Width is used for measurement.
func BorderBoxRawTitle(content, renderedTitle string, width int) string {
	borderStyle := lipgloss.NewStyle().Foreground(color.Border)

	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	var top string
	if renderedTitle != "" {
		titleLen := lipgloss.Width(renderedTitle)
		totalPad := innerWidth - titleLen
		if totalPad < 0 {
			totalPad = 0
		}
		leftPad := totalPad / 2
		rightPad := totalPad - leftPad
		top = borderStyle.Render(borderChars.TopLeft+strings.Repeat(borderChars.Top, leftPad)) +
			renderedTitle +
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

// LogoHeight returns the number of lines in the logo.
func LogoHeight() int {
	return lipgloss.Height(logo)
}

// HeaderContent renders the inner content for the header box:
// status info on the left (bottom-aligned), key hints in the center (bottom-aligned, 2-column),
// logo on the right (top-aligned).
func HeaderContent(controller, modelName, cloud, region string, hints []KeyHint, innerWidth int) string {
	// Logo height determines header height.
	logoHeight := LogoHeight()

	// Right block: logo (top-aligned, no padding needed).
	logoStyle := lipgloss.NewStyle().
		Foreground(color.LogoColor).
		Bold(true)
	logoRendered := logoStyle.Render(logo)
	logoWidth := lipgloss.Width(logoRendered)

	// Left block: status info (bottom-aligned).
	labelStyle := lipgloss.NewStyle().Foreground(color.InfoLabel)
	valueStyle := lipgloss.NewStyle().Foreground(color.InfoValue)

	infoLines := []string{
		labelStyle.Render("Controller: ") + valueStyle.Render(controller),
		labelStyle.Render("Model:      ") + valueStyle.Render(modelName),
		labelStyle.Render("Cloud:      ") + valueStyle.Render(cloud+"/"+region),
	}

	// Pad info block at the top to bottom-align (prepend empty lines).
	for len(infoLines) < logoHeight {
		infoLines = append([]string{""}, infoLines...)
	}
	infoBlock := strings.Join(infoLines, "\n")
	infoWidth := lipgloss.Width(infoBlock)

	// Center block: key hints in 2-column layout (bottom-aligned).
	keyStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(color.HintDesc)

	// Arrange hints in 2 columns with fixed column width.
	var hintLines []string
	hintCount := len(hints)
	mid := (hintCount + 1) / 2 // ceiling division for uneven splits

	// Calculate max width of left column for alignment.
	var maxLeftWidth int
	for i := 0; i < mid && i < hintCount; i++ {
		h1 := keyStyle.Render("<"+hints[i].Key+">") + descStyle.Render(" "+hints[i].Desc)
		w := lipgloss.Width(h1)
		if w > maxLeftWidth {
			maxLeftWidth = w
		}
	}

	// Build hint lines with aligned columns.
	for i := 0; i < mid; i++ {
		var line string
		// Left column
		if i < hintCount {
			h1 := keyStyle.Render("<"+hints[i].Key+">") + descStyle.Render(" "+hints[i].Desc)
			w := lipgloss.Width(h1)
			pad := maxLeftWidth - w
			if pad < 0 {
				pad = 0
			}
			line = h1 + strings.Repeat(" ", pad)
		} else {
			line = strings.Repeat(" ", maxLeftWidth)
		}
		// Right column with gap.
		if i+mid < hintCount {
			h2 := keyStyle.Render("<"+hints[i+mid].Key+">") + descStyle.Render(" "+hints[i+mid].Desc)
			line += "  " + h2
		}
		hintLines = append(hintLines, line)
	}

	// Pad hint block at the top to bottom-align (prepend empty lines).
	for len(hintLines) < logoHeight {
		hintLines = append([]string{""}, hintLines...)
	}
	hintBlock := strings.Join(hintLines, "\n")
	hintWidth := lipgloss.Width(hintBlock)

	// Calculate spacing.
	remaining := innerWidth - infoWidth - hintWidth - logoWidth
	if remaining < 4 {
		remaining = 4
	}
	gap1 := remaining / 2
	gap2 := remaining - gap1

	spacer1 := strings.Repeat(" ", gap1)
	spacer2 := strings.Repeat(" ", gap2)

	return lipgloss.JoinHorizontal(lipgloss.Top, infoBlock, spacer1, hintBlock, spacer2, logoRendered)
}

// CrumbBar renders the k9s-style navigation crumbs row.
// In k9s this looks like: <resource_type> [context]
func CrumbBar(viewName string, context string, width int) string {
	crumbStyle := lipgloss.NewStyle().
		Foreground(color.CrumbFg).
		Background(color.CrumbBg).
		Bold(true).
		Padding(0, 1)

	parts := []string{crumbStyle.Render(viewName)}
	if context != "" {
		ctxStyle := lipgloss.NewStyle().
			Foreground(color.CrumbFg).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1)
		parts = append(parts, ctxStyle.Render(context))
	}

	crumbs := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	barStyle := lipgloss.NewStyle().Width(width)
	return barStyle.Render(crumbs)
}

// Footer renders the k9s-style bottom hint bar showing key bindings.
func Footer(hints []KeyHint, filterText string, width int) string {
	var parts []string

	keyStyle := lipgloss.NewStyle().Foreground(color.HintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(color.HintDesc)
	sepStyle := lipgloss.NewStyle().Foreground(color.Subtle)

	for i, h := range hints {
		part := keyStyle.Render("<"+h.Key+">") + descStyle.Render(" "+h.Desc)
		parts = append(parts, part)
		if i < len(hints)-1 {
			parts = append(parts, sepStyle.Render(" │ "))
		}
	}

	line := strings.Join(parts, "")

	if filterText != "" {
		filterLabel := keyStyle.Render(" Filter:") +
			lipgloss.NewStyle().Foreground(color.Title).Render(" "+filterText)
		line += "    " + filterLabel
	}

	barStyle := lipgloss.NewStyle().Width(width)
	return barStyle.Render(line)
}

// KeyHint represents a single key-description pair for the footer.
type KeyHint struct {
	Key  string
	Desc string
}

// HintsForView returns the appropriate key hints for the current view.
func HintsForView(viewName string) []KeyHint {
	common := []KeyHint{
		{Key: ":", Desc: "cmd"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}

	switch viewName {
	case "Controllers":
		return append([]KeyHint{
			{Key: "enter", Desc: "select"},
		}, common...)
	case "Models":
		return append([]KeyHint{
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "back"},
		}, common...)
	case "Model":
		return append([]KeyHint{
			{Key: "U", Desc: "units"},
			{Key: "R", Desc: "relations"},
			{Key: "L", Desc: "logs (app)"},
			{Key: "l", Desc: "logs"},
			{Key: "+/-", Desc: "scale"},
		}, common...)
	case "Applications":
		return append([]KeyHint{
			{Key: "enter", Desc: "units"},
			{Key: "L", Desc: "logs (app)"},
			{Key: "l", Desc: "logs"},
		}, common...)
	case "Units":
		return append([]KeyHint{
			{Key: "esc", Desc: "back"},
			{Key: "L", Desc: "logs (unit)"},
			{Key: "l", Desc: "logs"},
			{Key: "+/-", Desc: "scale"},
		}, common...)
	case "Machines":
		return append([]KeyHint{
			{Key: "esc", Desc: "back"},
			{Key: "L", Desc: "logs (machine)"},
			{Key: "l", Desc: "logs"},
		}, common...)
	case "Relations":
		return append([]KeyHint{
			{Key: "esc", Desc: "back"},
			{Key: "l", Desc: "logs"},
		}, common...)
	case "Debug Log":
		return append([]KeyHint{
			{Key: "esc", Desc: "back"},
			{Key: "G", Desc: "bottom"},
			{Key: "g", Desc: "top"},
			{Key: "F", Desc: "filter"},
			{Key: "D", Desc: "clear filter"},
			{Key: "/", Desc: "search"},
			{Key: "n/N", Desc: "next/prev match"},
		}, common...)
	default:
		return append([]KeyHint{
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "back"},
		}, common...)
	}
}

// StatusBar renders the bottom status/info line (item count + resource type).
func StatusBar(resourceCount int, resourceType string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(color.Muted)
	return style.Render(fmt.Sprintf(" %d %s", resourceCount, resourceType))
}
