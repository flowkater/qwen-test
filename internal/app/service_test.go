package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type fakeCollector struct {
	order      *[]string
	metadataFn func() ([]byte, error)
	tocFn      func() ([]byte, error)
	reviewFn   func() ([]byte, error)
	coursesFn  func() ([]byte, error)
	similarFn  func() ([]byte, error)
}

func (f fakeCollector) CollectMetadata(context.Context, domain.BookQuery) ([]byte, error) {
	if f.order != nil {
		*f.order = append(*f.order, "metadata")
	}
	if f.metadataFn != nil {
		return f.metadataFn()
	}
	return []byte(`{"title":{"original":"Clean Code","korean":"클린 코드"},"author":"Robert C. Martin","publisher":"Addison-Wesley Professional","isbn13":"978-0132350884"}`), nil
}
func (f fakeCollector) CollectTOC(context.Context, domain.BookQuery) ([]byte, error) {
	if f.order != nil {
		*f.order = append(*f.order, "toc")
	}
	if f.tocFn != nil {
		return f.tocFn()
	}
	return []byte(`[{"title":{"original":"Meaningful Names","korean":"의미있는 이름"},"depth":1,"children":[]}]`), nil
}
func (f fakeCollector) CollectReview(context.Context, domain.BookQuery) ([]byte, error) {
	if f.order != nil {
		*f.order = append(*f.order, "review")
	}
	if f.reviewFn != nil {
		return f.reviewFn()
	}
	return []byte(`{"rating":4.8,"summary":{"pros":[],"cons":[]},"recommended_level":"중급","prerequisites":[]}`), nil
}
func (f fakeCollector) CollectCourses(context.Context, domain.BookQuery) ([]byte, error) {
	if f.order != nil {
		*f.order = append(*f.order, "courses")
	}
	if f.coursesFn != nil {
		return f.coursesFn()
	}
	return []byte(`[{"title":"course","platform":"YouTube","instructor":"x","curriculum":[],"rating":4.5,"price":"무료","url":"https://example.com"}]`), nil
}
func (f fakeCollector) CollectSimilarBooks(context.Context, domain.BookQuery) ([]byte, error) {
	if f.order != nil {
		*f.order = append(*f.order, "similar")
	}
	if f.similarFn != nil {
		return f.similarFn()
	}
	return []byte(`[
		{"title":{"original":"Refactoring","korean":"리팩터링"},"author":"Martin Fowler","brief_description":"a","difficulty_comparison":"similar"},
		{"title":{"original":"DDD","korean":"도메인 주도 설계"},"author":"Eric Evans","brief_description":"b","difficulty_comparison":"harder"},
		{"title":{"original":"Pragmatic Programmer","korean":"실용주의 프로그래머"},"author":"Andy Hunt","brief_description":"c","difficulty_comparison":"similar"}
	]`), nil
}

type fakeValidator struct{}

func (f fakeValidator) ValidateMetadataJSON(raw []byte) (domain.BookMetadata, error) {
	if string(raw) == "bad" {
		return domain.BookMetadata{}, errors.New("bad metadata")
	}
	return domain.BookMetadata{
		Title:         domain.LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
		Author:        "Robert C. Martin",
		Publisher:     "Addison-Wesley Professional",
		ISBN13:        "978-0132350884",
		SelectionNote: "popular edition",
	}, nil
}
func (f fakeValidator) ValidateTOCJSON(raw []byte) ([]domain.TOCNode, error) {
	if string(raw) == "bad" {
		return nil, errors.New("bad toc")
	}
	return []domain.TOCNode{
		{
			Title: domain.LocalizedTitle{Original: "Chapter 1", Korean: "1장"},
			Depth: 1,
			Children: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "1.1", Korean: "1.1"}, Depth: 2},
			},
		},
	}, nil
}
func (f fakeValidator) ValidateReviewJSON([]byte) (domain.ReviewInfo, error) {
	return domain.ReviewInfo{
		Rating: 4.8,
		Summary: domain.ReviewSummary{
			Pros: []string{},
			Cons: []string{},
		},
		RecommendedLevel: "중급",
		Prerequisites:    []string{},
	}, nil
}
func (f fakeValidator) ValidateCoursesJSON([]byte) ([]domain.RelatedCourse, error) {
	return []domain.RelatedCourse{
		{Title: "course", Platform: "YouTube", Curriculum: []string{}},
	}, nil
}
func (f fakeValidator) ValidateSimilarBooksJSON([]byte) ([]domain.SimilarBook, error) {
	return []domain.SimilarBook{
		{Title: domain.LocalizedTitle{Original: "A", Korean: "에이"}},
		{Title: domain.LocalizedTitle{Original: "B", Korean: "비"}},
		{Title: domain.LocalizedTitle{Original: "C", Korean: "씨"}},
	}, nil
}

