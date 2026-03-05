package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

var ErrHelpRequested = errors.New("help requested")

func ParseArgs(args []string, stderr io.Writer) (domain.BookQuery, error) {
	query := domain.BookQuery{
		Model:  domain.DefaultModel,
		Format: domain.DefaultOutputFormat,
	}
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--isbn":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --isbn")
			}
			query.ISBN13 = strings.TrimSpace(args[i])
		case "--author", "-a":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for %s", arg)
			}
			query.Author = strings.TrimSpace(args[i])
		case "--lang", "-l":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for %s", arg)
			}
			query.Lang = strings.TrimSpace(args[i])
		case "--full", "-f":
			query.FullMode = true
		case "--format":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --format")
			}
			query.Format = strings.ToLower(strings.TrimSpace(args[i]))
		case "--api-key":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --api-key")
			}
			query.APIKey = strings.TrimSpace(args[i])
		case "--model", "-m":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for %s", arg)
			}
			query.Model = strings.TrimSpace(args[i])
		case "--output", "-o":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for %s", arg)
			}
			query.Output = strings.TrimSpace(args[i])
		case "--batch", "-b":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for %s", arg)
			}
			query.BatchPath = strings.TrimSpace(args[i])
		case "--country":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --country")
			}
			query.Country = strings.ToLower(strings.TrimSpace(args[i]))
		case "--lecture":
			query.Lecture = true
		case "--url":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --url")
			}
			query.URL = strings.TrimSpace(args[i])
		case "--workers":
			i++
			if i >= len(args) {
				return domain.BookQuery{}, fmt.Errorf("missing value for --workers")
			}
			w, err := strconv.Atoi(strings.TrimSpace(args[i]))
			if err != nil || w < 1 {
				return domain.BookQuery{}, fmt.Errorf("--workers must be a positive integer")
			}
			query.Workers = w
		case "--no-cache":
			query.NoCache = true
		case "--help", "-h":
			_, _ = io.WriteString(stderr, HelpText()+"\n")
			return domain.BookQuery{}, ErrHelpRequested
		default:
			if strings.HasPrefix(arg, "-") {
				return domain.BookQuery{}, fmt.Errorf("unknown flag: %s", arg)
			}
			positionals = append(positionals, arg)
		}
	}
	query.Title = strings.TrimSpace(strings.Join(positionals, " "))
	if strings.TrimSpace(query.Model) == "" {
		query.Model = domain.DefaultModel
	}

	if err := domain.ValidateQuery(query); err != nil {
		return domain.BookQuery{}, err
	}
	return query, nil
}

func HelpText() string {
	return strings.TrimSpace(`
Usage:
  bookinfo <title> [flags]
  bookinfo --isbn <isbn13> --country <country> [flags]
  bookinfo --lecture --url <url> --country <country> [flags]
  bookinfo --batch <file> [flags]

Flags:
  --isbn                    ISBN-13 query (requires --country)
  --author, -a              author
  --lang, -l                ko|en|ja|zh-tw
  --full, -f                collect metadata+toc+review+courses+similar
  --format                  output format (json|text, default json)
  --api-key                 override DASHSCOPE_API_KEY
  --model, -m               model (default qwen/qwen3.5-flash-02-23)
  --output, -o              output path
  --batch, -b               batch text file (one title/ISBN per line)
  --workers                 parallel workers for batch (default 1 = sequential)
  --no-cache                skip cache read but still write refreshed cache
  --country                 country code us|kr|jp|tw (required for --isbn and --lecture)
  --lecture                 lecture/course mode (requires --url and --country)
  --url                     lecture site URL (required for --lecture)
`)
}

func ValidateLangMessage(lang string) string {
	return fmt.Sprintf("invalid --lang value %q (허용값: ko, en, ja, zh-tw)", lang)
}
