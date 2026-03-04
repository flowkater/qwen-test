package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
		BaseURL: domain.DashScopeBaseURL,
		HTTP:    httpClient,
		Timeout: 180 * time.Second,
	}
}

// DashScope Responses API types

type RequestPayload struct {
	Model       string    `json:"model"`
	Input       []Message `json:"input"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Seed        *int      `json:"seed,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type string `json:"type"`
}

type ResponseOutput struct {
	Output []OutputItem `json:"output"`
	Usage  struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type OutputItem struct {
	Type    string        `json:"type"`
	Content []ContentItem `json:"content,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (c *Client) Collect(ctx context.Context, model, apiKey, systemPrompt, userPrompt string) ([]byte, error) {
	if model == "" {
		model = domain.DefaultModel
	}

	temp := 0.0
	seed := 42
	payload := RequestPayload{
		Model:       model,
		Temperature: &temp,
		Seed:        &seed,
		Input: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Tools: []Tool{
			{Type: "web_search"},
			{Type: "web_extractor"},
		},
	}

	var content string
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		content, err = c.collectContent(ctx, apiKey, payload)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "no message output found") {
			return nil, err
		}
		// DashScope sometimes returns incomplete responses (no message output).
		// Retry up to 3 times before giving up.
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "not found") || strings.Contains(lower, "찾을 수 없습니다") {
		return nil, domain.ErrBookNotFound
	}
	return extractJSON(content)
}

func (c *Client) collectContent(ctx context.Context, apiKey string, payload RequestPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: c.Timeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return "", &domain.RetryableError{Code: 408, Err: err}
		}
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		if isTimeoutError(err) {
			return "", &domain.RetryableError{Code: 408, Err: err}
		}
		return "", err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return "", &domain.RetryableError{Code: resp.StatusCode, Err: fmt.Errorf("dashscope status %d", resp.StatusCode)}
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", domain.ErrBookNotFound
	}
	if resp.StatusCode >= 400 {
		return "", &providerStatusError{
			Code:   resp.StatusCode,
			Detail: extractProviderErrorDetail(raw),
		}
	}

	var parsed ResponseOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("response decode: %w", err)
	}

	if parsed.Error != nil && parsed.Error.Code != "" {
		return "", fmt.Errorf("dashscope error %s: %s", parsed.Error.Code, parsed.Error.Message)
	}

	// Find the "message" output item
	for _, item := range parsed.Output {
		if item.Type == "message" && len(item.Content) > 0 {
			for _, c := range item.Content {
				if c.Type == "output_text" && strings.TrimSpace(c.Text) != "" {
					return c.Text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("response decode: no message output found")
}

type providerStatusError struct {
	Code   int
	Detail string
}

func (e *providerStatusError) Error() string {
	if strings.TrimSpace(e.Detail) == "" {
		return fmt.Sprintf("dashscope non-retryable status %d", e.Code)
	}
	return fmt.Sprintf("dashscope non-retryable status %d: %s", e.Code, e.Detail)
}

func extractJSON(content string) ([]byte, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, fmt.Errorf("response is not valid JSON")
	}
	if json.Valid([]byte(trimmed)) {
		return []byte(trimmed), nil
	}

	candidates := []string{trimmed}
	if fenced, ok := stripFencedContent(trimmed); ok {
		candidates = append([]string{fenced}, candidates...)
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if json.Valid([]byte(candidate)) {
			return []byte(candidate), nil
		}
		if extracted, ok := extractFirstJSONObjectOrArray(candidate); ok {
			return []byte(extracted), nil
		}
	}
	return nil, fmt.Errorf("response is not valid JSON")
}

func stripFencedContent(content string) (string, bool) {
	start := strings.Index(content, "```")
	if start < 0 {
		return "", false
	}
	rest := content[start+3:]
	if newline := strings.Index(rest, "\n"); newline >= 0 {
		rest = rest[newline+1:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return "", false
	}
	inner := strings.TrimSpace(rest[:end])
	if inner == "" {
		return "", false
	}
	return inner, true
}

func extractFirstJSONObjectOrArray(content string) (string, bool) {
	for i := 0; i < len(content); i++ {
		if content[i] != '{' && content[i] != '[' {
			continue
		}
		if end, ok := findJSONBoundary(content, i); ok {
			candidate := strings.TrimSpace(content[i:end])
			if json.Valid([]byte(candidate)) {
				return candidate, true
			}
		}
	}
	return "", false
}

func findJSONBoundary(content string, start int) (int, bool) {
	stack := make([]byte, 0, 8)
	inString := false
	escaped := false

	for i := start; i < len(content); i++ {
		ch := content[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, ch)
		case '}':
			if len(stack) == 0 || stack[len(stack)-1] != '{' {
				return 0, false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i + 1, true
			}
		case ']':
			if len(stack) == 0 || stack[len(stack)-1] != '[' {
				return 0, false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i + 1, true
			}
		}
	}

	return 0, false
}

func extractProviderErrorDetail(raw []byte) string {
	type responseError struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	var parsed responseError
	if err := json.Unmarshal(raw, &parsed); err == nil {
		msg := strings.TrimSpace(parsed.Error.Message)
		if msg != "" {
			return msg
		}
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "client.timeout")
}
