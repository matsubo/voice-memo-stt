package watcher_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/watcher"
)

func TestDefaultPlistPath(t *testing.T) {
	path := watcher.DefaultPlistPath()
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("expected path to contain LaunchAgents, got: %s", path)
	}
	if !strings.HasSuffix(path, "com.matsubo.vmt.watch.plist") {
		t.Errorf("expected path to end with plist filename, got: %s", path)
	}
}

func TestGeneratePlist(t *testing.T) {
	plist := watcher.GeneratePlist("/usr/local/bin/vmt")

	if !strings.Contains(plist, "com.matsubo.vmt.watch") {
		t.Errorf("plist missing label: %s", plist)
	}
	if !strings.Contains(plist, "/usr/local/bin/vmt") {
		t.Errorf("plist missing binary path: %s", plist)
	}
	if !strings.Contains(plist, "<true/>") {
		t.Errorf("plist missing RunAtLoad=true: %s", plist)
	}
}

func TestInstallUninstall(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/com.matsubo.vmt.watch.plist"

	if err := watcher.WritePlist("/usr/local/bin/vmt", path); err != nil {
		t.Fatalf("WritePlist: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("plist not written: %v", err)
	}
	if err := watcher.RemovePlist(path); err != nil {
		t.Fatalf("RemovePlist: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("plist should be removed")
	}
}

func TestRemovePlist_NonExistent(t *testing.T) {
	// RemovePlist on a non-existent path should succeed (idempotent).
	if err := watcher.RemovePlist("/tmp/does-not-exist-vmt-test.plist"); err != nil {
		t.Errorf("RemovePlist on non-existent path should not error: %v", err)
	}
}

func TestWritePlist_ContentValid(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.plist"

	if err := watcher.WritePlist("/usr/local/bin/vmt", path); err != nil {
		t.Fatalf("WritePlist: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plist: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "com.matsubo.vmt.watch") {
		t.Errorf("written plist missing label")
	}
	if !strings.Contains(content, "watch") {
		t.Errorf("written plist missing watch subcommand")
	}
}
