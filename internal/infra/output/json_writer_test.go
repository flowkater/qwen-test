package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func TestJSONWriterCreatesParentAndWritesUTF8(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out", "book.json")
	w := NewJSONWriter()
	q := domain.BookQuery{Title: "클린 코드", Output: path}
	info := domain.BookInfo{
		Book: domain.BookMetadata{
			Title:     domain.LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley Professional",
			ISBN13:    "978-0132350884",
		},
		Metadata: domain.CollectMetadata{
			CollectedAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
			Mode:        "basic",
			Query:       domain.CollectQuery{},
		},
	}
	abs, err := w.Write(q, info, time.Now())
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	book := decoded["book"].(map[string]any)
	title := book["title"].(map[string]any)
	if title["korean"] != "클린 코드" {
		t.Fatalf("utf8 title mismatch")
	}
}
