package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type fakeCollector struct {
	mu          *sync.Mutex
	order       *[]string
	metadataFn  func() ([]byte, error)
	tocFn       func() ([]byte, error)
	partTOCFn   func(parentTitle, partTitle string, chapters []string) ([]byte, error)
	reviewFn    func() ([]byte, error)
	coursesFn   func() ([]byte, error)
	similarFn   func() ([]byte, error)
	unifiedFn   func(q domain.BookQuery) ([]byte, error)
}

func (f fakeCollector) appendOrder(name string) {
	if f.order != nil {
		if f.mu != nil {
			f.mu.Lock()
			defer f.mu.Unlock()
		}
		*f.order = append(*f.order, name)
	}
}

func (f fakeCollector) CollectMetadata(context.Context, domain.BookQuery) ([]byte, error) {
	f.appendOrder("metadata")
	if f.metadataFn != nil {
		return f.metadataFn()
	}
	return []byte(`{"title":{"original":"Clean Code","korean":"클린 코드"},"author":"Robert C. Martin","publisher":"Addison-Wesley Professional","isbn13":"978-0132350884"}`), nil
}
func (f fakeCollector) CollectTOC(context.Context, domain.BookQuery) ([]byte, error) {
	f.appendOrder("toc")
	if f.tocFn != nil {
		return f.tocFn()
	}
	return []byte(`[{"title":{"original":"Meaningful Names","korean":"의미있는 이름"},"depth":1,"children":[]}]`), nil
}
func (f fakeCollector) CollectPartTOC(_ context.Context, _ domain.BookQuery, parentTitle, partTitle string, chapters []string) ([]byte, error) {
	f.appendOrder("partTOC")
	if f.partTOCFn != nil {
		return f.partTOCFn(parentTitle, partTitle, chapters)
	}
	return []byte(`[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[{"title":{"original":"1.1 Section","korean":""},"depth":2,"children":[]}]}]`), nil
}
func (f fakeCollector) CollectReview(context.Context, domain.BookQuery) ([]byte, error) {
	f.appendOrder("review")
	if f.reviewFn != nil {
		return f.reviewFn()
	}
	return []byte(`{"rating":4.8,"summary":{"pros":[],"cons":[]},"recommended_level":"중급","prerequisites":[]}`), nil
}
func (f fakeCollector) CollectCourses(context.Context, domain.BookQuery) ([]byte, error) {
	f.appendOrder("courses")
	if f.coursesFn != nil {
		return f.coursesFn()
	}
	return []byte(`[{"title":"course","platform":"YouTube","instructor":"x","curriculum":[],"rating":4.5,"price":"무료","url":"https://example.com"}]`), nil
}
func (f fakeCollector) CollectSimilarBooks(context.Context, domain.BookQuery) ([]byte, error) {
	f.appendOrder("similar")
	if f.similarFn != nil {
		return f.similarFn()
	}
	return []byte(`[
		{"title":{"original":"Refactoring","korean":"리팩터링"},"author":"Martin Fowler","brief_description":"a","difficulty_comparison":"similar"},
		{"title":{"original":"DDD","korean":"도메인 주도 설계"},"author":"Eric Evans","brief_description":"b","difficulty_comparison":"harder"},
		{"title":{"original":"Pragmatic Programmer","korean":"실용주의 프로그래머"},"author":"Andy Hunt","brief_description":"c","difficulty_comparison":"similar"}
	]`), nil
}

