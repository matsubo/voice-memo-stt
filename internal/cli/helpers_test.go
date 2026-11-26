package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"short", "***"},
		{"12345678", "***"},
		{"sk-abc123456def", "sk-a***6def"},
	}
	for _, tt := range tests {
		if got := maskKey(tt.in); got != tt.want {
			t.Errorf("maskKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"foo.m4a", "foo.m4a"},
		{"foo", "foo.m4a"},
		{"/path/to/foo.m4a", "foo.m4a"},
		{"/path/to/foo", "foo.m4a"},
	}
	for _, tt := range tests {
		if got := normalizePath(tt.in); got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStripExt(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"foo.m4a", "foo"},
		{"foo.bar.m4a", "foo.bar"},
		{"foo", "foo"},
	}
	for _, tt := range tests {
		if got := stripExt(tt.in); got != tt.want {
			t.Errorf("stripExt(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// countingEngine records how often it was asked to transcribe, so a test can
// prove a guarded run never reached (and never billed) the Engine.
type countingEngine struct{ calls int }

func (e *countingEngine) Name() string        { return "counting" }
func (e *countingEngine) Model() engine.Model { return engine.Model{ID: "counting-1"} }
func (e *countingEngine) Transcribe(context.Context, string, engine.TranscribeOptions) (*engine.TranscribeResult, error) {
	e.calls++
	return &engine.TranscribeResult{}, nil
}

// withTranscribeFlags sets the transcribe command's flags for one test.
func withTranscribeFlags(t *testing.T, yes, overwrite bool) {
	t.Helper()
	prevYes, prevOverwrite := transcribeYes, transcribeOverwrite
	transcribeYes, transcribeOverwrite = yes, overwrite
	t.Cleanup(func() { transcribeYes, transcribeOverwrite = prevYes, prevOverwrite })
}

func TestTranscribeOne_RetranscriptionNeedsOverwrite(t *testing.T) {
	dir := t.TempDir()
	edited := filepath.Join(dir, "rec.txt")
	if err := os.WriteFile(edited, []byte("edited by hand"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{OutputDir: dir, OutputFormats: []string{"txt"}}
	recs := []voicememos.Recording{{Title: "Rec", Path: "rec.m4a"}}

	withTranscribeFlags(t, true, false)
	eng := &countingEngine{}
	err := transcribeOne(context.Background(), recs, eng, cfg, "rec.m4a")
	if err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("--yes on a transcribed recording must fail and name --overwrite, got %v", err)
	}
	if eng.calls != 0 {
		t.Errorf("the Engine must not be called without --overwrite, got %d calls", eng.calls)
	}
	if data, _ := os.ReadFile(edited); string(data) != "edited by hand" {
		t.Errorf("the edited output must survive, got %q", data)
	}

	withTranscribeFlags(t, true, true)
	if err := transcribeOne(context.Background(), recs, eng, cfg, "rec.m4a"); err != nil {
		t.Fatalf("--yes --overwrite must re-transcribe: %v", err)
	}
	if eng.calls != 1 {
		t.Errorf("--overwrite must reach the Engine once, got %d calls", eng.calls)
	}
}

func TestTranscribeOne_PendingNeedsNoOverwrite(t *testing.T) {
	cfg := config.Config{OutputDir: t.TempDir(), OutputFormats: []string{"txt"}}
	recs := []voicememos.Recording{{Title: "Rec", Path: "rec.m4a"}}

	withTranscribeFlags(t, true, false)
	eng := &countingEngine{}
	if err := transcribeOne(context.Background(), recs, eng, cfg, "rec.m4a"); err != nil {
		t.Fatalf("--yes alone must transcribe a pending recording: %v", err)
	}
	if eng.calls != 1 {
		t.Errorf("want 1 Engine call, got %d", eng.calls)
	}
}

func TestWatchHandler_SkipsTranscribed(t *testing.T) {
	dir := t.TempDir()
	edited := filepath.Join(dir, "rec.txt")
	if err := os.WriteFile(edited, []byte("edited by hand"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{OutputDir: dir, OutputFormats: []string{"txt"}}
	eng := &countingEngine{}
	notified := 0

	handle := watchHandler(eng, cfg, func(string) { notified++ })
	if err := handle(context.Background(), "/voice-memos/rec.m4a"); err != nil {
		t.Fatal(err)
	}
	if eng.calls != 0 || notified != 0 {
		t.Errorf("a transcribed recording must be skipped, got %d Engine calls, %d notifications", eng.calls, notified)
	}
	if data, _ := os.ReadFile(edited); string(data) != "edited by hand" {
		t.Errorf("the edited output must survive, got %q", data)
	}
}

func TestWatchHandler_TranscribesPending(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{OutputDir: dir, OutputFormats: []string{"txt"}}
	eng := &countingEngine{}
	notified := 0

	handle := watchHandler(eng, cfg, func(string) { notified++ })
	if err := handle(context.Background(), "/voice-memos/new.m4a"); err != nil {
		t.Fatal(err)
	}
	if eng.calls != 1 || notified != 1 {
		t.Errorf("a pending recording must be transcribed once and notified, got %d calls, %d notifications", eng.calls, notified)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); err != nil {
		t.Errorf("the output must be written: %v", err)
	}
}
