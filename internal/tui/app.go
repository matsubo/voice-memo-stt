package tui

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matsubo/voice-memo-stt/internal/config"
	"github.com/matsubo/voice-memo-stt/internal/engine"
	"github.com/matsubo/voice-memo-stt/internal/engine/elevenlabs"
	"github.com/matsubo/voice-memo-stt/internal/formatter"
	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

type screen int

const (
	screenList screen = iota
	screenConfirm
	screenPreview
	screenSettings
	screenQuitConfirm
)

type model struct {
	cfg         config.Config
	cfgPath     string // where settings edits are persisted
	screen      screen
	prevScreen  screen // screen active before quit confirm was shown
	list        listModel
	confirm     confirmModel
	preview     previewModel
	settings    settingsModel
	recordings  []voicememos.Recording
	loadError   error
	selected    voicememos.Recording // the recording chosen from the list
	running     map[string]bool      // recording path -> transcription in flight
	failed      map[string]bool      // recording path -> last transcription failed
	statusMsg   string               // transient status shown in list header
	statusIsErr bool
	width       int // last known terminal size
	height      int
	bell        io.Writer // where the completion beep is written (nil → stderr)
}

// Run starts the bubbletea TUI program with alt-screen. cfgPath is where the
// settings screen persists changes.
func Run(cfg config.Config, cfgPath string) error {
	m := model{cfg: cfg, cfgPath: cfgPath}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return loadRecordingsCmd()
}

type recordingsLoadedMsg struct {
	recordings []voicememos.Recording
	err        error
}

func loadRecordingsCmd() tea.Cmd {
	return func() tea.Msg {
		recs, err := voicememos.Load(context.Background())
		return recordingsLoadedMsg{recordings: recs, err: err}
	}
}

func transcribeCmd(cfg config.Config, rec voicememos.Recording) tea.Cmd {
	return func() tea.Msg {
		done := transcribeDoneMsg{path: rec.Path, title: rec.Title}
		if cfg.Engines.ElevenLabs.APIKey == "" {
			done.err = errMissingKey
			return done
		}
		eng := elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, cfg.Engines.ElevenLabs.Model)
		audioPath := filepath.Join(voicememos.AudioDir(), rec.Path)
		result, err := eng.Transcribe(context.Background(), audioPath, engine.TranscribeOptions{
			LanguageCode: cfg.LanguageCode,
			Diarize:      cfg.Diarize,
		})
		if err != nil {
			done.err = err
			return done
		}
		outDir := config.ExpandPath(cfg.OutputDir)
		fmtCtx := formatter.Context{
			File:       rec.Path,
			RecordedAt: rec.Date,
			Duration:   rec.Duration,
			Engine:     eng.Name(),
			Model:      cfg.Engines.ElevenLabs.Model,
			Segments:   result.Segments,
		}
		if err := formatter.Write(outDir, fmtCtx, cfg.OutputFormats); err != nil {
			done.err = err
		}
		return done
	}
}

var errMissingKey = &missingKeyError{}

type missingKeyError struct{}

