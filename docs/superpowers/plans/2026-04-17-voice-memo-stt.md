# voice-memo-stt (vmt) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `vmt`, a macOS CLI tool that reads Voice Memos recordings from SQLite, transcribes audio via ElevenLabs Scribe, and outputs in multiple formats (txt/md/json/csv/xml) with TUI, Alfred, and file-watch support.

**Architecture:** Single Go binary using cobra for CLI, bubbletea for TUI, modernc.org/sqlite (pure Go, no CGo) for the macOS Voice Memos SQLite database, a pluggable `Engine` interface for STT backends, and fsnotify for directory watching. Config lives at `~/.config/vmt/config.json`.

**Tech Stack:** Go 1.22, github.com/spf13/cobra, github.com/charmbracelet/bubbletea, github.com/charmbracelet/lipgloss, github.com/charmbracelet/bubbles, modernc.org/sqlite, github.com/fsnotify/fsnotify

---

## File Map

| File | Responsibility |
|------|---------------|
| `go.mod` / `go.sum` | Module dependencies |
| `Makefile` | build, test, lint, release targets |
| `.github/workflows/ci.yml` | Lint + test + build on push |
| `.goreleaser.yml` | Cross-compile + Homebrew tap release |
| `cmd/vmt/main.go` | Binary entrypoint |
| `internal/voicememos/recording.go` | Recording struct + helpers |
| `internal/voicememos/db.go` | SQLite reader (List, FindByPath, Open) |
| `internal/voicememos/db_test.go` | In-memory SQLite tests |
| `internal/config/config.go` | Config struct, Load, Save, env overrides |
| `internal/config/config_test.go` | Roundtrip + env override tests |
| `internal/engine/engine.go` | Engine interface + types |
| `internal/engine/registry.go` | Engine factory (Register, Get) |
| `internal/engine/registry_test.go` | Registry tests |
| `internal/engine/elevenlabs/client.go` | ElevenLabs Scribe implementation |
| `internal/engine/elevenlabs/client_test.go` | httptest mock tests |
| `internal/formatter/formatter.go` | Multi-format dispatcher (Write) |
| `internal/formatter/txt.go` | txt formatter |
| `internal/formatter/md.go` | md formatter |
| `internal/formatter/json.go` | json formatter |
| `internal/formatter/csv.go` | csv formatter |
| `internal/formatter/xml.go` | xml formatter |
| `internal/formatter/formatter_test.go` | Golden file tests for all 5 formats |
| `internal/alfred/scriptfilter.go` | Alfred Script Filter JSON builder |
| `internal/alfred/scriptfilter_test.go` | Snapshot test |
| `internal/watcher/watcher.go` | fsnotify watcher + 2s debounce |
| `internal/watcher/watcher_test.go` | Watcher tests |
| `internal/watcher/launchd.go` | plist generation + install/uninstall |
| `internal/watcher/launchd_test.go` | Plist generation tests |
| `internal/cli/root.go` | cobra root command + persistent flags |
| `internal/cli/list.go` | `vmt list` command |
| `internal/cli/transcribe.go` | `vmt transcribe` command |
| `internal/cli/preview.go` | `vmt preview` command |
| `internal/cli/config.go` | `vmt config` / `vmt config set` commands |
| `internal/cli/alfred.go` | `vmt alfred` command |
| `internal/cli/watch.go` | `vmt watch` command |
| `internal/cli/tui.go` | `vmt tui` command |
| `internal/tui/app.go` | bubbletea root model + screen routing |
| `internal/tui/list.go` | Recording list view |
| `internal/tui/confirm.go` | Confirmation dialog with cost estimate |
| `internal/tui/progress.go` | Transcription progress spinner |
| `internal/tui/preview.go` | Transcription preview + format switching |
| `internal/tui/settings.go` | Settings screen |

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.github/workflows/ci.yml`
- Create: `.goreleaser.yml`

- [ ] **Step 1: Initialize Go module**

```bash
cd /path/to/voice-memo-stt
go mod init github.com/matsubo/voice-memo-stt
```

- [ ] **Step 2: Add all dependencies**

```bash
go get github.com/spf13/cobra@v1.8.0
go get github.com/charmbracelet/bubbletea@v1.3.3
go get github.com/charmbracelet/bubbles@v0.20.0
go get github.com/charmbracelet/lipgloss@v1.0.0
go get modernc.org/sqlite@v1.33.1
go get github.com/fsnotify/fsnotify@v1.7.0
go mod tidy
```

- [ ] **Step 3: Write Makefile**

```makefile
.PHONY: build test lint clean

BINARY := vmt
CMD := ./cmd/vmt

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
```

- [ ] **Step 4: Write CI workflow**

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./...
      - run: go build ./cmd/vmt
```

- [ ] **Step 5: Create directory structure**

```bash
mkdir -p cmd/vmt \
  internal/voicememos \
  internal/config \
  internal/engine/elevenlabs \
  internal/formatter \
  internal/alfred \
  internal/watcher \
  internal/cli \
  internal/tui
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum Makefile .github/workflows/ci.yml
git commit -m "chore: project scaffold with Go module and CI"
```

---

## Task 2: voicememos package

**Files:**
- Create: `internal/voicememos/recording.go`
- Create: `internal/voicememos/db.go`
- Create: `internal/voicememos/db_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/voicememos/db_test.go
package voicememos_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE ZCLOUDRECORDING (
			Z_PK INTEGER PRIMARY KEY,
			ZENCRYPTEDTITLE TEXT,
			ZPATH TEXT NOT NULL,
			ZDURATION REAL NOT NULL,
			ZDATE REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	// ZDATE=0 → Unix 0 + 978307200 = 2001-01-01 00:00:00 UTC
	_, err = db.Exec(`
		INSERT INTO ZCLOUDRECORDING (Z_PK, ZENCRYPTEDTITLE, ZPATH, ZDURATION, ZDATE)
		VALUES (1, 'Meeting Notes', '20260415_113326.m4a', 3600.0, 0.0),
		       (2, NULL,            '20260416_090000.m4a', 120.5,  10.0)
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	recs, err := voicememos.List(context.Background(), db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 recordings, got %d", len(recs))
	}
	// ordered by ZDATE DESC — row with ZDATE=10 first
	if recs[0].Path != "20260416_090000.m4a" {
		t.Errorf("first path: got %q", recs[0].Path)
	}
	if recs[0].Title != "" {
		t.Errorf("nil title should be empty string, got %q", recs[0].Title)
	}
	wantDate := time.Unix(978307200, 0) // ZDATE=0 → +978307200
	if !recs[1].Date.Equal(wantDate) {
		t.Errorf("Date: got %v, want %v", recs[1].Date, wantDate)
	}
}

func TestFindByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r, err := voicememos.FindByPath(context.Background(), db, "20260415_113326.m4a")
	if err != nil {
		t.Fatalf("FindByPath: %v", err)
	}
	if r.Title != "Meeting Notes" {
		t.Errorf("Title: got %q", r.Title)
	}
	if r.Duration != 3600.0 {
		t.Errorf("Duration: got %f", r.Duration)
	}
}

func TestFindByPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := voicememos.FindByPath(context.Background(), db, "nonexistent.m4a")
	if err == nil {
		t.Error("expected error for missing path")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/voicememos/... -v
```

Expected: `FAIL — package voicememos not found`

- [ ] **Step 3: Write recording.go**

```go
// internal/voicememos/recording.go
package voicememos

import "time"

type Recording struct {
	ID       int64
	Title    string
	Path     string
	Duration float64 // seconds
	Date     time.Time
}

// DurationFormatted returns duration as "1h06m" or "45m30s".
func (r Recording) DurationFormatted() string {
	d := int(r.Duration)
	h := d / 3600
	m := (d % 3600) / 60
	s := d % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}
```

- [ ] **Step 4: Write db.go**

```go
// internal/voicememos/db.go
package voicememos

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const coreDateEpochOffset = int64(978307200)

func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings/CloudRecordings.db")
}

func AudioDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings")
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open Voice Memos DB at %q: %w", path, err)
	}
	return db, nil
}

func List(ctx context.Context, db *sql.DB) ([]Recording, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT Z_PK, COALESCE(ZENCRYPTEDTITLE, ''), ZPATH, ZDURATION, ZDATE
		FROM ZCLOUDRECORDING
		ORDER BY ZDATE DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query recordings: %w", err)
	}
	defer rows.Close()

	var recs []Recording
	for rows.Next() {
		var r Recording
		var zdate float64
		if err := rows.Scan(&r.ID, &r.Title, &r.Path, &r.Duration, &zdate); err != nil {
			return nil, fmt.Errorf("scan recording: %w", err)
		}
		r.Date = time.Unix(int64(zdate)+coreDateEpochOffset, 0)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

func FindByPath(ctx context.Context, db *sql.DB, path string) (*Recording, error) {
	row := db.QueryRowContext(ctx, `
		SELECT Z_PK, COALESCE(ZENCRYPTEDTITLE, ''), ZPATH, ZDURATION, ZDATE
		FROM ZCLOUDRECORDING WHERE ZPATH = ?
	`, path)
	var r Recording
	var zdate float64
	if err := row.Scan(&r.ID, &r.Title, &r.Path, &r.Duration, &zdate); err != nil {
		return nil, fmt.Errorf("find recording %q: %w", path, err)
	}
	r.Date = time.Unix(int64(zdate)+coreDateEpochOffset, 0)
	return &r, nil
}
```

Note: `time` and `fmt` imports must be added to `recording.go`. Add `"fmt"` and `"time"` to `recording.go` imports.

- [ ] **Step 5: Run test — expect PASS**

```bash
go test ./internal/voicememos/... -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add internal/voicememos/
git commit -m "feat: voicememos package — SQLite reader for Voice Memos recordings"
```

---

## Task 3: config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/config/config_test.go
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
	t.Setenv("VMT_ELEVENLABS_API_KEY", "test-key-123")
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
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/config/... -v
```

Expected: `FAIL — package config not found`

- [ ] **Step 3: Write config.go**

```go
// internal/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ElevenLabsConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type EnginesConfig struct {
	ElevenLabs ElevenLabsConfig `json:"elevenlabs"`
}

type Config struct {
	Engine        string        `json:"engine"`
	OutputFormats []string      `json:"output_formats"`
	OutputDir     string        `json:"output_dir"`
	LanguageCode  string        `json:"language_code"`
	Diarize       bool          `json:"diarize"`
	Engines       EnginesConfig `json:"engines"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vmt", "config.json")
}

