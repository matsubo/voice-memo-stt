package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/config"
)

func TestDefaults(t *testing.T) {
	cfg := config.Defaults()
	if cfg.Engine != "elevenlabs" {
		t.Errorf("Engine: got %q", cfg.Engine)
	}
	if len(cfg.OutputFormats) == 0 {
		t.Error("OutputFormats should not be empty")
	}
	if cfg.Engines.ElevenLabs.Model != "scribe_v2" {
		t.Errorf("ElevenLabs model: got %q", cfg.Engines.ElevenLabs.Model)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("missing file should return defaults, got error: %v", err)
	}
	if cfg.Engine != "elevenlabs" {
		t.Errorf("Engine: got %q", cfg.Engine)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	original := config.Defaults()
	original.Engine = "whisper"
	original.LanguageCode = "eng"

	if err := config.Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Engine != "whisper" {
		t.Errorf("Engine: got %q", loaded.Engine)
	}
	if loaded.LanguageCode != "eng" {
		t.Errorf("LanguageCode: got %q", loaded.LanguageCode)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("ELEVENLABS_API_KEY", "test-key-123")
	t.Setenv("VMT_ENGINE", "whisper")

	cfg, err := config.Load("/nonexistent/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Engines.ElevenLabs.APIKey != "test-key-123" {
		t.Errorf("APIKey override: got %q", cfg.Engines.ElevenLabs.APIKey)
	}
	if cfg.Engine != "whisper" {
		t.Errorf("Engine override: got %q", cfg.Engine)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := config.ExpandPath("~/foo"); got != filepath.Join(home, "foo") {
		t.Errorf("ExpandPath(~/foo): got %q", got)
	}
	if got := config.ExpandPath("/absolute/path"); got != "/absolute/path" {
		t.Errorf("ExpandPath(/absolute/path): got %q", got)
	}
}

func TestDefaultPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "vmt", "config.json")
	if got := config.DefaultPath(); got != want {
		t.Errorf("DefaultPath: got %q, want %q", got, want)
	}
}

func TestEnvOverrideOutputDirAndLanguage(t *testing.T) {
	t.Setenv("VMT_OUTPUT_DIR", "/tmp/transcripts")
	t.Setenv("VMT_LANGUAGE", "eng")

	cfg, err := config.Load("/nonexistent/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutputDir != "/tmp/transcripts" {
		t.Errorf("OutputDir override: got %q", cfg.OutputDir)
	}
	if cfg.LanguageCode != "eng" {
		t.Errorf("LanguageCode override: got %q", cfg.LanguageCode)
	}
}

func TestSaveCreatesMissingDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "subdir", "config.json")

	cfg := config.Defaults()
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save with nested path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(path)
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
}
