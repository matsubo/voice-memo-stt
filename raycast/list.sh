#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title List Voice Memos Recordings
# @raycast.mode fullOutput
# @raycast.packageName vmt
# @raycast.icon 🎙
# @raycast.description Show all Voice Memos recordings with transcription status
# @raycast.author matsubo
# @raycast.authorURL https://github.com/matsubo

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
exec vmt list