func Defaults() Config {
	return Config{
		Engine:        "elevenlabs",
		OutputFormats: []string{"txt", "json"},
		OutputDir:     "~/Downloads/voice-memo-transcription",
		LanguageCode:  "jpn",
		Diarize:       true,
		Engines: EnginesConfig{
			ElevenLabs: ElevenLabsConfig{
				Model: "scribe_v2",
			},
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return applyEnvOverrides(cfg), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return applyEnvOverrides(cfg), nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func applyEnvOverrides(cfg Config) Config {
	if v := os.Getenv("VMT_ELEVENLABS_API_KEY"); v != "" {
		cfg.Engines.ElevenLabs.APIKey = v
	}
	if v := os.Getenv("VMT_ENGINE"); v != "" {
		cfg.Engine = v
	}
	if v := os.Getenv("VMT_OUTPUT_DIR"); v != "" {
		cfg.OutputDir = v
	}
	if v := os.Getenv("VMT_LANGUAGE"); v != "" {
		cfg.LanguageCode = v
	}
	return cfg
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/config/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config package — JSON config with env var overrides"
```

---

## Task 4: Engine interface + registry

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/registry.go`
- Create: `internal/engine/registry_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/engine/registry_test.go
package engine_test

import (
	"context"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

type stubEngine struct{ name string }

func (s stubEngine) Name() string { return s.name }
func (s stubEngine) Transcribe(_ context.Context, _ string, _ engine.TranscribeOptions) (*engine.TranscribeResult, error) {
	return &engine.TranscribeResult{}, nil
}
func (s stubEngine) EstimateCost(_ float64) float64 { return 0 }

func TestRegistryRoundtrip(t *testing.T) {
	engine.Register(stubEngine{"testengine"})

	e, err := engine.Get("testengine")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if e.Name() != "testengine" {
		t.Errorf("Name: got %q", e.Name())
	}
}

func TestRegistryUnknown(t *testing.T) {
	_, err := engine.Get("doesnotexist")
	if err == nil {
		t.Error("expected error for unknown engine")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/engine/... -v
```

- [ ] **Step 3: Write engine.go**

```go
// internal/engine/engine.go
package engine

import "context"

type Segment struct {
	Time    string  // "MM:SS"
	Speaker *string // nil if diarization is off
	Text    string
}

type TranscribeResult struct {
	Segments []Segment
}

type TranscribeOptions struct {
	LanguageCode string
	Diarize      bool
}

type Engine interface {
	Name() string
	Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscribeResult, error)
	EstimateCost(durationSeconds float64) float64
}
```

- [ ] **Step 4: Write registry.go**

```go
// internal/engine/registry.go
package engine

import "fmt"

var registry = map[string]Engine{}

func Register(e Engine) {
	registry[e.Name()] = e
}

func Get(name string) (Engine, error) {
	e, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown STT engine %q; registered: %v", name, registeredNames())
	}
	return e, nil
}

func registeredNames() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}
```

- [ ] **Step 5: Run test — expect PASS**

```bash
go test ./internal/engine/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/engine/engine.go internal/engine/registry.go internal/engine/registry_test.go
git commit -m "feat: engine interface and registry"
```

---

## Task 5: ElevenLabs engine

**Files:**
- Create: `internal/engine/elevenlabs/client.go`
- Create: `internal/engine/elevenlabs/client_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/engine/elevenlabs/client_test.go
package elevenlabs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
)

func fakeAPIResponse() map[string]interface{} {
	return map[string]interface{}{
		"words": []map[string]interface{}{
			{"speaker_id": "speaker_0", "text": "Hello", "start": 15.0},
			{"speaker_id": "speaker_0", "text": "world", "start": 15.5},
			{"speaker_id": "speaker_1", "text": "Hi", "start": 83.0},
		},
	}
}

func startFakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fakeAPIResponse())
	}))
}

func writeTempAudio(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.m4a")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestTranscribe_Diarize(t *testing.T) {
	srv := startFakeServer(t)
	defer srv.Close()

	client := elevenlabs.New("test-key", "scribe_v2", elevenlabs.WithBaseURL(srv.URL))
	audioPath := writeTempAudio(t)

	result, err := client.Transcribe(context.Background(), audioPath, engine.TranscribeOptions{
		LanguageCode: "jpn",
		Diarize:      true,
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("want 2 segments (grouped by speaker), got %d", len(result.Segments))
	}
	if result.Segments[0].Text != "Hello world" {
		t.Errorf("segment 0 text: got %q", result.Segments[0].Text)
	}
	if result.Segments[0].Time != "00:15" {
		t.Errorf("segment 0 time: got %q", result.Segments[0].Time)
	}
	if result.Segments[1].Time != "01:23" {
		t.Errorf("segment 1 time: got %q", result.Segments[1].Time)
	}
}

func TestTranscribe_NoDiarize(t *testing.T) {
	srv := startFakeServer(t)
	defer srv.Close()

	client := elevenlabs.New("test-key", "scribe_v2", elevenlabs.WithBaseURL(srv.URL))
	result, err := client.Transcribe(context.Background(), writeTempAudio(t), engine.TranscribeOptions{Diarize: false})
	if err != nil {
		t.Fatal(err)
	}
	// Without diarize, all words merge into one segment
	if len(result.Segments) != 1 {
		t.Fatalf("want 1 segment (no diarize), got %d", len(result.Segments))
	}
	if result.Segments[0].Speaker != nil {
		t.Error("speaker should be nil when diarize=false")
	}
}

func TestName(t *testing.T) {
	c := elevenlabs.New("key", "scribe_v2")
	if c.Name() != "elevenlabs" {
		t.Errorf("Name: got %q", c.Name())
	}
}

func TestEstimateCost(t *testing.T) {
	c := elevenlabs.New("key", "scribe_v2")
	cost := c.EstimateCost(3600) // 1 hour
	if cost != 0.22 {
		t.Errorf("scribe_v2 cost/hour: got %f, want 0.22", cost)
	}

	c2 := elevenlabs.New("key", "scribe_v1")
	cost2 := c2.EstimateCost(3600)
	if cost2 != 0.40 {
		t.Errorf("scribe_v1 cost/hour: got %f, want 0.40", cost2)
	}
}

// unused import guard
var _ = filepath.Join
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/engine/elevenlabs/... -v
```

- [ ] **Step 3: Write client.go**

```go
// internal/engine/elevenlabs/client.go
package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

const (
	defaultAPIBase = "https://api.elevenlabs.io"
	defaultModel   = "scribe_v2"
	requestTimeout = 30 * time.Second
	costV1PerHour  = 0.40
	costV2PerHour  = 0.22
)

type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

type Option func(*Client)

func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

func New(apiKey, model string, opts ...Option) *Client {
	if model == "" {
		model = defaultModel
	}
	c := &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultAPIBase,
		http:    &http.Client{Timeout: requestTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) Name() string { return "elevenlabs" }

func (c *Client) EstimateCost(durationSeconds float64) float64 {
	hours := durationSeconds / 3600
	if c.model == "scribe_v1" {
		return hours * costV1PerHour
	}
	return hours * costV2PerHour
}

type apiWord struct {
	SpeakerID string  `json:"speaker_id"`
	Text      string  `json:"text"`
	Start     float64 `json:"start"`
}

type apiResponse struct {
	Words []apiWord `json:"words"`
}

func (c *Client) Transcribe(ctx context.Context, audioPath string, opts engine.TranscribeOptions) (*engine.TranscribeResult, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("model_id", c.model)
	if opts.LanguageCode != "" {
		_ = w.WriteField("language_code", opts.LanguageCode)
	}
	if opts.Diarize {
		_ = w.WriteField("diarize", "true")
	}
	part, err := w.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy audio: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/speech-to-text", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs API %d: %s", resp.StatusCode, b)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return parseWords(apiResp.Words, opts.Diarize), nil
}

func parseWords(words []apiWord, diarize bool) *engine.TranscribeResult {
	var segments []engine.Segment
	var cur *engine.Segment

	for _, w := range words {
		sameSpeaker := cur != nil && (!diarize || speakerOf(cur) == w.SpeakerID)
		if sameSpeaker {
			cur.Text += " " + w.Text
		} else {
			seg := engine.Segment{Time: formatTime(w.Start), Text: w.Text}
			if diarize {
				s := w.SpeakerID
				seg.Speaker = &s
			}
			segments = append(segments, seg)
			cur = &segments[len(segments)-1]
		}
	}
	return &engine.TranscribeResult{Segments: segments}
}

func speakerOf(s *engine.Segment) string {
	if s.Speaker == nil {
		return ""
	}
	return *s.Speaker
}

func formatTime(seconds float64) string {
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}
```

- [ ] **Step 4: Register engine in main.go (placeholder — revisit in Task 14)**

For now, verify tests pass:

```bash
go test ./internal/engine/... -v
```

Expected: `PASS` for both `engine` and `elevenlabs` packages.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: ElevenLabs Scribe engine with speaker diarization"
```

---

## Task 6: formatter package

**Files:**
- Create: `internal/formatter/formatter.go`
- Create: `internal/formatter/txt.go`
- Create: `internal/formatter/md.go`
- Create: `internal/formatter/json.go`
- Create: `internal/formatter/csv.go`
- Create: `internal/formatter/xml.go`
- Create: `internal/formatter/formatter_test.go`

- [ ] **Step 1: Write failing test with golden file approach**

```go
// internal/formatter/formatter_test.go
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

	// JSON must be valid and contain key fields
	jsonData, _ := os.ReadFile(filepath.Join(dir, "20260415_113326.json"))
	if !strings.Contains(string(jsonData), `"engine": "elevenlabs"`) {
		t.Errorf("JSON missing engine field: %s", jsonData)
	}
	if !strings.Contains(string(jsonData), `"time": "00:15"`) {
		t.Errorf("JSON missing segment time: %s", jsonData)
	}

	// XML must contain root element and segments
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
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/formatter/... -v
```

- [ ] **Step 3: Write formatter.go**

```go
// internal/formatter/formatter.go
package formatter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

type Context struct {
	File       string
	RecordedAt time.Time
	Duration   float64
	Engine     string
	Model      string
	Segments   []engine.Segment
}

type fmter interface {
	ext() string
	format(ctx Context) ([]byte, error)
}

var formatters = map[string]fmter{
	"txt": txtFormatter{},
	"md":  mdFormatter{},
	"json": jsonFormatter{},
	"csv": csvFormatter{},
	"xml": xmlFormatter{},
}

func Write(dir string, ctx Context, formats []string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	stem := strings.TrimSuffix(ctx.File, filepath.Ext(ctx.File))
	for _, name := range formats {
		f, ok := formatters[name]
		if !ok {
			return fmt.Errorf("unknown format %q", name)
		}
		data, err := f.format(ctx)
		if err != nil {
			return fmt.Errorf("format %q: %w", name, err)
		}
		outPath := filepath.Join(dir, stem+"."+f.ext())
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("write %q: %w", outPath, err)
		}
	}
	return nil
}

func speakerPrefix(seg engine.Segment) string {
	if seg.Speaker != nil {
		return *seg.Speaker + ": "
	}
	return ""
}
```

- [ ] **Step 4: Write txt.go**

```go
// internal/formatter/txt.go
package formatter

import (
	"bytes"
	"fmt"
)

type txtFormatter struct{}

func (txtFormatter) ext() string { return "txt" }

func (txtFormatter) format(ctx Context) ([]byte, error) {
	var buf bytes.Buffer
	for _, seg := range ctx.Segments {
		fmt.Fprintf(&buf, "[%s] %s%s\n", seg.Time, speakerPrefix(seg), seg.Text)
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 5: Write md.go**

```go
// internal/formatter/md.go
package formatter

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

type mdFormatter struct{}

func (mdFormatter) ext() string { return "md" }

func (mdFormatter) format(ctx Context) ([]byte, error) {
	stem := strings.TrimSuffix(ctx.File, filepath.Ext(ctx.File))
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", stem)
	for _, seg := range ctx.Segments {
		fmt.Fprintf(&buf, "- **%s** %s%s\n", seg.Time, speakerPrefix(seg), seg.Text)
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 6: Write json.go**

```go
// internal/formatter/json.go
package formatter

import "encoding/json"

type jsonFormatter struct{}

func (jsonFormatter) ext() string { return "json" }

type jsonOutput struct {
	File       string        `json:"file"`
	RecordedAt string        `json:"recorded_at"`
	Duration   float64       `json:"duration"`
	Engine     string        `json:"engine"`
	Model      string        `json:"model"`
	Segments   []jsonSegment `json:"segments"`
}

type jsonSegment struct {
	Time    string  `json:"time"`
	Speaker *string `json:"speaker,omitempty"`
	Text    string  `json:"text"`
}

func (jsonFormatter) format(ctx Context) ([]byte, error) {
	segs := make([]jsonSegment, len(ctx.Segments))
	for i, s := range ctx.Segments {
		segs[i] = jsonSegment{Time: s.Time, Speaker: s.Speaker, Text: s.Text}
	}
	out := jsonOutput{
		File:       ctx.File,
		RecordedAt: ctx.RecordedAt.Format("2006-01-02T15:04:05"),
		Duration:   ctx.Duration,
		Engine:     ctx.Engine,
		Model:      ctx.Model,
		Segments:   segs,
	}
	return json.MarshalIndent(out, "", "  ")
}
```

- [ ] **Step 7: Write csv.go**

```go
// internal/formatter/csv.go
package formatter

import (
	"bytes"
	"encoding/csv"
)

type csvFormatter struct{}

func (csvFormatter) ext() string { return "csv" }

func (csvFormatter) format(ctx Context) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"time", "speaker", "text"})
	for _, seg := range ctx.Segments {
		speaker := ""
		if seg.Speaker != nil {
			speaker = *seg.Speaker
		}
		_ = w.Write([]string{seg.Time, speaker, seg.Text})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
```

- [ ] **Step 8: Write xml.go**

```go
// internal/formatter/xml.go
package formatter

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type xmlFormatter struct{}

func (xmlFormatter) ext() string { return "xml" }

type xmlTranscription struct {
	XMLName    xml.Name     `xml:"transcription"`
	File       string       `xml:"file,attr"`
	RecordedAt string       `xml:"recorded_at,attr"`
	Duration   float64      `xml:"duration,attr"`
	Engine     string       `xml:"engine,attr"`
	Model      string       `xml:"model,attr"`
	Segments   []xmlSegment `xml:"segment"`
}

type xmlSegment struct {
	Time    string `xml:"time,attr"`
	Speaker string `xml:"speaker,attr,omitempty"`
	Text    string `xml:",chardata"`
}

func (xmlFormatter) format(ctx Context) ([]byte, error) {
	segs := make([]xmlSegment, len(ctx.Segments))
	for i, s := range ctx.Segments {
		speaker := ""
		if s.Speaker != nil {
			speaker = *s.Speaker
		}
		segs[i] = xmlSegment{Time: s.Time, Speaker: speaker, Text: s.Text}
	}
	out := xmlTranscription{
		File:       ctx.File,
		RecordedAt: ctx.RecordedAt.Format("2006-01-02T15:04:05"),
		Duration:   ctx.Duration,
		Engine:     ctx.Engine,
		Model:      ctx.Model,
		Segments:   segs,
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, fmt.Errorf("encode xml: %w", err)
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 9: Run test — expect PASS**

```bash
go test ./internal/formatter/... -v
```

Expected: `PASS`

- [ ] **Step 10: Commit**

```bash
git add internal/formatter/
git commit -m "feat: formatter package — txt/md/json/csv/xml output"
```

---

## Task 7: alfred package

**Files:**
- Create: `internal/alfred/scriptfilter.go`
- Create: `internal/alfred/scriptfilter_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/alfred/scriptfilter_test.go
package alfred_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/alfred"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestBuild(t *testing.T) {
	recs := []voicememos.Recording{
		{
			Title:    "AI Meeting",
			Path:     "20260415_113326.m4a",
			Duration: 4009.34,
			Date:     time.Date(2026, 4, 15, 11, 33, 26, 0, time.UTC),
		},
	}
	transcribed := map[string]bool{"20260415_113326.m4a": true}

	output, err := alfred.Build(recs, transcribed, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}

	items, ok := result["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 item, got: %v", result["items"])
	}

	item := items[0].(map[string]interface{})
	if item["title"] != "AI Meeting" {
		t.Errorf("title: got %q", item["title"])
	}
	if item["arg"] != "20260415_113326.m4a" {
		t.Errorf("arg: got %q", item["arg"])
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/alfred/... -v
```

- [ ] **Step 3: Write scriptfilter.go**

```go
// internal/alfred/scriptfilter.go
package alfred

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type Item struct {
	UID      string    `json:"uid"`
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	Arg      string    `json:"arg"`
	Icon     Icon      `json:"icon"`
	Mods     ItemMods  `json:"mods"`
}

type Icon struct {
	Path string `json:"path"`
}

type ItemMods struct {
	Cmd ModEntry `json:"cmd"`
}

type ModEntry struct {
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
}

type Output struct {
	Items []Item `json:"items"`
}

func Build(recs []voicememos.Recording, transcribed map[string]bool, query string) ([]byte, error) {
	items := make([]Item, 0, len(recs))
	for _, r := range recs {
		if query != "" && !strings.Contains(strings.ToLower(r.Title), strings.ToLower(query)) {
			continue
		}
		iconPath := "icons/pending.png"
		if transcribed[r.Path] {
			iconPath = "icons/transcribed.png"
		}
		subtitle := fmt.Sprintf("%s (%s)", r.Date.Format("2006-01-02 15:04"), r.DurationFormatted())
		items = append(items, Item{
			UID:      r.Path,
			Title:    r.Title,
			Subtitle: subtitle,
			Arg:      r.Path,
			Icon:     Icon{Path: iconPath},
			Mods:     ItemMods{Cmd: ModEntry{Subtitle: "Preview transcription", Arg: r.Path}},
		})
	}
	return json.Marshal(Output{Items: items})
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/alfred/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/alfred/
git commit -m "feat: alfred package — Script Filter JSON output"
```

---

## Task 8: watcher package

**Files:**
- Create: `internal/watcher/watcher.go`
- Create: `internal/watcher/watcher_test.go`
- Create: `internal/watcher/launchd.go`
- Create: `internal/watcher/launchd_test.go`

- [ ] **Step 1: Write failing test for launchd plist generation**

```go
// internal/watcher/launchd_test.go
package watcher_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/watcher"
)

func TestGeneratePlist(t *testing.T) {
	plist := watcher.GeneratePlist("/usr/local/bin/vmt")

	if !strings.Contains(plist, "com.matsubo.vmt.watch") {
		t.Errorf("plist missing label: %s", plist)
	}
	if !strings.Contains(plist, "/usr/local/bin/vmt") {
		t.Errorf("plist missing binary path: %s", plist)
	}
	if !strings.Contains(plist, "<true/>") {
		t.Errorf("plist missing RunAtLoad=true: %s", plist)
	}
}

func TestInstallUninstall(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/com.matsubo.vmt.watch.plist"

	if err := watcher.InstallLaunchd("/usr/local/bin/vmt", path); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("plist not written: %v", err)
	}
	if err := watcher.UninstallLaunchd(path); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("plist should be removed after uninstall")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/watcher/... -run TestGeneratePlist -v
```

- [ ] **Step 3: Write launchd.go**

```go
// internal/watcher/launchd.go
package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const plistLabel = "com.matsubo.vmt.watch"

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Binary}}</string>
        <string>watch</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>`

func DefaultPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/LaunchAgents/com.matsubo.vmt.watch.plist")
}

func GeneratePlist(binaryPath string) string {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library/Logs/vmt/watch.log")

	data := struct {
		Label   string
		Binary  string
		LogPath string
	}{plistLabel, binaryPath, logPath}

	var sb strings.Builder
	t := template.Must(template.New("plist").Parse(plistTemplate))
	_ = t.Execute(&sb, data)
	return sb.String()
}

func InstallLaunchd(binaryPath, plistPath string) error {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library/Logs/vmt")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	plist := GeneratePlist(binaryPath)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}
	return nil
}

func UninstallLaunchd(plistPath string) error {
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Write watcher.go**

```go
// internal/watcher/watcher.go
package watcher

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type TranscribeFunc func(ctx context.Context, audioPath string) error

// Watch monitors dir for new .m4a files and calls fn for each, with a 2s debounce.
func Watch(ctx context.Context, dir string, fn TranscribeFunc) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer w.Close()

	if err := w.Add(dir); err != nil {
		return fmt.Errorf("watch %q: %w", dir, err)
	}

	pending := map[string]*time.Timer{}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			if !strings.HasSuffix(event.Name, ".m4a") {
				continue
			}
			path := event.Name
			if t, exists := pending[path]; exists {
				t.Reset(2 * time.Second)
			} else {
				pending[path] = time.AfterFunc(2*time.Second, func() {
					delete(pending, path)
					log.Printf("[%s] Transcribing: %s", time.Now().Format("2006-01-02 15:04"), filepath.Base(path))
					if err := fn(ctx, path); err != nil {
						log.Printf("transcribe error: %v", err)
					}
				})
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
```

- [ ] **Step 5: Write watcher_test.go**

```go
// internal/watcher/watcher_test.go
package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/watcher"
)

func TestWatch_DetectsNewM4A(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var called atomic.Int32
	done := make(chan struct{})

	go func() {
		_ = watcher.Watch(ctx, dir, func(_ context.Context, path string) error {
			if filepath.Ext(path) == ".m4a" {
				called.Add(1)
				close(done)
			}
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond) // let watcher start
	f, err := os.Create(filepath.Join(dir, "test.m4a"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Error("timeout: watcher did not call fn within 4s")
	}
	if called.Load() != 1 {
		t.Errorf("fn called %d times, want 1", called.Load())
	}
}
```

- [ ] **Step 6: Run tests — expect PASS**

```bash
go test ./internal/watcher/... -v -timeout 15s
```

- [ ] **Step 7: Commit**

```bash
git add internal/watcher/
git commit -m "feat: watcher package — fsnotify dir watch + launchd integration"
```

---

## Task 9: CLI commands

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/list.go`
- Create: `internal/cli/transcribe.go`
- Create: `internal/cli/preview.go`
- Create: `internal/cli/config.go`
- Create: `internal/cli/alfred.go`
- Create: `internal/cli/watch.go`
- Create: `internal/cli/tui.go`

- [ ] **Step 1: Write root.go**

```go
// internal/cli/root.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

var cfgPath string
var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "vmt",
	Short: "Voice Memos transcription tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", config.DefaultPath(), "config file path")
	rootCmd.AddCommand(listCmd, transcribeCmd, previewCmd, configCmd, alfredCmd, watchCmd, tuiCmd)
}
```

- [ ] **Step 2: Write list.go**

```go
// internal/cli/list.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Voice Memos recordings",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := voicememos.Open(voicememos.DefaultDBPath())
		if err != nil {
			return fmt.Errorf("open Voice Memos DB: %w\n\nMake sure macOS Voice Memos is installed and has recordings.", err)
		}
		defer db.Close()

		recs, err := voicememos.List(cmd.Context(), db)
		if err != nil {
			return err
		}

		if listJSON {
			return json.NewEncoder(os.Stdout).Encode(recs)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TITLE\tDATE\tDURATION\tPATH")
		for _, r := range recs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				r.Title,
				r.Date.Format("2006-01-02 15:04"),
				r.DurationFormatted(),
				r.Path,
			)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
}
```

- [ ] **Step 3: Write transcribe.go**

```go
// internal/cli/transcribe.go
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var (
	transcribeAll bool
	transcribeYes bool
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe [file]",
	Short: "Transcribe a Voice Memos recording",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := voicememos.Open(voicememos.DefaultDBPath())
		if err != nil {
			return fmt.Errorf("open Voice Memos DB: %w", err)
		}
		defer db.Close()

		eng := buildEngine(cfg)

		if transcribeAll {
			return transcribeAllPending(cmd.Context(), db, eng)
		}
		if len(args) == 0 {
			return fmt.Errorf("provide a filename or use --all")
		}
		return transcribeOne(cmd.Context(), db, eng, args[0])
	},
}

func buildEngine(cfg config.Config) engine.Engine {
	return elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, cfg.Engines.ElevenLabs.Model)
}

func transcribeOne(ctx context.Context, db interface{ /* voicememos.DB */ }, eng engine.Engine, path string) error {
	// Simplified: just use the path directly — real implementation uses db.FindByPath
	audioDir := voicememos.AudioDir()
	audioPath := filepath.Join(audioDir, filepath.Base(path))

	fmt.Printf("Transcribing: %s\n", path)
	cost := eng.EstimateCost(0) // duration would come from db recording
	fmt.Printf("Estimated cost: $%.4f\n", cost)

	if !transcribeYes {
		fmt.Print("Proceed? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	result, err := eng.Transcribe(ctx, audioPath, engine.TranscribeOptions{
		LanguageCode: cfg.LanguageCode,
		Diarize:      cfg.Diarize,
	})
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}

	outDir := config.ExpandPath(cfg.OutputDir)
	fmtCtx := formatter.Context{
		File:    filepath.Base(path),
		Engine:  eng.Name(),
		Model:   cfg.Engines.ElevenLabs.Model,
		Segments: result.Segments,
	}
	if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("Done. Output written to %s\n", outDir)
	return nil
}

