package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBookInfoSerializationHasRootFields(t *testing.T) {
	info := BookInfo{
		Book: BookMetadata{
			Title:         LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:        "Robert C. Martin",
			Publisher:     "Addison-Wesley Professional",
			ISBN13:        "978-0132350884",
			SelectionNote: "popular edition",
		},
		TableOfContents: []TOCNode{
			{Title: LocalizedTitle{Original: "Chapter 1", Korean: "1장"}, Depth: 1},
		},
		Metadata: CollectMetadata{
			CollectedAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.FixedZone("KST", 9*3600)),
			Mode:        "basic",
			Model:       DefaultModel,
			Query:       CollectQuery{Title: "Clean Code", ISBN13: "978-0132350884", Author: "Robert C. Martin", Lang: "en"},
		},
	}
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"book", "table_of_contents", "review", "related_courses", "similar_books", "data_availability", "metadata"} {
		if _, ok := obj[field]; !ok {
			t.Fatalf("missing root field %q", field)
		}
	}
}

func TestTOCNodeChildrenSerializedAsArray(t *testing.T) {
	node := TOCNode{Title: LocalizedTitle{Original: "Chapter", Korean: "챕터"}, Depth: 1}
	raw, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	children, ok := obj["children"].([]any)
	if !ok || len(children) != 0 {
		t.Fatalf("children must be empty array, got %#v", obj["children"])
	}
}

func TestSimilarBookSerializationHasDifficultyComparison(t *testing.T) {
	item := SimilarBook{
		Title:                LocalizedTitle{Original: "Refactoring", Korean: "리팩터링"},
		Author:               "Martin Fowler",
		BriefDescription:     "desc",
		DifficultyComparison: "비슷함",
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	if _, ok := obj["difficulty_comparison"]; !ok {
		t.Fatalf("difficulty_comparison missing")
	}
}

func TestCollectMetadataCollectedAtUTC(t *testing.T) {
	meta := CollectMetadata{
		CollectedAt: time.Date(2026, 3, 1, 23, 30, 0, 0, time.FixedZone("KST", 9*3600)),
		Mode:        "basic",
		Query:       CollectQuery{},
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	got := obj["collected_at"].(string)
	if got != "2026-03-01T14:30:00Z" {
		t.Fatalf("collected_at must be UTC RFC3339, got %s", got)
	}
}

func TestNormalizeAuthor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "parenthesized english preferred",
			input: "로버트 C. 마틴 (Robert C. Martin)",
			want:  "Robert C. Martin",
		},
		{
			name:  "plain preserved",
			input: "Robert C. Martin",
			want:  "Robert C. Martin",
		},
		{
			name:  "empty parenthesis ignored",
			input: "홍길동 ()",
			want:  "홍길동 ()",
		},
		{
			name:  "trimmed",
			input: "  Robert C. Martin  ",
			want:  "Robert C. Martin",
		},
		{
			name:  "empty input",
			input: "   ",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeAuthor(tc.input)
			if got != tc.want {
				t.Fatalf("NormalizeAuthor(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
