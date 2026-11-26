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

const defaultAPIBase = "https://api.elevenlabs.io"

// Name is the Engine name used in the `engine` setting.
const Name = "elevenlabs"

// Models are the Models POST /v1/speech-to-text accepts, recommended first, which
// makes the first one the default. Realtime Models (scribe_v3_live,
// scribe_v2_realtime) are left out because vmt uploads whole files. Prices are
// the self-serve batch list prices since 2026-11-17
// (https://elevenlabs.io/docs/changelog/2026/11/17); scribe_v1 was removed on
// 2026-07-09.
var Models = []engine.Model{
	{ID: "scribe_v3", PricePerHour: 0.20},
	{ID: "scribe_v2", PricePerHour: 0.20},
	{ID: "scribe_v2_medical", PricePerHour: 0.20},
}

// Client is an ElevenLabs Speech-to-Text client implementing engine.Engine.
type Client struct {
	apiKey  string
	model   engine.Model
	baseURL string
	http    *http.Client
}

// Option is a functional option for Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// New creates an ElevenLabs client that transcribes with model.
func New(apiKey string, model engine.Model, opts ...Option) *Client {
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
func (c *Client) Name() string { return Name }

// Model returns the Model this client transcribes with.
func (c *Client) Model() engine.Model { return c.model }

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
	_ = w.WriteField("model_id", c.model.ID)
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
