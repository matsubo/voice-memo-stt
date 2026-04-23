package voicememos

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		in        string
		wantTitle string
		wantY     int
		wantM     time.Month
		wantD     int
		wantOK    bool
	}{
		{"20191126 124120-A5868508.m4a", "20191126 124120", 2019, time.November, 26, true},
		{"20260415_113326.m4a", "20260415_113326", 2026, time.April, 15, true},
		{"random_filename.m4a", "random_filename", 0, 0, 0, false},
		{"20191126 124120.m4a", "20191126 124120", 2019, time.November, 26, true},
	}
	for _, tt := range tests {
		title, date, ok := parseFilename(tt.in)
		if ok != tt.wantOK {
			t.Errorf("%q: ok = %v, want %v", tt.in, ok, tt.wantOK)
			continue
		}
		if title != tt.wantTitle {
			t.Errorf("%q: title = %q, want %q", tt.in, title, tt.wantTitle)
		}
		if ok {
			if date.Year() != tt.wantY || date.Month() != tt.wantM || date.Day() != tt.wantD {
				t.Errorf("%q: date = %v, want %d-%v-%d", tt.in, date, tt.wantY, tt.wantM, tt.wantD)
			}
		}
	}
}

func TestScanAudioDir(t *testing.T) {
	dir := t.TempDir()
	// Create 3 m4a files + 1 non-m4a
	for _, name := range []string{
		"20250715 102648-17E72737.m4a",
		"20250712 001342-CC237CA4.m4a",
		"20191126 124120-A5868508.m4a",
		"notes.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	recs, err := ScanAudioDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanAudioDir: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("want 3 recordings, got %d", len(recs))
	}
	// Sorted by date descending
	if recs[0].Path != "20250715 102648-17E72737.m4a" {
		t.Errorf("first sorted: got %q", recs[0].Path)
	}
	if recs[2].Path != "20191126 124120-A5868508.m4a" {
		t.Errorf("last sorted: got %q", recs[2].Path)
	}
	for _, r := range recs {
		if r.Duration != 0 {
			t.Errorf("Duration should be 0 from filesystem scan, got %f for %s", r.Duration, r.Path)
		}
	}
}

func TestScanAudioDir_MissingDir(t *testing.T) {
	_, err := ScanAudioDir(context.Background(), "/nonexistent/dir")
	if err == nil {
		t.Error("expected error for missing dir")
	}
}
