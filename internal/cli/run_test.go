package cli

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/flowkater/qwen/bookinfo/internal/app"
	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type fakeService struct {
	processOneFn   func(context.Context, domain.BookQuery) (app.ProcessResult, error)
	processBatchFn func(context.Context, domain.BookQuery) ([]domain.BatchItemResult, error)
}

func (f fakeService) ProcessOne(ctx context.Context, q domain.BookQuery) (app.ProcessResult, error) {
	if f.processOneFn != nil {
		return f.processOneFn(ctx, q)
	}
	return app.ProcessResult{}, nil
}

func (f fakeService) ProcessBatch(ctx context.Context, q domain.BookQuery) ([]domain.BatchItemResult, error) {
	if f.processBatchFn != nil {
		return f.processBatchFn(ctx, q)
	}
	return nil, nil
}

func TestRunnerSingleSummaryAndErrors(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &Runner{
		Service: fakeService{
			processOneFn: func(_ context.Context, _ domain.BookQuery) (app.ProcessResult, error) {
				return app.ProcessResult{
					Info: domain.BookInfo{
						Book: domain.BookMetadata{
							Title:         domain.LocalizedTitle{Original: "Clean Code"},
							Author:        "Robert C. Martin",
							SelectionNote: "duplicate hint",
						},
						TableOfContents: []domain.TOCNode{{Title: domain.LocalizedTitle{Original: "Meaningful Names"}}},
						Review:          domain.ReviewInfo{Rating: 4.9},
					},
					OutputPath: "/tmp/clean.json",
				}, nil
			},
		},
		Env:    map[string]string{"OPENROUTER_API_KEY": "x"},
		Stdout: stdout,
		Stderr: stderr,
	}
	code := r.Run(context.Background(), []string{"Clean Code", "--full"})
	if code != 0 {
		t.Fatalf("expected success")
	}
	text := stdout.String()
	for _, token := range []string{"Title:", "Author:", "TOC(1st):", "Rating:", "Saved:", "동명 도서"} {
		if !bytes.Contains([]byte(text), []byte(token)) {
			t.Fatalf("missing summary token %q in %s", token, text)
		}
	}
}

func TestRunnerBatchOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &Runner{
		Service: fakeService{
			processBatchFn: func(_ context.Context, _ domain.BookQuery) ([]domain.BatchItemResult, error) {
				return []domain.BatchItemResult{
					{Query: "A", Success: true, OutputPath: "/tmp/a.json"},
					{Query: "B", Success: false, Error: "boom"},
				}, nil
			},
		},
		Env:    map[string]string{"OPENROUTER_API_KEY": "x"},
		Stdout: stdout,
		Stderr: stderr,
	}
	code := r.Run(context.Background(), []string{"--batch", "books.txt"})
	if code != 1 {
		t.Fatalf("batch with failures should return 1, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Batch summary: success=1 failed=1")) {
		t.Fatalf("missing batch summary: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunnerMissingAPIKey(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &Runner{
		Service: fakeService{
			processOneFn: func(_ context.Context, _ domain.BookQuery) (app.ProcessResult, error) {
				return app.ProcessResult{}, fmt.Errorf("should not be called")
			},
		},
		Env:    map[string]string{},
		Stdout: stdout,
		Stderr: stderr,
	}
	code := r.Run(context.Background(), []string{"Clean Code"})
	if code != 1 {
		t.Fatalf("expected missing api key error")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("OPENROUTER_API_KEY")) {
		t.Fatalf("expected api key guidance message, got: %s", stderr.String())
	}
}

func TestRunnerHelpExitZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &Runner{
		Service: fakeService{},
		Env:     map[string]string{},
		Stdout:  stdout,
		Stderr:  stderr,
	}
	code := r.Run(context.Background(), []string{"--help"})
	if code != 0 {
		t.Fatalf("help should exit 0, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage:")) {
		t.Fatalf("help text should be printed")
	}
}

func TestRunnerBookNotFoundMapping(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &Runner{
		Service: fakeService{
			processOneFn: func(_ context.Context, _ domain.BookQuery) (app.ProcessResult, error) {
				return app.ProcessResult{}, domain.ErrBookNotFound
			},
		},
		Env:    map[string]string{"OPENROUTER_API_KEY": "x"},
		Stdout: stdout,
		Stderr: stderr,
	}
	code := r.Run(context.Background(), []string{"Clean Code"})
	if code != 1 {
		t.Fatalf("expected error exit")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("책을 찾지 못했습니다")) {
		t.Fatalf("expected book not found guidance, got: %s", stderr.String())
	}
}