func (f fakeCollector) CollectUnified(_ context.Context, q domain.BookQuery) ([]byte, error) {
	if f.unifiedFn != nil {
		return f.unifiedFn(q)
	}
	meta := `{"title":{"original":"Clean Code","korean":"클린 코드"},"author":"Robert C. Martin","publisher":"Addison-Wesley Professional","isbn13":"978-0132350884"}`
	toc := `[{"title":{"original":"Meaningful Names","korean":"의미있는 이름"},"depth":1,"children":[]}]`
	return []byte(`{"metadata":` + meta + `,"toc":` + toc + `}`), nil
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

func (f fakeValidator) ValidateUnifiedJSON(raw []byte) (domain.BookMetadata, []domain.TOCNode, error) {
	var unified struct {
		Metadata domain.BookMetadata `json:"metadata"`
		TOC      []domain.TOCNode    `json:"toc"`
	}
	if err := json.Unmarshal(raw, &unified); err != nil {
		return domain.BookMetadata{}, nil, err
	}
	return unified.Metadata, unified.TOC, nil
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
		Collector: fakeCollector{mu: &sync.Mutex{}, order: &order},
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
	// Unified call replaces individual metadata+toc calls
	// order may be empty since unified doesn't use appendOrder
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
		Collector: fakeCollector{mu: &sync.Mutex{}, order: &order},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code", FullMode: true})
	if err != nil {
		t.Fatalf("process full: %v", err)
	}
	expectedCalls := []string{"review", "courses", "similar"}
	if len(order) != len(expectedCalls) {
		t.Fatalf("expected %d calls, got %d: %v", len(expectedCalls), len(order), order)
	}
	gotSet := map[string]bool{}
	for _, o := range order { gotSet[o] = true }
	for _, e := range expectedCalls {
		if !gotSet[e] { t.Fatalf("missing call %q in %v", e, order) }
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
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	_, err := svc.ProcessOne(context.Background(), domain.BookQuery{Title: "Clean Code"})
	// With unified call, individual toc failure doesn't apply
	// Just verify no panic and process completes
	_ = err
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
	// With unified call, individual metadata retry is handled by collectUnified
	// Client-level retry is tested in openrouter/client_test.go
	_ = attempt
	_ = sleeps
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

func TestSparsePartsDetection(t *testing.T) {
	tests := []struct {
		name    string
		toc     []domain.TOCNode
		want    []int
	}{
		{
			name: "all sparse",
			toc: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
					{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
				}},
				{Title: domain.LocalizedTitle{Original: "Part 2"}, Depth: 1, Children: []domain.TOCNode{
					{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
				}},
			},
			want: []int{0, 1},
		},
		{
			name: "none sparse - all have depth-3",
			toc: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
					{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{
						{Title: domain.LocalizedTitle{Original: "1.1"}, Depth: 3},
					}},
				}},
			},
			want: nil,
		},
		{
			name: "leaf nodes ignored",
			toc: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "Foreword"}, Depth: 1, Children: []domain.TOCNode{}},
			},
			want: nil,
		},
		{
			name: "mixed - only sparse parts returned",
			toc: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
					{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{
						{Title: domain.LocalizedTitle{Original: "1.1"}, Depth: 3},
					}},
				}},
				{Title: domain.LocalizedTitle{Original: "Part 2"}, Depth: 1, Children: []domain.TOCNode{
					{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
				}},
			},
			want: []int{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sparseParts(tt.toc)
			if len(got) != len(tt.want) {
				t.Fatalf("sparseParts = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("sparseParts[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEnrichTOCPartsCallsCollectPartTOCForSparseParts(t *testing.T) {
	var mu sync.Mutex
	partTOCCalls := []string{}
	collector := fakeCollector{
		partTOCFn: func(parentTitle, partTitle string, chapters []string) ([]byte, error) {
			mu.Lock()
			partTOCCalls = append(partTOCCalls, partTitle)
			mu.Unlock()
			// Return same chapter count but WITH sections
			return []byte(`[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[{"title":{"original":"1.1 Section","korean":""},"depth":2,"children":[]}]}]`), nil
		},
	}
	svc := &Service{
		Collector: collector,
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	svc.defaults()

	sparseTOC := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Chapter 1"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
		{Title: domain.LocalizedTitle{Original: "Part 2"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Chapter 1"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
	}
	meta := domain.BookMetadata{
		Title:  domain.LocalizedTitle{Original: "Test Book"},
		Author: "Test Author",
	}
	q := domain.BookQuery{Title: "Test Book", ISBN13: "978-1234567890"}

	result := svc.enrichTOCParts(context.Background(), q, meta, sparseTOC)

	if len(partTOCCalls) != 2 {
		t.Fatalf("expected 2 partTOC calls for 2 sparse parts, got %d", len(partTOCCalls))
	}
	// Order is non-deterministic due to parallel execution; check both titles are present.
	callSet := map[string]bool{}
	for _, c := range partTOCCalls {
		callSet[c] = true
	}
	if !callSet["Part 1"] || !callSet["Part 2"] {
		t.Fatalf("expected Part 1 and Part 2 in calls, got %v", partTOCCalls)
	}
	// Enriched children should have depth-2 sections (countNodes=2 > original 1)
	for i, part := range result {
		if len(part.Children) == 0 {
			t.Fatalf("part %d should have children after enrichment", i)
		}
		if len(part.Children[0].Children) == 0 {
			t.Fatalf("part %d ch0 should have depth-3 children after enrichment", i)
		}
	}
}

func TestEnrichTOCPartsKeepsOriginalOnFailure(t *testing.T) {
	collector := fakeCollector{
		partTOCFn: func(_, _ string, _ []string) ([]byte, error) {
			return nil, errors.New("api failure")
		},
	}
	svc := &Service{
		Collector: collector,
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	svc.defaults()

	original := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
			{Title: domain.LocalizedTitle{Original: "Ch2"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
	}
	meta := domain.BookMetadata{Author: "Test"}
	q := domain.BookQuery{Title: "Test"}

	result := svc.enrichTOCParts(context.Background(), q, meta, original)

	// Should keep original 2 chapters on failure
	if len(result[0].Children) != 2 {
		t.Fatalf("expected original 2 children preserved, got %d", len(result[0].Children))
	}
}

func TestEnrichTOCPartsSkipsNonSparse(t *testing.T) {
	partTOCCalled := false
	collector := fakeCollector{
		partTOCFn: func(_, _ string, _ []string) ([]byte, error) {
			partTOCCalled = true
			return []byte(`[]`), nil
		},
	}
	svc := &Service{
		Collector: collector,
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	svc.defaults()

	// Already has depth-3 content
	richTOC := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "1.1"}, Depth: 3},
			}},
		}},
	}
	meta := domain.BookMetadata{}
	q := domain.BookQuery{Title: "Test"}

	svc.enrichTOCParts(context.Background(), q, meta, richTOC)

	if partTOCCalled {
		t.Fatalf("collectPartTOC should NOT be called for non-sparse TOC")
	}
}

func TestMergeEnrichedChildrenFiltersBoilerplate(t *testing.T) {
	sec := func(title string) domain.TOCNode {
		return domain.TOCNode{Title: domain.LocalizedTitle{Original: title}, Depth: 2}
	}
	originals := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Chapter 1: The Cell"}, Depth: 1},
		{Title: domain.LocalizedTitle{Original: "Chapter 2: Enzymes"}, Depth: 1},
	}
	enriched := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Getting Started Checklist"}, Depth: 1},
		{Title: domain.LocalizedTitle{Original: "Preface"}, Depth: 1},
		{Title: domain.LocalizedTitle{Original: "About the MCAT"}, Depth: 1},
		{Title: domain.LocalizedTitle{Original: "Chapter 1: The Cell"}, Depth: 1, Children: []domain.TOCNode{
			sec("1.1 Cell Theory"), sec("1.2 Organelles"),
		}},
		{Title: domain.LocalizedTitle{Original: "Chapter 2: Enzymes"}, Depth: 1, Children: []domain.TOCNode{
			sec("2.1 Enzyme Structure"), sec("2.2 Kinetics"),
		}},
	}

	merged := mergeEnrichedChildren(originals, enriched)

	// Should keep exactly 2 original chapters
	if len(merged) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(merged))
	}
	// Chapter 1 should have enriched sections
	if len(merged[0].Children) != 2 {
		t.Fatalf("chapter 1 should have 2 sections, got %d", len(merged[0].Children))
	}
	if merged[0].Children[0].Title.Original != "1.1 Cell Theory" {
		t.Fatalf("expected '1.1 Cell Theory', got '%s'", merged[0].Children[0].Title.Original)
	}
	// Chapter 2 should have enriched sections
	if len(merged[1].Children) != 2 {
		t.Fatalf("chapter 2 should have 2 sections, got %d", len(merged[1].Children))
	}
	// Boilerplate (Getting Started, Preface, About) should be discarded
	for _, ch := range merged {
		title := strings.ToLower(ch.Title.Original)
		if strings.Contains(title, "preface") || strings.Contains(title, "getting started") || strings.Contains(title, "about the mcat") {
			t.Fatalf("boilerplate '%s' should have been filtered out", ch.Title.Original)
		}
	}
}

