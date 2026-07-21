# Raycast Script Commands

Integrate `vmt` with [Raycast](https://raycast.com/) via script commands.

## Setup

1. Install `vmt`: `brew install matsubo/tap/vmt` (or `make build && sudo cp bin/vmt /usr/local/bin/`)
2. Set your API key (put in `~/.zshrc` or shell rc):
   ```bash
   export ELEVENLABS_API_KEY=sk-xxxxx
   ```
3. **Grant Raycast Full Disk Access** — see below. Without it every command fails.
4. Add this directory to Raycast:
   - Raycast → Settings → Extensions → Script Commands
   - **Add Directory** → select this folder. A Homebrew install puts it at
     `$(brew --prefix vmt)/share/vmt/raycast`; otherwise use the `raycast/`
     folder of this repo.
5. Commands appear under package `vmt`. Assign aliases/hotkeys as you like.

## Full Disk Access

The Voice Memos database lives in a TCC-protected container, so macOS only lets
apps with **Full Disk Access** read it. The grant is per calling app: `vmt` works
in a Terminal that has it and fails under Raycast, which does not have it by
default.

```
System Settings → Privacy & Security → Full Disk Access → add Raycast
```

Quit and reopen Raycast afterwards — the permission is picked up at launch.

Without it, commands fail with:

```
Error: macOS denied access to the Voice Memos database — grant Full Disk Access …
```

## Commands

| Title | Icon | Description |
|---|---|---|
| **Transcribe All Pending** | 🎙 | Batch transcribe untranscribed recordings |
| **Copy Latest Transcription** | 📋 | Copy most recent `.txt` output to clipboard |
| **List Voice Memos Recordings** | 🎙 | Show all recordings with date + duration |
| **Open Voice Memos TUI** | 🖥 | Launch `vmt tui` in Terminal.app |
| **Toggle Watch Agent** | 👁 | Install/uninstall launchd agent for auto-transcribe |

## Customization

Each script is a plain bash file. Edit the `@raycast.*` metadata or the command body to fit your workflow.

**PATH note:** scripts prepend `/opt/homebrew/bin` and `/usr/local/bin` so `vmt` resolves under Raycast's sandboxed env regardless of your login shell config.
