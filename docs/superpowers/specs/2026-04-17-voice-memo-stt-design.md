# voice-memo-stt Design Spec

## Overview

A Go CLI tool (`vmt`) that transcribes macOS Voice Memos recordings using pluggable STT engines. Supports CLI, TUI, and Alfred Workflow interfaces. Initial release ships with ElevenLabs Scribe; the STT engine interface allows adding more providers later.

## Repository

`github.com/matsubo/voice-memo-stt` — MIT License (Yuki Matsukura)

## Architecture

```
vmt (single binary)
  ├── CLI commands (cobra)
  ├── TUI mode (bubbletea)
  └── Alfred Script Filter output

Input: macOS Voice Memos (fixed)
       └── SQLite DB → recording list
       └── .m4a files → STT engine

STT Engine Interface:
       ├── ElevenLabs Scribe (v1/v2) ← implemented at launch
       └── (future: Whisper, Google, etc.)

Output: multiple formats simultaneously
       ├── txt / md / json / csv / xml
       └── configurable output directory
```

## Data Source

macOS Voice Memos stores recordings in:

- DB: `~/Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings/CloudRecordings.db`
- Audio: same directory, `.m4a` files
- Table: `ZCLOUDRECORDING`
  - `ZENCRYPTEDTITLE` — user-set title (plain text despite the column name)
  - `ZPATH` — filename
  - `ZDURATION` — seconds (float)
  - `ZDATE` — Core Data timestamp (Unix epoch + 978307200)

## STT Engine Interface

```go
// engine.go

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
    // Name returns the engine identifier (e.g. "elevenlabs").
    Name() string

    // Transcribe sends audio to the STT service and returns segments.
    // ctx allows cancellation.
    Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscribeResult, error)

    // EstimateCost returns the estimated USD cost for the given audio duration.
    EstimateCost(durationSeconds float64) float64
}
```

### ElevenLabs Implementation

- Models: `scribe_v1` ($0.40/hour), `scribe_v2` ($0.22/hour)
- Default: `scribe_v2`
- API: `POST /v1/speech-to-text` with multipart file upload
- Diarization: supported via `diarize` parameter
- Response: `words` array with `speaker_id`, `text`, `start` fields → grouped into `Segment`s by consecutive speaker

### Adding a New Engine (future)

1. Create `internal/engine/<name>/client.go` implementing `Engine`
2. Register in engine factory
3. Add engine-specific config under `engines.<name>` in config file

No other files need to change.

## Commands

```
vmt list                          # list recordings (table or --json)
vmt transcribe <file>             # transcribe one file
vmt transcribe --all [--yes]      # batch transcribe all pending
vmt preview <file>                # display transcription result
vmt config                        # show current config
vmt config set <key> <value>      # update config value
vmt tui                           # launch TUI mode
vmt alfred                        # output Alfred Script Filter JSON
```

### `vmt list`

- Default: table format (title, date, duration, transcription status)
- `--json`: machine-readable JSON array
- Sorted by date descending

### `vmt transcribe <file>`

- Accepts filename (stem or full `.m4a` path)
- Shows confirmation with recording info + estimated cost
- `--yes` to skip confirmation
- Outputs to all enabled formats simultaneously
- Exit code 0 on success, 1 on error
- Supports `context.Context` cancellation (Ctrl+C)

### `vmt transcribe --all`

- Lists all recordings without existing transcriptions
- Shows total estimated cost
- Prompts `y/n` to proceed (skip with `--yes`)
- Processes sequentially, shows progress per file
- Ctrl+C cancels current file gracefully, asks whether to continue

### `vmt preview <file>`

- Displays the transcription content in terminal
- If multiple formats exist, uses the first enabled format from config
- `--format <fmt>` to override

### `vmt config`

- `vmt config`: prints current config as formatted JSON
- `vmt config set engine elevenlabs`: change engine
- `vmt config set output_formats txt,json,xml`: set formats
- `vmt config set output_dir ~/Documents/transcriptions`: change output dir
- Validates values before saving

### `vmt alfred`

- Outputs Alfred Script Filter JSON
- Items: recordings with title, subtitle (date + duration), icon (checkmark if transcribed)
- `arg`: recording filename (passed to `vmt transcribe --yes <file>`)
- Mod keys: Cmd → preview

### `vmt tui`

- Full-screen TUI (bubbletea)
- Recording list with format tags and search/filter
- Confirmation dialog with cost estimate (engine-aware)
- Progress spinner + elapsed time + Ctrl+C cancellation
- Preview with format switching (left/right arrows)
- Settings screen (engine / model / formats / language / diarize / output dir)

## Config

Path: `~/.config/vmt/config.json`

```json
{
  "engine": "elevenlabs",
  "output_formats": ["txt", "json"],
  "output_dir": "~/Downloads/voice-memo-transcription",
  "language_code": "jpn",
  "diarize": true,
  "engines": {
    "elevenlabs": {
      "api_key": "sk-...",
      "model": "scribe_v2"
    }
  }
}
```

Precedence: environment variable > config file > default.

