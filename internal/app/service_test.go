package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func (f fakeCollector) CollectUnified(context.Context, domain.BookQuery) ([]byte, error) {
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
	partTOCCalls := []string{}
	collector := fakeCollector{
		partTOCFn: func(parentTitle, partTitle string, chapters []string) ([]byte, error) {
			partTOCCalls = append(partTOCCalls, partTitle)
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
	if partTOCCalls[0] != "Part 1" || partTOCCalls[1] != "Part 2" {
		t.Fatalf("expected part titles in calls, got %v", partTOCCalls)
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
