package openrouter

import (
	"context"

	"github.com/flowkater/qwen/bookinfo/internal/app"
	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type Collector struct {
	Client *Client
}

func NewCollector(client *Client) *Collector {
	return &Collector{Client: client}
}

func (c *Collector) CollectMetadata(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return c.Client.Collect(ctx, q.Model, q.APIKey, app.GetSystemPrompt(), app.BuildMetadataPrompt(q))
}

func (c *Collector) CollectTOC(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return c.Client.Collect(ctx, q.Model, q.APIKey, app.GetSystemPrompt(), app.BuildTOCPrompt(q))
}

func (c *Collector) CollectReview(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return c.Client.Collect(ctx, q.Model, q.APIKey, app.GetSystemPrompt(), app.BuildReviewPrompt(q))
}

func (c *Collector) CollectCourses(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return c.Client.Collect(ctx, q.Model, q.APIKey, app.GetSystemPrompt(), app.BuildCoursesPrompt(q))
}

func (c *Collector) CollectSimilarBooks(ctx context.Context, q domain.BookQuery) ([]byte, error) {
	return c.Client.Collect(ctx, q.Model, q.APIKey, app.GetSystemPrompt(), app.BuildSimilarPrompt(q))
}
