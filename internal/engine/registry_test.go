package engine_test

import (
	"context"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/engine"
)

type stubEngine struct{ name string }

func (s stubEngine) Name() string { return s.name }
func (s stubEngine) Transcribe(_ context.Context, _ string, _ engine.TranscribeOptions) (*engine.TranscribeResult, error) {
	return &engine.TranscribeResult{}, nil
}
func (s stubEngine) EstimateCost(_ float64) float64 { return 0 }

func TestRegistryRoundtrip(t *testing.T) {
	engine.Register(stubEngine{"testengine"})

	e, err := engine.Get("testengine")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if e.Name() != "testengine" {
		t.Errorf("Name: got %q", e.Name())
	}
}

func TestRegistryUnknown(t *testing.T) {
	_, err := engine.Get("doesnotexist")
	if err == nil {
		t.Error("expected error for unknown engine")
	}
}
