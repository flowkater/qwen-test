package domain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var allowedLang = map[string]struct{}{
	"":      {},
	"ko":    {},
	"en":    {},
	"ja":    {},
	"zh-tw": {},
}

func ValidateQuery(q BookQuery) error {
	title := strings.TrimSpace(q.Title)
	isbn := strings.TrimSpace(q.ISBN13)
	batch := strings.TrimSpace(q.BatchPath)

	if batch == "" && title == "" && isbn == "" {
		return ErrMissingQuery
	}
	if q.Title != "" && title == "" {
		return ErrInvalidTitle
	}
	if q.ISBN13 != "" && isbn == "" {
		return ErrInvalidISBN
	}
	lang := strings.ToLower(strings.TrimSpace(q.Lang))
	if _, ok := allowedLang[lang]; !ok {
		return ErrInvalidLang
	}
	if q.BatchPath != "" && batch == "" {
		return ErrInvalidBatchPath
	}
	if q.Output != "" {
		info, err := os.Stat(q.Output)
		if err == nil && info.IsDir() {
			if strings.TrimSpace(q.BatchPath) != "" {
				return nil
			}
			return ErrInvalidOutput
		}
		if err == nil && !info.IsDir() && strings.TrimSpace(q.BatchPath) != "" {
			return ErrInvalidOutput
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return ErrInvalidOutput
		}
	}
	return nil
}

func ResolveOutputPath(q BookQuery, nowLocalDate string) string {
	if strings.TrimSpace(q.Output) != "" {
		return q.Output
	}
	base := sanitizeFilename(q.PreferredIdentifier())
	if base == "" {
		base = "bookinfo"
	}
	return base + "_" + nowLocalDate + ".json"
}

func sanitizeFilename(input string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	out := strings.TrimSpace(replacer.Replace(input))
	out = strings.ReplaceAll(out, " ", "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

func EnsureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