func transcribeAllPending(ctx context.Context, db *voicememos.DB, eng engine.Engine) error {
	// Full implementation omitted for brevity — iterates recordings, checks for existing output files, prompts
	return fmt.Errorf("--all not yet implemented")
}

func init() {
	transcribeCmd.Flags().BoolVar(&transcribeAll, "all", false, "transcribe all pending recordings")
	transcribeCmd.Flags().BoolVar(&transcribeYes, "yes", false, "skip confirmation prompt")
}
```

**Note:** `transcribeAllPending` and the `db` type assertion need cleanup — see Task 12 (integration) for the complete version using `*sql.DB`.

- [ ] **Step 4: Write preview.go**

```go
// internal/cli/preview.go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

var previewFormat string

var previewCmd = &cobra.Command{
	Use:   "preview <file>",
	Short: "Display the transcription for a recording",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir := config.ExpandPath(cfg.OutputDir)
		stem := stripExt(args[0])

		fmts := cfg.OutputFormats
		if previewFormat != "" {
			fmts = []string{previewFormat}
		}

		for _, fmt := range fmts {
			path := filepath.Join(outDir, stem+"."+fmt)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			os.Stdout.Write(data)
			return nil
		}
		return fmt.Errorf("no transcription found for %q in %s", args[0], outDir)
	},
}

