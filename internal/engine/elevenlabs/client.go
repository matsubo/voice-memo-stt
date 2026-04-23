package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

const (
	defaultAPIBase = "https://api.elevenlabs.io"
	defaultModel   = "scribe_v2"
	costV1PerHour  = 0.40
	costV2PerHour  = 0.22
)

// Client is an ElevenLabs Speech-to-Text client implementing engine.Engine.
type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

// Option is a functional option for Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// New creates a new ElevenLabs client. If model is empty, defaultModel is used.
func New(apiKey, model string, opts ...Option) *Client {
	if model == "" {
		model = defaultModel
	}
	c := &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultAPIBase,
		http:    &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name returns the engine identifier.
func (c *Client) Name() string { return "elevenlabs" }

// EstimateCost returns the estimated USD cost for the given audio duration.
func (c *Client) EstimateCost(durationSeconds float64) float64 {
	hours := durationSeconds / 3600
	if c.model == "scribe_v1" {
		return hours * costV1PerHour
	}
	return hours * costV2PerHour
}

type apiWord struct {
	SpeakerID string  `json:"speaker_id"`
	Text      string  `json:"text"`
	Start     float64 `json:"start"`
}

type apiResponse struct {
	Words []apiWord `json:"words"`
}

// Transcribe uploads the audio file to ElevenLabs and returns transcription segments.
func (c *Client) Transcribe(ctx context.Context, audioPath string, opts engine.TranscribeOptions) (*engine.TranscribeResult, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("model_id", c.model)
	if opts.LanguageCode != "" {
		_ = w.WriteField("language_code", opts.LanguageCode)
	}
	if opts.Diarize {
		_ = w.WriteField("diarize", "true")
	}
	part, err := w.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy audio: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/speech-to-text", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs API %d: %s", resp.StatusCode, b)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return parseWords(apiResp.Words, opts.Diarize), nil
}

// parseWords groups consecutive words into segments. When diarize is true, a new
// segment starts whenever the speaker changes. When diarize is false, all words
// are merged into a single segment with no Speaker set.
func parseWords(words []apiWord, diarize bool) *engine.TranscribeResult {
	var segments []engine.Segment
	var cur *engine.Segment

	for _, w := range words {
		sameSpeaker := cur != nil && (!diarize || speakerOf(cur) == w.SpeakerID)
		if sameSpeaker {
			cur.Text += " " + w.Text
		} else {
			seg := engine.Segment{Time: formatTime(w.Start), Text: w.Text}
			if diarize {
				s := w.SpeakerID
				seg.Speaker = &s
			}
			segments = append(segments, seg)
			// Reassign after append: the slice may have been reallocated.
			cur = &segments[len(segments)-1]
		}
	}
	return &engine.TranscribeResult{Segments: segments}
}

func speakerOf(s *engine.Segment) string {
	if s.Speaker == nil {
		return ""
	}
	return *s.Speaker
}

// formatTime converts a float64 seconds value to "MM:SS" format.
func formatTime(seconds float64) string {
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}
