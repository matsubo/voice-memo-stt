package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
)

// modelOptions are the ElevenLabs models the user can cycle between.
var modelOptions = []string{"scribe_v1", "scribe_v2"}

// languageOptions are the languages offered in the settings cycle. The empty
// code means auto-detect. ElevenLabs accepts ISO-639-1 or -3 codes and supports
// far more than these; any other code can be set via `vmt config set`.
var languageOptions = []struct{ code, name string }{
	{"", "auto"},
	{"ja", "Japanese"},
	{"en", "English"},
	{"zh", "Chinese"},
	{"ko", "Korean"},
	{"es", "Spanish"},
	{"fr", "French"},
	{"de", "German"},
	{"pt", "Portuguese"},
	{"ru", "Russian"},
	{"it", "Italian"},
}

type rowKind int

const (
	rowDisplay rowKind = iota // shown but not editable (Engine, Output Dir)
	rowChoice                 // cycle through options with ←/→ (Model)
	rowLang                   // cycle through languages with ←/→
	rowBool                   // toggle with ←/→ or space (Diarize)
	rowText                   // edit via a text input (API Key)
	rowFormat                 // a single output-format checkbox
)

type settingsRow struct {
	label   string
	kind    rowKind
	options []string // rowChoice
	format  string   // rowFormat
	masked  bool     // rowText: hide the stored value
}

type settingsModel struct {
	cfg     config.Config
	rows    []settingsRow
	cursor  int
	editing bool
	input   textinput.Model
	saveErr string // set by the app when persisting fails
}

func newSettingsModel(cfg config.Config) settingsModel {
	rows := []settingsRow{
		{label: "Engine", kind: rowDisplay},
		{label: "Model", kind: rowChoice, options: modelOptions},
		{label: "Language", kind: rowLang},
		{label: "Diarize", kind: rowBool},
		{label: "API Key", kind: rowText, masked: true},
		{label: "Output Dir", kind: rowDisplay},
	}
	for _, f := range formatter.SupportedFormats {
		rows = append(rows, settingsRow{label: f, kind: rowFormat, format: f})
	}
	return settingsModel{cfg: cfg, rows: rows, input: textinput.New()}
}

func (m settingsModel) Init() tea.Cmd { return nil }

// changed announces the edited config so the app can persist it.
func (m settingsModel) changed() tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg { return configChangedMsg{cfg: cfg} }
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.editing {
		return m.updateEditing(key)
	}

	switch key.String() {
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return m, nil
	case "esc":
		return m, func() tea.Msg { return backMsg{} }
	}

	return m.editRow(m.rows[m.cursor], key)
}

// editRow applies a key to the focused row according to its kind.
func (m settingsModel) editRow(row settingsRow, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch row.kind {
	case rowChoice:
		switch key.String() {
		case "left", "right", "enter", " ":
			m.cfg.Engines.ElevenLabs.Model = cycle(row.options, currentModel(m.cfg), dir(key))
			return m, m.changed()
		}
	case rowLang:
		switch key.String() {
		case "left", "right", "enter", " ":
			m.cfg.LanguageCode = cycle(languageCycle(m.cfg.LanguageCode), m.cfg.LanguageCode, dir(key))
			return m, m.changed()
		}
	case rowBool:
		switch key.String() {
		case "left", "right", "enter", " ":
			m.cfg.Diarize = !m.cfg.Diarize
			return m, m.changed()
		}
	case rowFormat:
		switch key.String() {
		case "enter", " ":
			m.cfg.OutputFormats = toggleFormat(m.cfg.OutputFormats, row.format)
			return m, m.changed()
		}
	case rowText:
		if key.String() == "enter" {
			m.input = textinput.New()
			m.input.SetValue(m.textValue(row))
			m.input.CursorEnd()
			m.input.Focus()
			m.editing = true
			return m, nil
		}
	}
	return m, nil
}