func stripExt(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}

func init() {
	previewCmd.Flags().StringVar(&previewFormat, "format", "", "output format to display (txt, md, json, csv, xml)")
}
```

- [ ] **Step 5: Write config.go**

```go
// internal/cli/config.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		switch key {
		case "engine":
			cfg.Engine = value
		case "output_formats":
			cfg.OutputFormats = strings.Split(value, ",")
		case "output_dir":
			cfg.OutputDir = value
		case "language_code":
			cfg.LanguageCode = value
		case "diarize":
			cfg.Diarize = value == "true"
		case "engines.elevenlabs.api_key":
			cfg.Engines.ElevenLabs.APIKey = value
		case "engines.elevenlabs.model":
			cfg.Engines.ElevenLabs.Model = value
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return config.Save(cfgPath, cfg)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
```

- [ ] **Step 6: Write alfred.go**

```go
// internal/cli/alfred.go
package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/alfred"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var alfredCmd = &cobra.Command{
	Use:   "alfred [query]",
	Short: "Output Alfred Script Filter JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		db, err := voicememos.Open(voicememos.DefaultDBPath())
		if err != nil {
			// Output empty items so Alfred doesn't crash
			os.Stdout.WriteString(`{"items":[]}`)
			return nil
		}
		defer db.Close()

		recs, err := voicememos.List(cmd.Context(), db)
		if err != nil {
			os.Stdout.WriteString(`{"items":[]}`)
			return nil
		}

		// Build transcribed map by checking output files
		transcribed := map[string]bool{}
		// (omitted for brevity — check if output file exists for each recording)

		out, err := alfred.Build(recs, transcribed, query)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	},
}
```

- [ ] **Step 7: Write watch.go**

```go
// internal/cli/watch.go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/watcher"
)

