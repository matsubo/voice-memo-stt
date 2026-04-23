#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title Copy Latest Transcription
# @raycast.mode compact
# @raycast.packageName vmt
# @raycast.icon 📋
# @raycast.description Copy the most recent transcription to clipboard
# @raycast.author matsubo
# @raycast.authorURL https://github.com/matsubo

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
set -e

OUTPUT_DIR=$(vmt config | sed -n 's/.*"output_dir": "\([^"]*\)".*/\1/p')
OUTPUT_DIR="${OUTPUT_DIR/#\~/$HOME}"

if [ -z "$OUTPUT_DIR" ] || [ ! -d "$OUTPUT_DIR" ]; then
  echo "Output dir not found: $OUTPUT_DIR"
  exit 1
fi

LATEST=$(ls -t "$OUTPUT_DIR"/*.txt 2>/dev/null | head -1)
if [ -z "$LATEST" ]; then
  echo "No transcriptions in $OUTPUT_DIR"
  exit 1
fi

pbcopy < "$LATEST"
echo "Copied to clipboard: $(basename "$LATEST")"
