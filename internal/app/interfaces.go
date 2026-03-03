package app

import (
	"context"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type Collector interface {
	CollectMetadata(ctx context.Context, q domain.BookQuery) ([]byte, error)
	CollectTOC(ctx context.Context, q domain.BookQuery) ([]byte, error)
	CollectPartTOC(ctx context.Context, q domain.BookQuery, parentTitle, partTitle string, existingChapters []string) ([]byte, error)
	CollectReview(ctx context.Context, q domain.BookQuery) ([]byte, error)
	CollectCourses(ctx context.Context, q domain.BookQuery) ([]byte, error)
	CollectSimilarBooks(ctx context.Context, q domain.BookQuery) ([]byte, error)
	CollectUnified(ctx context.Context, q domain.BookQuery) ([]byte, error)
}

type Validator interface {
	ValidateMetadataJSON(raw []byte) (domain.BookMetadata, error)
	ValidateTOCJSON(raw []byte) ([]domain.TOCNode, error)
	ValidateReviewJSON(raw []byte) (domain.ReviewInfo, error)
	ValidateCoursesJSON(raw []byte) ([]domain.RelatedCourse, error)
	ValidateSimilarBooksJSON(raw []byte) ([]domain.SimilarBook, error)
	ValidateUnifiedJSON(raw []byte) (domain.BookMetadata, []domain.TOCNode, error)
}

type Cache interface {
	Get(key string) (domain.BookInfo, bool, error)
	Set(key string, value domain.BookInfo) error
}

type Writer interface {
	Write(q domain.BookQuery, data domain.BookInfo, now time.Time) (string, error)
}

type Clock func() time.Time
type Sleeper func(time.Duration)
