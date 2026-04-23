package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type listModel struct {
	table      table.Model
	recordings []voicememos.Recording
}

// hasTranscriptionOutput returns true if any configured output format exists
// for the given recording in outputDir.
func hasTranscriptionOutput(recPath, outputDir string, formats []string) bool {
	if len(formats) == 0 {
		return false
	}
	stem := strings.TrimSuffix(recPath, filepath.Ext(recPath))
	dir := config.ExpandPath(outputDir)
	for _, f := range formats {
		if _, err := os.Stat(filepath.Join(dir, stem+"."+f)); err == nil {
			return true
		}
	}
	return false
}

func newListModel(recs []voicememos.Recording, outputDir string, formats []string) listModel {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "Title", Width: 40},
		{Title: "Date", Width: 17},
		{Title: "Duration", Width: 10},
	}
	rows := make([]table.Row, len(recs))
	for i, r := range recs {
		mark := " "
		if hasTranscriptionOutput(r.Path, outputDir, formats) {
			mark = "✓"
		}
		rows[i] = table.Row{mark, r.Title, r.Date.Format("2006-01-02 15:04"), r.DurationFormatted()}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())
	return listModel{table: t, recordings: recs}
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) selected() (voicememos.Recording, bool) {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.recordings) {
		return voicememos.Recording{}, false
	}
	return m.recordings[idx], true
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if rec, ok := m.selected(); ok {
				return m, func() tea.Msg { return startTranscribeMsg{recording: rec} }
			}
		case "p":
			if _, ok := m.selected(); ok {
				return m, func() tea.Msg { return navigateMsg{to: screenPreview} }
			}
		case "s":
			return m, func() tea.Msg { return navigateMsg{to: screenSettings} }
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	return fmt.Sprintf("%s\n\n✓ = transcribed • ↑/↓ navigate • enter transcribe • p preview • s settings • q quit (confirm)",
		tableStyle.Render(m.table.View()))
}
