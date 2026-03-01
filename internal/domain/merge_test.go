package domain

import (
	"testing"
	"time"
)

func TestMergeBasicModeDefaults(t *testing.T) {
	q := BookQuery{Title: "Clean Code", FullMode: false}
	parts := CollectedParts{
		Metadata: BookMetadata{
			Title:     LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley Professional",
			ISBN13:    "978-0132350884",
		},
		TOC: []TOCNode{
			{
				Title: LocalizedTitle{Original: "Chapter 1", Korean: "1장"},
				Depth: 1,
				Children: []TOCNode{
					{Title: LocalizedTitle{Original: "Section 1", Korean: "1.1절"}, Depth: 2},
				},
			},
		},
	}
	info, err := MergeBookInfo(q, parts, time.Now())
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(info.RelatedCourses) != 0 || len(info.SimilarBooks) != 0 {
		t.Fatalf("basic mode must set empty collections")
	}
	if info.DataAvailability.Review || info.DataAvailability.RelatedCourses || info.DataAvailability.SimilarBooks {
		t.Fatalf("basic mode availability must be false for full-only fields")
	}
}

func TestMergeFullModeValidation(t *testing.T) {
	q := BookQuery{Title: "Clean Code", FullMode: true}
	parts := CollectedParts{
		Metadata: BookMetadata{
			Title:     LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley Professional",
			ISBN13:    "978-0132350884",
		},
		TOC: []TOCNode{
			{Title: LocalizedTitle{Original: "Chapter 1", Korean: "1장"}, Depth: 1},
		},
		Review: &ReviewInfo{
			Rating: 4.5,
		},
		SimilarBooks: []SimilarBook{
			{}, {},
		},
	}
	if _, err := MergeBookInfo(q, parts, time.Now()); err == nil {
		t.Fatalf("expected similar books count validation error")
	}
}

func TestMergeTOCDepthAndTitleValidation(t *testing.T) {
	q := BookQuery{Title: "Clean Code"}
	parts := CollectedParts{
		Metadata: BookMetadata{
			Title:     LocalizedTitle{Original: "Clean Code", Korean: "클린 코드"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley Professional",
			ISBN13:    "978-0132350884",
		},
		TOC: []TOCNode{
			{
				Title:    LocalizedTitle{Original: "Chapter", Korean: "챕터"},
				Depth:    1,
				Children: []TOCNode{{Title: LocalizedTitle{Original: "", Korean: "소절"}, Depth: 1}},
			},
		},
	}
	if _, err := MergeBookInfo(q, parts, time.Now()); err == nil {
		t.Fatalf("expected toc validation failure")
	}
}
