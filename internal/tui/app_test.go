package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestTranscribeCmd_MissingKeyReturnsError(t *testing.T) {
	cfg := config.Config{}
	cmd := transcribeCmd(cfg, voicememos.Recording{Path: "test.m4a"})
	msg := cmd()
	done, ok := msg.(transcribeDoneMsg)
	if !ok {
		t.Fatalf("expected transcribeDoneMsg, got %T", msg)
	}
	if done.err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(done.err.Error(), "ElevenLabs API key not set") {
		t.Errorf("error message: got %q, want to contain 'ElevenLabs API key not set'", done.err.Error())
	}
}

func TestStartTranscribe_NoKeySetsStatus(t *testing.T) {
	m := model{cfg: config.Config{}}
	updated, _ := m.Update(startTranscribeMsg{recording: voicememos.Recording{Title: "test"}})
	got := updated.(model)
	if got.screen != screenList {
		t.Errorf("screen: got %v, want screenList (stay on list)", got.screen)
	}
	if !got.statusIsErr {
		t.Error("statusIsErr should be true when key is missing")
	}
	if !strings.Contains(got.statusMsg, "API key not set") {
		t.Errorf("statusMsg: got %q, want to contain 'API key not set'", got.statusMsg)
	}
}

func TestTranscribeDone_WithErrorSetsStatus(t *testing.T) {
	m := model{
		cfg:      config.Config{},
		selected: voicememos.Recording{Title: "Meeting"},
	}
	updated, _ := m.Update(transcribeDoneMsg{err: errMissingKey})
	got := updated.(model)
	if !got.statusIsErr {
		t.Error("statusIsErr should be true on error")
	}
	if !strings.Contains(got.statusMsg, "Error:") {
		t.Errorf("statusMsg: got %q, want to contain 'Error:'", got.statusMsg)
	}
}

func TestTranscribeDone_SuccessSetsStatus(t *testing.T) {
	m := model{
		cfg:      config.Config{},
		selected: voicememos.Recording{Title: "Meeting"},
	}
	updated, _ := m.Update(transcribeDoneMsg{})
	got := updated.(model)
	if got.statusIsErr {
		t.Error("statusIsErr should be false on success")
	}
	if !strings.Contains(got.statusMsg, "Meeting") {
		t.Errorf("statusMsg: got %q, want to contain recording title", got.statusMsg)
	}
}

func TestQuit_QOnListShowsConfirm(t *testing.T) {
	m := model{screen: screenList}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(model)
	if got.screen != screenQuitConfirm {
		t.Errorf("screen: got %v, want screenQuitConfirm", got.screen)
	}
	if got.prevScreen != screenList {
		t.Errorf("prevScreen: got %v, want screenList", got.prevScreen)
	}
	if cmd != nil {
		t.Error("q on list should NOT return tea.Quit cmd")
	}
}

func TestQuit_YConfirms(t *testing.T) {
	m := model{screen: screenQuitConfirm, prevScreen: screenList}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y on quit confirm should return tea.Quit cmd")
	}
	// Invoke the cmd and verify it produces QuitMsg
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg from tea.Quit")
	}
}

func TestQuit_NReturns(t *testing.T) {
	m := model{screen: screenQuitConfirm, prevScreen: screenList}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := updated.(model)
	if got.screen != screenList {
		t.Errorf("n on quit confirm should return to prevScreen, got %v", got.screen)
	}
}

func TestQuit_EscReturns(t *testing.T) {
	m := model{screen: screenQuitConfirm, prevScreen: screenPreview}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.screen != screenPreview {
		t.Errorf("esc on quit confirm should return to prevScreen, got %v", got.screen)
	}
}

func TestQuit_CtrlCFromNonListShowsConfirm(t *testing.T) {
	m := model{screen: screenPreview}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := updated.(model)
	if got.screen != screenQuitConfirm {
		t.Errorf("ctrl+c should show quit confirm, got screen %v", got.screen)
	}
	if got.prevScreen != screenPreview {
		t.Errorf("prevScreen: got %v, want screenPreview", got.prevScreen)
	}
	if cmd != nil {
		t.Error("ctrl+c should NOT return tea.Quit directly anymore")
	}
}
