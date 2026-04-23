package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/config"
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
			cfg.Engine = value
		case "output_formats":
			cfg.OutputFormats = strings.Split(value, ",")
		case "output_dir":
			cfg.OutputDir = value
		case "language_code":
			cfg.LanguageCode = value
		case "diarize":
			cfg.Diarize = value == "true"
		case "engines.elevenlabs.api_key":
			cfg.Engines.ElevenLabs.APIKey = value
		case "engines.elevenlabs.model":
			cfg.Engines.ElevenLabs.Model = value
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return config.Save(cfgPath, cfg)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
