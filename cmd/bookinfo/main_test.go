package main

import (
	"os"
	"testing"
	"time"
)

func TestResolveHTTPTimeout(t *testing.T) {
	t.Setenv("BOOKINFO_HTTP_TIMEOUT_SEC", "")
	if got := resolveHTTPTimeout(); got != 90*time.Second {
		t.Fatalf("default timeout mismatch: %v", got)
	}

	t.Setenv("BOOKINFO_HTTP_TIMEOUT_SEC", "120")
	if got := resolveHTTPTimeout(); got != 120*time.Second {
		t.Fatalf("env timeout mismatch: %v", got)
	}

	t.Setenv("BOOKINFO_HTTP_TIMEOUT_SEC", "invalid")
	if got := resolveHTTPTimeout(); got != 90*time.Second {
		t.Fatalf("invalid env should fallback default: %v", got)
	}

	t.Setenv("BOOKINFO_HTTP_TIMEOUT_SEC", "-1")
	if got := resolveHTTPTimeout(); got != 90*time.Second {
		t.Fatalf("negative env should fallback default: %v", got)
	}

	_ = os.Unsetenv("BOOKINFO_HTTP_TIMEOUT_SEC")
}
