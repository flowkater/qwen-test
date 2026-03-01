package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type FileCache struct {
	RootDir string
}

func NewFileCache(root string) *FileCache {
	return &FileCache{RootDir: root}
}

func (c *FileCache) filePath(key string) string {
	return filepath.Join(c.RootDir, key+".json")
}

func (c *FileCache) Get(key string) (domain.BookInfo, bool, error) {
	path := c.filePath(key)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.BookInfo{}, false, nil
		}
		return domain.BookInfo{}, false, err
	}
	var entry domain.CacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		// corrupted cache must behave as miss
		return domain.BookInfo{}, false, nil
	}
	return entry.Value, true, nil
}

func (c *FileCache) Set(key string, value domain.BookInfo) error {
	if err := os.MkdirAll(c.RootDir, 0o755); err != nil {
		return err
	}
	entry := domain.CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: value.Metadata.CollectedAt.UTC(),
	}
	raw, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath(key), raw, 0o644)
}
