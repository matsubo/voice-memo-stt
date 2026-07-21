# Voice Memos Transcription — Raycast Extension

Browse macOS Voice Memos recordings in Raycast, transcribe them with
[`vmt`](../README.md), and read, copy, or open the results — all from the list
with **⌘K**.

```
Search Voice Memos
┌──────────────────────────────────────────────────────────────────────┐
│ ✓  AI Moderator Meeting     2,730 chars   1h06m    3 months ago      │
│ ✓  Lunch notes                412 chars   15m23s   3 months ago      │
│ ○  Product review Q2                      45m12s   3 months ago      │
│ ○  Quick idea                             0m45s    3 months ago      │
└──────────────────────────────────────────────────────────────────────┘
                                                  ⌘K  Actions
```

`✓` is transcribed, `○` is not. The dropdown on the right filters to one or the
other.

## Actions (⌘K)

| Action | Shortcut | Available on |
|---|---|---|
| View Transcription | `↵` | transcribed |
| Transcribe | `↵` | not transcribed |
| Copy Transcription | `⌘C` | transcribed |
| Open in Default App | `⌘O` | transcribed |
| Open With… | | transcribed |
| Show Transcription in Finder | `⌘⇧F` | transcribed |
| Show Recording in Finder | | any |
| Transcribe Again | `⌘R` | transcribed |
| Copy Recording Name | `⌘⇧C` | any |

Transcription runs in the background with a progress toast, and the list
refreshes itself when it finishes.

## Requirements

1. **`vmt` installed** — `brew install matsubo/tap/vmt`, with an ElevenLabs API
   key set (`vmt config set engines.elevenlabs.api_key sk-...`).
2. **Raycast has Full Disk Access.** The Voice Memos database is in a
   TCC-protected container, and the permission is granted per calling app, so
   Raycast needs its own grant:

   ```
   System Settings → Privacy & Security → Full Disk Access → add Raycast
   ```

   Relaunch Raycast afterwards. Without it the extension shows
   *"Raycast cannot read Voice Memos"* with the same instructions.

If `vmt` lives somewhere other than `/opt/homebrew/bin` or `/usr/local/bin`, set
its path in the extension preferences (`⌘,`) — Raycast runs commands without a
login shell, so `PATH` is not inherited.

## Install

Not published to the Raycast Store. Import it locally:

```bash
cd raycast-extension
npm install
npm run dev        # imports into Raycast and watches for changes
```

The extension stays in Raycast after you stop `npm run dev`. Run it again to
pick up code changes.

## Development

```bash
npm run test       # bun test — pure logic (output selection, error mapping)
npm run build      # ray build — type-checks and bundles
npm run lint       # ray lint
```

`ray lint` reports `Invalid author "matsubo"` because it validates the handle
against raycast.com. That only matters for publishing to the Store; everything
else passes.

## How it talks to vmt

The extension shells out and parses `vmt list --json`, whose shape is a contract
declared in [`internal/listing`](../internal/listing/listing.go). Each item
carries the transcription state and absolute paths to every written format, so
the extension never rebuilds vmt's output-path rule itself. `src/types.ts`
mirrors that document.
