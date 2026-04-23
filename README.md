# vmt — Voice Memos Transcription CLI

[![CI](https://github.com/matsubo/voice-memo-stt/actions/workflows/ci.yml/badge.svg)](https://github.com/matsubo/voice-memo-stt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/matsubo/voice-memo-stt.svg)](https://pkg.go.dev/github.com/matsubo/voice-memo-stt)
[![Go Report Card](https://goreportcard.com/badge/github.com/matsubo/voice-memo-stt)](https://goreportcard.com/report/github.com/matsubo/voice-memo-stt)
[![Go Version](https://img.shields.io/github/go-mod/go-version/matsubo/voice-memo-stt)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Release](https://img.shields.io/github/v/release/matsubo/voice-memo-stt?include_prereleases&sort=semver)](https://github.com/matsubo/voice-memo-stt/releases)
[![Platform: macOS](https://img.shields.io/badge/platform-macOS-lightgrey)](https://www.apple.com/macos/)

Transcribe macOS Voice Memos recordings via ElevenLabs Scribe. Single Go binary with CLI, TUI, Alfred/Raycast integration, and file-watch auto-transcription.

## Preview

### TUI (`vmt tui`)

```
┌───┬──────────────────────────────────────────┬───────────────────┬──────────┐
│   │ Title                                    │ Date              │ Duration │
├───┼──────────────────────────────────────────┼───────────────────┼──────────┤
│   │ AI Moderator Meeting                     │ 2026-04-15 11:33  │ 1h06m    │
│   │ Lunch notes                              │ 2026-04-14 12:00  │ 15m23s   │
│ ✓ │ Product review Q2                        │ 2026-04-10 15:30  │ 45m12s   │
│   │ Quick idea                               │ 2026-04-08 09:05  │ 0m45s    │
│ ✓ │ Team standup                             │ 2026-04-05 10:00  │ 30m34s   │
└───┴──────────────────────────────────────────┴───────────────────┴──────────┘

✓ = transcribed • ↑/↓ navigate • enter transcribe • p preview • s settings • q quit
```

### Preview screen (`p` on a row)

```
[txt] ←/→ switch format • c copy • esc back

[00:15] speaker_0: Good morning everyone, let's get started.
[00:23] speaker_1: Thanks. I'll share the updated numbers.
[01:04] speaker_0: Sounds good. What about the Q2 outlook?
[01:23] speaker_1: Up 18% year over year — details in the deck.
...
```

### Transcribe confirmation (`enter` on a row)

```
Transcribe: AI Moderator Meeting
Duration: 1h06m
Estimated cost: $0.2447

[y] confirm  [n/esc] cancel
```

### CLI (`vmt list`)

```
$ vmt list
TITLE                 DATE              DURATION  PATH
AI Moderator Meeting  2026-04-15 11:33  1h06m     20260415_113326.m4a
Lunch notes           2026-04-14 12:00  15m23s    20260414_120000.m4a
Product review Q2     2026-04-10 15:30  45m12s    20260410_153000.m4a
Quick idea            2026-04-08 09:05  0m45s     20260408_090515.m4a
Team standup          2026-04-05 10:00  30m34s    20260405_100000.m4a
```

## Features

- **Read macOS Voice Memos directly** from its SQLite database (read-only, no data mutation)
- **Pluggable STT engines** — ElevenLabs Scribe v1/v2 at launch, easy to add more
- **Multi-format output** — txt, md, json, csv, xml generated from a single API call
- **Speaker diarization** (via ElevenLabs `diarize`)
- **Interactive TUI** (bubbletea) — list, transcribe, preview, settings, clipboard copy
- **Alfred Script Filter** — see [alfred-workflow/](alfred-workflow/)
- **Raycast Script Commands** — see [raycast/](raycast/)
- **File watcher** — auto-transcribe new recordings (foreground or launchd agent)
- **Cost estimation** before transcription

## Install

### Homebrew (planned)

```bash
brew install matsubo/tap/vmt
```

### From source

```bash
git clone https://github.com/matsubo/voice-memo-stt.git
cd voice-memo-stt
make build
# Binary at ./bin/vmt — move to your PATH:
sudo cp bin/vmt /usr/local/bin/
```

Requires Go 1.25+.

## Setup

Get an API key from [ElevenLabs](https://elevenlabs.io/app/settings/api-keys), then configure one of:

### Environment variable (recommended)

```bash
export ELEVENLABS_API_KEY=sk-xxxxx
```

Put it in `~/.zshrc` / `~/.config/fish/config.fish` to persist.

### Config file

```bash
vmt config set engines.elevenlabs.api_key sk-xxxxx
```

Written to `~/.config/vmt/config.json` with `0600` permissions. The key is masked in `vmt config` output.

Precedence: env var > config file > default.

## Usage

```bash
vmt list                       # list recordings
vmt list --json                # machine-readable
vmt transcribe <file>          # transcribe one (with cost confirmation)
vmt transcribe --all --yes     # batch, skip prompts
vmt preview <file>             # print transcription (uses first enabled format)
vmt preview <file> --format md # specific format
vmt config                     # show config (API key masked)
vmt config set <key> <value>   # update config
vmt alfred [query]             # Alfred Script Filter JSON
vmt tui                        # interactive TUI
vmt watch                      # foreground watch + auto-transcribe
vmt watch --install            # register launchd agent (auto-start on login)
vmt watch --uninstall          # remove launchd agent
```

### TUI keys

| Screen   | Keys                                                           |
|----------|----------------------------------------------------------------|
| list     | `↑/↓` navigate • `enter` transcribe • `p` preview • `s` settings • `q` quit |
| confirm  | `y` confirm • `n`/`esc` cancel                                 |
| preview  | `←/→` switch format • `c` copy to clipboard (pbcopy) • `esc` back |
| settings | `↑/↓` navigate • `esc` back                                    |

In the list, a `✓` mark in the leftmost column means that recording has at least one output file in your configured `output_dir`.

## Config

`~/.config/vmt/config.json`:

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

| Key | Env var | Default |
|---|---|---|
| `engines.elevenlabs.api_key` | `ELEVENLABS_API_KEY` | (none, required) |
| `engine` | `VMT_ENGINE` | `elevenlabs` |
| `output_dir` | `VMT_OUTPUT_DIR` | `~/Downloads/voice-memo-transcription` |
| `language_code` | `VMT_LANGUAGE` | `jpn` |

ElevenLabs models:

| Model | Price |
|---|---|
| `scribe_v1` | $0.40 / hour |
| `scribe_v2` (default) | $0.22 / hour |

## Output formats

All enabled formats are generated from one API call, written as `{stem}.{ext}` under `output_dir`.

- **txt** — `[00:15] speaker_0: Hello`
- **md** — `- **00:15** speaker_0: Hello`
- **json** — structured: file, recorded_at, duration, engine, model, segments
- **csv** — `time,speaker,text`
- **xml** — `<transcription ...><segment .../></transcription>`

## Watch mode

`vmt watch` monitors the Voice Memos directory with `fsnotify`. When a new `.m4a` appears, it waits 2 s (debounce — Voice Memos writes incrementally), transcribes, writes output, and fires a macOS notification.

`vmt watch --install` generates `~/Library/LaunchAgents/com.matsubo.vmt.watch.plist` and loads it with `launchctl`. Logs go to `~/Library/Logs/vmt/watch.log`.

## Alfred workflow

The `alfred-workflow/` directory has a skeleton `info.plist`. Suggested wiring in Alfred:

- Keyword trigger: `vm`
- Script Filter: `/usr/local/bin/vmt alfred {query}`
- Run Script on selection: `/usr/local/bin/vmt transcribe --yes {query}`
- Cmd modifier: `/usr/local/bin/vmt preview {query}`

Transcribed recordings show a `✓` icon; pending ones show a dash.

## Raycast

The [`raycast/`](raycast/) directory contains 5 Raycast Script Commands:

- **Transcribe All Pending** — batch transcribe untranscribed recordings
- **Copy Latest Transcription** — `.txt` → clipboard
- **List Voice Memos Recordings** — show recordings with `✓` marks
- **Open Voice Memos TUI** — launch `vmt tui` in Terminal.app
- **Toggle Watch Agent** — install/uninstall launchd watch agent

Setup: Raycast → Settings → Extensions → Script Commands → **Add Directory** → point to `raycast/`. See [raycast/README.md](raycast/README.md) for details.

## Data source

`vmt` reads the macOS Voice Memos SQLite DB at:

```
~/Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings/CloudRecordings.db
```

Audio files (`.m4a`) live in the same directory. If the file is missing, `vmt` returns `no Voice Memos recordings found`.

**The DB is opened in read-only mode.** `vmt` never writes to Voice Memos data.

## Architecture

```
cmd/vmt/main.go             — entrypoint, engine registration
internal/
  voicememos/               — SQLite reader
  engine/                   — Engine interface + registry
    elevenlabs/             — ElevenLabs Scribe client
  formatter/                — txt/md/json/csv/xml output
  config/                   — JSON config + env var overrides
  alfred/                   — Alfred Script Filter JSON
  watcher/                  — fsnotify + debounce + launchd
  cli/                      — cobra commands
  tui/                      — bubbletea screens
```

Adding a new STT engine:

1. Create `internal/engine/<name>/client.go` implementing the `Engine` interface.
2. `engine.Register(newengine.New(...))` in `cmd/vmt/main.go`.
3. Add `engines.<name>.*` to config.

## Development

```bash
make build           # build to bin/vmt
make test            # go test ./...
make lint            # golangci-lint run ./...
```

Minimum test coverage per package: 80 %. Golden-file tests for formatters, in-memory SQLite for voicememos, httptest for the ElevenLabs client.

## License

MIT
