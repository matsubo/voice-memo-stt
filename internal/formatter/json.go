package formatter

import "encoding/json"

type jsonFormatter struct{}

func (jsonFormatter) ext() string { return "json" }

type jsonOutput struct {
	File       string        `json:"file"`
	RecordedAt string        `json:"recorded_at"`
	Duration   float64       `json:"duration"`
	Engine     string        `json:"engine"`
	Model      string        `json:"model"`
	Segments   []jsonSegment `json:"segments"`
}

type jsonSegment struct {
	Time    string  `json:"time"`
	Speaker *string `json:"speaker,omitempty"`
	Text    string  `json:"text"`
}

func (jsonFormatter) format(ctx Context) ([]byte, error) {
	segs := make([]jsonSegment, len(ctx.Segments))
	for i, s := range ctx.Segments {
		segs[i] = jsonSegment{Time: s.Time, Speaker: s.Speaker, Text: s.Text}
	}
	out := jsonOutput{
		File:       ctx.File,
		RecordedAt: ctx.RecordedAt.Format("2006-01-02T15:04:05"),
		Duration:   ctx.Duration,
		Engine:     ctx.Engine,
		Model:      ctx.Model,
		Segments:   segs,
	}
	return json.MarshalIndent(out, "", "  ")
}
