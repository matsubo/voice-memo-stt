# Raycast Script Commands

Integrate `vmt` with [Raycast](https://raycast.com/) via script commands.

## Setup

1. Install `vmt`: `make build && sudo cp bin/vmt /usr/local/bin/`
2. Set your API key (put in `~/.zshrc` or shell rc):
   ```bash
   export ELEVENLABS_API_KEY=sk-xxxxx
   ```
3. Add this directory to Raycast:
   - Raycast → Settings → Extensions → Script Commands
   - **Add Directory** → select the `raycast/` folder of this repo
4. Commands appear under package `vmt`. Assign aliases/hotkeys as you like.

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
