package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type progressModel struct {
	spinner   spinner.Model
	title     string
	startTime time.Time
}

func newProgressModel(title string) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return progressModel{spinner: s, title: title, startTime: time.Now()}
}

func (m progressModel) Init() tea.Cmd { return m.spinner.Tick }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m progressModel) View() string {
	elapsed := time.Since(m.startTime).Round(time.Second)
	return fmt.Sprintf("%s Transcribing: %s (%s)\n\nCtrl+C to cancel", m.spinner.View(), m.title, elapsed)
}