| Config key | Env var | Default |
|---|---|---|
| `engines.elevenlabs.api_key` | `VMT_ELEVENLABS_API_KEY` | (none, required) |
| `engine` | `VMT_ENGINE` | `elevenlabs` |
| `output_dir` | `VMT_OUTPUT_DIR` | `~/Downloads/voice-memo-transcription` |
| `language_code` | `VMT_LANGUAGE` | `jpn` |

## Output Formats

All enabled formats are generated from a single API call.

### txt

```
[00:15] speaker_0: Hello
[01:23] speaker_1: Hi there
```

### md

```markdown
# Recording Title

- **00:15** speaker_0: Hello
- **01:23** speaker_1: Hi there
```

### json

```json
{
  "file": "20260415 113326.m4a",
  "recorded_at": "2026-04-15T11:33:26",
  "duration": 4009.34,
  "engine": "elevenlabs",
  "model": "scribe_v2",
  "segments": [
    {"time": "00:15", "speaker": "speaker_0", "text": "Hello"}
  ]
}
```

### csv

```csv
time,speaker,text
00:15,speaker_0,Hello
01:23,speaker_1,Hi there
```

### xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<transcription file="20260415 113326.m4a" recorded_at="2026-04-15T11:33:26" duration="4009.34" engine="elevenlabs" model="scribe_v2">
  <segment time="00:15" speaker="speaker_0">Hello</segment>
  <segment time="01:23" speaker="speaker_1">Hi there</segment>
</transcription>
```

## Alfred Workflow

### Installation

1. `brew install matsubo/tap/vmt`
2. Download and import `voice-memo-stt.alfredworkflow`

### Workflow Structure

- Keyword trigger: `vm`
- Script Filter: `/usr/local/bin/vmt alfred {query}`
- Run Script (on selection): `/usr/local/bin/vmt transcribe --yes "{query}" && osascript -e 'display notification "Transcription complete" with title "vmt"'`
- Cmd modifier: preview via `/usr/local/bin/vmt preview "{query}"`

### Script Filter JSON Example

```json
{
  "items": [
    {
      "uid": "20260415 113326",
      "title": "AI Moderator Meeting",
      "subtitle": "2026-04-15 11:33 (1h06m) [txt,json]",
      "arg": "20260415 113326.m4a",
      "icon": { "path": "icons/transcribed.png" },
      "mods": {
        "cmd": {
          "subtitle": "Preview transcription",
          "arg": "20260415 113326.m4a"
        }
      }
    }
  ]
}
```

## File Structure

```
cmd/vmt/main.go                — entrypoint
internal/
  voicememos/
    db.go                       — SQLite DB reader (recording list)
    db_test.go
    recording.go                — Recording struct and helpers
  engine/
    engine.go                   — Engine interface definition
    registry.go                 — Engine registry/factory
    elevenlabs/
      client.go                 — ElevenLabs Scribe implementation
      client_test.go
  formatter/
    formatter.go                — format dispatcher (multi-format output)
    txt.go
    md.go
    json.go
    csv.go
    xml.go
    formatter_test.go
  config/
    config.go                   — config read/write with env var override
    config_test.go
  tui/
    app.go                      — bubbletea root model
    list.go                     — recording list view
    confirm.go                  — confirmation dialog
    progress.go                 — transcription progress
    preview.go                  — transcription preview
    settings.go                 — settings screen
  alfred/
    scriptfilter.go             — Alfred Script Filter JSON builder
    scriptfilter_test.go
  cli/
    root.go                     — cobra root command
    list.go
    transcribe.go
    preview.go
    config.go
    tui.go
    alfred.go
alfred-workflow/
  info.plist                    — Alfred Workflow definition
  icons/
    transcribed.png
    pending.png
go.mod
go.sum
Makefile                        — build, test, lint, release targets
.goreleaser.yml                 — cross-compile + Homebrew tap
.github/workflows/ci.yml        — lint + test + build
README.md
LICENSE
SECURITY.md
```

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (spinner, table, etc.) |
| `modernc.org/sqlite` | SQLite (pure Go, no CGo) |
| `encoding/xml` (stdlib) | XML output |
| `encoding/json` (stdlib) | JSON output/config |
| `encoding/csv` (stdlib) | CSV output |

## Testing

| Package | Strategy |
|---|---|
| `voicememos` | In-memory SQLite with test fixtures |
| `engine/elevenlabs` | `httptest.Server` mocking ElevenLabs API |
| `formatter` | Golden file tests for all 5 formats |
| `config` | Roundtrip read/write, env var override |
| `alfred` | Snapshot test for Script Filter JSON |
| `cli` | Integration tests with test fixtures |

## Error Handling

- API errors: retry once with backoff, then fail with clear message
- Missing API key: exit with setup instructions
- DB not found: exit with message explaining Voice Memos location
- Cancellation (Ctrl+C): graceful shutdown, partial results not saved
- Network timeout: 30s default, configurable

## Cost Estimation

Per-engine cost table (stored in engine implementation, not config):

| Engine | Model | Cost |
|---|---|---|
| ElevenLabs | scribe_v1 | $0.40/hour |
| ElevenLabs | scribe_v2 | $0.22/hour |

Displayed in confirmation dialogs and `--all` batch summary.
