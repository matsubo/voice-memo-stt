package cli

import (
	"fmt"
	"os"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/spf13/cobra"
)

var cfgPath string
var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "vmt",
	Short: "Voice Memos transcription tool",
	// A command that fails at runtime is not a usage mistake, and burying the
	// error under a flag listing is how a Raycast panel ends up showing help
	// text instead of what went wrong.
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", config.DefaultPath(), "config file path")
	rootCmd.AddCommand(listCmd, transcribeCmd, previewCmd, configCmd, alfredCmd, watchCmd, tuiCmd)
}
