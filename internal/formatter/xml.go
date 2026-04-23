package formatter

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type xmlFormatter struct{}

func (xmlFormatter) ext() string { return "xml" }

type xmlTranscription struct {
	XMLName    xml.Name     `xml:"transcription"`
	File       string       `xml:"file,attr"`
	RecordedAt string       `xml:"recorded_at,attr"`
	Duration   float64      `xml:"duration,attr"`
	Engine     string       `xml:"engine,attr"`
	Model      string       `xml:"model,attr"`
	Segments   []xmlSegment `xml:"segment"`
}

type xmlSegment struct {
	Time    string `xml:"time,attr"`
	Speaker string `xml:"speaker,attr,omitempty"`
	Text    string `xml:",chardata"`
}

func (xmlFormatter) format(ctx Context) ([]byte, error) {
	segs := make([]xmlSegment, len(ctx.Segments))
	for i, s := range ctx.Segments {
		speaker := ""
		if s.Speaker != nil {
			speaker = *s.Speaker
		}
		segs[i] = xmlSegment{Time: s.Time, Speaker: speaker, Text: s.Text}
	}
	out := xmlTranscription{
		File:       ctx.File,
		RecordedAt: ctx.RecordedAt.Format("2006-01-02T15:04:05"),
		Duration:   ctx.Duration,
		Engine:     ctx.Engine,
		Model:      ctx.Model,
		Segments:   segs,
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, fmt.Errorf("encode xml: %w", err)
	}
	return buf.Bytes(), nil
}
