package ui

import (
	"fmt"
	"strings"

	"github.com/bschimke95/jara/pkg/env"
	"github.com/bschimke95/jara/pkg/types/juju"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// HeaderInfo contains the information to display in the header
type HeaderInfo struct {
	Controller juju.Controller
	KeyHints   interface {
		ShortHelp() []key.Binding
		FullHelp() [][]key.Binding
	}
}

// GetDefaultKeyMap returns the default key map from the env package
func GetDefaultKeyMap() []key.Binding {
	return env.DefaultKeyMap.ShortHelp()
}

// Header renders a header with controller info, key hints, and app info
func Header(width int, info HeaderInfo) string {
	// Get theme colors
	theme := env.CanonicalTheme

	// Background color for all header elements
	headerBg := lipgloss.Color(theme.HeaderBg)

	// Define styles using theme colors
	controllerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color(theme.PrimaryColor)).
		Background(headerBg)

	keyHintStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Background(headerBg)

	appNameStyle := lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		Foreground(lipgloss.Color(theme.SecondaryColor)).
		Background(headerBg).
		PaddingRight(1).
		Align(lipgloss.Right)

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SecondaryText)).
		Background(headerBg).
		Padding(0, 1).
		Align(lipgloss.Right)

	// Calculate section widths (using ratios 1:2:1)
	controllerWidth := width / 4
	keyHintsWidth := width / 2
	appInfoWidth := width - controllerWidth - keyHintsWidth

	// 1. Controller section (left)
	controllerName := info.Controller.Name
	controllerText := fmt.Sprintf("Controller: %s", controllerName)
	controllerSection := controllerStyle.Width(controllerWidth).Render(controllerText)

	// 2. Key hints section (middle)
	keyHintsSection := ""

	// Style the help model with theme colors
	keyHintsSection += keyHintStyle.Width(keyHintsWidth).Align(lipgloss.Center).Background(headerBg).Render("")
	keyHintsSection += "\n"

	// Rather than using the help component, we'll build our key hints manually
	// to have complete control over styling
	var keyBindings []string

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.PrimaryColor)).
		Background(headerBg)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.PrimaryText)).
		Background(headerBg)

	bulletStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SecondaryText)).
		Background(headerBg)

	// Get key bindings and style each one
	for _, binding := range info.KeyHints.ShortHelp() {
		// Create a combined style with background for both key and description
		combinedStyle := lipgloss.NewStyle().Background(headerBg)

		// Style the key and description with proper background
		keyText := keyStyle.Render(binding.Help().Key)

		// Add a space with the correct background color
		spaceWithBg := lipgloss.NewStyle().Background(headerBg).Render(" ")

		descText := descStyle.Render(binding.Help().Desc)

		// Combine them in a container with background
		fullText := combinedStyle.Render(keyText + spaceWithBg + descText)
		keyBindings = append(keyBindings, fullText)
	}

	// Join all key bindings with styled bullet points
	separator := bulletStyle.Render(" • ")
	keyHintsText := strings.Join(keyBindings, separator)

	// Apply container styling with proper background color
	keyHintsSection += keyHintStyle.
		Width(keyHintsWidth).
		Align(lipgloss.Center).
		Background(headerBg).
		Render(keyHintsText)

	// 3. App info section (right)
	appName := appNameStyle.Render("Jara")
	// TODO(ben): Get version from build info
	version := versionStyle.Render(fmt.Sprintf("v%s", "0.1.0"))
	appInfoText := fmt.Sprintf("%s\n%s", appName, version)
	// Ensure the entire app info section has the background color, including any empty space
	appInfoSection := lipgloss.NewStyle().
		Width(appInfoWidth).
		Align(lipgloss.Right).
		Background(headerBg).
		Render(appInfoText)

	// Combine all sections
	return lipgloss.JoinHorizontal(lipgloss.Top, controllerSection, keyHintsSection, appInfoSection)
}