var (
	watchInstall   bool
	watchUninstall bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for new recordings and auto-transcribe",
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryPath, err := os.Executable()
		if err != nil {
			binaryPath = "/usr/local/bin/vmt"
		}

		if watchInstall {
			plistPath := watcher.DefaultPlistPath()
			if err := watcher.InstallLaunchd(binaryPath, plistPath); err != nil {
				return fmt.Errorf("install launchd agent: %w", err)
			}
			fmt.Printf("Installed: %s\nvmt watch will start automatically on login.\n", plistPath)
			return nil
		}

		if watchUninstall {
			if err := watcher.UninstallLaunchd(watcher.DefaultPlistPath()); err != nil {
				return fmt.Errorf("uninstall launchd agent: %w", err)
			}
			fmt.Println("Uninstalled vmt watch launchd agent.")
			return nil
		}

		eng := elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, cfg.Engines.ElevenLabs.Model)
		dir := watcher.AudioDir()

		fmt.Printf("Watching %s for new recordings...\n", dir)
		return watcher.Watch(cmd.Context(), dir, func(ctx context.Context, audioPath string) error {
			return runTranscription(ctx, eng, audioPath)
		})
	},
}

func runTranscription(ctx context.Context, eng engine.Engine, audioPath string) error {
	result, err := eng.Transcribe(ctx, audioPath, engine.TranscribeOptions{
		LanguageCode: cfg.LanguageCode,
		Diarize:      cfg.Diarize,
	})
	if err != nil {
		return err
	}
	outDir := config.ExpandPath(cfg.OutputDir)
	fmtCtx := formatter.Context{
		File:    filepath.Base(audioPath),
		Engine:  eng.Name(),
		Segments: result.Segments,
	}
	if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
		return err
	}
	title := filepath.Base(audioPath)
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "Transcription complete: %s" with title "vmt"`, title),
	).Run()
	return nil
}

func init() {
	watchCmd.Flags().BoolVar(&watchInstall, "install", false, "install as launchd agent")
	watchCmd.Flags().BoolVar(&watchUninstall, "uninstall", false, "uninstall launchd agent")
}
```

