package voicememos_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE ZCLOUDRECORDING (
			Z_PK INTEGER PRIMARY KEY,
			ZENCRYPTEDTITLE TEXT,
			ZPATH TEXT NOT NULL,
			ZDURATION REAL NOT NULL,
			ZDATE REAL NOT NULL,
			ZEVICTIONDATE REAL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	// ZDATE=0 → Unix 0 + 978307200 = 2001-01-01 00:00:00 UTC
	// Row 3 is marked as evicted (deleted) — must NOT appear in List output.
	_, err = db.Exec(`
		INSERT INTO ZCLOUDRECORDING (Z_PK, ZENCRYPTEDTITLE, ZPATH, ZDURATION, ZDATE, ZEVICTIONDATE)
		VALUES (1, 'Meeting Notes', '20260415_113326.m4a', 3600.0, 0.0,  NULL),
		       (2, NULL,            '20260416_090000.m4a', 120.5,  10.0, NULL),
		       (3, 'Deleted memo',  '20260401_080000.m4a', 45.0,   20.0, 100.0)
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	recs, err := voicememos.List(context.Background(), db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 recordings, got %d", len(recs))
	}
	// ordered by ZDATE DESC — row with ZDATE=10 first
	if recs[0].Path != "20260416_090000.m4a" {
		t.Errorf("first path: got %q", recs[0].Path)
	}
	if recs[0].Title != "" {
		t.Errorf("nil title should be empty string, got %q", recs[0].Title)
	}
	wantDate := time.Unix(978307200, 0) // ZDATE=0 → +978307200
	if !recs[1].Date.Equal(wantDate) {
		t.Errorf("Date: got %v, want %v", recs[1].Date, wantDate)
	}
}

func TestFindByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r, err := voicememos.FindByPath(context.Background(), db, "20260415_113326.m4a")
	if err != nil {
		t.Fatalf("FindByPath: %v", err)
	}
	if r.Title != "Meeting Notes" {
		t.Errorf("Title: got %q", r.Title)
	}
	if r.Duration != 3600.0 {
		t.Errorf("Duration: got %f", r.Duration)
	}
}

func TestFindByPath_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := voicememos.FindByPath(context.Background(), db, "nonexistent.m4a")
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestList_ExcludesEvicted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	recs, err := voicememos.List(context.Background(), db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 non-evicted recordings, got %d", len(recs))
	}
	for _, r := range recs {
		if r.Path == "20260401_080000.m4a" {
			t.Errorf("evicted recording %q should not appear in List", r.Path)
		}
	}
}

func TestFindByPath_ExcludesEvicted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := voicememos.FindByPath(context.Background(), db, "20260401_080000.m4a")
	if err == nil {
		t.Error("FindByPath should not return evicted recording")
	}
}

func TestDurationFormatted(t *testing.T) {
	tests := []struct {
		duration float64
		want     string
	}{
		{0, "—"},
		{-1, "—"},
		{45, "0m45s"},
		{65, "1m05s"},
		{3600, "1h00m"},
		{3665, "1h01m"},
		{7200, "2h00m"},
	}
	for _, tt := range tests {
		r := voicememos.Recording{Duration: tt.duration}
		if got := r.DurationFormatted(); got != tt.want {
			t.Errorf("Duration=%v: got %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestOpen_ReadOnlyEnforced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Create schema first with a writable connection
	setupDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setupDB.Exec("CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	setupDB.Close()

	// Now open via our Open() and verify writes fail
	db, err := voicememos.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO t VALUES (1)")
	if err == nil {
		t.Error("expected write to fail on read-only DB, but it succeeded")
	}
}

func TestOpen_MissingFile(t *testing.T) {
	_, err := voicememos.Open("/nonexistent/path/does_not_exist.db")
	if err == nil {
		t.Fatal("expected error for missing DB file")
	}
	if !errors.Is(err, voicememos.ErrNoRecordings) {
		t.Errorf("expected ErrNoRecordings, got %v", err)
	}
}

func TestDefaultDBPath(t *testing.T) {
	p := voicememos.DefaultDBPath()
	if !strings.HasSuffix(p, "CloudRecordings.db") {
		t.Errorf("DefaultDBPath should end with CloudRecordings.db, got %q", p)
	}
}

func TestAudioDir(t *testing.T) {
	d := voicememos.AudioDir()
	if !strings.HasSuffix(d, "Recordings") {
		t.Errorf("AudioDir should end with Recordings, got %q", d)
	}
}
