package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func TestFileCacheReadWriteAndCorruptionHandling(t *testing.T) {
	root := t.TempDir()
	c := NewFileCache(root)
	key := "abc"
	value := domain.BookInfo{
		Book: domain.BookMetadata{
			Title:     domain.LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley Professional",
			ISBN13:    "978-0132350884",
		},
		Metadata: domain.CollectMetadata{
			CollectedAt: time.Now().UTC(),
			Mode:        "basic",
			Query:       domain.CollectQuery{},
		},
	}
	if err := c.Set(key, value); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	got, ok, err := c.Get(key)
	if err != nil || !ok {
		t.Fatalf("get cache: ok=%v err=%v", ok, err)
	}
	if got.Book.ISBN13 != value.Book.ISBN13 {
		t.Fatalf("cache value mismatch")
	}

	// corrupted file must become cache miss (no hard failure)
	path := filepath.Join(root, key+".json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("corrupt write: %v", err)
	}
	_, ok, err = c.Get(key)
	if err != nil {
		t.Fatalf("corrupted cache must not return error: %v", err)
	}
	if ok {
		t.Fatalf("corrupted cache must be treated as miss")
	}
}
