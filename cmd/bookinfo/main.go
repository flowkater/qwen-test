package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/flowkater/qwen/bookinfo/internal/app"
	"github.com/flowkater/qwen/bookinfo/internal/cli"
	"github.com/flowkater/qwen/bookinfo/internal/infra/cache"
	"github.com/flowkater/qwen/bookinfo/internal/infra/openrouter"
	"github.com/flowkater/qwen/bookinfo/internal/infra/output"
	"github.com/flowkater/qwen/bookinfo/internal/infra/validator"
)

func main() {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	collector := openrouter.NewCollector(openrouter.NewClient(httpClient))
	service := &app.Service{
		Collector: collector,
		Validator: validator.NewJSONValidator(),
		Cache:     cache.NewFileCache(defaultCacheDir()),
		Writer:    output.NewJSONWriter(),
	}
	runner := &cli.Runner{Service: service}
	os.Exit(runner.Run(context.Background(), os.Args[1:]))
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cache/bookinfo"
	}
	return filepath.Join(home, ".cache", "bookinfo")
}
