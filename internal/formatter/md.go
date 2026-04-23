package formatter

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

type mdFormatter struct{}

func (mdFormatter) ext() string { return "md" }

func (mdFormatter) format(ctx Context) ([]byte, error) {
	stem := strings.TrimSuffix(ctx.File, filepath.Ext(ctx.File))
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", stem)
	for _, seg := range ctx.Segments {
		fmt.Fprintf(&buf, "- **%s** %s%s\n", seg.Time, speakerPrefix(seg), seg.Text)
	}
	return buf.Bytes(), nil
}