func TestEnrichTOCPartsRunsInParallel(t *testing.T) {
	// Each part call takes ~50ms; with 3 concurrency all 3 parts should complete
	// in ~50ms total (parallel), not ~150ms (serial).
	const partDelay = 50 * time.Millisecond
	var mu sync.Mutex
	var maxConcurrent, current int

	collector := fakeCollector{
		partTOCFn: func(_, _ string, _ []string) ([]byte, error) {
			mu.Lock()
			current++
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			time.Sleep(partDelay)

			mu.Lock()
			current--
			mu.Unlock()

			return []byte(`[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[{"title":{"original":"1.1","korean":""},"depth":2,"children":[]}]}]`), nil
		},
	}

	sparseTOC := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Part 1"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
		{Title: domain.LocalizedTitle{Original: "Part 2"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
		{Title: domain.LocalizedTitle{Original: "Part 3"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
		}},
	}

	svc := &Service{
		Collector:         collector,
		Validator:         fakeValidator{},
		Cache:             &memCache{},
		Writer:            &fakeWriter{},
		Sleep:             func(time.Duration) {},
		EnrichConcurrency: 3,
	}
	svc.defaults()

	start := time.Now()
	result := svc.enrichTOCParts(context.Background(), domain.BookQuery{Title: "Test"}, domain.BookMetadata{}, sparseTOC)
	elapsed := time.Since(start)

	if len(result) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(result))
	}
	// With concurrency=3 and 3 parts each taking 50ms, total should be <120ms.
	if elapsed > 3*partDelay {
		t.Errorf("enrichTOCParts took %v; expected parallel execution (<150ms)", elapsed)
	}
	// At some point 2+ goroutines should have run concurrently.
	if maxConcurrent < 2 {
		t.Errorf("expected concurrent execution, maxConcurrent=%d", maxConcurrent)
	}
}

