package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/matsubo/voice-memo-stt/internal/config"
)

func baseCfg() config.Config {
	c := config.Defaults()
	c.Engines.ElevenLabs.Model = "scribe_v2"
	c.LanguageCode = "jpn"
	c.Diarize = true
	c.OutputFormats = []string{"txt", "json"}
	c.Engines.ElevenLabs.APIKey = "sk-secret-1234"
	return c
}

// cursorOn moves the settings cursor onto the row whose label matches.
func cursorOn(t *testing.T, m settingsModel, label string) settingsModel {
	t.Helper()
	for i, r := range m.rows {
		if r.label == label {
			m.cursor = i
			return m
		}
	}
	t.Fatalf("no settings row labelled %q", label)
	return m
}

// changedCfg runs cmd and extracts the config it announced, failing if it did not.
func changedCfg(t *testing.T, cmd tea.Cmd) config.Config {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a configChangedMsg command, got nil")
	}
	msg, ok := cmd().(configChangedMsg)
	if !ok {
		t.Fatalf("expected configChangedMsg, got %T", cmd())
	}
	return msg.cfg
}

func TestSettings_ToggleDiarize(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "Diarize")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if changedCfg(t, cmd).Diarize {
		t.Error("→ on Diarize should flip true to false")
	}
	if updated.(settingsModel).cfg.Diarize {
		t.Error("the settings model must hold the flipped value")
	}
}

func TestSettings_CycleModel(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "Model")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := changedCfg(t, cmd).Engines.ElevenLabs.Model; got != "scribe_v1" {
		t.Errorf("model after →: got %q, want scribe_v1", got)
	}
}

func TestSettings_ToggleFormatOffAndOn(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "json")

	// json is on; space removes it.
	off, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if got := changedCfg(t, cmd).OutputFormats; contains(got, "json") {
		t.Errorf("json should be removed, got %v", got)
	}

	// space again restores it, in canonical order (txt before json).
	m2 := cursorOn(t, off.(settingsModel), "json")
	_, cmd = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := changedCfg(t, cmd).OutputFormats
	if !contains(got, "json") {
		t.Errorf("json should be restored, got %v", got)
	}
	if indexOf(got, "txt") > indexOf(got, "json") {
		t.Errorf("formats should stay in canonical order, got %v", got)
	}
}

func TestSettings_CycleLanguage(t *testing.T) {
	cfg := baseCfg()
	cfg.LanguageCode = "" // auto
	m := cursorOn(t, newSettingsModel(cfg), "Language")

	// → from auto moves to the first real language.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := changedCfg(t, cmd).LanguageCode; got != "ja" {
		t.Errorf("language after → from auto: got %q, want ja", got)
	}

	// ← from auto wraps to the last option (a real language, not empty twice).
	back, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if got := changedCfg(t, cmd).LanguageCode; got == "" {
		t.Error("← from auto should wrap to a language, not stay on auto")
	}
	_ = next
	_ = back
}

func TestSettings_CycleLanguageKeepsCustomCode(t *testing.T) {
	cfg := baseCfg()
	cfg.LanguageCode = "swa" // not in the curated list
	m := cursorOn(t, newSettingsModel(cfg), "Language")

	// The custom code is displayed as-is and stays reachable in the cycle.
	if !strings.Contains(m.View(), "swa") {
		t.Errorf("a custom language code should be shown, view:\n%s", m.View())
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := changedCfg(t, cmd).LanguageCode; got == "swa" {
		t.Error("→ should advance away from the custom code")
	}
	// Cycling all the way around must return to the custom code, not drop it.
	cur := "swa"
	codes := languageCycle("swa")
	if indexOf(codes, "swa") < 0 {
		t.Fatalf("custom code missing from cycle: %v", codes)
	}
	_ = cur
}

func TestSettings_LanguageDisplaysAutoAndName(t *testing.T) {
	cfg := baseCfg()
	cfg.LanguageCode = ""
	if !strings.Contains(newSettingsModel(cfg).View(), "auto") {
		t.Error("an empty language code should read as 'auto'")
	}

	cfg.LanguageCode = "ja"
	if !strings.Contains(newSettingsModel(cfg).View(), "Japanese") {
		t.Error("a known language should show its name")
	}
}

func TestSettings_APIKeyMaskedInView(t *testing.T) {
	m := newSettingsModel(baseCfg())
	view := m.View()
	if strings.Contains(view, "sk-secret-1234") {
		t.Error("the API key must never be shown in full")
	}
	if !strings.Contains(view, "***") {
		t.Error("the API key row should show a masked placeholder")
	}
}

func TestSettings_EscExitsToList(t *testing.T) {
	m := newSettingsModel(baseCfg())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should return to the list")
	}
	if _, ok := cmd().(backMsg); !ok {
		t.Errorf("esc should emit backMsg, got %T", cmd())
	}
}

func TestSettings_EngineRowNotEditable(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "Engine")
	// Engine is fixed to elevenlabs; cycling keys must do nothing.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd != nil {
		t.Errorf("Engine row must not be editable, got a change: %v", cmd())
	}
}

func contains(s []string, v string) bool {
	return indexOf(s, v) >= 0
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
