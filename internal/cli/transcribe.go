package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
	"github.com/spf13/cobra"
)

var (
	transcribeAll bool
	transcribeYes bool
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe [file]",
	Short: "Transcribe a Voice Memos recording",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recs, err := voicememos.Load(cmd.Context())
		if err != nil {
			return err
		}

		eng, err := buildEngine(cfg)
		if err != nil {
			return err
		}

		if transcribeAll {
			return transcribeAllPending(cmd.Context(), recs, eng, cfg)
		}
		if len(args) == 0 {
			return fmt.Errorf("provide a filename or use --all")
		}
		return transcribeOne(cmd.Context(), recs, eng, cfg, args[0])
	},
}

func buildEngine(cfg config.Config) (engine.Engine, error) {
	apiKey := cfg.Engines.ElevenLabs.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key not configured.\n\nSet via:\n  export ELEVENLABS_API_KEY=sk-...\nor:\n  vmt config set engines.elevenlabs.api_key sk-...")
	}
	return elevenlabs.New(apiKey, cfg.Engines.ElevenLabs.Model), nil
}

func normalizePath(filename string) string {
	base := filepath.Base(filename)
	if !strings.HasSuffix(base, ".m4a") {
		base += ".m4a"
	}
	return base
}

func transcribeOne(ctx context.Context, recs []voicememos.Recording, eng engine.Engine, cfg config.Config, filename string) error {
	path := normalizePath(filename)
	rec, ok := voicememos.Find(recs, path)
	if !ok {
		return fmt.Errorf("recording %q not found", path)
	}

	audioPath := filepath.Join(voicememos.AudioDir(), rec.Path)
	cost := eng.EstimateCost(rec.Duration)

	fmt.Printf("Recording: %s\n", rec.Title)
	fmt.Printf("Duration:  %s\n", rec.DurationFormatted())
	fmt.Printf("Est. cost: $%.4f\n", cost)

	if !transcribeYes {
		fmt.Print("Proceed? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Println("Transcribing...")
	result, err := eng.Transcribe(ctx, audioPath, engine.TranscribeOptions{
		LanguageCode: cfg.LanguageCode,
		Diarize:      cfg.Diarize,
	})
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}

	outDir := config.ExpandPath(cfg.OutputDir)
	fmtCtx := formatter.Context{
		File:       rec.Path,
		RecordedAt: rec.Date,
		Duration:   rec.Duration,
		Engine:     eng.Name(),
		Model:      cfg.Engines.ElevenLabs.Model,
		Segments:   result.Segments,
	}
	if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("Done. Output written to %s\n", outDir)
	return nil
}

func hasTranscription(outDir, recPath string, formats []string) bool {
	if len(formats) == 0 {
		return false
	}
	stem := strings.TrimSuffix(recPath, filepath.Ext(recPath))
	check := filepath.Join(outDir, stem+"."+formats[0])
	_, err := os.Stat(check)
	return err == nil
}

func transcribeAllPending(ctx context.Context, recs []voicememos.Recording, eng engine.Engine, cfg config.Config) error {
	outDir := config.ExpandPath(cfg.OutputDir)

	var pending []voicememos.Recording
	var totalCost float64
	var totalDur float64
	for _, r := range recs {
		if hasTranscription(outDir, r.Path, cfg.OutputFormats) {
			continue
		}
		pending = append(pending, r)
		totalDur += r.Duration
		totalCost += eng.EstimateCost(r.Duration)
	}

	if len(pending) == 0 {
		fmt.Println("All recordings already transcribed.")
		return nil
	}

	fmt.Printf("Pending: %d recordings, total %.1f minutes\n", len(pending), totalDur/60)
	fmt.Printf("Est. total cost: $%.4f\n", totalCost)

	if !transcribeYes {
		fmt.Print("Proceed? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	for i, r := range pending {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(pending), r.Title, r.DurationFormatted())
		audioPath := filepath.Join(voicememos.AudioDir(), r.Path)
		result, err := eng.Transcribe(ctx, audioPath, engine.TranscribeOptions{
			LanguageCode: cfg.LanguageCode,
			Diarize:      cfg.Diarize,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}
		fmtCtx := formatter.Context{
			File:       r.Path,
			RecordedAt: r.Date,
			Duration:   r.Duration,
			Engine:     eng.Name(),
			Model:      cfg.Engines.ElevenLabs.Model,
			Segments:   result.Segments,
		}
		if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
			fmt.Fprintf(os.Stderr, "  error writing output: %v\n", err)
		}
	}
	return nil
}

func init() {
	transcribeCmd.Flags().BoolVar(&transcribeAll, "all", false, "transcribe all pending recordings")
	transcribeCmd.Flags().BoolVar(&transcribeYes, "yes", false, "skip confirmation prompt")
}
