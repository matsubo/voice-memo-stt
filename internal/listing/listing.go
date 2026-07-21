// Package listing turns recordings plus on-disk transcription state into the
// JSON document `vmt list --json` emits. The field names are a published
// contract: the Raycast extension and any other consumer read them, so they are
// declared here once rather than falling out of Go struct names.
package listing

import (
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

// Item is one recording and everything a consumer needs to act on it without
// re-deriving paths or re-reading the config.
type Item struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	Path              string    `json:"path"`       // ZPATH, the identifier other vmt commands take
	AudioPath         string    `json:"audio_path"` // absolute path to the .m4a
	Duration          float64   `json:"duration"`   // seconds
	DurationFormatted string    `json:"duration_formatted"`
	Date              time.Time `json:"date"`
	Transcribed       bool      `json:"transcribed"`
	Chars             int       `json:"chars"` // runes in the preferred output, 0 when pending
	// Outputs maps each written format to its absolute path, so a consumer can
	// open or reveal a transcription without knowing the naming rule.
	Outputs map[string]string `json:"outputs"`
}

// Build assembles the listing for recs under the given configuration. The
// result is never nil, so it marshals as [] rather than null.
func Build(recs []voicememos.Recording, cfg config.Config) []Item {
	outDir := config.ExpandPath(cfg.OutputDir)
	audioDir := voicememos.AudioDir()

	items := make([]Item, 0, len(recs))
	for _, r := range recs {
		outputs := formatter.ExistingOutputs(outDir, r.Path, cfg.OutputFormats)
		items = append(items, Item{
			ID:                r.ID,
			Title:             r.Title,
			Path:              r.Path,
			AudioPath:         filepath.Join(audioDir, r.Path),
			Duration:          r.Duration,
			DurationFormatted: r.DurationFormatted(),
			Date:              r.Date,
			Transcribed:       len(outputs) > 0,
			Chars:             chars(outputs, cfg.OutputFormats),
			Outputs:           outputs,
		})
	}
	return items
}

// chars counts the runes of the preferred output, which is what a reader thinks
// of as the length of a transcription. Bytes would over-count every language
// that is not ASCII.
func chars(outputs map[string]string, formats []string) int {
	format, ok := formatter.PreferredFormat(formats)
	if !ok {
		return 0
	}
	path, ok := outputs[format]
	if !ok {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return utf8.RuneCount(data)
}
