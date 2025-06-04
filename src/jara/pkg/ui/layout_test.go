package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	. "github.com/onsi/gomega"
)

func TestLayout(t *testing.T) {
	g := NewWithT(t)

	t.Run("NewLayout returns a valid layout", func(t *testing.T) {
		layout := NewLayout()
		g.Expect(layout).NotTo(BeNil(), "NewLayout() should not return nil")
	})

	t.Run("SetDimensions updates layout dimensions", func(t *testing.T) {
		layout := NewLayout()
		layout.SetDimensions()
		g.Expect(layout.Width).To(Equal(80), "Width should be set to 80")
		g.Expect(layout.Height).To(Equal(24), "Height should be set to 24")
	})

	t.Run("Style helpers return valid styles", func(t *testing.T) {
		layout := NewLayout()

		tests := []struct {
			name   string
			method func() lipgloss.Style
		}{
			{"HeaderStyle", func() lipgloss.Style { return layout.HeaderStyle }},
			{"BodyStyle", func() lipgloss.Style { return layout.BodyStyle }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				style := tt.method()
				g.Expect(style).To(BeAssignableToTypeOf(lipgloss.Style{}), "%s should return a lipgloss.Style", tt.name)
			})
		}
	})
}
