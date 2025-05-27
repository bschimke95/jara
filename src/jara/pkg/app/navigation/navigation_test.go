package navigation_test

import (
	"testing"

	"github.com/bschimke95/jara/pkg/app/navigation"
	"github.com/charmbracelet/bubbletea"
	. "github.com/onsi/gomega"
)

// mockModel is a simple tea.Model implementation for testing
type mockModel struct {
	id string
}

func (m mockModel) Init() tea.Cmd { return nil }
func (m mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m mockModel) View() string { return m.id }

func TestNavigation(t *testing.T) {
	g := NewWithT(t)

	t.Run("GoTo creates a command with the correct page", func(t *testing.T) {
		// Arrange
		page := mockModel{id: "test-page"}

		// Act
		cmd := navigation.GoTo(page)
		msg := cmd()

		// Assert
		goToMsg, ok := msg.(navigation.GoToMsg)
		g.Expect(ok).To(BeTrue(), "should be a GoToMsg")
		g.Expect(goToMsg.Page).To(Equal(page), "should contain the provided page")
		g.Expect(goToMsg.Opts.SkipHistory).To(BeFalse(), "should not skip history by default")
	})

	t.Run("GoTo with SkipHistory creates correct options", func(t *testing.T) {
		// Arrange
		page := mockModel{id: "test-page"}

		// Act
		cmd := navigation.GoTo(page, navigation.GoToOpts{SkipHistory: true})
		msg := cmd()

		// Assert
		goToMsg, ok := msg.(navigation.GoToMsg)
		g.Expect(ok).To(BeTrue(), "should be a GoToMsg")
		g.Expect(goToMsg.Opts.SkipHistory).To(BeTrue(), "should skip history when requested")
	})

	t.Run("GoBack creates a GoBackMsg", func(t *testing.T) {
		// Act
		cmd := navigation.GoBack()
		msg := cmd()

		// Assert
		_, ok := msg.(navigation.GoBackMsg)
		g.Expect(ok).To(BeTrue(), "should be a GoBackMsg")
	})
}
