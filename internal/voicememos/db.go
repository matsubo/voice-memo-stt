package voicememos

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const coreDateEpochOffset = int64(978307200)

// DefaultDBPath returns the macOS Voice Memos SQLite database path for the current user.
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings/CloudRecordings.db")
}

// AudioDir returns the directory containing Voice Memos audio files.
func AudioDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Group Containers/group.com.apple.VoiceMemos.shared/Recordings")
}

// ErrNoRecordings is returned when the Voice Memos database file is missing,
// typically meaning the user has not recorded any memos yet.
var ErrNoRecordings = fmt.Errorf("no Voice Memos recordings found — record at least one memo in the Voice Memos app, then try again")

// Open opens the Voice Memos SQLite database at path in read-only mode.
// Returns ErrNoRecordings if the database file does not exist.
func Open(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrNoRecordings
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open Voice Memos DB at %q: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open Voice Memos DB at %q: %w", path, err)
	}
	return db, nil
}

// List returns all recordings ordered by date descending (most recent first).
func List(ctx context.Context, db *sql.DB) ([]Recording, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT Z_PK, COALESCE(ZENCRYPTEDTITLE, ''), ZPATH, ZDURATION, ZDATE
		FROM ZCLOUDRECORDING
		WHERE ZEVICTIONDATE IS NULL
		ORDER BY ZDATE DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query recordings: %w", err)
	}
	defer rows.Close()

	var recs []Recording
	for rows.Next() {
		var r Recording
		var zdate float64
		if err := rows.Scan(&r.ID, &r.Title, &r.Path, &r.Duration, &zdate); err != nil {
			return nil, fmt.Errorf("scan recording: %w", err)
		}
		r.Date = time.Unix(int64(zdate)+coreDateEpochOffset, 0)
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recordings: %w", err)
	}
	return recs, nil
}

// FindByPath returns the recording with the given ZPATH, or an error if not found.
func FindByPath(ctx context.Context, db *sql.DB, path string) (*Recording, error) {
	row := db.QueryRowContext(ctx, `
		SELECT Z_PK, COALESCE(ZENCRYPTEDTITLE, ''), ZPATH, ZDURATION, ZDATE
		FROM ZCLOUDRECORDING WHERE ZPATH = ? AND ZEVICTIONDATE IS NULL
	`, path)
	var r Recording
	var zdate float64
	if err := row.Scan(&r.ID, &r.Title, &r.Path, &r.Duration, &zdate); err != nil {
		return nil, fmt.Errorf("find recording %q: %w", path, err)
	}
	r.Date = time.Unix(int64(zdate)+coreDateEpochOffset, 0)
	return &r, nil
}
