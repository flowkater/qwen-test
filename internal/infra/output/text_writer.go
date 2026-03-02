package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type TextWriter struct{}

func NewTextWriter() *TextWriter { return &TextWriter{} }

func (w *TextWriter) Write(q domain.BookQuery, data domain.BookInfo, now time.Time) (string, error) {
	localDate := now.Local().Format("20060102")
	path := domain.ResolveOutputPath(q, localDate)
	if err := domain.EnsureParentDir(path); err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Title: %s", data.Book.Title.Original))
	if strings.TrimSpace(data.Book.Title.Korean) != "" {
		b.WriteString(fmt.Sprintf(" (%s)", data.Book.Title.Korean))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Author: %s\n", data.Book.Author))
	if strings.TrimSpace(data.Book.Publisher) != "" {
		b.WriteString(fmt.Sprintf("Publisher: %s\n", data.Book.Publisher))
	}
	if strings.TrimSpace(data.Book.ISBN13) != "" {
		b.WriteString(fmt.Sprintf("ISBN13: %s\n", data.Book.ISBN13))
	}
	b.WriteString("\n[Table of Contents]\n")
	if len(data.TableOfContents) == 0 {
		b.WriteString("- (none)\n")
	} else {
		for _, n := range data.TableOfContents {
			writeTOCNode(&b, n, 0)
		}
	}

	if data.DataAvailability.Review {
		b.WriteString("\n[Review]\n")
		b.WriteString(fmt.Sprintf("Rating: %.1f\n", data.Review.Rating))
		if len(data.Review.Summary.Pros) > 0 {
			b.WriteString("Pros:\n")
			for _, p := range data.Review.Summary.Pros {
				b.WriteString("- " + p + "\n")
			}
		}
		if len(data.Review.Summary.Cons) > 0 {
			b.WriteString("Cons:\n")
			for _, c := range data.Review.Summary.Cons {
				b.WriteString("- " + c + "\n")
			}
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	return abs, nil
}

func writeTOCNode(b *strings.Builder, node domain.TOCNode, depth int) {
	indent := strings.Repeat("  ", depth)
	title := node.Title.Original
	if title == "" {
		title = node.Title.Korean
	}
	b.WriteString(fmt.Sprintf("%s- %s\n", indent, title))
	for _, child := range node.Children {
		writeTOCNode(b, child, depth+1)
	}
}