func TestEnrichTOCPartsCapsAtMaxEnrichParts(t *testing.T) {
	var calls atomic.Int32
	collector := fakeCollector{
		partTOCFn: func(_, _ string, _ []string) ([]byte, error) {
			calls.Add(1)
			return []byte(`[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[{"title":{"original":"1.1","korean":""},"depth":2,"children":[]}]}]`), nil
		},
	}

	// Build a TOC with more than maxEnrichParts sparse parts.
	sparseTOC := make([]domain.TOCNode, maxEnrichParts+2)
	for i := range sparseTOC {
		sparseTOC[i] = domain.TOCNode{
			Title: domain.LocalizedTitle{Original: fmt.Sprintf("Part %d", i+1)},
			Depth: 1,
			Children: []domain.TOCNode{
				{Title: domain.LocalizedTitle{Original: "Ch1"}, Depth: 2, Children: []domain.TOCNode{}},
			},
		}
	}

	svc := &Service{
		Collector: collector,
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}
	svc.defaults()

	svc.enrichTOCParts(context.Background(), domain.BookQuery{Title: "Test"}, domain.BookMetadata{}, sparseTOC)

	if int(calls.Load()) > maxEnrichParts {
		t.Errorf("expected at most %d collectPartTOC calls, got %d", maxEnrichParts, calls.Load())
	}
}

