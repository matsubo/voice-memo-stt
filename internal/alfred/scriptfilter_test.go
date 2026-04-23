package alfred_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matsubo/voice-memo-stt/internal/alfred"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestBuild(t *testing.T) {
	recs := []voicememos.Recording{
		{
			Title:    "AI Meeting",
			Path:     "20260415_113326.m4a",
			Duration: 4009.34,
			Date:     time.Date(2026, 4, 15, 11, 33, 26, 0, time.UTC),
		},
	}
	transcribed := map[string]bool{"20260415_113326.m4a": true}

	output, err := alfred.Build(recs, transcribed, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}

	items, ok := result["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 item, got: %v", result["items"])
	}

	item := items[0].(map[string]interface{})
	if item["title"] != "AI Meeting" {
		t.Errorf("title: got %q", item["title"])
	}
	if item["arg"] != "20260415_113326.m4a" {
		t.Errorf("arg: got %q", item["arg"])
	}
}

func TestBuild_QueryFilter(t *testing.T) {
	recs := []voicememos.Recording{
		{Title: "AI Meeting", Path: "a.m4a", Date: time.Now()},
		{Title: "Lunch chat", Path: "b.m4a", Date: time.Now()},
	}
	output, _ := alfred.Build(recs, nil, "lunch")
	var result map[string]interface{}
	json.Unmarshal(output, &result)
	items := result["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("want 1 filtered item, got %d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["title"] != "Lunch chat" {
		t.Errorf("filtered title: got %q", item["title"])
	}
}

func TestBuild_IconSwitch(t *testing.T) {
	recs := []voicememos.Recording{
		{Title: "done", Path: "a.m4a", Date: time.Now()},
		{Title: "pending", Path: "b.m4a", Date: time.Now()},
	}
	transcribed := map[string]bool{"a.m4a": true}
	output, _ := alfred.Build(recs, transcribed, "")
	var result map[string]interface{}
	json.Unmarshal(output, &result)
	items := result["items"].([]interface{})
	iconA := items[0].(map[string]interface{})["icon"].(map[string]interface{})["path"]
	iconB := items[1].(map[string]interface{})["icon"].(map[string]interface{})["path"]
	if iconA == iconB {
		t.Errorf("transcribed and pending should have different icons, both = %v", iconA)
	}
}
