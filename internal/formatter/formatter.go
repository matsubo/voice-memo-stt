package formatter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

type Context struct {
	File       string
	RecordedAt time.Time
	Duration   float64
	Engine     string
	Model      string
	Segments   []engine.Segment
}

type fmter interface {
	ext() string
	format(ctx Context) ([]byte, error)
}

var formatters = map[string]fmter{
	"txt":  txtFormatter{},
	"md":   mdFormatter{},
	"json": jsonFormatter{},
	"csv":  csvFormatter{},
	"xml":  xmlFormatter{},
}

func Write(dir string, ctx Context, formats []string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	stem := strings.TrimSuffix(ctx.File, filepath.Ext(ctx.File))
	for _, name := range formats {
		f, ok := formatters[name]
		if !ok {
			return fmt.Errorf("unknown format %q", name)
		}
		data, err := f.format(ctx)
		if err != nil {
			return fmt.Errorf("format %q: %w", name, err)
		}
		outPath := filepath.Join(dir, stem+"."+f.ext())
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("write %q: %w", outPath, err)
		}
	}
	return nil
}

func speakerPrefix(seg engine.Segment) string {
	if seg.Speaker != nil {
		return *seg.Speaker + ": "
	}
	return ""
}
