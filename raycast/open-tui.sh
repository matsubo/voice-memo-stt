#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title Open Voice Memos TUI
# @raycast.mode silent
# @raycast.packageName vmt
# @raycast.icon 🖥
# @raycast.description Launch vmt interactive TUI in Terminal.app
# @raycast.author matsubo
# @raycast.authorURL https://github.com/matsubo

osascript <<'EOF'
tell application "Terminal"
    activate
    do script "vmt tui"
end tell
EOF
