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

// ErrAccessDenied is returned when the database exists but cannot be read.
// On macOS this is almost always TCC: the Voice Memos container is protected,
// and only apps granted Full Disk Access may read it. The grant is per calling
// app, so vmt works from a Terminal that has it and fails from a launcher
// (Raycast, Alfred, a launchd agent) that does not.
var ErrAccessDenied = fmt.Errorf("macOS denied access to the Voice Memos database — grant Full Disk Access to the app running vmt (Terminal, iTerm, Raycast, Alfred, …) in System Settings → Privacy & Security → Full Disk Access, then try again")

// Open opens the Voice Memos SQLite database at path in read-only mode.
// Returns ErrNoRecordings if the database file does not exist, and
// ErrAccessDenied if it exists but cannot be read.
func Open(path string) (*sql.DB, error) {
	if err := checkReadable(path); err != nil {
		return nil, err
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

// checkReadable classifies the file before sqlite gets a chance to: a denied
// read comes back from the driver as "unable to open database file: out of
// memory (14)", which sends users looking for a memory problem they do not have.
//
// Opening the file is what settles it. TCC lets a protected path be stat'ed and
// blocks the read, so a successful Stat proves nothing on its own.
func checkReadable(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNoRecordings
		}
		if os.IsPermission(err) {
			return fmt.Errorf("%w (%q)", ErrAccessDenied, path)
		}
		return fmt.Errorf("open Voice Memos DB at %q: %w", path, err)
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w (%q)", ErrAccessDenied, path)
		}
		return fmt.Errorf("open Voice Memos DB at %q: %w", path, err)
	}
	return f.Close()
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