func (m settingsModel) updateEditing(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		m.commit(m.rows[m.cursor], m.input.Value())
		m.editing = false
		return m, m.changed()
	case "esc":
		m.editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

// textValue is the raw (unmasked) value seeded into the editor for a text row.
func (m settingsModel) textValue(row settingsRow) string {
	if row.label == "API Key" {
		return m.cfg.Engines.ElevenLabs.APIKey
	}
	return ""
}

func (m *settingsModel) commit(row settingsRow, value string) {
	if row.label == "API Key" {
		m.cfg.Engines.ElevenLabs.APIKey = value
	}
}

func (m settingsModel) displayValue(row settingsRow) string {
	switch row.kind {
	case rowChoice:
		return "< " + currentModel(m.cfg) + " >"
	case rowLang:
		return "< " + languageDisplay(m.cfg.LanguageCode) + " >"
	case rowBool:
		return fmt.Sprintf("< %v >", m.cfg.Diarize)
	case rowFormat:
		mark := " "
		if formatEnabled(m.cfg.OutputFormats, row.format) {
			mark = "x"
		}
		return "[" + mark + "] " + row.format
	case rowText:
		v := m.textValue(row)
		if row.masked {
			return maskSecret(v)
		}
		return v
	}
	// rowDisplay
	switch row.label {
	case "Engine":
		return m.cfg.Engine
	case "Output Dir":
		return m.cfg.OutputDir
	}
	return ""
}

func (m settingsModel) View() string {
	var sb strings.Builder
	sb.WriteString("Settings\n\n")
	for i, row := range m.rows {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		label := row.label
		if row.kind == rowFormat {
			label = "" // the checkbox already names the format
		}
		if m.editing && i == m.cursor {
			fmt.Fprintf(&sb, "%s%-12s %s\n", cursor, label, m.input.View())
		} else {
			fmt.Fprintf(&sb, "%s%-12s %s\n", cursor, label, m.displayValue(row))
		}
	}
	sb.WriteString("\n")
	if m.saveErr != "" {
		sb.WriteString("save failed: " + m.saveErr + "\n")
	}
	sb.WriteString(m.helpLine())
	return sb.String()
}

func (m settingsModel) helpLine() string {
	if m.editing {
		return "type value • enter save • esc cancel"
	}
	switch m.rows[m.cursor].kind {
	case rowChoice, rowLang, rowBool:
		return "←/→ change • ↑/↓ navigate • esc back"
	case rowFormat:
		return "space toggle • ↑/↓ navigate • esc back"
	case rowText:
		return "enter edit • ↑/↓ navigate • esc back"
	}
	return "↑/↓ navigate • esc back"
}

// --- helpers ---

func currentModel(cfg config.Config) string {
	if cfg.Engines.ElevenLabs.Model == "" {
		return modelOptions[len(modelOptions)-1] // scribe_v2 default
	}
	return cfg.Engines.ElevenLabs.Model
}

// dir maps a key to a cycle direction; unknown keys advance forward.
func dir(key tea.KeyMsg) int {
	if key.String() == "left" {
		return -1
	}
	return 1
}

// languageCycle is the ordered set of language codes to cycle through, including
// the current code if it is not one of the curated options (set via config),
// so cycling never silently discards it.
func languageCycle(current string) []string {
	codes := make([]string, 0, len(languageOptions)+1)
	for _, o := range languageOptions {
		codes = append(codes, o.code)
	}
	if current != "" && indexOfStr(codes, current) < 0 {
		codes = append(codes, current)
	}
	return codes
}

// languageDisplay renders a code as a readable label ("ja — Japanese", "auto").
func languageDisplay(code string) string {
	for _, o := range languageOptions {
		if o.code == code {
			if code == "" {
				return o.name // "auto"
			}
			return code + " — " + o.name
		}
	}
	return code // a custom code set outside the curated list
}

func cycle(options []string, current string, delta int) string {
	idx := indexOfStr(options, current)
	if idx < 0 {
		idx = 0
	}
	return options[(idx+delta+len(options))%len(options)]
}

func indexOfStr(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func formatEnabled(formats []string, f string) bool {
	return indexOfStr(formats, f) >= 0
}

// toggleFormat flips f in the set and returns the result in canonical order, so
// output_formats never drifts into an arbitrary order as it is edited.
func toggleFormat(formats []string, f string) []string {
	enabled := map[string]bool{}
	for _, x := range formats {
		enabled[x] = true
	}
	enabled[f] = !enabled[f]

	var out []string
	for _, name := range formatter.SupportedFormats {
		if enabled[name] {
			out = append(out, name)
		}
	}
	return out
}

func maskSecret(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