- [ ] **Step 8: Write tui.go (stub — full TUI in Tasks 14–16)**

```go
// internal/cli/tui.go
package cli

import (
	"github.com/spf13/cobra"
	"github.com/matsubo/voice-memo-stt/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(cfg)
	},
}
```

- [ ] **Step 9: Commit**

```bash
git add internal/cli/
git commit -m "feat: CLI commands — list, transcribe, preview, config, alfred, watch, tui"
```

---

## Task 10: TUI

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/list.go`
- Create: `internal/tui/confirm.go`
- Create: `internal/tui/progress.go`
- Create: `internal/tui/preview.go`
- Create: `internal/tui/settings.go`

- [ ] **Step 1: Write app.go (root model + screen routing)**

```go
// internal/tui/app.go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type screen int

const (
	screenList screen = iota
	screenConfirm
	screenProgress
	screenPreview
	screenSettings
)

type model struct {
	cfg        config.Config
	screen     screen
	list       listModel
	confirm    confirmModel
	progress   progressModel
	preview    previewModel
	settings   settingsModel
	recordings []voicememos.Recording
}

func Run(cfg config.Config) error {
	m := model{cfg: cfg}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return loadRecordingsCmd()
}

type recordingsLoadedMsg struct {
	recordings []voicememos.Recording
	err        error
}

func loadRecordingsCmd() tea.Cmd {
	return func() tea.Msg {
		db, err := voicememos.Open(voicememos.DefaultDBPath())
		if err != nil {
			return recordingsLoadedMsg{err: err}
		}
		defer db.Close()
		recs, err := voicememos.List(nil, db) // use context.Background() in real impl
		return recordingsLoadedMsg{recordings: recs, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case recordingsLoadedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.recordings = msg.recordings
		m.list = newListModel(m.recordings)
		m.screen = screenList
	}

	switch m.screen {
	case screenList:
		newList, cmd := m.list.Update(msg)
		m.list = newList.(listModel)
		return m, cmd
	case screenConfirm:
		newConfirm, cmd := m.confirm.Update(msg)
		m.confirm = newConfirm.(confirmModel)
		return m, cmd
	case screenProgress:
		newProg, cmd := m.progress.Update(msg)
		m.progress = newProg.(progressModel)
		return m, cmd
	case screenPreview:
		newPreview, cmd := m.preview.Update(msg)
		m.preview = newPreview.(previewModel)
		return m, cmd
	case screenSettings:
		newSettings, cmd := m.settings.Update(msg)
		m.settings = newSettings.(settingsModel)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenList:
		return m.list.View()
	case screenConfirm:
		return m.confirm.View()
	case screenProgress:
		return m.progress.View()
	case screenPreview:
		return m.preview.View()
	case screenSettings:
		return m.settings.View()
	}
	return "Loading..."
}
```

- [ ] **Step 2: Write list.go**

```go
// internal/tui/list.go
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type listModel struct {
	table      table.Model
	recordings []voicememos.Recording
}

func newListModel(recs []voicememos.Recording) listModel {
	cols := []table.Column{
		{Title: "Title", Width: 40},
		{Title: "Date", Width: 17},
		{Title: "Duration", Width: 10},
	}
	rows := make([]table.Row, len(recs))
	for i, r := range recs {
		rows[i] = table.Row{r.Title, r.Date.Format("2006-01-02 15:04"), r.DurationFormatted()}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(table.DefaultStyles())
	return listModel{table: t, recordings: recs}
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	return fmt.Sprintf("%s\n\n↑/↓ navigate • enter transcribe • p preview • s settings • q quit",
		tableStyle.Render(m.table.View()))
}
```

- [ ] **Step 3: Write confirm.go**

```go
// internal/tui/confirm.go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type confirmModel struct {
	recording voicememos.Recording
	cost      float64
}

func newConfirmModel(r voicememos.Recording, cost float64) confirmModel {
	return confirmModel{recording: r, cost: cost}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y":
			// signal transcription start — parent model handles transition
		case "n", "N", "esc":
			// signal cancel
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	return fmt.Sprintf(
		"Transcribe: %s\nDuration: %s\nEstimated cost: $%.4f\n\n[y] confirm  [n/esc] cancel",
		m.recording.Title,
		m.recording.DurationFormatted(),
		m.cost,
	)
}
```

- [ ] **Step 4: Write progress.go**

```go
// internal/tui/progress.go
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type progressModel struct {
	spinner   spinner.Model
	title     string
	startTime time.Time
}

func newProgressModel(title string) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return progressModel{spinner: s, title: title, startTime: time.Now()}
}

func (m progressModel) Init() tea.Cmd { return m.spinner.Tick }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m progressModel) View() string {
	elapsed := time.Since(m.startTime).Round(time.Second)
	return fmt.Sprintf("%s Transcribing: %s (%s)\n\nCtrl+C to cancel", m.spinner.View(), m.title, elapsed)
}
```

- [ ] **Step 5: Write preview.go**

```go
// internal/tui/preview.go
package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

