package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type Service struct {
	Collector Collector
	Validator Validator
	Cache     Cache
	Writer    Writer

	Backoff domain.BackoffConfig
	Clock   Clock
	Sleep   Sleeper
	Jitter  func(max time.Duration) time.Duration
}

type ProcessResult struct {
	Info       domain.BookInfo
	OutputPath string
	FromCache  bool
}

func (s *Service) defaults() {
	if s.Backoff.MaxRetries == 0 {
		s.Backoff = domain.DefaultBackoffConfig()
	}
	if s.Clock == nil {
		s.Clock = time.Now
	}
	if s.Sleep == nil {
		s.Sleep = time.Sleep
	}
	if s.Jitter == nil {
		s.Jitter = func(max time.Duration) time.Duration {
			if max <= 0 {
				return 0
			}
			// #nosec G404 - non-crypto jitter for retry backoff
			return time.Duration(rand.Int63n(int64(max) + 1))
		}
	}
}

func (s *Service) ProcessOne(ctx context.Context, q domain.BookQuery) (ProcessResult, error) {
	s.defaults()
	if err := domain.ValidateQuery(q); err != nil {
		return ProcessResult{}, err
	}
	if q.Model == "" {
		q.Model = domain.DefaultModel
	}
	key := domain.CacheKey(q)

	if s.Cache != nil && !q.NoCache {
		if hit, ok, err := s.Cache.Get(key); err == nil && ok {
			path := ""
			if s.Writer != nil {
				path, _ = s.Writer.Write(q, hit, s.Clock())
			}
			return ProcessResult{Info: hit, OutputPath: path, FromCache: true}, nil
		}
	}

	var parts domain.CollectedParts
	// Single unified call: metadata + TOC in one DashScope request
	unifiedRaw, err := s.collectUnified(ctx, q)
	if err != nil {
		return ProcessResult{}, err
	}
	meta, toc, err := s.Validator.ValidateUnifiedJSON(unifiedRaw)
	if err != nil {
		return ProcessResult{}, err
	}
	parts.Metadata = meta
	parts.TOC = toc

	// 2nd pass: enrich sparse TOC parts (multi-volume sets)
	parts.TOC = s.enrichTOCParts(ctx, q, meta, parts.TOC)

	if q.FullMode {
		review, err := s.collectReview(ctx, q)
		if err != nil {
			return ProcessResult{}, err
		}
		parts.Review = &review

		if parts.RelatedCourse, err = s.collectCourses(ctx, q); err != nil {
			return ProcessResult{}, err
		}
		if parts.SimilarBooks, err = s.collectSimilar(ctx, q); err != nil {
			return ProcessResult{}, err
		}
	}

	info, err := domain.MergeBookInfo(q, parts, s.Clock())
	if err != nil {
		return ProcessResult{}, err
	}

	if s.Cache != nil {
		_ = s.Cache.Set(key, info)
	}
	var output string
	if s.Writer != nil {
		output, err = s.Writer.Write(q, info, s.Clock())
		if err != nil {
			// spec requires summary still be possible even when save fails
			return ProcessResult{Info: info, OutputPath: ""}, err
		}
	}
	return ProcessResult{Info: info, OutputPath: output, FromCache: false}, nil
}

