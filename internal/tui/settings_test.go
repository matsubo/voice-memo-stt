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

func TestSettings_EditLanguageCommits(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "Language")

	// enter starts editing; no config change yet.
	editing, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	em := editing.(settingsModel)
	if !em.editing {
		t.Fatal("enter on a text row should start editing")
	}
	if cmd != nil {
		t.Error("starting an edit must not announce a config change yet")
	}

	em.input.SetValue("eng")
	done, cmd := em.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if done.(settingsModel).editing {
		t.Error("enter should commit and leave edit mode")
	}
	if got := changedCfg(t, cmd).LanguageCode; got != "eng" {
		t.Errorf("language: got %q, want eng", got)
	}
}

func TestSettings_EditEscapeCancels(t *testing.T) {
	m := cursorOn(t, newSettingsModel(baseCfg()), "Language")
	editing, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	em := editing.(settingsModel)
	em.input.SetValue("eng")

	cancelled, cmd := em.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cm := cancelled.(settingsModel)
	if cm.editing {
		t.Error("esc should leave edit mode")
	}
	if cm.cfg.LanguageCode != "jpn" {
		t.Errorf("esc must discard the edit, language changed to %q", cm.cfg.LanguageCode)
	}
	if cmd != nil {
		t.Error("a cancelled edit must not announce a config change")
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
