package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

const (
	markDone    = "✓"
	markFailed  = "✗"
	markPending = " "

	// Width of the Chars cell, so counts line up right-aligned.
	charsWidth = 7
)

// rowCells holds the parts of a row that only change when a transcription is
// written, so the spinner can repaint at 10fps without re-reading the disk.
type rowCells struct {
	path     string
	title    string
	date     string
	duration string
	chars    string
	done     bool
}

type listModel struct {
	table      table.Model
	recordings []voicememos.Recording
	cells      []rowCells
	spinner    spinner.Model
	running    map[string]bool // recording path -> transcription in flight
	failed     map[string]bool // recording path -> last transcription failed
}

// hasTranscriptionOutput returns true if any configured output format exists
// for the given recording in outputDir.
func hasTranscriptionOutput(recPath, outputDir string, formats []string) bool {
	dir := config.ExpandPath(outputDir)
	return len(formatter.ExistingOutputs(dir, recPath, formats)) > 0
}

// transcriptionChars counts the runes in a recording's transcription. The second
// return is false when no transcription exists. Plain text is preferred as the
// measure of how much was said; sizing the JSON output instead would mostly
// measure timestamp metadata.
func transcriptionChars(recPath, outputDir string, formats []string) (int, bool) {
	format, ok := formatter.PreferredFormat(formats)
	if !ok {
		return 0, false
	}
	path := formatter.OutputPath(config.ExpandPath(outputDir), recPath, format)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	return utf8.RuneCount(data), true
}

// formatChars renders a rune count with thousands separators, or "-" when there
// is no transcription to measure.
func formatChars(n int, ok bool) string {
	if !ok {
		return "-"
	}
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func newListModel(recs []voicememos.Recording, outputDir string, formats []string) listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	cells := make([]rowCells, len(recs))
	for i, r := range recs {
		chars, ok := transcriptionChars(r.Path, outputDir, formats)
		cells[i] = rowCells{
			path:     r.Path,
			title:    r.Title,
			date:     r.Date.Format("2006-01-02 15:04"),
			duration: r.DurationFormatted(),
			chars:    formatChars(chars, ok),
			done:     hasTranscriptionOutput(r.Path, outputDir, formats),
		}
	}

	t := table.New(
		table.WithColumns([]table.Column{
			{Title: " ", Width: 2},
			{Title: "Title", Width: 40},
			{Title: "Date", Width: 17},
			{Title: "Duration", Width: 10},
			{Title: "Chars", Width: charsWidth + 2},
		}),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())

	m := listModel{table: t, recordings: recs, cells: cells}
	return m.refresh()
}

// withJobs returns a copy of the list reflecting a new set of in-flight and
// failed jobs.
func (m listModel) withJobs(running, failed map[string]bool) listModel {
	m.running, m.failed = running, failed
	return m.refresh()
}

// status renders the leftmost cell: pending, running, done, or failed.
func (m listModel) status(c rowCells) string {
	switch {
	case m.running[c.path]:
		return m.spinner.View()
	case m.failed[c.path]:
		return markFailed
	case c.done:
		return markDone
	}
	return markPending
}

// refresh rebuilds the table rows from the cached cells plus current job state,
// keeping the cursor where the user left it.
func (m listModel) refresh() listModel {
	rows := make([]table.Row, len(m.cells))
	for i, c := range m.cells {
		rows[i] = table.Row{
			m.status(c),
			c.title,
			c.date,
			c.duration,
			fmt.Sprintf("%*s", charsWidth, c.chars),
		}
	}
	cursor := m.table.Cursor()
	m.table.SetRows(rows)
	m.table.SetCursor(cursor)
	return m
}

func (m listModel) runningCount() int {
	return len(m.running)
}

// spinnerTick starts the animation loop for the running-job indicator.
func (m listModel) spinnerTick() tea.Cmd { return m.spinner.Tick }

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) selected() (voicememos.Recording, bool) {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.recordings) {
		return voicememos.Recording{}, false
	}
	return m.recordings[idx], true
}

// selectedDone reports whether the highlighted recording has a transcription on
// disk, so the preview is only offered when there is something to show.
func (m listModel) selectedDone() bool {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.cells) {
		return false
	}
	return m.cells[idx].done
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Let the animation loop die once the last job finishes; it is restarted
		// when the next one starts.
		if m.runningCount() == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m.refresh(), cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if rec, ok := m.selected(); ok {
				return m, func() tea.Msg { return startTranscribeMsg{recording: rec} }
			}
		case "p":
			// Preview is only meaningful once something has been transcribed; on
			// any other row p does nothing, and the footer hides the hint to match.
			if _, ok := m.selected(); ok && m.selectedDone() {
				return m, func() tea.Msg { return navigateMsg{to: screenPreview} }
			}
			return m, nil
		case "s":
			return m, func() tea.Msg { return navigateMsg{to: screenSettings} }
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// jobsLine summarises in-flight transcriptions, or is empty when the queue is idle.
func (m listModel) jobsLine() string {
	n := m.runningCount()
	if n == 0 {
		return ""
	}
	job := "job"
	if n > 1 {
		job = "jobs"
	}
	return fmt.Sprintf("%s %d transcription %s running", m.spinner.View(), n, job)
}

// helpLine lists the actions available for the row under the cursor, so keys
// that would do nothing (preview on an untranscribed recording) are not offered.
func (m listModel) helpLine() string {
	parts := []string{"✓ done • ✗ failed", "↑/↓ navigate", "enter transcribe"}
	if m.selectedDone() {
		parts = append(parts, "p preview")
	}
	parts = append(parts, "s settings", "q quit (confirm)")
	return strings.Join(parts, " • ")
}

func (m listModel) View() string {
	help := m.helpLine()
	if jobs := m.jobsLine(); jobs != "" {
		help = jobs + "\n" + help
	}
	return fmt.Sprintf("%s\n\n%s", tableStyle.Render(m.table.View()), help)
}
