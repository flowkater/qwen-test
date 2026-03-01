package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type JSONWriter struct{}

func NewJSONWriter() *JSONWriter { return &JSONWriter{} }

func (w *JSONWriter) Write(q domain.BookQuery, data domain.BookInfo, now time.Time) (string, error) {
	localDate := now.Local().Format("20060102")
	path := domain.ResolveOutputPath(q, localDate)
	if err := domain.EnsureParentDir(path); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	return abs, nil
}
