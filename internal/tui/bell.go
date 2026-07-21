package tui

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// bell is the ASCII BEL character. The terminal turns it into whatever alert
// the user configured — an audible beep, or a visual flash if they prefer one.
const bell = "\a"

// bellCmd rings the terminal bell. It writes to stderr rather than stdout
// because stdout carries the rendered frame, and a byte injected there would be
// wiped out by the next repaint. A nil writer means the real terminal.
func bellCmd(w io.Writer) tea.Cmd {
	if w == nil {
		w = os.Stderr
	}
	return func() tea.Msg {
		fmt.Fprint(w, bell)
		return nil
	}
}
