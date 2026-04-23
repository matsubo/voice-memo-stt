package tui

import (
	"context"
	"path/filepath"
	"strings"

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
	screenProgress
	screenPreview
	screenSettings
	screenQuitConfirm
)

type model struct {
	cfg         config.Config
	screen      screen
	prevScreen  screen // screen active before quit confirm was shown
	list        listModel
	confirm     confirmModel
	progress    progressModel
	preview     previewModel
	settings    settingsModel
	recordings  []voicememos.Recording
	loadError   error
	selected    voicememos.Recording // the recording chosen from the list
	statusMsg   string               // transient status shown in list header
	statusIsErr bool
}

// Run starts the bubbletea TUI program with alt-screen.
func Run(cfg config.Config) error {
	m := model{cfg: cfg}
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
		if cfg.Engines.ElevenLabs.APIKey == "" {
			return transcribeDoneMsg{err: errMissingKey}
		}
		eng := elevenlabs.New(cfg.Engines.ElevenLabs.APIKey, cfg.Engines.ElevenLabs.Model)
		audioPath := filepath.Join(voicememos.AudioDir(), rec.Path)
		result, err := eng.Transcribe(context.Background(), audioPath, engine.TranscribeOptions{
			LanguageCode: cfg.LanguageCode,
			Diarize:      cfg.Diarize,
		})
		if err != nil {
			return transcribeDoneMsg{err: err}
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
			return transcribeDoneMsg{err: err}
		}
		return transcribeDoneMsg{}
	}
}

var errMissingKey = &missingKeyError{}

type missingKeyError struct{}

func (*missingKeyError) Error() string {
	return "ElevenLabs API key not set — run: vmt config set engines.elevenlabs.api_key sk-..."
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.list = newListModel(m.recordings, m.cfg.OutputDir, m.cfg.OutputFormats)
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
		m.selected = msg.recording
		eng := elevenlabs.New(m.cfg.Engines.ElevenLabs.APIKey, m.cfg.Engines.ElevenLabs.Model)
		cost := eng.EstimateCost(msg.recording.Duration)
		m.confirm = newConfirmModel(msg.recording, cost)
		m.screen = screenConfirm
		return m, nil

	case navigateMsg:
		m.statusMsg = "" // clear stale status on any screen change
		m.statusIsErr = false
		switch msg.to {
		case screenProgress:
			m.progress = newProgressModel(m.selected.Title)
			m.screen = screenProgress
			return m, tea.Batch(m.progress.Init(), transcribeCmd(m.cfg, m.selected))
		case screenPreview:
			stem := strings.TrimSuffix(m.selected.Path, filepath.Ext(m.selected.Path))
			if rec, ok := m.list.selected(); ok {
				stem = strings.TrimSuffix(rec.Path, filepath.Ext(rec.Path))
			}
			m.preview = newPreviewModel(stem, m.cfg.OutputDir, m.cfg.OutputFormats)
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

	case backMsg:
		m.screen = screenList
		return m, nil

	case transcribeDoneMsg:
		m.screen = screenList
		if msg.err != nil {
			m.statusMsg = "Error: " + msg.err.Error()
			m.statusIsErr = true
		} else {
			m.statusMsg = "Transcription complete: " + m.selected.Title
			m.statusIsErr = false
			// Rebuild list so the new ✓ mark appears
			m.list = newListModel(m.recordings, m.cfg.OutputDir, m.cfg.OutputFormats)
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
	case screenProgress:
		updated, cmd := m.progress.Update(msg)
		m.progress = updated.(progressModel)
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
	case screenProgress:
		return m.progress.View()
	case screenPreview:
		return m.preview.View()
	case screenSettings:
		return m.settings.View()
	case screenQuitConfirm:
		return "Quit vmt?\n\n[y] yes  [n/esc] cancel"
	}
	return "Loading recordings..."
}
