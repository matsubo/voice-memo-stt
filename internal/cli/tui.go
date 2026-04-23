package cli

import (
	"github.com/matsubo/voice-memo-stt/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(cfg)
	},
}
