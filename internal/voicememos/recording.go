package voicememos

import (
	"fmt"
	"time"
)

// Recording represents a single Voice Memos recording row from ZCLOUDRECORDING.
type Recording struct {
	ID       int64
	Title    string
	Path     string
	Duration float64 // seconds
	Date     time.Time
}

// DurationFormatted returns duration as "1h06m" or "45m30s", or "—" if unknown.
func (r Recording) DurationFormatted() string {
	if r.Duration <= 0 {
		return "—"
	}
	d := int(r.Duration)
	h := d / 3600
	m := (d % 3600) / 60
	s := d % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}
