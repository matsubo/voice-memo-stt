#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title Transcribe All Pending
# @raycast.mode fullOutput
# @raycast.packageName vmt
# @raycast.icon 🎙
# @raycast.description Transcribe all untranscribed Voice Memos recordings
# @raycast.author matsubo
# @raycast.authorURL https://github.com/matsubo

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
exec vmt transcribe --all --yes
