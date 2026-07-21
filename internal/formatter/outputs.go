package formatter

import (
	"os"
	"path/filepath"
	"strings"
)

// OutputPath returns where the transcription of recording recPath is written in
// the given format. Every consumer that looks for a transcription — the list,
// the preview, the Alfred filter — has to agree on this rule, so it lives here
// next to the writer that produces the files.
func OutputPath(dir, recPath, format string) string {
	stem := strings.TrimSuffix(recPath, filepath.Ext(recPath))
	return filepath.Join(dir, stem+"."+format)
}

// PreferredFormat picks the format that best represents a transcription to a
// human: txt if it is enabled, otherwise whichever comes first. It reports false
// when no format is configured at all.
func PreferredFormat(formats []string) (string, bool) {
	if len(formats) == 0 {
		return "", false
	}
	for _, f := range formats {
		if f == "txt" {
			return "txt", true
		}
	}
	return formats[0], true
}

// ExistingOutputs maps each of formats that has actually been written for
// recPath to its path on disk. Formats still pending are absent, so an empty
// result means the recording has not been transcribed.
func ExistingOutputs(dir, recPath string, formats []string) map[string]string {
	out := make(map[string]string, len(formats))
	for _, f := range formats {
		path := OutputPath(dir, recPath, f)
		if _, err := os.Stat(path); err == nil {
			out[f] = path
		}
	}
	return out
}
