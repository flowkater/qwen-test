package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func CacheKey(q BookQuery) string {
	identifier := strings.ToLower(strings.TrimSpace(q.PreferredIdentifier()))
	author := strings.ToLower(strings.TrimSpace(q.Author))
	lang := strings.ToLower(strings.TrimSpace(q.Lang))
	model := strings.ToLower(strings.TrimSpace(q.Model))
	if model == "" {
		model = DefaultModel
	}
	mode := q.Mode()
	raw := strings.Join([]string{identifier, author, lang, model, mode}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
