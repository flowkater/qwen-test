package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL string
	HTTP    HTTPClient
	Timeout time.Duration
}

func NewClient(httpClient HTTPClient) *Client {
	return &Client{
		BaseURL: "https://openrouter.ai/api/v1/chat/completions",
		HTTP:    httpClient,
		Timeout: 30 * time.Second,
	}
}

type RequestPayload struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (c *Client) Collect(ctx context.Context, model, apiKey, prompt string) ([]byte, error) {
	if model == "" {
		model = domain.DefaultModel
	}
	payload := RequestPayload{Model: model}
	payload.Messages = append(payload.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: prompt})

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: c.Timeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, &domain.RetryableError{Code: resp.StatusCode, Err: fmt.Errorf("openrouter status %d", resp.StatusCode)}
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrBookNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openrouter non-retryable status %d", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("response decode: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("response decode: empty choices")
	}
	content := parsed.Choices[0].Message.Content
	lower := strings.ToLower(content)
	if strings.Contains(lower, "not found") || strings.Contains(lower, "찾을 수 없습니다") {
		return nil, domain.ErrBookNotFound
	}
	return []byte(content), nil
}
