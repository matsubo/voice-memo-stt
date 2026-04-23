//go:build !test

package watcher

import (
	"fmt"
	"os/exec"
)

// InstallLaunchd writes the plist and runs launchctl load.
func InstallLaunchd(binaryPath, plistPath string) error {
	if err := WritePlist(binaryPath, plistPath); err != nil {
		return err
	}
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}
	return nil
}

// UninstallLaunchd runs launchctl unload and removes the plist.
func UninstallLaunchd(plistPath string) error {
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	return RemovePlist(plistPath)
}
