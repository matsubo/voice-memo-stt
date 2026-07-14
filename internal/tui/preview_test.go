package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
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
	if !strings.Contains(last, "←/→ switch format • c copy • e edit • esc back") {
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

func TestPreviewUpdate_CopyFlashesAndSetsNotice(t *testing.T) {
	if _, err := exec.LookPath("pbcopy"); err != nil {
		t.Skip("pbcopy not available")
	}
	m := newTestPreview(t, "hello", []string{"txt"}, 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got := updated.(previewModel)

	if !got.flashing {
		t.Error("copy should start a screen flash")
	}
	if got.notice != "copied!" {
		t.Errorf("notice: got %q, want %q", got.notice, "copied!")
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
	m.notice = "copied!"

	updated, _ := m.Update(noticeExpiredMsg{})
	if got := updated.(previewModel).notice; got != "" {
		t.Errorf("noticeExpiredMsg should clear the status, got %q", got)
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

func TestEditorCommand_PassesFlagsThenPath(t *testing.T) {
	cmd := editorCommand([]string{"nvim"}, "/tmp/a.txt")
	if got := cmd.Args; len(got) != 2 || got[0] != "nvim" || got[1] != "/tmp/a.txt" {
		t.Errorf("args: got %v, want [nvim /tmp/a.txt]", got)
	}

	cmd = editorCommand([]string{"code", "-w"}, "/tmp/a.txt")
	if got := cmd.Args; len(got) != 3 || got[1] != "-w" || got[2] != "/tmp/a.txt" {
		t.Errorf("args: got %v, want [code -w /tmp/a.txt] — the path must come after the flags", got)
	}
}

func TestPreviewCurrentPath_TracksDisplayedFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rec.txt", "plain")
	writeFile(t, dir, "rec.md", "# markdown")

	m := previewModel{stem: "rec", outputDir: dir, formats: []string{"txt", "md"}}

	path, ok := m.currentPath()
	if !ok || filepath.Base(path) != "rec.txt" {
		t.Errorf("currentPath: got %q (ok=%v), want rec.txt", path, ok)
	}

	m.formatIdx = 1
	path, ok = m.currentPath()
	if !ok || filepath.Base(path) != "rec.md" {
		t.Errorf("currentPath after switching format: got %q (ok=%v), want rec.md", path, ok)
	}
}

func TestPreviewCurrentPath_MissingTranscription(t *testing.T) {
	m := previewModel{stem: "rec", outputDir: t.TempDir(), formats: []string{"txt"}}
	if _, ok := m.currentPath(); ok {
		t.Error("currentPath must report false when the transcription does not exist")
	}
}

func TestPreviewUpdate_EditWithoutTranscriptionShowsNotice(t *testing.T) {
	m := previewModel{stem: "rec", outputDir: t.TempDir(), formats: []string{"txt"}, editor: []string{"nvim"}}
	m.setSize(80, 24)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	got := updated.(previewModel)

	if got.notice == "" {
		t.Error("pressing e with nothing to edit should explain why, not open an empty buffer")
	}
	if !strings.Contains(got.View(), got.notice) {
		t.Error("the notice should be visible in the footer")
	}
}

func TestPreviewUpdate_EditOpensEditor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rec.txt", "before")

	m := previewModel{stem: "rec", outputDir: dir, formats: []string{"txt"}, editor: []string{"nvim"}}
	m.setSize(80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("pressing e should hand the terminal to the editor")
	}
	if got := updated.(previewModel).notice; got != "" {
		t.Errorf("no notice expected when the editor actually opens, got %q", got)
	}
}

func TestPreviewAfterEdit_ReloadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rec.txt", "before")

	m := newPreviewModel(config.Config{OutputDir: dir, OutputFormats: []string{"txt"}}, "rec", 80, 24)
	if !strings.Contains(m.View(), "before") {
		t.Fatal("preview should start with the original content")
	}

	// The editor saved new content behind our back.
	writeFile(t, dir, "rec.txt", "after editing")

	got, cmd := m.afterEdit(nil)
	if !strings.Contains(got.View(), "after editing") {
		t.Error("returning from the editor must re-read the file, not show a stale buffer")
	}
	if cmd == nil {
		t.Error("expected a command to expire the notice")
	}
}

func TestPreviewAfterEdit_ReportsEditorFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rec.txt", "text")

	m := newPreviewModel(config.Config{OutputDir: dir, OutputFormats: []string{"txt"}}, "rec", 80, 24)

	got, _ := m.afterEdit(errors.New("exit status 1"))
	if !strings.Contains(got.notice, "exit status 1") {
		t.Errorf("notice: got %q, want it to surface the editor failure", got.notice)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