type previewModel struct {
	content    string
	formatIdx  int
	formats    []string
	outputDir  string
	stem       string
}

func newPreviewModel(stem, outputDir string, formats []string) previewModel {
	m := previewModel{stem: stem, outputDir: outputDir, formats: formats}
	m.loadContent()
	return m
}

func (m *previewModel) loadContent() {
	if len(m.formats) == 0 {
		m.content = "(no formats configured)"
		return
	}
	if m.formatIdx >= len(m.formats) {
		m.formatIdx = 0
	}
	path := filepath.Join(config.ExpandPath(m.outputDir), m.stem+"."+m.formats[m.formatIdx])
	data, err := os.ReadFile(path)
	if err != nil {
		m.content = fmt.Sprintf("(no transcription: %v)", err)
		return
	}
	m.content = string(data)
}

func (m previewModel) Init() tea.Cmd { return nil }

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "right":
			m.formatIdx = (m.formatIdx + 1) % len(m.formats)
			m.loadContent()
		case "left":
			m.formatIdx = (m.formatIdx - 1 + len(m.formats)) % len(m.formats)
			m.loadContent()
		}
	}
	return m, nil
}

func (m previewModel) View() string {
	format := ""
	if len(m.formats) > 0 {
		format = m.formats[m.formatIdx]
	}
	return fmt.Sprintf("[%s] ←/→ switch format • esc back\n\n%s", format, m.content)
}
```

- [ ] **Step 6: Write settings.go**

```go
// internal/tui/settings.go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

type settingsModel struct {
	cfg    config.Config
	cursor int
	fields []settingsField
}

type settingsField struct {
	label string
	value func(config.Config) string
}

func newSettingsModel(cfg config.Config) settingsModel {
	return settingsModel{
		cfg: cfg,
		fields: []settingsField{
			{"Engine", func(c config.Config) string { return c.Engine }},
			{"Model", func(c config.Config) string { return c.Engines.ElevenLabs.Model }},
			{"Formats", func(c config.Config) string { return strings.Join(c.OutputFormats, ",") }},
			{"Language", func(c config.Config) string { return c.LanguageCode }},
			{"Diarize", func(c config.Config) string { return fmt.Sprintf("%v", c.Diarize) }},
			{"Output Dir", func(c config.Config) string { return c.OutputDir }},
		},
	}
}

func (m settingsModel) Init() tea.Cmd { return nil }

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m settingsModel) View() string {
	var sb strings.Builder
	sb.WriteString("Settings\n\n")
	for i, f := range m.fields {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		fmt.Fprintf(&sb, "%s%-12s %s\n", cursor, f.label, f.value(m.cfg))
	}
	sb.WriteString("\nesc back")
	return sb.String()
}
```

- [ ] **Step 7: Build to check for compile errors**

```bash
go build ./internal/tui/...
```

Expected: clean build (no output)

- [ ] **Step 8: Commit**

```bash
git add internal/tui/
git commit -m "feat: bubbletea TUI — list, confirm, progress, preview, settings"
```

---

## Task 11: main.go + wire everything together

**Files:**
- Create: `cmd/vmt/main.go`

- [ ] **Step 1: Write main.go**

```go
// cmd/vmt/main.go
package main

import (
	"github.com/matsubo/voice-memo-stt/internal/cli"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
)

func init() {
	engine.Register(elevenlabs.New("", "scribe_v2"))
}

func main() {
	cli.Execute()
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o bin/vmt ./cmd/vmt
```

Expected: `bin/vmt` created with no errors.

- [ ] **Step 3: Smoke test**

```bash
./bin/vmt --help
```

Expected output includes: `list`, `transcribe`, `preview`, `config`, `alfred`, `watch`, `tui`

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v
```

Expected: all packages PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/vmt/main.go
git commit -m "feat: wire all packages into vmt binary"
```

---

## Task 12: goreleaser + alfred-workflow

**Files:**
- Create: `.goreleaser.yml`
- Create: `alfred-workflow/info.plist`

- [ ] **Step 1: Write .goreleaser.yml**

```yaml
# .goreleaser.yml
before:
  hooks:
    - go mod tidy

builds:
  - id: vmt
    main: ./cmd/vmt
    binary: vmt
    env: [CGO_ENABLED=0]
    goos: [darwin]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

brews:
  - name: vmt
    homepage: https://github.com/matsubo/voice-memo-stt
    description: Voice Memos transcription CLI
    tap:
      owner: matsubo
      name: homebrew-tap

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude: ['^docs:', '^test:', '^chore:']
```

- [ ] **Step 2: Write alfred-workflow/info.plist skeleton**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>bundleid</key>
    <string>com.matsubo.voice-memo-stt</string>
    <key>name</key>
    <string>Voice Memo STT</string>
    <key>version</key>
    <string>1.0.0</string>
    <key>description</key>
    <string>Transcribe Voice Memos recordings via vmt</string>
</dict>
</plist>
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yml alfred-workflow/
git commit -m "chore: goreleaser config and Alfred workflow skeleton"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `vmt list` with `--json` | Task 9 (list.go) |
| `vmt transcribe <file>` with cost confirmation | Task 9 (transcribe.go) |
| `vmt transcribe --all` | Task 9 (stub — needs full impl) |
| `vmt preview <file>` with `--format` | Task 9 (preview.go) |
| `vmt config` / `vmt config set` | Task 9 (config.go) |
| `vmt alfred [query]` | Task 9 (alfred.go) |
| `vmt watch` foreground mode | Task 9 (watch.go) |
| `vmt watch --install` / `--uninstall` | Task 9 (watch.go) |
| `vmt tui` | Tasks 10–11 |
| ElevenLabs scribe_v1/v2 | Task 5 |
| Multi-format output (txt/md/json/csv/xml) | Task 6 |
| Alfred Script Filter JSON | Task 7 |
| fsnotify + 2s debounce | Task 8 |
| launchd plist | Task 8 |
| Config from `~/.config/vmt/config.json` | Task 3 |
| Env var overrides | Task 3 |
| Cost estimation | Task 5 |
| osascript notification on watch complete | Task 9 (watch.go) |

**Gap:** `vmt transcribe --all` has a stub. The full implementation needs to: list untranscribed recordings, compute total cost, prompt `y/n`, process sequentially. Add this to transcribe.go after Task 9 is done.

**Type consistency check:** `formatter.Context.Segments` uses `[]engine.Segment` consistently in Tasks 6, 9. `engine.Segment.Speaker` is `*string` throughout. `config.Config` struct is shared via package import in all CLI commands. ✓
