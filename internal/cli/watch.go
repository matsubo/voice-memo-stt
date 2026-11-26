package cli

import (
	"context"
	"fmt"
	"log"
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
		return watcher.Watch(cmd.Context(), dir, watchHandler(eng, cfg, notifyComplete))
	},
}

// watchHandler returns what Watch runs for each settled audio file. Watch only
// ever transcribes a Pending recording: a write to a Transcribed one (a trim in
// Voice Memos, an iCloud re-sync) would otherwise bill again and overwrite its
// outputs, edits included, without anyone asking for it.
func watchHandler(eng engine.Engine, cfg config.Config, notify func(recPath string)) func(context.Context, string) error {
	return func(ctx context.Context, audioPath string) error {
		recPath := filepath.Base(audioPath)
		if formatter.Transcribed(config.ExpandPath(cfg.OutputDir), recPath, cfg.OutputFormats) {
			log.Printf("Skipping %s: already transcribed", recPath)
			return nil
		}
		if err := runTranscription(ctx, eng, audioPath, cfg); err != nil {
			return err
		}
		notify(recPath)
		return nil
	}
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
	return formatter.Write(outDir, fmtCtx, cfg.OutputFormats)
}

// notifyComplete posts a macOS notification for a finished transcription.
func notifyComplete(recPath string) {
	title := strings.ReplaceAll(recPath, `"`, "")
	_ = exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "Transcription complete: %s" with title "vmt"`, title),
	).Run()
}

func init() {
	watchCmd.Flags().BoolVar(&watchInstall, "install", false, "install as launchd agent")
	watchCmd.Flags().BoolVar(&watchUninstall, "uninstall", false, "uninstall launchd agent")
}
