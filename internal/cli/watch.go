package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
	"github.com/matsubo/voice-memo-stt/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	watchInstall   bool
	watchUninstall bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for new recordings and auto-transcribe",
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryPath, err := os.Executable()
		if err != nil {
			binaryPath = "/usr/local/bin/vmt"
		}

		if watchInstall {
			plistPath := watcher.DefaultPlistPath()
			if err := watcher.InstallLaunchd(binaryPath, plistPath); err != nil {
				return fmt.Errorf("install launchd agent: %w", err)
			}
			fmt.Printf("Installed: %s\nvmt watch will start automatically on login.\n", plistPath)
			return nil
		}

		if watchUninstall {
			if err := watcher.UninstallLaunchd(watcher.DefaultPlistPath()); err != nil {
				return fmt.Errorf("uninstall launchd agent: %w", err)
			}
			fmt.Println("Uninstalled vmt watch launchd agent.")
			return nil
		}

		eng, err := buildEngine(cfg)
		if err != nil {
			return err
		}
		dir := voicememos.AudioDir()

		fmt.Printf("Watching %s for new recordings...\n", dir)
		return watcher.Watch(cmd.Context(), dir, func(ctx context.Context, audioPath string) error {
			return runTranscription(ctx, eng, audioPath, cfg)
		})
	},
}

func runTranscription(ctx context.Context, eng engine.Engine, audioPath string, cfg config.Config) error {
	result, err := eng.Transcribe(ctx, audioPath, engine.TranscribeOptions{
		LanguageCode: cfg.LanguageCode,
		Diarize:      cfg.Diarize,
	})
	if err != nil {
		return err
	}
	outDir := config.ExpandPath(cfg.OutputDir)
	fmtCtx := formatter.Context{
		File:     filepath.Base(audioPath),
		Engine:   eng.Name(),
		Model:    cfg.Engines.ElevenLabs.Model,
		Segments: result.Segments,
	}
	if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
		return err
	}
	title := strings.ReplaceAll(filepath.Base(audioPath), `"`, "")
	_ = exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "Transcription complete: %s" with title "vmt"`, title),
	).Run()
	return nil
}

func init() {
	watchCmd.Flags().BoolVar(&watchInstall, "install", false, "install as launchd agent")
	watchCmd.Flags().BoolVar(&watchUninstall, "uninstall", false, "uninstall launchd agent")
}
