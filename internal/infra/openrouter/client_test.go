package openrouter

import (
	"context"
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