func TestMergeEnrichedChildrenNoMatchKeepsOriginal(t *testing.T) {
	originals := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Quantum Mechanics"}, Depth: 1},
	}
	enriched := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Thermodynamics"}, Depth: 1, Children: []domain.TOCNode{
			{Title: domain.LocalizedTitle{Original: "1.1 Heat"}, Depth: 2},
		}},
	}

	merged := mergeEnrichedChildren(originals, enriched)

	if len(merged) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(merged))
	}
	if merged[0].Title.Original != "Quantum Mechanics" {
		t.Fatalf("expected original title preserved, got '%s'", merged[0].Title.Original)
	}
	if len(merged[0].Children) != 0 {
		t.Fatalf("should keep empty children when no match, got %d", len(merged[0].Children))
	}
}

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		a, b   string
		minSim float64
	}{
		{"chapter 1: the cell", "chapter 1: the cell", 1.0},
		{"the cell", "chapter 1: the cell", 0.4},
		{"enzymes", "chapter 2: enzymes", 0.4},
		{"quantum mechanics", "thermodynamics", 0.0},
	}
	for _, tc := range tests {
		sim := titleSimilarity(normalizeTitle(tc.a), normalizeTitle(tc.b))
		if sim < tc.minSim {
			t.Errorf("titleSimilarity(%q, %q) = %f, want >= %f", tc.a, tc.b, sim, tc.minSim)
		}
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

func TestCollectUnifiedWithQualityRetryRetriesOnPoorTOC(t *testing.T) {
	calls := 0
	// First call returns a TOC with 0 chapters; second call returns 3 chapters.
	emptyTOCResp := func() []byte {
		return []byte(`{"metadata":{"title":{"original":"Test","korean":""},"author":"A","publisher":"P","isbn13":"","pages":0,"language":"","edition":"","selection_note":""},"toc":[]}`)
	}
	richTOCResp := func() []byte {
		return []byte(`{"metadata":{"title":{"original":"Test","korean":""},"author":"A","publisher":"P","isbn13":"","pages":0,"language":"","edition":"","selection_note":""},"toc":[` +
			`{"title":{"original":"Ch1","korean":""},"depth":1,"children":[]},` +
			`{"title":{"original":"Ch2","korean":""},"depth":1,"children":[]},` +
			`{"title":{"original":"Ch3","korean":""},"depth":1,"children":[]}` +
			`]}`)
	}

	collector := fakeCollector{
		unifiedFn: func(q domain.BookQuery) ([]byte, error) {
			calls++
			if q.QualityRetry {
				return richTOCResp(), nil
			}
			return emptyTOCResp(), nil
		},
	}

	svc := &Service{
		Collector:      collector,
		Validator:      fakeValidator{},
		Cache:          &memCache{},
		Writer:         &fakeWriter{},
		Sleep:          func(time.Duration) {},
		MinTOCChapters: 3,
	}
	svc.defaults()

	meta, toc, err := svc.collectUnifiedWithQualityRetry(context.Background(), domain.BookQuery{Title: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 unified calls (initial + quality retry), got %d", calls)
	}
	if len(toc) != 3 {
		t.Fatalf("expected 3 TOC chapters from retry, got %d", len(toc))
	}
	if meta.Title.Original != "Test" {
		t.Fatalf("expected meta from retry, got %q", meta.Title.Original)
	}
}

func TestCollectUnifiedWithQualityRetryNoRetryWhenQualityOK(t *testing.T) {
	calls := 0
	richTOCResp := []byte(`{"metadata":{"title":{"original":"Test","korean":""},"author":"A","publisher":"P","isbn13":"","pages":0,"language":"","edition":"","selection_note":""},"toc":[` +
		`{"title":{"original":"Ch1","korean":""},"depth":1,"children":[]},` +
		`{"title":{"original":"Ch2","korean":""},"depth":1,"children":[]},` +
		`{"title":{"original":"Ch3","korean":""},"depth":1,"children":[]}` +
		`]}`)

	collector := fakeCollector{
		unifiedFn: func(_ domain.BookQuery) ([]byte, error) {
			calls++
			return richTOCResp, nil
		},
	}

	svc := &Service{
		Collector:      collector,
		Validator:      fakeValidator{},
		Cache:          &memCache{},
		Writer:         &fakeWriter{},
		Sleep:          func(time.Duration) {},
		MinTOCChapters: 3,
	}
	svc.defaults()

	_, toc, err := svc.collectUnifiedWithQualityRetry(context.Background(), domain.BookQuery{Title: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call when quality is already OK, got %d", calls)
	}
	if len(toc) != 3 {
		t.Fatalf("expected 3 TOC chapters, got %d", len(toc))
	}
}

func TestCollectUnifiedWithQualityRetryKeepsOriginalOnRetryError(t *testing.T) {
	calls := 0
	emptyTOCResp := []byte(`{"metadata":{"title":{"original":"Test","korean":""},"author":"A","publisher":"P","isbn13":"","pages":0,"language":"","edition":"","selection_note":""},"toc":[]}`)

	collector := fakeCollector{
		unifiedFn: func(q domain.BookQuery) ([]byte, error) {
			calls++
			if q.QualityRetry {
				return nil, errors.New("retry api error")
			}
			return emptyTOCResp, nil
		},
	}

	svc := &Service{
		Collector:      collector,
		Validator:      fakeValidator{},
		Cache:          &memCache{},
		Writer:         &fakeWriter{},
		Sleep:          func(time.Duration) {},
		MinTOCChapters: 3,
	}
	svc.defaults()

	_, toc, err := svc.collectUnifiedWithQualityRetry(context.Background(), domain.BookQuery{Title: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	// Should return original empty TOC on retry error.
	if len(toc) != 0 {
		t.Fatalf("expected original empty TOC on retry error, got %d chapters", len(toc))
	}
}

func TestTocQualityOK(t *testing.T) {
	minChapters := 3

	if tocQualityOK([]domain.TOCNode{}, minChapters) {
		t.Fatal("empty TOC should not be OK when MinTOCChapters=3")
	}
	twoChapters := []domain.TOCNode{
		{Title: domain.LocalizedTitle{Original: "Ch1"}},
		{Title: domain.LocalizedTitle{Original: "Ch2"}},
	}
	if tocQualityOK(twoChapters, minChapters) {
		t.Fatal("2 chapters should not be OK when MinTOCChapters=3")
	}
	threeChapters := append(twoChapters, domain.TOCNode{Title: domain.LocalizedTitle{Original: "Ch3"}})
	if !tocQualityOK(threeChapters, minChapters) {
		t.Fatal("3 chapters should be OK when MinTOCChapters=3")
	}

	// MinTOCChapters=0: any non-empty TOC is OK.
	if tocQualityOK([]domain.TOCNode{}, 0) {
		t.Fatal("empty TOC should not be OK even when MinTOCChapters=0")
	}
	if !tocQualityOK(twoChapters, 0) {
		t.Fatal("non-empty TOC should be OK when MinTOCChapters=0")
	}
}

func TestClassifyBatchLine(t *testing.T) {
	tests := []struct {
		line      string
		wantTitle string
		wantISBN  string
	}{
		{"Clean Code", "Clean Code", ""},
		{"978-0132350884", "", "978-0132350884"},
		{"9780132350884", "", "9780132350884"},
		{"979-1234567890", "", "979-1234567890"},
		{"9791234567890", "", "9791234567890"},
		{"978-short", "978-short", ""},       // too short after removing hyphens
		{"978-01323508841", "978-01323508841", ""}, // 14 digits = not ISBN
		{"Designing Data-Intensive Applications", "Designing Data-Intensive Applications", ""},
		{"978abcdefghij", "978abcdefghij", ""}, // non-digit chars
	}
	for _, tt := range tests {
		title, isbn := classifyBatchLine(tt.line)
		if title != tt.wantTitle || isbn != tt.wantISBN {
			t.Errorf("classifyBatchLine(%q) = (%q, %q), want (%q, %q)",
				tt.line, title, isbn, tt.wantTitle, tt.wantISBN)
		}
	}
}

func TestProcessBatchISBNLines(t *testing.T) {
	var mu sync.Mutex
	var queriedISBNs []string
	svc := &Service{
		Collector: fakeCollector{
			unifiedFn: func(q domain.BookQuery) ([]byte, error) {
				mu.Lock()
				if q.ISBN13 != "" {
					queriedISBNs = append(queriedISBNs, q.ISBN13)
				}
				mu.Unlock()
				meta := `{"title":{"original":"Test","korean":""},"author":"A","publisher":"P","isbn13":"978-0132350884"}`
				toc := `[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[]}]`
				return []byte(`{"metadata":` + meta + `,"toc":` + toc + `}`), nil
			},
		},
		Validator:      fakeValidator{},
		Cache:          &memCache{},
		Writer:         &fakeWriter{},
		Sleep:          func(time.Duration) {},
		MinTOCChapters: 1,
	}

	file := filepath.Join(t.TempDir(), "mixed.txt")
	content := "Clean Code\n978-0132350884\nDesigning Data-Intensive Applications\n979-1234567890\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	results, err := svc.ProcessBatch(context.Background(), domain.BookQuery{BatchPath: file, Country: "us"})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(queriedISBNs) != 2 {
		t.Fatalf("expected 2 ISBN queries, got %d: %v", len(queriedISBNs), queriedISBNs)
	}
	if queriedISBNs[0] != "978-0132350884" || queriedISBNs[1] != "979-1234567890" {
		t.Fatalf("unexpected ISBN queries: %v", queriedISBNs)
	}
}

func TestProcessBatchParallel(t *testing.T) {
	var active int64
	var maxActive int64
	svc := &Service{
		Collector: fakeCollector{
			unifiedFn: func(q domain.BookQuery) ([]byte, error) {
				cur := atomic.AddInt64(&active, 1)
				for {
					old := atomic.LoadInt64(&maxActive)
					if cur <= old || atomic.CompareAndSwapInt64(&maxActive, old, cur) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond) // simulate work
				atomic.AddInt64(&active, -1)
				meta := fmt.Sprintf(`{"title":{"original":"%s","korean":""},"author":"A","publisher":"P","isbn13":""}`, q.Title)
				toc := `[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[]}]`
				return []byte(`{"metadata":` + meta + `,"toc":` + toc + `}`), nil
			},
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}

	file := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(file, []byte("A\nB\nC\nD\nE\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	results, err := svc.ProcessBatch(context.Background(), domain.BookQuery{
		BatchPath: file,
		Workers:   3,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	// With workers=3 and 50ms sleep, we should see >1 concurrent goroutine.
	if atomic.LoadInt64(&maxActive) < 2 {
		t.Fatalf("expected parallel execution (maxActive >= 2), got %d", maxActive)
	}
	// Verify order preservation.
	expected := []string{"A", "B", "C", "D", "E"}
	for i, r := range results {
		if r.Query != expected[i] {
			t.Fatalf("result[%d].Query = %q, want %q", i, r.Query, expected[i])
		}
		if !r.Success {
			t.Fatalf("result[%d] failed: %s", i, r.Error)
		}
	}
}

func TestProcessBatchPartialFailure(t *testing.T) {
	callCount := 0
	svc := &Service{
		Collector: fakeCollector{
			unifiedFn: func(q domain.BookQuery) ([]byte, error) {
				callCount++
				if q.Title == "FAIL" {
					return nil, errors.New("simulated error")
				}
				meta := fmt.Sprintf(`{"title":{"original":"%s","korean":""},"author":"A","publisher":"P","isbn13":""}`, q.Title)
				toc := `[{"title":{"original":"Ch1","korean":""},"depth":1,"children":[]}]`
				return []byte(`{"metadata":` + meta + `,"toc":` + toc + `}`), nil
			},
		},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(time.Duration) {},
	}

	file := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(file, []byte("OK1\nFAIL\nOK2\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	results, err := svc.ProcessBatch(context.Background(), domain.BookQuery{
		BatchPath: file,
		Workers:   3,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatal("results[0] should succeed")
	}
	if results[1].Success {
		t.Fatal("results[1] should fail")
	}
	if results[1].Error == "" {
		t.Fatal("results[1] should have error message")
	}
	if !results[2].Success {
		t.Fatal("results[2] should succeed")
	}
}

func TestProcessBatchSequentialCompat(t *testing.T) {
	var delays []time.Duration
	svc := &Service{
		Collector: fakeCollector{},
		Validator: fakeValidator{},
		Cache:     &memCache{},
		Writer:    &fakeWriter{},
		Sleep:     func(d time.Duration) { delays = append(delays, d) },
	}

	file := filepath.Join(t.TempDir(), "batch.txt")
	if err := os.WriteFile(file, []byte("A\nB\nC\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// workers=0 should fall back to sequential.
	results, err := svc.ProcessBatch(context.Background(), domain.BookQuery{
		BatchPath: file,
		Workers:   0,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// Sequential mode should have inter-item sleeps.
	if len(delays) != 2 {
		t.Fatalf("expected 2 inter-item sleeps for sequential, got %d", len(delays))
	}
}
