package voicememos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// filenamePattern matches Voice Memos filenames like "20191126 124120-A5868508.m4a"
// and "20260415_113326.m4a".
var filenamePattern = regexp.MustCompile(`^(\d{8})[ _](\d{6})`)

// parseFilename extracts a title and timestamp from a Voice Memos audio filename.
// Returns (title, date, true) if the filename matches the expected pattern.
func parseFilename(name string) (string, time.Time, bool) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	m := filenamePattern.FindStringSubmatch(stem)
	if m == nil {
		return stem, time.Time{}, false
	}
	t, err := time.ParseInLocation("20060102 150405", m[1]+" "+m[2], time.Local)
	if err != nil {
		return stem, time.Time{}, false
	}
	// Drop the trailing "-HASH" if present so the title is the date portion only.
	title := stem
	if idx := strings.LastIndex(stem, "-"); idx > 0 {
		title = stem[:idx]
	}
	return title, t, true
}

// ScanAudioDir lists .m4a files in dir and returns Recording entries derived
// from filenames only (no duration data). Used as fallback when the Voice
// Memos SQLite database is missing.
func ScanAudioDir(ctx context.Context, dir string) ([]Recording, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read audio dir %q: %w", dir, err)
	}
	var recs []Recording
	for _, e := range entries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".m4a") {
			continue
		}
		title, date, ok := parseFilename(e.Name())
		if !ok {
			if info, err := e.Info(); err == nil {
				date = info.ModTime()
			}
		}
		recs = append(recs, Recording{
			Title:    title,
			Path:     e.Name(),
			Date:     date,
			Duration: 0, // unknown without DB
		})
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Date.After(recs[j].Date)
	})
	return recs, nil
}
