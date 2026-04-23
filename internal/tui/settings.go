package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
)

type settingsModel struct {
	cfg    config.Config
	cursor int
	fields []settingsField
}

type settingsField struct {
	label string
	value func(config.Config) string
}

func newSettingsModel(cfg config.Config) settingsModel {
	return settingsModel{
		cfg: cfg,
		fields: []settingsField{
			{"Engine", func(c config.Config) string { return c.Engine }},
			{"Model", func(c config.Config) string { return c.Engines.ElevenLabs.Model }},
			{"Formats", func(c config.Config) string { return strings.Join(c.OutputFormats, ",") }},
			{"Language", func(c config.Config) string { return c.LanguageCode }},
			{"Diarize", func(c config.Config) string { return fmt.Sprintf("%v", c.Diarize) }},
			{"Output Dir", func(c config.Config) string { return c.OutputDir }},
		},
	}
}

func (m settingsModel) Init() tea.Cmd { return nil }

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m settingsModel) View() string {
	var sb strings.Builder
	sb.WriteString("Settings\n\n")
	for i, f := range m.fields {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		fmt.Fprintf(&sb, "%s%-12s %s\n", cursor, f.label, f.value(m.cfg))
	}
	sb.WriteString("\nesc back")
	return sb.String()
}
