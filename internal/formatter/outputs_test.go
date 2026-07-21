package formatter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/formatter"
)

func TestOutputPath(t *testing.T) {
	got := formatter.OutputPath("/out", "20260415 113326-AB.m4a", "txt")
	want := filepath.Join("/out", "20260415 113326-AB.txt")
	if got != want {
		t.Errorf("OutputPath: got %q, want %q", got, want)
	}
}

func TestPreferredFormat(t *testing.T) {
	// txt wins wherever it appears: it is the plain-text view every consumer
	// can read without parsing.
	if got, ok := formatter.PreferredFormat([]string{"json", "txt"}); !ok || got != "txt" {
		t.Errorf("PreferredFormat: got %q ok=%v, want txt", got, ok)
	}
	if got, ok := formatter.PreferredFormat([]string{"json", "csv"}); !ok || got != "json" {
		t.Errorf("PreferredFormat without txt: got %q ok=%v, want json", got, ok)
	}
	if _, ok := formatter.PreferredFormat(nil); ok {
		t.Error("PreferredFormat of nothing should report no format")
	}
}

func TestExistingOutputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rec.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	got := formatter.ExistingOutputs(dir, "rec.m4a", []string{"txt", "md", "json"})
	if len(got) != 1 {
		t.Fatalf("only the written format should be reported, got %v", got)
	}
	if got["txt"] != filepath.Join(dir, "rec.txt") {
		t.Errorf("txt path: got %q", got["txt"])
	}
}

func TestExistingOutputs_NoneIsEmptyNotNilPanic(t *testing.T) {
	got := formatter.ExistingOutputs(t.TempDir(), "rec.m4a", []string{"txt"})
	if len(got) != 0 {
		t.Errorf("nothing on disk should yield no outputs, got %v", got)
	}
}
