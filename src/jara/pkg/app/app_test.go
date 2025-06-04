package app_test

import (
	"testing"

	"github.com/bschimke95/jara/pkg/app"
	"github.com/bschimke95/jara/pkg/app/navigation"
	tea "github.com/charmbracelet/bubbletea"
	. "github.com/onsi/gomega"
)

// testModel is a simple tea.Model implementation for testing
type testModel struct {
	id string
}

func (m testModel) Init() tea.Cmd                           { return nil }
func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m testModel) View() string                            { return m.id }

func TestAppNavigation(t *testing.T) {
	g := NewWithT(t)

	t.Run("initial state has no history", func(t *testing.T) {
		// Arrange
		a := app.New(testModel{id: "initial"})

		// Act
		prevPage := a.PopFromHistory()

		// Assert
		g.Expect(prevPage).To(BeNil(), "should not have previous page initially")
	})

	t.Run("navigating to a page adds to history", func(t *testing.T) {
		// Arrange
		a := app.New(testModel{id: "initial"})
		page1 := testModel{id: "page1"}
		page2 := testModel{id: "page2"}

		// Act - navigate to page1
		updated, _ := a.Update(navigation.GoToMsg{Page: page1})
		a = updated.(app.Model)

		// Act - navigate to page2 (should add page1 to history)
		updated, _ = a.Update(navigation.GoToMsg{Page: page2})
		a = updated.(app.Model)

		// Assert
		prevPage := a.PopFromHistory()
		g.Expect(prevPage).ToNot(BeNil(), "should have previous page")
		g.Expect(prevPage.View()).To(Equal("page1"), "should have page1 in history")
	})

	t.Run("navigating with SkipHistory does not add to history", func(t *testing.T) {
		// Arrange
		a := app.New(testModel{id: "initial"})
		page1 := testModel{id: "page1"}
		page2 := testModel{id: "page2"}

		// Act - navigate to page1 with SkipHistory
		updated, _ := a.Update(navigation.GoToMsg{
			Page: page1,
			Opts: navigation.GoToOpts{SkipHistory: true},
		})
		a = updated.(app.Model)

		// Act - navigate to page2 with SkipHistory (should not add page1 to history)
		updated, _ = a.Update(navigation.GoToMsg{
			Page: page2,
			Opts: navigation.GoToOpts{SkipHistory: true},
		})
		a = updated.(app.Model)

		// Assert
		prevPage := a.PopFromHistory()
		g.Expect(prevPage).To(BeNil(), "should not have previous page when SkipHistory is true")
	})

	t.Run("GoBackMsg navigates to previous page", func(t *testing.T) {
		// Arrange
		a := app.New(testModel{id: "initial"})
		page1 := testModel{id: "page1"}
		page2 := testModel{id: "page2"}

		// Act - navigate to page1
		updated, _ := a.Update(navigation.GoToMsg{Page: page1})
		a = updated.(app.Model)

		// Act - navigate to page2 (adds page1 to history)
		updated, _ = a.Update(navigation.GoToMsg{Page: page2})
		a = updated.(app.Model)

		// Act - go back
		updated, _ = a.Update(navigation.GoBackMsg{})
		a = updated.(app.Model)

		// Assert
		g.Expect(a.CurrentPage().View()).To(Equal("page1"), "should navigate back to page1")
	})

	t.Run("GoBackMsg with empty history does nothing", func(t *testing.T) {
		// Arrange
		initialPage := testModel{id: "initial"}
		a := app.New(initialPage)

		// Act - try to go back with empty history
		updated, _ := a.Update(navigation.GoBackMsg{})
		a = updated.(app.Model)

		// Assert
		g.Expect(a.CurrentPage()).To(Equal(initialPage), "should stay on the same page")
	})
}
