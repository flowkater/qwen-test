package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/app"
	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type Service interface {
	ProcessOne(ctx context.Context, q domain.BookQuery) (app.ProcessResult, error)
	ProcessBatch(ctx context.Context, q domain.BookQuery) ([]domain.BatchItemResult, error)
}

type Runner struct {
	Service Service
	Now     func() time.Time
	Env     map[string]string
	Stdout  io.Writer
	Stderr  io.Writer
}

func (r *Runner) defaults() {
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}
	if r.Stderr == nil {
		r.Stderr = os.Stderr
	}
	if r.Env == nil {
		r.Env = map[string]string{}
		for _, kv := range os.Environ() {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				r.Env[parts[0]] = parts[1]
			}
		}
	}
}

func (r *Runner) Run(ctx context.Context, args []string) int {
	r.defaults()
	query, err := ParseArgs(args, r.Stderr)
	if err != nil {
		if errors.Is(err, ErrHelpRequested) {
			return 0
		}
		fmt.Fprintf(r.Stderr, "error: %v\n%s\n", err, HelpText())
		return 1
	}
	key, source, err := domain.ResolveAPIKey(query.APIKey, r.Env)
	if err != nil {
		fmt.Fprintf(r.Stderr, "error: %v\n", err)
		return 1
	}
	query.APIKey = key
	_ = source

	if query.BatchPath != "" {
		return r.runBatch(ctx, query)
	}
	return r.runSingle(ctx, query)
}

func (r *Runner) runSingle(ctx context.Context, query domain.BookQuery) int {
	res, err := r.Service.ProcessOne(ctx, query)
	r.printSummary(res.Info, res.OutputPath, query.FullMode)
	if err != nil {
		fmt.Fprintf(r.Stderr, "error: %v\n", mapError(err))
		return 1
	}
	return 0
}

func (r *Runner) runBatch(ctx context.Context, query domain.BookQuery) int {
	results, err := r.Service.ProcessBatch(ctx, query)
	if err != nil {
		fmt.Fprintf(r.Stderr, "error: %v\n", mapError(err))
		return 1
	}
	success, failed := 0, 0
	for idx, item := range results {
		fmt.Fprintf(r.Stdout, "[%d/%d] %q 처리 중...\n", idx+1, len(results), item.Query)
		if item.Success {
			success++
			fmt.Fprintf(r.Stdout, "  ✅ success: %s\n", item.OutputPath)
		} else {
			failed++
			fmt.Fprintf(r.Stdout, "  ❌ failed: %s\n", item.Error)
		}
	}
	fmt.Fprintf(r.Stdout, "Batch summary: success=%d failed=%d\n", success, failed)
	if failed > 0 {
		return 1
	}
	return 0
}

func (r *Runner) printSummary(info domain.BookInfo, outputPath string, full bool) {
	fmt.Fprintf(r.Stdout, "Title: %s\n", info.Book.Title.Original)
	fmt.Fprintf(r.Stdout, "Author: %s\n", info.Book.Author)
	if len(info.TableOfContents) > 0 {
		fmt.Fprintf(r.Stdout, "TOC(1st): %s\n", info.TableOfContents[0].Title.Original)
	}
	if full {
		fmt.Fprintf(r.Stdout, "Rating: %.1f\n", info.Review.Rating)
	}
	if strings.TrimSpace(outputPath) != "" {
		fmt.Fprintf(r.Stdout, "Saved: %s\n", outputPath)
	}
	if strings.TrimSpace(info.Book.SelectionNote) != "" {
		fmt.Fprintf(r.Stdout, "동명 도서가 있을 수 있습니다. --author로 특정하세요\n")
	}
}

func mapError(err error) error {
	switch {
	case errorsIs(err, domain.ErrMissingAPIKey):
		return fmt.Errorf("API 키가 없습니다. OPENROUTER_API_KEY 또는 --api-key를 설정하세요")
	case errorsIs(err, domain.ErrMissingQuery):
		return fmt.Errorf("제목 또는 --isbn 중 하나는 필수입니다")
	case errorsIs(err, domain.ErrInvalidLang):
		return fmt.Errorf("허용값: ko, en, ja, zh-tw")
	case errorsIs(err, domain.ErrBookNotFound):
		return fmt.Errorf("책을 찾지 못했습니다. 제목/ISBN을 확인하거나 --author를 추가해 주세요")
	default:
		return err
	}
}

func errorsIs(err, target error) bool {
	if err == nil || target == nil {
		return false
	}
	return strings.Contains(err.Error(), target.Error())
}
