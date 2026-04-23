package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

type previewModel struct {
	content    string
	formatIdx  int
	formats    []string
	outputDir  string
	stem       string
	copyStatus string    // "copied!" / error message
	copyShown  time.Time // when the status was set
}

func newPreviewModel(stem, outputDir string, formats []string) previewModel {
	m := previewModel{stem: stem, outputDir: outputDir, formats: formats}
	m.loadContent()
	return m
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

// copyToClipboard pipes s into pbcopy. macOS only.
func copyToClipboard(s string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func (m previewModel) Init() tea.Cmd { return nil }

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "right":
			m.formatIdx = (m.formatIdx + 1) % len(m.formats)
			m.loadContent()
		case "left":
			m.formatIdx = (m.formatIdx - 1 + len(m.formats)) % len(m.formats)
			m.loadContent()
		case "c":
			if err := copyToClipboard(m.content); err != nil {
				m.copyStatus = fmt.Sprintf("copy failed: %v", err)
			} else {
				m.copyStatus = "copied!"
			}
			m.copyShown = time.Now()
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m previewModel) View() string {
	format := ""
	if len(m.formats) > 0 {
		format = m.formats[m.formatIdx]
	}
	footer := "←/→ switch format • c copy • esc back"
	if m.copyStatus != "" && time.Since(m.copyShown) < 2*time.Second {
		footer = m.copyStatus + " • " + footer
	}
	return fmt.Sprintf("[%s] %s\n\n%s", format, footer, m.content)
}
