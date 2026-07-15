package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

func TestTranscribeCmd_MissingKeyReturnsError(t *testing.T) {
	cfg := config.Config{}
	cmd := transcribeCmd(cfg, voicememos.Recording{Path: "test.m4a", Title: "Test"})
	msg := cmd()
	done, ok := msg.(transcribeDoneMsg)
	if !ok {
		t.Fatalf("expected transcribeDoneMsg, got %T", msg)
	}
	if done.err == nil {
		t.Fatal("expected error for missing API key")
	}
	if done.path != "test.m4a" {
		t.Errorf("path: got %q, want %q — the result must name its recording", done.path, "test.m4a")
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

func TestStartJob_ReturnsToListAndRunsInBackground(t *testing.T) {
	m := withKey(model{})
	rec := voicememos.Recording{Path: "a.m4a", Title: "Meeting"}

	updated, cmd := m.Update(startJobMsg{recording: rec})
	got := updated.(model)

	if got.screen != screenList {
		t.Errorf("screen: got %v, want screenList — the user must keep control while it transcribes", got.screen)
	}
	if !got.running["a.m4a"] {
		t.Error("the recording should be marked as in flight")
	}
	if cmd == nil {
		t.Fatal("expected a background transcription command")
	}
}

func TestStartJob_SecondJobRunsAlongsideTheFirst(t *testing.T) {
	m := withKey(model{})
	first, _ := m.Update(startJobMsg{recording: voicememos.Recording{Path: "a.m4a", Title: "A"}})
	second, _ := first.(model).Update(startJobMsg{recording: voicememos.Recording{Path: "b.m4a", Title: "B"}})
	got := second.(model)

	if len(got.running) != 2 {
		t.Errorf("running jobs: got %d, want 2 — jobs must not block each other", len(got.running))
	}
}

func TestStartTranscribe_AlreadyRunningIsRejected(t *testing.T) {
	m := withKey(model{running: map[string]bool{"a.m4a": true}})

	updated, _ := m.Update(startTranscribeMsg{recording: voicememos.Recording{Path: "a.m4a", Title: "Meeting"}})
	got := updated.(model)

	if got.screen != screenList {
		t.Errorf("screen: got %v, want screenList (no confirm for an in-flight job)", got.screen)
	}
	if !strings.Contains(got.statusMsg, "Already transcribing") {
		t.Errorf("statusMsg: got %q, want it to say the job is already running", got.statusMsg)
	}
}

func TestTranscribeDone_ClearsRunningJob(t *testing.T) {
	m := model{cfg: config.Config{}, running: map[string]bool{"a.m4a": true}}

	updated, _ := m.Update(transcribeDoneMsg{path: "a.m4a", title: "Meeting"})
	got := updated.(model)

	if got.running["a.m4a"] {
		t.Error("a finished job must not stay marked as running")
	}
	if got.failed["a.m4a"] {
		t.Error("a successful job must not be marked as failed")
	}
}

func TestTranscribeDone_WithErrorSetsStatus(t *testing.T) {
	m := model{cfg: config.Config{}, running: map[string]bool{"a.m4a": true}}
	updated, _ := m.Update(transcribeDoneMsg{path: "a.m4a", title: "Meeting", err: errMissingKey})
	got := updated.(model)
	if !got.statusIsErr {
		t.Error("statusIsErr should be true on error")
	}
	if !got.failed["a.m4a"] {
		t.Error("a failed job should be marked as failed")
	}
	if got.running["a.m4a"] {
		t.Error("a failed job must not stay marked as running")
	}
	if !strings.Contains(got.statusMsg, "Error:") {
		t.Errorf("statusMsg: got %q, want to contain 'Error:'", got.statusMsg)
	}
}

func TestTranscribeDone_SuccessSetsStatus(t *testing.T) {
	// The status must come from the message, not from the last selected recording:
	// another job may have been started since.
	m := model{cfg: config.Config{}, selected: voicememos.Recording{Title: "Something Else"}}
	updated, _ := m.Update(transcribeDoneMsg{path: "a.m4a", title: "Meeting"})
	got := updated.(model)
	if got.statusIsErr {
		t.Error("statusIsErr should be false on success")
	}
	if !strings.Contains(got.statusMsg, "Meeting") {
		t.Errorf("statusMsg: got %q, want the title of the finished job", got.statusMsg)
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

func TestQuitConfirmView_WarnsAboutRunningJobs(t *testing.T) {
	idle := model{}
	if strings.Contains(idle.quitConfirmView(), "running") {
		t.Error("idle quit confirm should not warn about jobs")
	}

	busy := model{running: map[string]bool{"a.m4a": true, "b.m4a": true}}
	if !strings.Contains(busy.quitConfirmView(), "2 transcription jobs are still running") {
		t.Errorf("quit confirm should warn that work will be lost, got:\n%s", busy.quitConfirmView())
	}
}

// withKey returns m with an API key set, so the confirm/job path is reachable.
func withKey(m model) model {
	m.cfg.Engines.ElevenLabs.APIKey = "sk-test"
	return m
}

func TestConfigChanged_PersistsAndAdopts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := model{cfg: config.Defaults(), cfgPath: path}
	m.settings = newSettingsModel(m.cfg)

	next := config.Defaults()
	next.LanguageCode = "eng"
	next.Engines.ElevenLabs.APIKey = "sk-new"

	updated, _ := m.Update(configChangedMsg{cfg: next})
	got := updated.(model)

	if got.cfg.LanguageCode != "eng" {
		t.Errorf("app must adopt the edited config, language is %q", got.cfg.LanguageCode)
	}

	saved, err := config.Load(path)
	if err != nil {
		t.Fatalf("reload saved config: %v", err)
	}
	if saved.LanguageCode != "eng" || saved.Engines.ElevenLabs.APIKey != "sk-new" {
		t.Errorf("settings edit was not written to disk: %+v", saved)
	}
}

func TestConfigChanged_FormatChangeRebuildsList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rec.md"), []byte("# hi"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.OutputDir = dir
	cfg.OutputFormats = []string{"txt"} // rec.md not counted yet
	m := model{cfg: cfg, cfgPath: filepath.Join(dir, "config.json")}
	m.recordings = []voicememos.Recording{{Path: "rec.m4a", Title: "Rec"}}
	m.list = newListModel(m.recordings, cfg.OutputDir, cfg.OutputFormats)

	if m.list.cells[0].done {
		t.Fatal("precondition: rec should not be marked done for txt-only")
	}

	next := cfg
	next.OutputFormats = []string{"txt", "md"} // now md counts

	updated, _ := m.Update(configChangedMsg{cfg: next})
	if !updated.(model).list.cells[0].done {
		t.Error("enabling the md format should re-mark the recording as transcribed")
	}
}
