package tui

import (
	"os"
	"path/filepath"
	"testing"
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
