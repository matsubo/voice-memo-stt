package catalog_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine/catalog"
)

func cfgWith(model, key string) config.Config {
	cfg := config.Defaults()
	cfg.Engines.ElevenLabs.Model = model
	cfg.Engines.ElevenLabs.APIKey = key
	return cfg
}

func TestBuild_UsesSelectedModel(t *testing.T) {
	eng, err := catalog.Build(cfgWith("scribe_v2_medical", "k"))
	if err != nil {
		t.Fatal(err)
	}
	if eng.Name() != "elevenlabs" || eng.Model().ID != "scribe_v2_medical" {
		t.Errorf("got %s/%s, want elevenlabs/scribe_v2_medical", eng.Name(), eng.Model().ID)
	}
}

func TestBuild_EmptyModelIsDefault(t *testing.T) {
	eng, err := catalog.Build(cfgWith("", "k"))
	if err != nil {
		t.Fatal(err)
	}
	if eng.Model().ID != "scribe_v3" {
		t.Errorf("default model: got %q, want scribe_v3", eng.Model().ID)
	}
}

// scribe_v1 was removed by ElevenLabs; a config still naming it must fail with
// the Models that do exist, not when the API rejects the upload.
func TestBuild_UnknownModelListsAvailable(t *testing.T) {
	_, err := catalog.Build(cfgWith("scribe_v1", "k"))
	if err == nil || !strings.Contains(err.Error(), "scribe_v3") {
		t.Fatalf("want an error listing the available models, got %v", err)
	}
}

func TestBuild_UnknownEngine(t *testing.T) {
	cfg := cfgWith("", "k")
	cfg.Engine = "whisper"
	if _, err := catalog.Build(cfg); err == nil || !strings.Contains(err.Error(), "elevenlabs") {
		t.Fatalf("want an error listing the available engines, got %v", err)
	}
}

func TestBuild_MissingKey(t *testing.T) {
	if _, err := catalog.Build(cfgWith("", "")); !errors.Is(err, catalog.ErrMissingAPIKey) {
		t.Fatalf("want ErrMissingAPIKey, got %v", err)
	}
}
