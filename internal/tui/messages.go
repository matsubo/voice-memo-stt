package tui

import "github.com/matsubo/voice-memo-stt/internal/voicememos"

type navigateMsg struct {
	to screen
}

type startTranscribeMsg struct {
	recording voicememos.Recording
}

// startJobMsg is emitted once the cost has been confirmed: the transcription is
// handed to a background command and the user is returned to the list.
type startJobMsg struct {
	recording voicememos.Recording
}

// transcribeDoneMsg names its recording because several jobs may be in flight.
type transcribeDoneMsg struct {
	path  string
	title string
	err   error
}

type backMsg struct{}
