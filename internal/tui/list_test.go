package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestHasTranscriptionOutput(t *testing.T) {
	dir := t.TempDir()
	// Create a txt output for "recording.m4a"
	if err := os.WriteFile(filepath.Join(dir, "recording.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	if !hasTranscriptionOutput("recording.m4a", dir, []string{"txt", "json"}) {
		t.Error("should find existing .txt")
	}
	if hasTranscriptionOutput("other.m4a", dir, []string{"txt"}) {
		t.Error("should not find nonexistent output")
	}
	if hasTranscriptionOutput("recording.m4a", dir, nil) {
		t.Error("empty formats should return false")
	}
	// Any configured format match counts as transcribed
	if !hasTranscriptionOutput("recording.m4a", dir, []string{"json", "txt"}) {
		t.Error("should find .txt even when .json is first in formats")
	}
}

func TestTranscriptionChars_PrefersTxtOverOtherFormats(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "recording.txt", "hello")                            // 5 runes
	write(t, dir, "recording.json", `{"segments":[{"text":"hello"}]}`) // much larger
	write(t, dir, "recording.md", "# Transcription\n\nhello")          // larger too

	// json is first in formats, but txt is the honest measure of transcription volume.
	got, ok := transcriptionChars("recording.m4a", dir, []string{"json", "md", "txt"})
	if !ok {
		t.Fatal("expected a count for an existing transcription")
	}
	if got != 5 {
		t.Errorf("chars: got %d, want 5 (the .txt rune count)", got)
	}
}

func TestTranscriptionChars_FallsBackToFirstFormatWhenNoTxt(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "recording.md", "# Hi")

	got, ok := transcriptionChars("recording.m4a", dir, []string{"md", "json"})
	if !ok {
		t.Fatal("expected a count when the first format exists")
	}
	if got != 4 {
		t.Errorf("chars: got %d, want 4 (the .md rune count)", got)
	}
}

func TestTranscriptionChars_CountsRunesNotBytes(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "recording.txt", "こんにちは") // 5 runes, 15 UTF-8 bytes

	got, ok := transcriptionChars("recording.m4a", dir, []string{"txt"})
	if !ok {
		t.Fatal("expected a count")
	}
	if got != 5 {
		t.Errorf("chars: got %d, want 5 runes (not 15 bytes)", got)
	}
}

func TestTranscriptionChars_MissingTranscription(t *testing.T) {
	dir := t.TempDir()

	if _, ok := transcriptionChars("recording.m4a", dir, []string{"txt"}); ok {
		t.Error("missing transcription should report no count")
	}
	if _, ok := transcriptionChars("recording.m4a", dir, nil); ok {
		t.Error("empty formats should report no count")
	}
}

func TestFormatChars(t *testing.T) {
	tests := []struct {
		n    int
		ok   bool
		want string
	}{
		{0, false, "-"},
		{0, true, "0"},
		{47, true, "47"},
		{999, true, "999"},
		{1000, true, "1,000"},
		{2730, true, "2,730"},
		{12345, true, "12,345"},
		{1234567, true, "1,234,567"},
	}
	for _, tt := range tests {
		if got := formatChars(tt.n, tt.ok); got != tt.want {
			t.Errorf("formatChars(%d, %v): got %q, want %q", tt.n, tt.ok, got, tt.want)
		}
	}
}

func TestNewListModel_ShowsCharsColumn(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "recording.txt", strings.Repeat("あ", 2730))

	m := newListModel(testRecordings(), dir, []string{"txt", "json"})

	cols := m.table.Columns()
	if cols[len(cols)-1].Title != "Chars" {
		t.Fatalf("last column: got %q, want %q", cols[len(cols)-1].Title, "Chars")
	}
	if got := m.cells[0].chars; got != "2,730" {
		t.Errorf("transcribed row chars: got %q, want %q", got, "2,730")
	}
	if got := m.cells[1].chars; got != "-" {
		t.Errorf("untranscribed row chars: got %q, want %q", got, "-")
	}
	if !strings.Contains(m.table.View(), "2,730") {
		t.Errorf("table should render the char count:\n%s", m.table.View())
	}
}

func TestListStatus_PendingRunningDoneFailed(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "recording.txt", "done")

	m := newListModel(testRecordings(), dir, []string{"txt"})

	// 実行完了: output on disk.
	if got := m.status(m.cells[0]); got != markDone {
		t.Errorf("transcribed status: got %q, want %q", got, markDone)
	}
	// 未実行: nothing on disk, no job.
	if got := m.status(m.cells[1]); got != markPending {
		t.Errorf("untouched status: got %q, want %q", got, markPending)
	}
	// 実行中: a job is in flight — the spinner takes over the cell.
	running := m.withJobs(map[string]bool{"untouched.m4a": true}, nil)
	if got := running.status(running.cells[1]); got == markPending || got == markDone {
		t.Errorf("running status should be a spinner frame, got %q", got)
	}
	// A failed job is neither pending nor silently "done".
	failed := m.withJobs(nil, map[string]bool{"untouched.m4a": true})
	if got := failed.status(failed.cells[1]); got != markFailed {
		t.Errorf("failed status: got %q, want %q", got, markFailed)
	}
}

func TestListView_ShowsRunningJobCount(t *testing.T) {
	dir := t.TempDir()
	m := newListModel(testRecordings(), dir, []string{"txt"})

	if got := m.jobsLine(); got != "" {
		t.Errorf("idle list should not show a jobs line, got %q", got)
	}

	one := m.withJobs(map[string]bool{"a.m4a": true}, nil)
	if !strings.Contains(one.jobsLine(), "1 transcription job running") {
		t.Errorf("jobs line: got %q, want it to report 1 running job", one.jobsLine())
	}

	two := m.withJobs(map[string]bool{"a.m4a": true, "b.m4a": true}, nil)
	if !strings.Contains(two.jobsLine(), "2 transcription jobs running") {
		t.Errorf("jobs line: got %q, want it to report 2 running jobs", two.jobsLine())
	}
	if !strings.Contains(two.View(), "2 transcription jobs running") {
		t.Error("list view should surface the running job count in the footer")
	}
}

func TestListRefresh_KeepsCursor(t *testing.T) {
	dir := t.TempDir()
	m := newListModel(testRecordings(), dir, []string{"txt"})
	m.table.SetCursor(1)

	refreshed := m.withJobs(map[string]bool{"untouched.m4a": true}, nil)
	if got := refreshed.table.Cursor(); got != 1 {
		t.Errorf("cursor: got %d, want 1 — a job update must not move the user's selection", got)
	}
}

func testRecordings() []voicememos.Recording {
	return []voicememos.Recording{
		{Path: "recording.m4a", Title: "Transcribed", Date: time.Now()},
		{Path: "untouched.m4a", Title: "Untouched", Date: time.Now()},
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
