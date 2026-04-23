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

type Engine interface {
	Name() string
	Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscribeResult, error)
	EstimateCost(durationSeconds float64) float64
}
