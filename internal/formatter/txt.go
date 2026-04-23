package formatter

import (
	"bytes"
	"fmt"
)

type txtFormatter struct{}

func (txtFormatter) ext() string { return "txt" }

func (txtFormatter) format(ctx Context) ([]byte, error) {
	var buf bytes.Buffer
	for _, seg := range ctx.Segments {
		fmt.Fprintf(&buf, "[%s] %s%s\n", seg.Time, speakerPrefix(seg), seg.Text)
	}
	return buf.Bytes(), nil
}
