package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/spf13/cobra"
)

var previewFormat string

var previewCmd = &cobra.Command{
	Use:   "preview <file>",
	Short: "Display the transcription for a recording",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir := config.ExpandPath(cfg.OutputDir)
		stem := stripExt(normalizePath(args[0]))

		fmts := cfg.OutputFormats
		if previewFormat != "" {
			fmts = []string{previewFormat}
		}

		for _, f := range fmts {
			path := filepath.Join(outDir, stem+"."+f)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			_, werr := os.Stdout.Write(data)
			return werr
		}
		return fmt.Errorf("no transcription found for %q in %s", args[0], outDir)
	},
}

func stripExt(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}

func init() {
	previewCmd.Flags().StringVar(&previewFormat, "format", "", "output format to display (txt, md, json, csv, xml)")
}
