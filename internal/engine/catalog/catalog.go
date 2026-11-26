// Package catalog knows which Engines vmt can use and which Models each one
// offers, and builds the Engine the configuration selects. Every caller (CLI,
// TUI, Watch) goes through Build, so the `engine` setting and the Model check
// apply everywhere.
package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
)

// ErrMissingAPIKey reports that the selected Engine has no API key configured.
var ErrMissingAPIKey = errors.New("ElevenLabs API key not configured.\n\nSet via:\n  export ELEVENLABS_API_KEY=sk-...\nor:\n  vmt config set engines.elevenlabs.api_key sk-...")

type definition struct {
	models     []engine.Model // the first is the default
	configured func(config.Config) string
	build      func(config.Config, engine.Model) (engine.Engine, error)
}

var definitions = map[string]definition{
	elevenlabs.Name: {
		models:     elevenlabs.Models,
		configured: func(cfg config.Config) string { return cfg.Engines.ElevenLabs.Model },
		build: func(cfg config.Config, m engine.Model) (engine.Engine, error) {
			if cfg.Engines.ElevenLabs.APIKey == "" {
				return nil, ErrMissingAPIKey
			}
			return elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, m), nil
		},
	},
}

// Engines lists the Engine names, sorted.
func Engines() []string {
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func lookup(engineName string) (definition, error) {
	def, ok := definitions[engineName]
	if !ok {
		return definition{}, fmt.Errorf("unknown engine %q; available: %s", engineName, strings.Join(Engines(), ", "))
	}
	return def, nil
}

// Models lists the Models engineName offers, default first.
func Models(engineName string) ([]engine.Model, error) {
	def, err := lookup(engineName)
	if err != nil {
		return nil, err
	}
	return def.models, nil
}

// FindModel returns engineName's Model with the given id. An empty id means the
// Engine's default Model.
func FindModel(engineName, id string) (engine.Model, error) {
	def, err := lookup(engineName)
	if err != nil {
		return engine.Model{}, err
	}
	if id == "" {
		return def.models[0], nil
	}
	ids := make([]string, len(def.models))
	for i, m := range def.models {
		if m.ID == id {
			return m, nil
		}
		ids[i] = m.ID
	}
	return engine.Model{}, fmt.Errorf("unknown model %q for engine %q; available: %s", id, engineName, strings.Join(ids, ", "))
}

// SelectedModel is the Model cfg selects for its Engine.
func SelectedModel(cfg config.Config) (engine.Model, error) {
	def, err := lookup(cfg.Engine)
	if err != nil {
		return engine.Model{}, err
	}
	return FindModel(cfg.Engine, def.configured(cfg))
}

// Build returns the Engine cfg selects, set to the selected Model.
func Build(cfg config.Config) (engine.Engine, error) {
	m, err := SelectedModel(cfg)
	if err != nil {
		return nil, err
	}
	return definitions[cfg.Engine].build(cfg, m)
}
