package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func TestFormatWriterJSONDefault(t *testing.T) {
	w := NewFormatWriter()
	root := t.TempDir()
	q := domain.BookQuery{Title: "Clean Code", Output: filepath.Join(root, "a.json")}
	info := domain.BookInfo{
		Book: domain.BookMetadata{
			Title:  domain.LocalizedTitle{Original: "Clean Code"},
			Author: "Robert C. Martin",
		},
	}
	path, err := w.Write(q, info, time.Now())
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Fatalf("expected json output, got %s", path)
	}
}

func TestFormatWriterTextMode(t *testing.T) {
	w := NewFormatWriter()
	root := t.TempDir()
	q := domain.BookQuery{
		Title:  "Clean Code",
		Format: "text",
		Output: filepath.Join(root, "a.txt"),
	}
	info := domain.BookInfo{
		Book: domain.BookMetadata{
			Title:  domain.LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author: "Robert C. Martin",
		},
		TableOfContents: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Meaningful Names"}, Depth: 1},
		},
	}
	path, err := w.Write(q, info, time.Now())
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "Title: Clean Code (클린 코드)") {
		t.Fatalf("missing plain text title: %s", text)
	}
	if !strings.Contains(text, "- Meaningful Names") {
		t.Fatalf("missing toc line: %s", text)
	}
}