func (s *Service) ProcessBatch(ctx context.Context, q domain.BookQuery) ([]domain.BatchItemResult, error) {
	s.defaults()
	if strings.TrimSpace(q.BatchPath) == "" {
		return nil, domain.ErrInvalidBatchPath
	}
	file, err := os.Open(q.BatchPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	var titles []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			titles = append(titles, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	delay := 3 * time.Second
	successStreak := 0
	results := make([]domain.BatchItemResult, 0, len(titles))
	usedOutputs := map[string]struct{}{}
	date := s.Clock().Local().Format("20060102")
	for idx, title := range titles {
		itemQuery := q
		itemQuery.Title = title
		itemQuery.ISBN13 = ""
		itemQuery.BatchPath = ""
		itemQuery.Output = s.batchOutputPath(itemQuery, q.Output, date, usedOutputs)

		res, err := s.ProcessOne(ctx, itemQuery)
		if err != nil {
			results = append(results, domain.BatchItemResult{
				Query:   title,
				Success: false,
				Error:   err.Error(),
			})
			successStreak = 0
			var retryable *domain.RetryableError
			if domain.AsRetryable(err, &retryable) && retryable.Code == 429 {
				delay *= 2
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
		} else {
			results = append(results, domain.BatchItemResult{
				Query:      title,
				Success:    true,
				OutputPath: res.OutputPath,
			})
			successStreak++
			if successStreak >= 3 {
				delay = 3 * time.Second
			}
		}
		if idx < len(titles)-1 {
			s.Sleep(delay)
		}
	}
	return results, nil
}

func (s *Service) batchOutputPath(itemQuery domain.BookQuery, outputRoot string, localDate string, used map[string]struct{}) string {
	temp := itemQuery
	temp.Output = ""
	base := domain.ResolveOutputPath(temp, localDate)
	candidate := base
	if strings.TrimSpace(outputRoot) != "" {
		candidate = filepath.Join(outputRoot, base)
	}

	ext := filepath.Ext(candidate)
	stem := strings.TrimSuffix(candidate, ext)
	i := 1
	for {
		if _, ok := used[candidate]; !ok {
			if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
				used[candidate] = struct{}{}
				return candidate
			}
		}
		i++
		candidate = stem + "-" + strconv.Itoa(i) + ext
	}
}

func (s *Service) collectMetadata(ctx context.Context, q domain.BookQuery) (domain.BookMetadata, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectMetadata(ctx, q)
	}, s.Validator.ValidateMetadataJSON)
}

func (s *Service) collectTOC(ctx context.Context, q domain.BookQuery) ([]domain.TOCNode, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectTOC(ctx, q)
	}, s.Validator.ValidateTOCJSON)
}

func (s *Service) collectReview(ctx context.Context, q domain.BookQuery) (domain.ReviewInfo, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectReview(ctx, q)
	}, s.Validator.ValidateReviewJSON)
}

func (s *Service) collectCourses(ctx context.Context, q domain.BookQuery) ([]domain.RelatedCourse, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectCourses(ctx, q)
	}, s.Validator.ValidateCoursesJSON)
}

func (s *Service) collectSimilar(ctx context.Context, q domain.BookQuery) ([]domain.SimilarBook, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectSimilarBooks(ctx, q)
	}, s.Validator.ValidateSimilarBooksJSON)
}

func collectWithRetry[T any](svc *Service, ctx context.Context, collect func() ([]byte, error), validate func([]byte) (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 1; attempt <= svc.Backoff.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}
		raw, err := collect()
		if err != nil {
			lastErr = err
			var retryable *domain.RetryableError
			if !domain.AsRetryable(err, &retryable) || attempt == svc.Backoff.MaxRetries {
				return zero, err
			}
			svc.Sleep(svc.Backoff.DelayForAttempt(attempt, svc.Jitter(svc.Backoff.MaxJitter)))
			continue
		}
		item, err := validate(raw)
		if err == nil {
			return item, nil
		}
		lastErr = err
		if attempt == svc.Backoff.MaxRetries {
			break
		}
		svc.Sleep(svc.Backoff.DelayForAttempt(attempt, svc.Jitter(svc.Backoff.MaxJitter)))
	}
	if lastErr == nil {
		lastErr = domain.ErrInvalidResponse
	}
	if errors.Is(lastErr, domain.ErrInvalidResponse) {
		return zero, lastErr
	}
	return zero, fmt.Errorf("%w: %v", domain.ErrInvalidResponse, lastErr)
}

