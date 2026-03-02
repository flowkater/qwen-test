package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func TestClientPayloadIncludesSystemUserAndSamplingParams(t *testing.T) {
	var authHeader string
	var captured RequestPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`)
	}))
	defer server.Close()

	client := NewClient(&http.Client{Timeout: 30 * time.Second})
	client.BaseURL = server.URL

	raw, err := client.Collect(context.Background(), "", "secret-token", "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if authHeader != "Bearer secret-token" {
		t.Fatalf("missing auth header: %s", authHeader)
	}
	if captured.Model != domain.DefaultModel {
		t.Fatalf("default model not used: %s", captured.Model)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(captured.Messages))
	}
	if captured.Messages[0].Role != "system" || captured.Messages[0].Content != "system prompt" {
		t.Fatalf("unexpected system message: %+v", captured.Messages[0])
	}
	if captured.Messages[1].Role != "user" || captured.Messages[1].Content != "user prompt" {
		t.Fatalf("unexpected user message: %+v", captured.Messages[1])
	}
	if captured.Temperature == nil || math.Abs(*captured.Temperature-0.1) > 0.000001 {
		t.Fatalf("expected temperature=0.1, got %+v", captured.Temperature)
	}
	if captured.MaxTokens != 4000 {
		t.Fatalf("expected max_tokens=4000, got %d", captured.MaxTokens)
	}
	if captured.ResponseFormat == nil || captured.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected response_format json_object, got %+v", captured.ResponseFormat)
	}
	if strings.TrimSpace(string(raw)) != `{"ok":true}` {
		t.Fatalf("unexpected response payload: %s", raw)
	}
}

func TestClientStatusMapping(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		retryable bool
		notFound  bool
	}{
		{name: "429 retryable", status: 429, retryable: true},
		{name: "500 retryable", status: 500, retryable: true},
		{name: "404 book not found", status: 404, retryable: false, notFound: true},
		{name: "400 non retryable", status: 400, retryable: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{"error":"x"}`)
			}))
			defer server.Close()
			client := NewClient(server.Client())
			client.BaseURL = server.URL

			_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "system", "prompt")
			if err == nil {
				t.Fatalf("expected error")
			}
			var re *domain.RetryableError
			gotRetry := domain.AsRetryable(err, &re)
			if gotRetry != tc.retryable {
				t.Fatalf("retryable mismatch want=%v got=%v err=%v", tc.retryable, gotRetry, err)
			}
			if tc.notFound && !strings.Contains(err.Error(), domain.ErrBookNotFound.Error()) {
				t.Fatalf("expected not found mapping, got %v", err)
			}
		})
	}
}

func TestClientMapsNotFoundMessageContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"Book not found for the given ISBN"}}]}`)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.BaseURL = server.URL

	_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "system", "prompt")
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if !strings.Contains(err.Error(), domain.ErrBookNotFound.Error()) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

type timeoutHTTPClient struct{}

func (timeoutHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

type errorReadCloser struct{}

func (errorReadCloser) Read([]byte) (int, error) { return 0, context.DeadlineExceeded }
func (errorReadCloser) Close() error             { return nil }

type readTimeoutHTTPClient struct{}

func (readTimeoutHTTPClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       errorReadCloser{},
	}, nil
}

func TestClientTimeoutErrorsAreRetryable(t *testing.T) {
	t.Run("do timeout", func(t *testing.T) {
		client := NewClient(timeoutHTTPClient{})
		_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "system", "prompt")
		if err == nil {
			t.Fatalf("expected timeout error")
		}
		var re *domain.RetryableError
		if !domain.AsRetryable(err, &re) {
			t.Fatalf("timeout should be retryable: %v", err)
		}
	})

	t.Run("read timeout", func(t *testing.T) {
		client := NewClient(readTimeoutHTTPClient{})
		_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "system", "prompt")
		if err == nil {
			t.Fatalf("expected timeout error")
		}
		var re *domain.RetryableError
		if !domain.AsRetryable(err, &re) {
			t.Fatalf("read timeout should be retryable: %v", err)
		}
	})
}

func TestIsTimeoutError(t *testing.T) {
	if !isTimeoutError(context.DeadlineExceeded) {
		t.Fatalf("context deadline should be timeout")
	}
	if !isTimeoutError(errors.New("Client.Timeout exceeded while awaiting headers")) {
		t.Fatalf("Client.Timeout string should be timeout")
	}
	if isTimeoutError(errors.New("plain error")) {
		t.Fatalf("plain error should not be timeout")
	}
}

func TestClientFallbackWhenResponseFormatUnsupported(t *testing.T) {
	call := 0
	var payloads []RequestPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		var payload RequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		payloads = append(payloads, payload)

		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"response_format is unsupported for this model"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.BaseURL = server.URL

	raw, err := client.Collect(context.Background(), domain.DefaultModel, "token", "system", "prompt")
	if err != nil {
		t.Fatalf("collect with fallback: %v", err)
	}
	if strings.TrimSpace(string(raw)) != `{"ok":true}` {
		t.Fatalf("unexpected payload: %s", raw)
	}
	if call != 2 {
		t.Fatalf("expected 2 requests, got %d", call)
	}
	if len(payloads) != 2 {
		t.Fatalf("expected captured payloads for 2 requests, got %d", len(payloads))
	}
	if payloads[0].ResponseFormat == nil || payloads[0].ResponseFormat.Type != "json_object" {
		t.Fatalf("first request must include response_format, got %+v", payloads[0].ResponseFormat)
	}
	if payloads[1].ResponseFormat != nil {
		t.Fatalf("fallback request must omit response_format, got %+v", payloads[1].ResponseFormat)
	}
}

func TestExtractJSONFromFencedContent(t *testing.T) {
	got, err := extractJSON("Sure, here's the data:\n```json\n{\"foo\":\"bar\"}\n```\nThanks.")
	if err != nil {
		t.Fatalf("extract json: %v", err)
	}
	if strings.TrimSpace(string(got)) != `{"foo":"bar"}` {
		t.Fatalf("unexpected extracted json: %s", got)
	}
}
