package domain

import "testing"

func TestCacheKeyStabilityAndDifferences(t *testing.T) {
	base := BookQuery{
		Title:    "Clean Code",
		Author:   "Robert C. Martin",
		Lang:     "en",
		Model:    DefaultModel,
		FullMode: false,
	}
	key1 := CacheKey(base)
	key2 := CacheKey(base)
	if key1 != key2 {
		t.Fatalf("same inputs must yield same key")
	}

	diffModel := base
	diffModel.Model = "another/model"
	if CacheKey(diffModel) == key1 {
		t.Fatalf("different model must produce different key")
	}

	diffMode := base
	diffMode.FullMode = true
	if CacheKey(diffMode) == key1 {
		t.Fatalf("different mode must produce different key")
	}
}
