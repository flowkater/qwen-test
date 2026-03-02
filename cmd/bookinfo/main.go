package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/app"
	"github.com/flowkater/qwen/bookinfo/internal/cli"
	"github.com/flowkater/qwen/bookinfo/internal/infra/cache"
	"github.com/flowkater/qwen/bookinfo/internal/infra/openrouter"
	"github.com/flowkater/qwen/bookinfo/internal/infra/output"
	"github.com/flowkater/qwen/bookinfo/internal/infra/validator"
)

func main() {
	httpClient := &http.Client{Timeout: resolveHTTPTimeout()}
	collector := openrouter.NewCollector(openrouter.NewClient(httpClient))
	service := &app.Service{
		Collector: collector,
		Validator: validator.NewJSONValidator(),
		Cache:     cache.NewFileCache(defaultCacheDir()),
		Writer:    output.NewFormatWriter(),
	}
	runner := &cli.Runner{Service: service}
	os.Exit(runner.Run(context.Background(), os.Args[1:]))
}

func resolveHTTPTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BOOKINFO_HTTP_TIMEOUT_SEC"))
	if raw == "" {
		return 90 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 90 * time.Second
	}
	return time.Duration(sec) * time.Second
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cache/bookinfo"
	}
	return filepath.Join(home, ".cache", "bookinfo")
}
