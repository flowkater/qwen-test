package cli

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestParseArgsContracts(t *testing.T) {
	t.Run("title positional", func(t *testing.T) {
		q, err := ParseArgs([]string{"Clean Code"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if q.Title != "Clean Code" {
			t.Fatalf("title mismatch: %q", q.Title)
		}
		if q.Model == "" {
			t.Fatalf("default model must be set")
		}
		if q.Format != "json" {
			t.Fatalf("default format must be json, got %q", q.Format)
		}
	})

	t.Run("isbn wins as preferred identifier", func(t *testing.T) {
		q, err := ParseArgs([]string{"Clean Code", "--isbn", "978-0132350884"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if q.ISBN13 != "978-0132350884" {
			t.Fatalf("isbn mismatch: %q", q.ISBN13)
		}
	})

	t.Run("missing title/isbn/batch fails", func(t *testing.T) {
		if _, err := ParseArgs([]string{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("expected parse validation error")
		}
	})

	t.Run("batch bypasses missing title", func(t *testing.T) {
		q, err := ParseArgs([]string{"--batch", "books.txt"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if q.BatchPath != "books.txt" {
			t.Fatalf("batch mismatch")
		}
	})

	t.Run("batch accepts directory output", func(t *testing.T) {
		dir := t.TempDir()
		q, err := ParseArgs([]string{"--batch", "books.txt", "--output", dir}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if filepath.Clean(q.Output) != filepath.Clean(dir) {
			t.Fatalf("output mismatch: %s", q.Output)
		}
	})

	t.Run("format text", func(t *testing.T) {
		q, err := ParseArgs([]string{"Clean Code", "--format", "text"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if q.Format != "text" {
			t.Fatalf("format mismatch: %q", q.Format)
		}
	})
}

func TestHelpIncludesRequiredFlags(t *testing.T) {
	help := HelpText()
	for _, f := range []string{"--isbn", "--full", "--lang", "--no-cache", "--format"} {
		if !bytes.Contains([]byte(help), []byte(f)) {
			t.Fatalf("help missing flag %s", f)
		}
	}
}
