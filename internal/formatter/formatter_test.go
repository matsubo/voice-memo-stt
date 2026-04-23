package formatter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
)

var speaker0 = "speaker_0"
var speaker1 = "speaker_1"

var testCtx = formatter.Context{
	File:       "20260415_113326.m4a",
	RecordedAt: time.Date(2026, 4, 15, 11, 33, 26, 0, time.UTC),
	Duration:   4009.34,
	Engine:     "elevenlabs",
	Model:      "scribe_v2",
	Segments: []engine.Segment{
		{Time: "00:15", Speaker: &speaker0, Text: "Hello"},
		{Time: "01:23", Speaker: &speaker1, Text: "Hi there"},
	},
}

func TestWrite_AllFormats(t *testing.T) {
	dir := t.TempDir()
	formats := []string{"txt", "md", "json", "csv", "xml"}

	if err := formatter.Write(dir, testCtx, formats); err != nil {
		t.Fatalf("Write: %v", err)
	}

	expected := map[string]string{
		"20260415_113326.txt": "[00:15] speaker_0: Hello\n[01:23] speaker_1: Hi there\n",
		"20260415_113326.md":  "# 20260415_113326\n\n- **00:15** speaker_0: Hello\n- **01:23** speaker_1: Hi there\n",
		"20260415_113326.csv": "time,speaker,text\n00:15,speaker_0,Hello\n01:23,speaker_1,Hi there\n",
	}

	for name, want := range expected {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		if string(data) != want {
			t.Errorf("%s:\ngot:  %q\nwant: %q", name, string(data), want)
		}
	}

	jsonData, _ := os.ReadFile(filepath.Join(dir, "20260415_113326.json"))
	if !strings.Contains(string(jsonData), `"engine": "elevenlabs"`) {
		t.Errorf("JSON missing engine field: %s", jsonData)
	}
	if !strings.Contains(string(jsonData), `"time": "00:15"`) {
		t.Errorf("JSON missing segment time: %s", jsonData)
	}

	xmlData, _ := os.ReadFile(filepath.Join(dir, "20260415_113326.xml"))
	if !strings.Contains(string(xmlData), `<transcription`) {
		t.Errorf("XML missing root element: %s", xmlData)
	}
	if !strings.Contains(string(xmlData), `speaker_0`) {
		t.Errorf("XML missing speaker: %s", xmlData)
	}
}

func TestWrite_UnknownFormat(t *testing.T) {
	err := formatter.Write(t.TempDir(), testCtx, []string{"pdf"})
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestWrite_NoDiarize(t *testing.T) {
	ctx := testCtx
	ctx.Segments = []engine.Segment{
		{Time: "00:00", Text: "hello world"},
	}
	dir := t.TempDir()
	if err := formatter.Write(dir, ctx, []string{"txt"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "20260415_113326.txt"))
	if string(data) != "[00:00] hello world\n" {
		t.Errorf("no-diarize txt: got %q", data)
	}
}