func (*missingKeyError) Error() string {
	return "ElevenLabs API key not set — run: vmt config set engines.elevenlabs.api_key sk-..."
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// withJob returns a copy of set with key added or removed, so a previous model
// value never sees a job flip underneath it.
func withJob(set map[string]bool, key string, member bool) map[string]bool {
	next := make(map[string]bool, len(set)+1)
	for k := range set {
		next[k] = true
	}
	if member {
		next[key] = true
	} else {
		delete(next, key)
	}
	return next
}

// rebuildList re-reads transcription output from disk (marks, char counts) while
// preserving the cursor and current job state.
func (m model) rebuildList() model {
	cursor := m.list.table.Cursor()
	m.list = newListModel(m.recordings, m.cfg.OutputDir, m.cfg.OutputFormats).withJobs(m.running, m.failed)
	m.list.table.SetCursor(cursor)
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Remember the terminal size even while another screen is active, so a screen
	// created later (preview) can lay itself out without waiting for a resize.
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
	}

	// The running-job spinner must keep animating on whichever screen the user
	// wandered off to, so its ticks bypass the per-screen dispatch below.
	if _, ok := msg.(spinner.TickMsg); ok {
		updated, cmd := m.list.Update(msg)
		m.list = updated.(listModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Quit confirm: 'y' exits, 'n'/'esc' returns to previous screen.
		if m.screen == screenQuitConfirm {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			case "n", "N", "esc":
				m.screen = m.prevScreen
				return m, nil
			}
			return m, nil
		}
		// ctrl+c anywhere, or 'q' on list — show quit confirm instead of exiting directly.
		if msg.String() == "ctrl+c" || (msg.String() == "q" && m.screen == screenList) {
			m.prevScreen = m.screen
			m.screen = screenQuitConfirm
			return m, nil
		}

	case recordingsLoadedMsg:
		m.loadError = msg.err
		if msg.err == nil {
			m.recordings = msg.recordings
			m.list = newListModel(m.recordings, m.cfg.OutputDir, m.cfg.OutputFormats).withJobs(m.running, m.failed)
		}
		m.screen = screenList
		return m, nil

	case startTranscribeMsg:
		if m.cfg.Engines.ElevenLabs.APIKey == "" {
			m.statusMsg = errMissingKey.Error()
			m.statusIsErr = true
			m.screen = screenList
			return m, nil
		}
		if m.running[msg.recording.Path] {
			m.statusMsg = "Already transcribing: " + msg.recording.Title
			m.statusIsErr = false
			m.screen = screenList
			return m, nil
		}
		m.selected = msg.recording
		eng := elevenlabs.New(m.cfg.Engines.ElevenLabs.APIKey, m.cfg.Engines.ElevenLabs.Model)
		cost := eng.EstimateCost(msg.recording.Duration)
		m.confirm = newConfirmModel(msg.recording, cost)
		m.screen = screenConfirm
		return m, nil

	case startJobMsg:
		// Hand the transcription to a background command and go straight back to
		// the list, so the user can keep browsing and queue more work.
		rec := msg.recording
		wasIdle := len(m.running) == 0
		m.running = withJob(m.running, rec.Path, true)
		m.failed = withJob(m.failed, rec.Path, false)
		m.statusMsg = "Transcribing in background: " + rec.Title
		m.statusIsErr = false
		m.screen = screenList
		m = m.rebuildList()

		cmds := []tea.Cmd{transcribeCmd(m.cfg, rec)}
		if wasIdle {
			// Only one animation loop, no matter how many jobs are queued.
			cmds = append(cmds, m.list.spinnerTick())
		}
		return m, tea.Batch(cmds...)

	case navigateMsg:
		m.statusMsg = "" // clear stale status on any screen change
		m.statusIsErr = false
		switch msg.to {
		case screenPreview:
			stem := strings.TrimSuffix(m.selected.Path, filepath.Ext(m.selected.Path))
			if rec, ok := m.list.selected(); ok {
				stem = strings.TrimSuffix(rec.Path, filepath.Ext(rec.Path))
			}
			m.preview = newPreviewModel(m.cfg, stem, m.width, m.height)
			m.screen = screenPreview
			return m, nil
		case screenSettings:
			m.settings = newSettingsModel(m.cfg)
			m.screen = screenSettings
			return m, nil
		case screenList:
			m.screen = screenList
			return m, nil
		}

	case configChangedMsg:
		// Persist settings edits immediately. A format change also alters the
		// list's ✓ marks and char counts, so rebuild the list when it happens.
		formatsChanged := !equalStrings(m.cfg.OutputFormats, msg.cfg.OutputFormats)
		m.cfg = msg.cfg
		m.settings.cfg = msg.cfg
		if err := config.Save(m.cfgPath, m.cfg); err != nil {
			m.settings.saveErr = err.Error()
		} else {
			m.settings.saveErr = ""
		}
		if formatsChanged {
			m = m.rebuildList()
		}
		return m, nil

	case backMsg:
		m.screen = screenList
		return m, nil

	case editorFinishedMsg:
		// The edit may have changed the transcription's length, so the list's
		// char count has to be recomputed alongside the preview's buffer.
		updated, cmd := m.preview.afterEdit(msg.err)
		m.preview = updated
		m = m.rebuildList()
		return m, cmd

	case transcribeDoneMsg:
		m.running = withJob(m.running, msg.path, false)
		m.failed = withJob(m.failed, msg.path, msg.err != nil)
		if msg.err != nil {
			m.statusMsg = "Error: " + msg.title + ": " + msg.err.Error()
			m.statusIsErr = true
		} else {
			m.statusMsg = "Transcription complete: " + msg.title
			m.statusIsErr = false
		}
		// Re-read the outputs so the ✓ mark and char count appear.
		m = m.rebuildList()
		// The point of the beep is to call the user back to a job they left
		// running, so a failure rings too — it needs attention more, not less.
		if m.cfg.BeepOnComplete {
			return m, bellCmd(m.bell)
		}
		return m, nil
	}

	switch m.screen {
	case screenList:
		updated, cmd := m.list.Update(msg)
		m.list = updated.(listModel)
		return m, cmd
	case screenConfirm:
		updated, cmd := m.confirm.Update(msg)
		m.confirm = updated.(confirmModel)
		return m, cmd
	case screenPreview:
		updated, cmd := m.preview.Update(msg)
		m.preview = updated.(previewModel)
		return m, cmd
	case screenSettings:
		updated, cmd := m.settings.Update(msg)
		m.settings = updated.(settingsModel)
		return m, cmd
	}
	return m, nil
}

// quitConfirmView warns about work that would be lost, since transcriptions now
// run in the background and quitting kills them.
func (m model) quitConfirmView() string {
	if n := len(m.running); n > 0 {
		job := "job is"
		if n > 1 {
			job = "jobs are"
		}
		return fmt.Sprintf("Quit vmt?\n\n%d transcription %s still running and will be lost.\n\n[y] yes  [n/esc] cancel", n, job)
	}
	return "Quit vmt?\n\n[y] yes  [n/esc] cancel"
}

func (m model) View() string {
	if m.loadError != nil {
		return "Failed to load recordings: " + m.loadError.Error() + "\n\nPress q to quit."
	}
	switch m.screen {
	case screenList:
		header := ""
		if m.statusMsg != "" {
			prefix := "✓ "
			if m.statusIsErr {
				prefix = "✗ "
			}
			header = prefix + m.statusMsg + "\n\n"
		}
		return header + m.list.View()
	case screenConfirm:
		return m.confirm.View()
	case screenPreview:
		return m.preview.View()
	case screenSettings:
		return m.settings.View()
	case screenQuitConfirm:
		return m.quitConfirmView()
	}
	return "Loading recordings..."
}
