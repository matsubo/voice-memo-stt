package voicememos

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// Load returns recordings using the Voice Memos SQLite DB when available,
// or falls back to scanning the audio directory for .m4a files when the
// DB is missing (common on fresh installs or when the DB is being rebuilt
// by the Voice Memos app).
func Load(ctx context.Context) ([]Recording, error) {
	db, err := Open(DefaultDBPath())
	if err == nil {
		defer db.Close()
		return List(ctx, db)
	}
	if !errors.Is(err, ErrNoRecordings) {
		return nil, err
	}
	// DB missing — try scanning the audio directory.
	if _, statErr := os.Stat(AudioDir()); statErr != nil {
		return nil, ErrNoRecordings
	}
	recs, scanErr := ScanAudioDir(ctx, AudioDir())
	if scanErr != nil {
		return nil, fmt.Errorf("%w (fallback scan failed: %v)", ErrNoRecordings, scanErr)
	}
	if len(recs) == 0 {
		return nil, ErrNoRecordings
	}
	return recs, nil
}

// Find returns the recording with the given Path from recs.
func Find(recs []Recording, path string) (*Recording, bool) {
	for i := range recs {
		if recs[i].Path == path {
			return &recs[i], true
		}
	}
	return nil, false
}
