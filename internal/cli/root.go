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
