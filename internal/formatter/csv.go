package formatter

import (
	"bytes"
	"encoding/csv"
)

type csvFormatter struct{}

func (csvFormatter) ext() string { return "csv" }

func (csvFormatter) format(ctx Context) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"time", "speaker", "text"})
	for _, seg := range ctx.Segments {
		speaker := ""
		if seg.Speaker != nil {
			speaker = *seg.Speaker
		}
		_ = w.Write([]string{seg.Time, speaker, seg.Text})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
