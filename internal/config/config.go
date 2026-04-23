package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ElevenLabsConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type EnginesConfig struct {
	ElevenLabs ElevenLabsConfig `json:"elevenlabs"`
}

type Config struct {
	Engine        string        `json:"engine"`
	OutputFormats []string      `json:"output_formats"`
	OutputDir     string        `json:"output_dir"`
	LanguageCode  string        `json:"language_code"`
	Diarize       bool          `json:"diarize"`
	Engines       EnginesConfig `json:"engines"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vmt", "config.json")
}

func Defaults() Config {
	return Config{
		Engine:        "elevenlabs",
		OutputFormats: []string{"txt", "json"},
		OutputDir:     "~/Downloads/voice-memo-transcription",
		LanguageCode:  "jpn",
		Diarize:       true,
		Engines: EnginesConfig{
			ElevenLabs: ElevenLabsConfig{
				Model: "scribe_v2",
			},
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return applyEnvOverrides(cfg), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return applyEnvOverrides(cfg), nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func applyEnvOverrides(cfg Config) Config {
	if v := os.Getenv("ELEVENLABS_API_KEY"); v != "" {
		cfg.Engines.ElevenLabs.APIKey = v
	}
	if v := os.Getenv("VMT_ENGINE"); v != "" {
		cfg.Engine = v
	}
	if v := os.Getenv("VMT_OUTPUT_DIR"); v != "" {
		cfg.OutputDir = v
	}
	if v := os.Getenv("VMT_LANGUAGE"); v != "" {
		cfg.LanguageCode = v
	}
	return cfg
}
