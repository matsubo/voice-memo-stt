package alfred

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
)

// Item represents a single Alfred Script Filter result item.
type Item struct {
	UID      string   `json:"uid"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Arg      string   `json:"arg"`
	Icon     Icon     `json:"icon"`
	Mods     ItemMods `json:"mods"`
}

// Icon holds the icon path for an Alfred item.
type Icon struct {
	Path string `json:"path"`
}

// ItemMods holds modifier key overrides for an Alfred item.
type ItemMods struct {
	Cmd ModEntry `json:"cmd"`
}

// ModEntry is a single modifier key entry with its own subtitle and arg.
type ModEntry struct {
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
}

// Output is the top-level Alfred Script Filter JSON envelope.
type Output struct {
	Items []Item `json:"items"`
}

// Build converts a slice of recordings into Alfred Script Filter JSON.
// transcribed maps recording paths to whether they have been transcribed.
// query, if non-empty, filters recordings by case-insensitive title substring.
func Build(recs []voicememos.Recording, transcribed map[string]bool, query string) ([]byte, error) {
	items := make([]Item, 0, len(recs))
	lowerQuery := strings.ToLower(query)

	for _, r := range recs {
		if query != "" && !strings.Contains(strings.ToLower(r.Title), lowerQuery) {
			continue
		}

		iconPath := "icons/pending.png"
		if transcribed[r.Path] {
			iconPath = "icons/transcribed.png"
		}

		subtitle := fmt.Sprintf("%s (%s)", r.Date.Format("2006-01-02 15:04"), r.DurationFormatted())

		items = append(items, Item{
			UID:      r.Path,
			Title:    r.Title,
			Subtitle: subtitle,
			Arg:      r.Path,
			Icon:     Icon{Path: iconPath},
			Mods: ItemMods{
				Cmd: ModEntry{
					Subtitle: "Preview transcription",
					Arg:      r.Path,
				},
			},
		})
	}

	return json.Marshal(Output{Items: items})
}
