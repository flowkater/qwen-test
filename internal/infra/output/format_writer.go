package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type FormatWriter struct {
	JSON *JSONWriter
	Text *TextWriter
}

func NewFormatWriter() *FormatWriter {
	return &FormatWriter{
		JSON: NewJSONWriter(),
		Text: NewTextWriter(),
	}
}

func (w *FormatWriter) Write(q domain.BookQuery, data domain.BookInfo, now time.Time) (string, error) {
	format := strings.ToLower(strings.TrimSpace(q.Format))
	switch format {
	case "", domain.DefaultOutputFormat:
		return w.JSON.Write(q, data, now)
	case "text":
		return w.Text.Write(q, data, now)
	default:
		return "", fmt.Errorf("unsupported format: %s", q.Format)
	}
}