// enrichTOCParts detects depth-1 parts whose chapters all have empty children,
// then makes individual CollectPartTOC calls per part to get section-level detail.
// This solves the output-token limitation of single-call TOC for multi-volume sets.
func (s *Service) enrichTOCParts(ctx context.Context, q domain.BookQuery, meta domain.BookMetadata, toc []domain.TOCNode) []domain.TOCNode {
	sparse := sparseParts(toc)
	if len(sparse) == 0 {
		return toc
	}

	enriched := make([]domain.TOCNode, len(toc))
	copy(enriched, toc)

	for _, idx := range sparse {
		part := toc[idx]
		existingChapters := make([]string, len(part.Children))
		for i, ch := range part.Children {
			existingChapters[i] = ch.Title.Original
		}

		children, err := s.collectPartTOC(ctx, q, meta.Title.Original, part.Title.Original, existingChapters)
		if err != nil {
			continue // keep original on failure
		}
		// Merge: map enriched chapters back to originals by title similarity.
		// This filters out boilerplate (Preface, Getting Started, etc.) that
		// the LLM may add despite prompt instructions.
		merged := mergeEnrichedChildren(part.Children, children)
		if countTOCNodes(merged) > countTOCNodes(part.Children) {
			enriched[idx].Children = merged
		}
	}
	return enriched
}

func countTOCNodes(nodes []domain.TOCNode) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countTOCNodes(n.Children)
	}
	return count
}

// sparseParts returns indices of depth-1 TOC nodes that have chapters (children)
// but none of those chapters have any sections (grandchildren).
func sparseParts(toc []domain.TOCNode) []int {
	var indices []int
	for i, node := range toc {
		if len(node.Children) == 0 {
			continue // leaf node, no enrichment needed
		}
		hasAnySections := false
		for _, ch := range node.Children {
			if len(ch.Children) > 0 {
				hasAnySections = true
				break
			}
		}
		if !hasAnySections {
			indices = append(indices, i)
		}
	}
	return indices
}

// mergeEnrichedChildren maps enriched chapters back to original chapters by
// title similarity. For each original chapter, it finds the best-matching
// enriched chapter and copies its children (sections). Enriched chapters
// that don't match any original (e.g. boilerplate front matter) are discarded.
func mergeEnrichedChildren(originals, enriched []domain.TOCNode) []domain.TOCNode {
	if len(enriched) == 0 {
		return originals
	}
	result := make([]domain.TOCNode, len(originals))
	copy(result, originals)

	used := make([]bool, len(enriched))
	for i, orig := range originals {
		origNorm := normalizeTitle(orig.Title.Original)
		best := -1
		bestScore := 0.0
		for j, enr := range enriched {
			if used[j] {
				continue
			}
			enrNorm := normalizeTitle(enr.Title.Original)
			score := titleSimilarity(origNorm, enrNorm)
			if score > bestScore {
				bestScore = score
				best = j
			}
		}
		if best >= 0 && bestScore >= 0.4 && len(enriched[best].Children) > 0 {
			result[i].Children = enriched[best].Children
			used[best] = true
		}
	}
	return result
}

// normalizeTitle lowercases and strips leading numbering like "Chapter 1:", "1.", etc.
func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Strip common prefixes: "chapter N:", "part N:", numbering like "1." "1.1"
	for _, prefix := range []string{"chapter ", "part "} {
		if strings.HasPrefix(s, prefix) {
			rest := s[len(prefix):]
			// skip the number and colon/space after it
			idx := strings.IndexFunc(rest, func(r rune) bool {
				return !unicode.IsDigit(r) && r != '.' && r != ':' && r != ' '
			})
			if idx > 0 {
				s = strings.TrimSpace(rest[idx:])
			}
		}
	}
	return s
}

// titleSimilarity returns a score [0,1] based on word overlap (Jaccard index).
func titleSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 0.9
	}
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	setB := make(map[string]bool, len(wordsB))
	for _, w := range wordsB {
		setB[w] = true
	}
	overlap := 0
	for _, w := range wordsA {
		if setB[w] {
			overlap++
		}
	}
	union := len(wordsA) + len(wordsB) - overlap
	if union == 0 {
		return 0
	}
	return float64(overlap) / float64(union)
}

func (s *Service) collectPartTOC(ctx context.Context, q domain.BookQuery, parentTitle, partTitle string, existingChapters []string) ([]domain.TOCNode, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectPartTOC(ctx, q, parentTitle, partTitle, existingChapters)
	}, s.Validator.ValidateTOCJSON)
}

func (s *Service) collectUnified(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return collectWithRetry(s, ctx, func() ([]byte, error) {
		return s.Collector.CollectUnified(ctx, q)
	}, func(raw []byte) ([]byte, error) { return raw, nil })
}
