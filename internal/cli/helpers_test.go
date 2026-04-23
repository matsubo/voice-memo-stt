package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/config"
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

func TestHasTranscription(t *testing.T) {
	dir := t.TempDir()
	// Create a fake transcription file
	f, err := os.Create(filepath.Join(dir, "recording.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if !hasTranscription(dir, "recording.m4a", []string{"txt"}) {
		t.Error("should find existing .txt output")
	}
	if hasTranscription(dir, "missing.m4a", []string{"txt"}) {
		t.Error("should not find nonexistent .txt output")
	}
	if hasTranscription(dir, "recording.m4a", nil) {
		t.Error("empty formats slice → no transcription")
	}
}

func TestBuildEngine_MissingKey(t *testing.T) {
	cfg := config.Config{}
	_, err := buildEngine(cfg)
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestBuildEngine_WithKey(t *testing.T) {
	cfg := config.Config{}
	cfg.Engines.ElevenLabs.APIKey = "test-key"
	cfg.Engines.ElevenLabs.Model = "scribe_v2"
	eng, err := buildEngine(cfg)
	if err != nil {
		t.Fatalf("buildEngine: %v", err)
	}
	if eng.Name() != "elevenlabs" {
		t.Errorf("engine Name: got %q", eng.Name())
	}
}
