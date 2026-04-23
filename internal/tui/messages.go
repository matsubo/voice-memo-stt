package tui

import "github.com/matsubo/voice-memo-stt/internal/voicememos"

type navigateMsg struct {
	to screen
}

type startTranscribeMsg struct {
	recording voicememos.Recording
}

type transcribeDoneMsg struct {
	err error
}

type backMsg struct{}
