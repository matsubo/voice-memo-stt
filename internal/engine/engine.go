package engine

import "context"

type Segment struct {
	Time    string  // "MM:SS"
	Speaker *string // nil if diarization is off
	Text    string
}

type TranscribeResult struct {
	Segments []Segment
}

type TranscribeOptions struct {
	LanguageCode string
	Diarize      bool
}

// Model is one variant of an Engine. Its price is fixed per Model, so the Cost
// Estimate of a Transcription Job depends only on the Model and the duration.
type Model struct {
	ID           string  // the identifier the Engine's API takes
	PricePerHour float64 // USD per hour of audio, batch list price
}

// EstimateCost is the Cost Estimate for durationSeconds of audio.
func (m Model) EstimateCost(durationSeconds float64) float64 {
	return durationSeconds / 3600 * m.PricePerHour
}

// Engine is a speech-to-text service, configured with the Model it uses.
type Engine interface {
	Name() string
	Model() Model
	Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscribeResult, error)
}
