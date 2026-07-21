package listing_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/listing"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func testCfg(dir string) config.Config {
	cfg := config.Defaults()
	cfg.OutputDir = dir
	cfg.OutputFormats = []string{"txt", "json"}
	return cfg
}

func recs() []voicememos.Recording {
	return []voicememos.Recording{
		{ID: 1, Title: "Meeting", Path: "a.m4a", Duration: 3600, Date: time.Unix(978307200, 0)},
		{ID: 2, Title: "Idea", Path: "b.m4a", Duration: 45, Date: time.Unix(978307200, 0)},
	}
}

func TestBuild_MarksTranscribedRecordings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("こんにちは"), 0644); err != nil {
		t.Fatal(err)
	}

	items := listing.Build(recs(), testCfg(dir))
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if !items[0].Transcribed {
		t.Error("a.m4a has a .txt on disk and must be reported as transcribed")
	}
	if items[1].Transcribed {
		t.Error("b.m4a has no output and must not be reported as transcribed")
	}
}

func TestBuild_CountsRunesNotBytes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("こんにちは"), 0644); err != nil {
		t.Fatal(err)
	}

	items := listing.Build(recs(), testCfg(dir))
	if items[0].Chars != 5 {
		t.Errorf("Chars: got %d, want 5 — 5 Japanese runes, not 15 bytes", items[0].Chars)
	}
	if items[1].Chars != 0 {
		t.Errorf("an untranscribed recording has no characters, got %d", items[1].Chars)
	}
}

func TestBuild_ExposesOutputPaths(t *testing.T) {
	// A consumer that wants to open or reveal the transcription must not have to
	// rebuild the path rule itself.
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "a.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	items := listing.Build(recs(), testCfg(dir))
	if got := items[0].Outputs["txt"]; got != filepath.Join(dir, "a.txt") {
		t.Errorf("outputs[txt]: got %q", got)
	}
	if got := items[0].Outputs["json"]; got != filepath.Join(dir, "a.json") {
		t.Errorf("outputs[json]: got %q", got)
	}
	if len(items[1].Outputs) != 0 {
		t.Errorf("an untranscribed recording has no outputs, got %v", items[1].Outputs)
	}
}

func TestBuild_ExpandsTildeInOutputDir(t *testing.T) {
	cfg := testCfg("~/somewhere")
	items := listing.Build(recs(), cfg)
	for _, p := range items[0].Outputs {
		if strings.HasPrefix(p, "~") {
			t.Errorf("output paths must be absolute, got %q", p)
		}
	}
	// The audio path is absolute too, so callers can play or reveal the file.
	if !filepath.IsAbs(items[0].AudioPath) {
		t.Errorf("AudioPath: got %q, want an absolute path", items[0].AudioPath)
	}
}

func TestBuild_JSONContract(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(listing.Build(recs(), testCfg(dir)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	// The field names are a contract with external consumers (the Raycast
	// extension); renaming one is a breaking change and should fail here.
	for _, key := range []string{
		"id", "title", "path", "audio_path", "duration", "duration_formatted",
		"date", "transcribed", "chars", "outputs",
	} {
		if _, ok := decoded[0][key]; !ok {
			t.Errorf("missing field %q in %v", key, decoded[0])
		}
	}
	if decoded[0]["duration_formatted"] != "1h00m" {
		t.Errorf("duration_formatted: got %v", decoded[0]["duration_formatted"])
	}
}

func TestBuild_EmptyInput(t *testing.T) {
	items := listing.Build(nil, testCfg(t.TempDir()))
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	// An empty list must serialise as [], not null: consumers iterate it.
	if string(data) != "[]" {
		t.Errorf("empty listing: got %s, want []", data)
	}
}
