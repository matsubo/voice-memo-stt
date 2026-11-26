package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine/catalog"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		display := cfg
		display.Engines.ElevenLabs.APIKey = maskKey(display.Engines.ElevenLabs.APIKey)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(display)
	},
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		switch key {
		case "engine":
			if _, err := catalog.Models(value); err != nil {
				return err
			}
			cfg.Engine = value
		case "output_formats":
			cfg.OutputFormats = strings.Split(value, ",")
		case "output_dir":
			cfg.OutputDir = value
		case "language_code":
			cfg.LanguageCode = value
		case "diarize":
			cfg.Diarize = value == "true"
		case "editor":
			cfg.Editor = value
		case "beep_on_complete":
			cfg.BeepOnComplete = value == "true"
		case "engines.elevenlabs.api_key":
			cfg.Engines.ElevenLabs.APIKey = value
		case "engines.elevenlabs.model":
			if _, err := catalog.FindModel(elevenlabs.Name, value); err != nil {
				return err
			}
			cfg.Engines.ElevenLabs.Model = value
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return config.Save(cfgPath, cfg)
	},
}

// configModelsCmd lists the Models the configured Engine offers, so a user can
// pick one without looking up the provider's documentation.
var configModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List the models the configured engine offers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		models, err := catalog.Models(cfg.Engine)
		if err != nil {
			return err
		}
		selected, _ := catalog.SelectedModel(cfg)
		for i, m := range models {
			var notes []string
			if i == 0 {
				notes = append(notes, "default")
			}
			if m.ID == selected.ID {
				notes = append(notes, "selected")
			}
			fmt.Printf("%-20s $%.2f / hour  %s\n", m.ID, m.PricePerHour, strings.Join(notes, ", "))
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configModelsCmd)
}
