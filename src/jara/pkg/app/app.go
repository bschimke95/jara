package app

import (
	"github.com/bschimke95/jara/pkg/app/startup"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	currentPage tea.Model
}

func New() Model {
	return Model{
		currentPage: startup.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.currentPage.Init()
}

func (m Model) View() string {
	return m.currentPage.View()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.currentPage, cmd = m.currentPage.Update(msg)
	return m, cmd
}
