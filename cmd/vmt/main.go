package main

import (
	"github.com/matsubo/voice-memo-stt/internal/cli"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
)

func main() {
	cfg, _ := config.Load(config.DefaultPath())
	engine.Register(elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, cfg.Engines.ElevenLabs.Model))
	cli.Execute()
}
