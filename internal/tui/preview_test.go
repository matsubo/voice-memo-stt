package tui

import (
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCopyToClipboard(t *testing.T) {
	// Only run if pbcopy/pbpaste available
	if _, err := exec.LookPath("pbcopy"); err != nil {
		t.Skip("pbcopy not available")
	}
	if _, err := exec.LookPath("pbpaste"); err != nil {
		t.Skip("pbpaste not available")
	}

	content := "test clipboard content 12345"
	if err := copyToClipboard(content); err != nil {
		t.Fatalf("copyToClipboard: %v", err)
	}

	got, err := exec.Command("pbpaste").Output()
	if err != nil {
		t.Fatalf("pbpaste: %v", err)
	}
	if string(got) != content {
		t.Errorf("clipboard: got %q, want %q", got, content)
	}
}

// newTestPreview builds a preview model without touching the filesystem.
func newTestPreview(t *testing.T, content string, formats []string, w, h int) previewModel {
	t.Helper()
	m := previewModel{content: content, formats: formats}
	m.setSize(w, h)
	return m
}

func TestPreviewView_FooterVisibleWithLongContent(t *testing.T) {
	const height = 24
	long := strings.Repeat("a transcription line that goes on\n", 500)
	m := newTestPreview(t, long, []string{"txt", "md"}, 80, height)

	lines := strings.Split(m.View(), "\n")
	if len(lines) > height {
		t.Fatalf("view is %d lines, exceeds terminal height %d — top scrolls off screen", len(lines), height)
	}

	last := lines[len(lines)-1]
	if !strings.Contains(last, "←/→ switch format • c copy • esc back") {
		t.Errorf("footer must be pinned to the last line, got %q", last)
	}
	if !strings.Contains(last, "[txt]") {
		t.Errorf("footer must show the active format, got %q", last)
	}
}

func TestPreviewView_ShowsContent(t *testing.T) {
	m := newTestPreview(t, "hello transcription", []string{"txt"}, 80, 24)
	if !strings.Contains(m.View(), "hello transcription") {
		t.Error("view should render the transcription content")
	}
}

func TestPreviewUpdate_CopyFlashesAndSetsStatus(t *testing.T) {
	if _, err := exec.LookPath("pbcopy"); err != nil {
		t.Skip("pbcopy not available")
	}
	m := newTestPreview(t, "hello", []string{"txt"}, 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got := updated.(previewModel)

	if !got.flashing {
		t.Error("copy should start a screen flash")
	}
	if got.copyStatus != "copied!" {
		t.Errorf("copyStatus: got %q, want %q", got.copyStatus, "copied!")
	}
	if cmd == nil {
		t.Fatal("copy should return a command that ends the flash")
	}
}

func TestPreviewView_FlashInvertsEveryLine(t *testing.T) {
	m := newTestPreview(t, "hello", []string{"txt"}, 40, 10)
	m.flashing = true

	for i, line := range strings.Split(m.View(), "\n") {
		if !strings.HasPrefix(line, ansiReverseOn) || !strings.HasSuffix(line, ansiReverseOff) {
			t.Fatalf("line %d not inverted during flash: %q", i, line)
		}
	}
}

func TestPreviewView_NoFlashByDefault(t *testing.T) {
	m := newTestPreview(t, "hello", []string{"txt"}, 40, 10)
	if strings.Contains(m.View(), ansiReverseOn) {
		t.Error("view should not be inverted when not flashing")
	}
}

func TestPreviewUpdate_FlashOffClearsFlash(t *testing.T) {
	m := newTestPreview(t, "hello", []string{"txt"}, 80, 24)
	m.flashing = true

	updated, _ := m.Update(flashOffMsg{})
	if updated.(previewModel).flashing {
		t.Error("flashOffMsg should end the flash")
	}
}

func TestPreviewUpdate_CopyStatusExpires(t *testing.T) {
	m := newTestPreview(t, "hello", []string{"txt"}, 80, 24)
	m.copyStatus = "copied!"

	updated, _ := m.Update(copyStatusExpiredMsg{})
	if got := updated.(previewModel).copyStatus; got != "" {
		t.Errorf("copyStatusExpiredMsg should clear the status, got %q", got)
	}
}

func TestPreviewUpdate_WindowSizeResizesViewport(t *testing.T) {
	m := newTestPreview(t, strings.Repeat("line\n", 200), []string{"txt"}, 80, 24)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	got := updated.(previewModel)

	if lines := strings.Split(got.View(), "\n"); len(lines) > 40 {
		t.Errorf("view is %d lines after resize, want <= 40", len(lines))
	}
}

func TestPreviewUpdate_NoFormatsDoesNotPanic(t *testing.T) {
	m := newTestPreview(t, "hello", nil, 80, 24)
	// left/right used to divide by zero when no formats were configured.
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
}
