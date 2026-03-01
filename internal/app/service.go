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
	var err error
	if parts.Metadata, err = s.collectMetadata(ctx, q); err != nil {
		return ProcessResult{}, err
	}
	if parts.TOC, err = s.collectTOC(ctx, q); err != nil {
		return ProcessResult{}, err
	}
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
