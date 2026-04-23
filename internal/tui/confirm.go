package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type confirmModel struct {
	recording voicememos.Recording
	cost      float64
}

func newConfirmModel(r voicememos.Recording, cost float64) confirmModel {
	return confirmModel{recording: r, cost: cost}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			return m, func() tea.Msg { return navigateMsg{to: screenProgress} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	return fmt.Sprintf(
		"Transcribe: %s\nDuration: %s\nEstimated cost: $%.4f\n\n[y] confirm  [n/esc] cancel",
		m.recording.Title,
		m.recording.DurationFormatted(),
		m.cost,
	)
}
