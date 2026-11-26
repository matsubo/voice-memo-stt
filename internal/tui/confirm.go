package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type confirmModel struct {
	recording voicememos.Recording
	cost      float64
	// overwrite is set when the recording is already transcribed, so confirming
	// replaces its outputs, edits included.
	overwrite bool
}

func newConfirmModel(r voicememos.Recording, cost float64, overwrite bool) confirmModel {
	return confirmModel{recording: r, cost: cost, overwrite: overwrite}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			rec := m.recording
			return m, func() tea.Msg { return startJobMsg{recording: rec} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	warning := ""
	if m.overwrite {
		warning = "\nAlready transcribed: its outputs, including any edits, will be overwritten.\n"
	}
	return fmt.Sprintf(
		"Transcribe: %s\nDuration: %s\nEstimated cost: $%.4f\n%s\n[y] confirm  [n/esc] cancel",
		m.recording.Title,
		m.recording.DurationFormatted(),
		m.cost,
		warning,
	)
}
