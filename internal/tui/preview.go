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
	flashDuration  = 120 * time.Millisecond
	noticeDuration = 2 * time.Second

	// Blank separator line plus the footer line.
	footerHeight = 2

	defaultWidth  = 80
	defaultHeight = 24
)

type flashOffMsg struct{}

type noticeExpiredMsg struct{}

// editorFinishedMsg arrives once the external editor has released the terminal.
type editorFinishedMsg struct {
	err error
}

type previewModel struct {
	content   string
	formatIdx int
	formats   []string
	outputDir string
	stem      string
	editor    []string // command + flags that open a transcription
	notice    string   // transient footer message ("copied!", errors)
	flashing  bool
	viewport  viewport.Model
	width     int
	height    int
}

func newPreviewModel(cfg config.Config, stem string, width, height int) previewModel {
	m := previewModel{
		stem:      stem,
		outputDir: cfg.OutputDir,
		formats:   cfg.OutputFormats,
		editor:    cfg.ResolveEditor(),
	}
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

// currentPath is the file backing the format on screen. The second return is
// false when nothing has been transcribed for it yet.
func (m previewModel) currentPath() (string, bool) {
	if len(m.formats) == 0 {
		return "", false
	}
	idx := m.formatIdx
	if idx >= len(m.formats) {
		idx = 0
	}
	path := filepath.Join(config.ExpandPath(m.outputDir), m.stem+"."+m.formats[idx])
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
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

// afterEdit re-reads the file the editor just had, so an edit that was saved
// outside the TUI is not masked by a stale buffer.
func (m previewModel) afterEdit(err error) (previewModel, tea.Cmd) {
	if err != nil {
		m.notice = fmt.Sprintf("editor failed: %v", err)
	} else {
		m.notice = "reloaded"
	}
	m.loadContent()
	m.viewport.SetContent(m.content)
	return m, noticeExpiredCmd()
}

// copyToClipboard pipes s into pbcopy. macOS only.
func copyToClipboard(s string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

// editorCommand builds the editor invocation: its own flags first, then the file.
func editorCommand(editor []string, path string) *exec.Cmd {
	args := append(append([]string{}, editor[1:]...), path)
	return exec.Command(editor[0], args...)
}

// openEditorCmd suspends the TUI and hands the terminal to the editor, restoring
// the TUI once it exits.
func openEditorCmd(editor []string, path string) tea.Cmd {
	return tea.ExecProcess(editorCommand(editor, path), func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func flashOffCmd() tea.Cmd {
	return tea.Tick(flashDuration, func(time.Time) tea.Msg { return flashOffMsg{} })
}

func noticeExpiredCmd() tea.Cmd {
	return tea.Tick(noticeDuration, func(time.Time) tea.Msg { return noticeExpiredMsg{} })
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

	case noticeExpiredMsg:
		m.notice = ""
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
				m.notice = fmt.Sprintf("copy failed: %v", err)
				return m, noticeExpiredCmd()
			}
			m.notice = "copied!"
			m.flashing = true
			return m, tea.Batch(flashOffCmd(), noticeExpiredCmd())
		case "e":
			path, ok := m.currentPath()
			if !ok {
				m.notice = "nothing to edit — transcribe this recording first"
				return m, noticeExpiredCmd()
			}
			if len(m.editor) == 0 {
				m.notice = "no editor configured"
				return m, noticeExpiredCmd()
			}
			return m, openEditorCmd(m.editor, path)
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
	footer := fmt.Sprintf("[%s] ←/→ switch format • c copy • e edit • esc back", format)
	if m.notice != "" {
		footer = m.notice + " • " + footer
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
