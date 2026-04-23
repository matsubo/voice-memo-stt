package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const plistLabel = "com.matsubo.vmt.watch"

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Binary}}</string>
        <string>watch</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>`

// DefaultPlistPath returns the standard macOS LaunchAgents path for the plist.
func DefaultPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/LaunchAgents/com.matsubo.vmt.watch.plist")
}

// GeneratePlist renders the launchd plist XML for the given binary path.
func GeneratePlist(binaryPath string) string {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library/Logs/vmt/watch.log")

	data := struct {
		Label, Binary, LogPath string
	}{plistLabel, binaryPath, logPath}

	var sb strings.Builder
	t := template.Must(template.New("plist").Parse(plistTemplate))
	_ = t.Execute(&sb, data)
	return sb.String()
}

// WritePlist writes the plist file only (no launchctl calls). Used internally and in tests.
func WritePlist(binaryPath, plistPath string) error {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library/Logs/vmt")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	plist := GeneratePlist(binaryPath)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	return nil
}

// RemovePlist removes the plist file (no launchctl calls).
func RemovePlist(plistPath string) error {
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

