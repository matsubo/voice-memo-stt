package tui

import (
	"os/exec"
	"testing"
)

func TestCopyToClipboard(t *testing.T) {
	// Only run if pbcopy/pbpaste available
	if _, err := exec.LookPath("pbcopy"); err != nil {
		t.Skip("pbcopy not available")
	}
	if _, err := exec.LookPath("pbpaste"); err != nil {
		t.Skip("pbpaste not available")
	}

	content := "test clipboard content 12345"
	if err := copyToClipboard(content); err != nil {
		t.Fatalf("copyToClipboard: %v", err)
	}

	got, err := exec.Command("pbpaste").Output()
	if err != nil {
		t.Fatalf("pbpaste: %v", err)
	}
	if string(got) != content {
		t.Errorf("clipboard: got %q, want %q", got, content)
	}
}
