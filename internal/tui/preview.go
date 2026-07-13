package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

// Raw SGR codes rather than lipgloss styles: lipgloss strips attributes when the
// color profile degrades to Ascii, which would silently drop the copy flash.
const (
	ansiReverseOn  = "\x1b[7m"
	ansiReverseOff = "\x1b[27m"
)

const (
	flashDuration      = 120 * time.Millisecond
	copyStatusDuration = 2 * time.Second

	// Blank separator line plus the footer line.
	footerHeight = 2

	defaultWidth  = 80
	defaultHeight = 24
)

type flashOffMsg struct{}

type copyStatusExpiredMsg struct{}

type previewModel struct {
	content    string
	formatIdx  int
	formats    []string
	outputDir  string
	stem       string
	copyStatus string // "copied!" / error message
	flashing   bool
	viewport   viewport.Model
	width      int
	height     int
}

func newPreviewModel(stem, outputDir string, formats []string, width, height int) previewModel {
	m := previewModel{stem: stem, outputDir: outputDir, formats: formats}
	m.loadContent()
	m.setSize(width, height)
	return m
}

// setSize lays the screen out as a scrollable content viewport with the footer
// pinned below it, so the whole frame always fits the terminal.
func (m *previewModel) setSize(width, height int) {
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	m.width, m.height = width, height

	viewportHeight := height - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport = viewport.New(width, viewportHeight)
	m.viewport.SetContent(m.content)
}

func (m *previewModel) loadContent() {
	if len(m.formats) == 0 {
		m.content = "(no formats configured)"
		return
	}
	if m.formatIdx >= len(m.formats) {
		m.formatIdx = 0
	}
	path := filepath.Join(config.ExpandPath(m.outputDir), m.stem+"."+m.formats[m.formatIdx])
	data, err := os.ReadFile(path)
	if err != nil {
		m.content = fmt.Sprintf("(no transcription: %v)", err)
		return
	}
	m.content = string(data)
}

// switchFormat moves to another output format and shows it from the top.
func (m *previewModel) switchFormat(delta int) {
	if len(m.formats) == 0 {
		return
	}
	m.formatIdx = (m.formatIdx + delta + len(m.formats)) % len(m.formats)
	m.loadContent()
	m.viewport.SetContent(m.content)
	m.viewport.GotoTop()
}

// copyToClipboard pipes s into pbcopy. macOS only.
func copyToClipboard(s string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func flashOffCmd() tea.Cmd {
	return tea.Tick(flashDuration, func(time.Time) tea.Msg { return flashOffMsg{} })
}

func copyStatusExpiredCmd() tea.Cmd {
	return tea.Tick(copyStatusDuration, func(time.Time) tea.Msg { return copyStatusExpiredMsg{} })
}

func (m previewModel) Init() tea.Cmd { return nil }

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil

	case flashOffMsg:
		m.flashing = false
		return m, nil

	case copyStatusExpiredMsg:
		m.copyStatus = ""
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "right":
			m.switchFormat(1)
			return m, nil
		case "left":
			m.switchFormat(-1)
			return m, nil
		case "c":
			if err := copyToClipboard(m.content); err != nil {
				m.copyStatus = fmt.Sprintf("copy failed: %v", err)
				return m, copyStatusExpiredCmd()
			}
			m.copyStatus = "copied!"
			m.flashing = true
			return m, tea.Batch(flashOffCmd(), copyStatusExpiredCmd())
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m previewModel) footer() string {
	format := ""
	if len(m.formats) > 0 {
		format = m.formats[m.formatIdx]
	}
	footer := fmt.Sprintf("[%s] ←/→ switch format • c copy • esc back", format)
	if m.copyStatus != "" {
		footer = m.copyStatus + " • " + footer
	}
	return footer
}

// invert reverses every line of the frame, padded to full width, so a copy
// registers as a whole-screen blink.
func invert(frame string, width int) string {
	lines := strings.Split(frame, "\n")
	inverted := make([]string, len(lines))
	for i, line := range lines {
		if pad := width - lipgloss.Width(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		inverted[i] = ansiReverseOn + line + ansiReverseOff
	}
	return strings.Join(inverted, "\n")
}

func (m previewModel) View() string {
	frame := m.viewport.View() + "\n\n" + m.footer()
	if m.flashing {
		return invert(frame, m.width)
	}
	return frame
}
