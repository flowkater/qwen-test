package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateQueryRules(t *testing.T) {
	t.Run("missing all query", func(t *testing.T) {
		if err := ValidateQuery(BookQuery{}); err == nil {
			t.Fatalf("expected error")
		}
	})
	t.Run("invalid lang", func(t *testing.T) {
		err := ValidateQuery(BookQuery{Title: "Clean Code", Lang: "xyz"})
		if err == nil || err.Error() != ErrInvalidLang.Error() {
			t.Fatalf("expected invalid lang, got %v", err)
		}
	})
	t.Run("batch bypasses title requirement", func(t *testing.T) {
		err := ValidateQuery(BookQuery{BatchPath: "books.txt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("output path cannot be directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ValidateQuery(BookQuery{Title: "Clean Code", Output: dir})
		if err == nil || err.Error() != ErrInvalidOutput.Error() {
			t.Fatalf("expected invalid output, got %v", err)
		}
	})
	t.Run("batch output can be directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ValidateQuery(BookQuery{BatchPath: "books.txt", Output: dir})
		if err != nil {
			t.Fatalf("batch output dir should be valid: %v", err)
		}
	})
	t.Run("batch output existing file is invalid", func(t *testing.T) {
		root := t.TempDir()
		file := filepath.Join(root, "out.json")
		if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		err := ValidateQuery(BookQuery{BatchPath: "books.txt", Output: file})
		if err == nil {
			t.Fatalf("batch output as file should be invalid")
		}
	})
}

func TestResolveOutputPath(t *testing.T) {
	q := BookQuery{Title: "Clean Code"}
	path := ResolveOutputPath(q, "20260301")
	if path != "Clean-Code_20260301.json" {
		t.Fatalf("unexpected path: %s", path)
	}
	q = BookQuery{ISBN13: "978-0132350884"}
	path = ResolveOutputPath(q, "20260301")
	if path != "978-0132350884_20260301.json" {
		t.Fatalf("unexpected isbn path: %s", path)
	}
}

func TestEnsureParentDir(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "a", "b", "c.json")
	if err := EnsureParentDir(out); err != nil {
		t.Fatalf("ensure parent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "b")); err != nil {
		t.Fatalf("parent dir missing: %v", err)
	}
}
