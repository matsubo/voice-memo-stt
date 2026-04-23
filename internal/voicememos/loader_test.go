package voicememos_test

import (
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestFind(t *testing.T) {
	recs := []voicememos.Recording{
		{Path: "a.m4a", Title: "A"},
		{Path: "b.m4a", Title: "B"},
	}
	r, ok := voicememos.Find(recs, "b.m4a")
	if !ok {
		t.Fatal("Find: not found")
	}
	if r.Title != "B" {
		t.Errorf("Title: got %q", r.Title)
	}

	_, ok = voicememos.Find(recs, "missing.m4a")
	if ok {
		t.Error("Find: should return false for missing path")
	}
}
