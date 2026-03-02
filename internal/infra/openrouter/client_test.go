package openrouter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func TestClientHeadersAndTimeoutAndSuccess(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`)
	}))
	defer server.Close()

	client := NewClient(&http.Client{Timeout: 30 * time.Second})
	client.BaseURL = server.URL

	raw, err := client.Collect(context.Background(), "", "secret-token", "prompt")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if authHeader != "Bearer secret-token" {
		t.Fatalf("missing auth header: %s", authHeader)
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

			_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "prompt")
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

	_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "prompt")
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
		_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "prompt")
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
		_, err := client.Collect(context.Background(), domain.DefaultModel, "token", "prompt")
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
