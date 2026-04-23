package elevenlabs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
)

func fakeAPIResponse() map[string]interface{} {
	return map[string]interface{}{
		"words": []map[string]interface{}{
			{"speaker_id": "speaker_0", "text": "Hello", "start": 15.0},
			{"speaker_id": "speaker_0", "text": "world", "start": 15.5},
			{"speaker_id": "speaker_1", "text": "Hi", "start": 83.0},
		},
	}
}

func startFakeServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fakeAPIResponse())
	}))
}

func writeTempAudio(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.m4a")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestTranscribe_Diarize(t *testing.T) {
	srv := startFakeServer(t)
	defer srv.Close()

	client := elevenlabs.New("test-key", "scribe_v2", elevenlabs.WithBaseURL(srv.URL))
	audioPath := writeTempAudio(t)

	result, err := client.Transcribe(context.Background(), audioPath, engine.TranscribeOptions{
		LanguageCode: "jpn",
		Diarize:      true,
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("want 2 segments (grouped by speaker), got %d", len(result.Segments))
	}
	if result.Segments[0].Text != "Hello world" {
		t.Errorf("segment 0 text: got %q", result.Segments[0].Text)
	}
	if result.Segments[0].Time != "00:15" {
		t.Errorf("segment 0 time: got %q", result.Segments[0].Time)
	}
	if result.Segments[1].Time != "01:23" {
		t.Errorf("segment 1 time: got %q", result.Segments[1].Time)
	}
}

func TestTranscribe_NoDiarize(t *testing.T) {
	srv := startFakeServer(t)
	defer srv.Close()

	client := elevenlabs.New("test-key", "scribe_v2", elevenlabs.WithBaseURL(srv.URL))
	result, err := client.Transcribe(context.Background(), writeTempAudio(t), engine.TranscribeOptions{Diarize: false})
	if err != nil {
		t.Fatal(err)
	}
	// Without diarize, all words merge into one segment
	if len(result.Segments) != 1 {
		t.Fatalf("want 1 segment (no diarize), got %d", len(result.Segments))
	}
	if result.Segments[0].Speaker != nil {
		t.Error("speaker should be nil when diarize=false")
	}
}

func TestName(t *testing.T) {
	c := elevenlabs.New("key", "scribe_v2")
	if c.Name() != "elevenlabs" {
		t.Errorf("Name: got %q", c.Name())
	}
}

func TestEstimateCost(t *testing.T) {
	c := elevenlabs.New("key", "scribe_v2")
	cost := c.EstimateCost(3600) // 1 hour
	if cost != 0.22 {
		t.Errorf("scribe_v2 cost/hour: got %f, want 0.22", cost)
	}

	c2 := elevenlabs.New("key", "scribe_v1")
	cost2 := c2.EstimateCost(3600)
	if cost2 != 0.40 {
		t.Errorf("scribe_v1 cost/hour: got %f, want 0.40", cost2)
	}
}

func TestTranscribe_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := elevenlabs.New("test-key", "scribe_v2", elevenlabs.WithBaseURL(srv.URL))
	_, err := client.Transcribe(context.Background(), writeTempAudio(t), engine.TranscribeOptions{})
	if err == nil {
		t.Error("expected error from API 429")
	}
}
