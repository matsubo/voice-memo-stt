package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/alfred"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
	"github.com/spf13/cobra"
)

var alfredCmd = &cobra.Command{
	Use:   "alfred [query]",
	Short: "Output Alfred Script Filter JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		recs, err := voicememos.Load(cmd.Context())
		if err != nil {
			os.Stdout.WriteString(`{"items":[]}`)
			return nil
		}

		outDir := config.ExpandPath(cfg.OutputDir)
		transcribed := map[string]bool{}
		for _, r := range recs {
			stem := strings.TrimSuffix(r.Path, filepath.Ext(r.Path))
			if len(cfg.OutputFormats) > 0 {
				check := filepath.Join(outDir, stem+"."+cfg.OutputFormats[0])
				if _, err := os.Stat(check); err == nil {
					transcribed[r.Path] = true
				}
			}
		}

		out, err := alfred.Build(recs, transcribed, query)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	},
}
