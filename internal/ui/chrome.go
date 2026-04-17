// Package ui provides the chrome layer for jara: the header, footer, border
// boxes, key-hint rendering, and the shared key-binding map used across all
// views.
package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/bschimke95/jara/internal/color"
)

// MaxHintsPerColumn is the maximum number of key hints rendered per column
// in both the header hint block and the help modal sections.
const MaxHintsPerColumn = 6

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
func BorderBox(content, title string, width int, s *color.Styles) string {
	borderStyle := s.BorderStyle
	titleStyle := s.BorderTitleStyle

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
func BorderBoxRawTitle(content, renderedTitle string, width int, s *color.Styles) string {
	borderStyle := s.BorderStyle

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
func HeaderContent(controller, modelName, cloud, region, jaraVersion, jujuVersion string, hints []KeyHint, innerWidth int, s *color.Styles) string {
	// Logo height determines header height.
	logoHeight := LogoHeight()

	// Right block: logo (top-aligned, no padding needed).
	logoRendered := s.Logo.Render(logo)
	logoWidth := lipgloss.Width(logoRendered)

	// Left block: status info (bottom-aligned).
	labelStyle := s.InfoLabel
	valueStyle := s.InfoValue

	infoLines := []string{
		labelStyle.Render("Controller: ") + valueStyle.Render(controller),
		labelStyle.Render("Model:      ") + valueStyle.Render(modelName),
		labelStyle.Render("Cloud:      ") + valueStyle.Render(cloud+"/"+region),
		labelStyle.Render("Juju:       ") + valueStyle.Render(jujuVersion),
		labelStyle.Render("Jara:       ") + valueStyle.Render(jaraVersion),
	}

	// Pad info block at the top to bottom-align (prepend empty lines).
	for len(infoLines) < logoHeight {
		infoLines = append([]string{""}, infoLines...)
	}
	infoBlock := strings.Join(infoLines, "\n")
	infoWidth := lipgloss.Width(infoBlock)

	// Center block: key hints in 2-column layout (bottom-aligned).
	keyStyle := s.HintKey
	descStyle := s.HintDesc

	// Arrange hints in 2 columns with keys and descriptions aligned vertically.
	// Cap the left column at MaxHintsPerColumn rows; overflow spills right.
	var hintLines []string
	hintCount := len(hints)
	mid := min(hintCount, MaxHintsPerColumn)

	// Find the max rendered key width across all hints so descriptions line up.
	var maxKeyWidth int
	for i := 0; i < hintCount; i++ {
		if w := lipgloss.Width(keyStyle.Render("<" + hints[i].Key + ">")); w > maxKeyWidth {
			maxKeyWidth = w
		}
	}

	// renderHint returns "<key><pad> desc" with key padded to maxKeyWidth.
	renderHint := func(h KeyHint) string {
		k := keyStyle.Render("<" + h.Key + ">")
		pad := maxKeyWidth - lipgloss.Width(k)
		if pad < 0 {
			pad = 0
		}
		return k + strings.Repeat(" ", pad) + descStyle.Render(" "+h.Desc)
	}

	// Calculate max width of left column for inter-column gap.
	var maxLeftWidth int
	for i := 0; i < mid && i < hintCount; i++ {
		if w := lipgloss.Width(renderHint(hints[i])); w > maxLeftWidth {
			maxLeftWidth = w
		}
	}

	// Build hint lines with aligned columns.
	for i := 0; i < mid; i++ {
		var line string
		// Left column
		if i < hintCount {
			h1 := renderHint(hints[i])
			pad := maxLeftWidth - lipgloss.Width(h1)
			if pad < 0 {
				pad = 0
			}
			line = h1 + strings.Repeat(" ", pad)
		} else {
			line = strings.Repeat(" ", maxLeftWidth)
		}
		// Right column with gap.
		if i+mid < hintCount {
			line += "  " + renderHint(hints[i+mid])
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

// CrumbBar renders the full navigation stack as a breadcrumb bar.
// Each entry in crumbs is rendered as a pill; the last (current) entry is
// highlighted with CrumbBg; ancestors use CrumbBgAlt. Entries are joined
// by a › separator.
func CrumbBar(crumbs []string, width int, s *color.Styles) string {
	if len(crumbs) == 0 {
		return lipgloss.NewStyle().Width(width).Render("")
	}

	activeStyle := s.CrumbActive
	ancestorStyle := s.CrumbAncestor
	sepStyle := s.CrumbSep

	var parts []string
	for i, crumb := range crumbs {
		if i == len(crumbs)-1 {
			parts = append(parts, activeStyle.Render(crumb))
		} else {
			parts = append(parts, ancestorStyle.Render(crumb))
			parts = append(parts, sepStyle.Render(" › "))
		}
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	barStyle := lipgloss.NewStyle().Width(width)
	return " " + barStyle.Render(bar) + "\n"
}

// Footer renders the k9s-style bottom hint bar showing key bindings.
func Footer(hints []KeyHint, filterText string, width int, s *color.Styles) string {
	var parts []string

	keyStyle := s.HintKey
	descStyle := s.HintDesc
	sepStyle := s.HintSep

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
			s.TitleText.Render(" "+filterText)
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
// Key labels are derived from the actual KeyMap bindings so they stay
// consistent when the user overrides key bindings via config.
//
// Deprecated: Views now provide their own hints via the KeyHints() method on
// the view.View interface. This function remains for backward compatibility
// and will be removed in a future release.
func HintsForView(viewName string, keys KeyMap) []KeyHint {
	bk := func(b key.Binding) string { return b.Help().Key }

	common := []KeyHint{
		{Key: bk(keys.Command), Desc: "cmd"},
		{Key: bk(keys.Help), Desc: "help"},
		{Key: bk(keys.Quit), Desc: "quit"},
	}

	switch viewName {
	case "Controllers":
		return append([]KeyHint{
			{Key: bk(keys.Enter), Desc: "select"},
		}, common...)
	case "Models":
		return append([]KeyHint{
			{Key: bk(keys.Enter), Desc: "select"},
			{Key: bk(keys.Back), Desc: "back"},
		}, common...)
	case "Model":
		return append([]KeyHint{
			{Key: bk(keys.UnitsNav), Desc: "units"},
			{Key: bk(keys.RelationsNav), Desc: "relations"},
			{Key: bk(keys.LogsJump), Desc: "logs (app)"},
			{Key: bk(keys.ScaleUp) + "/" + bk(keys.ScaleDown), Desc: "scale"},
		}, common...)
	case "Applications":
		return append([]KeyHint{
			{Key: bk(keys.Enter), Desc: "units"},
			{Key: bk(keys.LogsJump), Desc: "logs (app)"},
		}, common...)
	case "Units":
		return append([]KeyHint{
			{Key: bk(keys.Back), Desc: "back"},
			{Key: bk(keys.LogsJump), Desc: "logs (unit)"},
			{Key: bk(keys.ScaleUp) + "/" + bk(keys.ScaleDown), Desc: "scale"},
		}, common...)
	case "Machines":
		return append([]KeyHint{
			{Key: bk(keys.Back), Desc: "back"},
			{Key: bk(keys.LogsJump), Desc: "logs (machine)"},
		}, common...)
	case "Relations":
		return append([]KeyHint{
			{Key: bk(keys.Back), Desc: "back"},
			{Key: bk(keys.LogsJump), Desc: "logs"},
		}, common...)
	case "Debug Log":
		return append([]KeyHint{
			{Key: bk(keys.Back), Desc: "back"},
			{Key: bk(keys.Bottom), Desc: "bottom"},
			{Key: bk(keys.Top), Desc: "top"},
			{Key: bk(keys.FilterOpen), Desc: "filter"},
			{Key: bk(keys.ClearFilter), Desc: "clear filter"},
			{Key: bk(keys.SearchOpen), Desc: "search"},
			{Key: bk(keys.SearchNext) + "/" + bk(keys.SearchPrev), Desc: "next/prev match"},
		}, common...)
	default:
		return append([]KeyHint{
			{Key: bk(keys.Enter), Desc: "select"},
			{Key: bk(keys.Back), Desc: "back"},
		}, common...)
	}
}

// StatusBar renders the bottom status/info line (item count + resource type).
func StatusBar(resourceCount int, resourceType string, width int, s *color.Styles) string {
	return s.MutedText.Render(fmt.Sprintf(" %d %s", resourceCount, resourceType))
}
