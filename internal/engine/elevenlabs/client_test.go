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

	client := elevenlabs.New("test-key", elevenlabs.Models[0], elevenlabs.WithBaseURL(srv.URL))
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

	client := elevenlabs.New("test-key", elevenlabs.Models[0], elevenlabs.WithBaseURL(srv.URL))
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
	c := elevenlabs.New("key", elevenlabs.Models[0])
	if c.Name() != "elevenlabs" {
		t.Errorf("Name: got %q", c.Name())
	}
}

// The Model the user picked must be the one the API is asked to use.
func TestTranscribe_SendsSelectedModel(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModel = r.FormValue("model_id")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fakeAPIResponse())
	}))
	defer srv.Close()

	medical := engine.Model{ID: "scribe_v2_medical", PricePerHour: 0.20}
	client := elevenlabs.New("test-key", medical, elevenlabs.WithBaseURL(srv.URL))
	if _, err := client.Transcribe(context.Background(), writeTempAudio(t), engine.TranscribeOptions{}); err != nil {
		t.Fatal(err)
	}
	if gotModel != "scribe_v2_medical" {
		t.Errorf("model_id sent: got %q, want scribe_v2_medical", gotModel)
	}
}

func TestTranscribe_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := elevenlabs.New("test-key", elevenlabs.Models[0], elevenlabs.WithBaseURL(srv.URL))
	_, err := client.Transcribe(context.Background(), writeTempAudio(t), engine.TranscribeOptions{})
	if err == nil {
		t.Error("expected error from API 429")
	}
}
