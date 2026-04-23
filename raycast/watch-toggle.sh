#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title Toggle Watch Agent
# @raycast.mode compact
# @raycast.packageName vmt
# @raycast.icon 👁
# @raycast.description Install/uninstall vmt launchd watch agent
# @raycast.author matsubo
# @raycast.authorURL https://github.com/matsubo

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

PLIST="$HOME/Library/LaunchAgents/com.matsubo.vmt.watch.plist"

if [ -f "$PLIST" ]; then
  vmt watch --uninstall
  echo "Watch agent stopped"
else
  vmt watch --install
  echo "Watch agent started (runs on login)"
fi
