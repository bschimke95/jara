package app

import (
	"github.com/bschimke95/jara/pkg/app/navigation"
	"github.com/bschimke95/jara/pkg/app/startup"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	currentPage tea.Model
	history     []tea.Model // Navigation history stack
}

func New() Model {
	return Model{
		currentPage: startup.New(),
		history:     make([]tea.Model, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return m.currentPage.Init()
}

func (m Model) View() string {
	return m.currentPage.View()
}

// PushToHistory adds the current page to the history and sets a new current page
func (m *Model) PushToHistory(page tea.Model) {
	m.history = append(m.history, m.currentPage)
	m.currentPage = page
}

// PopFromHistory returns to the previous page in the history
func (m *Model) PopFromHistory() (tea.Model, bool) {
	if len(m.history) == 0 {
		return nil, false
	}
	lastIdx := len(m.history) - 1
	prevPage := m.history[lastIdx]
	m.history = m.history[:lastIdx]
	return prevPage, true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			return m, navigation.GoBack()
		}
	case navigation.GoBackMsg:
		if prevPage, ok := m.PopFromHistory(); ok {
			m.currentPage = prevPage
		}
		return m, nil
	case navigation.GoToMsg:
		if !msg.Opts.SkipHistory {
			m.PushToHistory(m.currentPage)
		}
		m.currentPage = msg.Page
		return m, m.currentPage.Init()
	}

	// Ignore model updates from child models since they are handled by the navigation system
	_, cmd = m.currentPage.Update(msg)
	return m, cmd
}