type memCache struct {
	value    domain.BookInfo
	hit      bool
	getCalls int
	setCalls int
	setErr   error
}

func (m *memCache) Get(string) (domain.BookInfo, bool, error) {
	m.getCalls++
	if m.hit {
		return m.value, true, nil
	}
	return domain.BookInfo{}, false, nil
}
func (m *memCache) Set(_ string, value domain.BookInfo) error {
	m.setCalls++
	m.value = value
	return m.setErr
}

type fakeWriter struct {
	lastPath string
	paths    []string
	err      error
}

func (w *fakeWriter) Write(q domain.BookQuery, _ domain.BookInfo, _ time.Time) (string, error) {
	if w.err != nil {
		return "", w.err
	}
	if q.Output != "" {
		w.lastPath = q.Output
		w.paths = append(w.paths, q.Output)
		return q.Output, nil
	}
	w.lastPath = "/tmp/out.json"
	w.paths = append(w.paths, w.lastPath)
	return w.lastPath, nil
}

func TestProcessOneBasicFlowOrderAndCache(t *testing.T) {
	var order []string
	cache := &memCache{}
	writer := &fakeWriter{}
	svc := &Service{
		Collector: fakeCollector{order: &order},
		Validator: fakeValidator{},
		Cache:     cache,
		Writer:    writer,
		Clock: func() time.Time {
			return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		},
		Sleep: func(time.Duration) {},
	}
	res, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code"})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	gotOrder := fmt.Sprint(order)
	if gotOrder != "[metadata toc]" {
		t.Fatalf("unexpected order: %s", gotOrder)
	}
	if cache.getCalls != 1 || cache.setCalls != 1 {
		t.Fatalf("cache calls mismatch get=%d set=%d", cache.getCalls, cache.setCalls)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
}

func TestProcessOneFullModeCallsAllCollectors(t *testing.T) {
	var order []string
	svc := &Service{
		Collector: fakeCollector{order: &order},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code", FullMode: true})
	if err != nil {
		t.Fatalf("process full: %v", err)
	}
	if fmt.Sprint(order) != "[metadata toc review courses similar]" {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestProcessOneNoCacheSkipsReadStillWrites(t *testing.T) {
	cache := &memCache{hit: true}
	svc := &Service{
		Collector: fakeCollector{},
		Validator: fakeValidator{},
		Cache:     cache,
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code", NoCache: true})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if cache.getCalls != 0 {
		t.Fatalf("no-cache should skip cache read")
	}
	if cache.setCalls != 1 {
		t.Fatalf("no-cache should still write cache")
	}
}

func TestProcessOneCacheHitSkipsCollector(t *testing.T) {
	cache := &memCache{
		hit: true,
		value: domain.BookInfo{
			Book: domain.BookMetadata{Title: domain.LocalizedTitle{Original: "Hit"}},
		},
	}
	called := false
	svc := &Service{
		Collector: fakeCollector{
			metadataFn: func() ([]byte, error) { called = true; return nil, nil },
		},
		Validator: fakeValidator{},
		Cache:     cache,
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	res, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code"})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !res.FromCache {
		t.Fatalf("expected from cache")
	}
	if called {
		t.Fatalf("collector should not be called on cache hit")
	}
}

func TestProcessOneStopsOnMiddleFailure(t *testing.T) {
	var order []string
	svc := &Service{
		Collector: fakeCollector{
			order: &order,
			tocFn: func() ([]byte, error) { return nil, errors.New("toc fail") },
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if fmt.Sprint(order) != "[metadata toc]" {
		t.Fatalf("unexpected order on failure: %v", order)
	}
}

func TestProcessOneRetryOnRetryable(t *testing.T) {
	attempt := 0
	sleeps := []time.Duration{}
	svc := &Service{
		Collector: fakeCollector{
			metadataFn: func() ([]byte, error) {
				attempt++
				if attempt < 3 {
					return nil, &domain.RetryableError{Code: 429, Err: errors.New("rate")}
				}
				return []byte(`ok`), nil
			},
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(d time.Duration) { sleeps = append(sleeps, d) },
		Jitter: func(max time.Duration) time.Duration {
			return 500 * time.Millisecond
		},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code"})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if attempt != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempt)
	}
	if len(sleeps) < 2 {
		t.Fatalf("expected backoff sleeps")
	}
	if sleeps[0] != 2500*time.Millisecond || sleeps[1] != 4500*time.Millisecond {
		t.Fatalf("unexpected jittered sleeps: %v", sleeps)
	}
}

func TestProcessBatchSequentialAndAdaptiveDelay(t *testing.T) {
	delays := []time.Duration{}
	attempt := 0
	svc := &Service{
		Collector: fakeCollector{
			metadataFn: func() ([]byte, error) {
				attempt++
				if attempt == 2 {
					return nil, &domain.RetryableError{Code: 429, Err: errors.New("rate")}
				}
				return []byte(`ok`), nil
			},
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(d time.Duration) { delays = append(delays, d) },
	}

	file := t.TempDir() + "/batch.txt"
	if err := os.WriteFile(file, []byte("A\n\nB\nC\n"), 0o644); err != nil {
		t.Fatalf("write batch: %v", err)
	}
	q := domain.BookQuery{BatchPath: file}
	results, err := svc.ProcessBatch(context.Background(), q)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 valid batch items, got %d", len(results))
	}
	if len(delays) < 2 {
		t.Fatalf("expected at least 2 sleeps, got %d", len(delays))
	}
	if delays[0] != 3*time.Second {
		t.Fatalf("first inter-item delay must start at 3s, got %v", delays[0])
	}
}

func TestProcessBatchOutputPathsArePerItemWhenOutputDirGiven(t *testing.T) {
	outDir := t.TempDir()
	writer := &fakeWriter{}
	svc := &Service{
		Collector: fakeCollector{},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    writer,
		Sleep:     func(time.Duration) {},
		Clock: func() time.Time {
			return time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
		},
	}
	file := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(file, []byte("Clean Code\nRefactoring\nDDD\n"), 0o644); err != nil {
		t.Fatalf("write batch: %v", err)
	}
	results, err := svc.ProcessBatch(context.Background(), domain.BookQuery{
		BatchPath: file,
		Output:    outDir,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if len(writer.paths) != 3 {
		t.Fatalf("expected 3 output writes, got %d", len(writer.paths))
	}
	uniq := map[string]struct{}{}
	for _, p := range writer.paths {
		uniq[p] = struct{}{}
		if filepath.Dir(p) != outDir {
			t.Fatalf("path must be under output dir: %s", p)
		}
	}
	if len(uniq) != 3 {
		t.Fatalf("batch outputs must be unique per item: %v", writer.paths)
	}
}

func TestProcessBatchDuplicateTitlesStillUseUniquePaths(t *testing.T) {
	outDir := t.TempDir()
	writer := &fakeWriter{}
	svc := &Service{
		Collector: fakeCollector{},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    writer,
		Sleep:     func(time.Duration) {},
		Clock: func() time.Time {
			return time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
		},
	}
	file := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(file, []byte("Clean Code\nClean Code\nClean/Code\n"), 0o644); err != nil {
		t.Fatalf("write batch: %v", err)
	}
	_, err := svc.ProcessBatch(context.Background(), domain.BookQuery{
		BatchPath: file,
		Output:    outDir,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(writer.paths) != 3 {
		t.Fatalf("expected 3 output writes")
	}
	if writer.paths[0] == writer.paths[1] || writer.paths[1] == writer.paths[2] || writer.paths[0] == writer.paths[2] {
		t.Fatalf("duplicate/sanitized-equivalent titles must not overwrite: %v", writer.paths)
	}
}
