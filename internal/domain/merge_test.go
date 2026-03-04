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

func TestIsLecture(t *testing.T) {
	tests := []struct {
		name     string
		meta     BookMetadata
		expected bool
	}{
		{"book with ISBN and pages", BookMetadata{ISBN13: "978-0132350884", Pages: 464}, false},
		{"book with ISBN only", BookMetadata{ISBN13: "978-0132350884", Pages: 0}, false},
		{"lecture no ISBN no pages", BookMetadata{ISBN13: "", Pages: 0}, true},
		{"lecture whitespace ISBN", BookMetadata{ISBN13: "  ", Pages: 0}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.meta.IsLecture(); got != tc.expected {
				t.Errorf("IsLecture() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestMergeLectureTruncatesDepth3(t *testing.T) {
	q := BookQuery{Title: "시스템 디자인"}
	parts := CollectedParts{
		Metadata: BookMetadata{
			Title:     LocalizedTitle{Original: "시스템 디자인 첫걸음"},
			Author:    "성장랜턴",
			Publisher: "인프런",
			ISBN13:    "", // no ISBN → lecture
			Pages:     0,  // no pages → lecture
		},
		TOC: []TOCNode{
			{Title: LocalizedTitle{Original: "섹션 1"}, Depth: 1, Children: []TOCNode{
				{Title: LocalizedTitle{Original: "강의 1"}, Depth: 2, Children: []TOCNode{
					{Title: LocalizedTitle{Original: "환각 서브토픽"}, Depth: 3}, // should be removed
				}},
				{Title: LocalizedTitle{Original: "강의 2"}, Depth: 2},
			}},
		},
	}
	info, err := MergeBookInfo(q, parts, time.Now())
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	// depth-1 섹션 preserved
	if len(info.TableOfContents) != 1 {
		t.Fatalf("expected 1 section, got %d", len(info.TableOfContents))
	}
	// depth-2 강의 preserved
	if len(info.TableOfContents[0].Children) != 2 {
		t.Fatalf("expected 2 lectures, got %d", len(info.TableOfContents[0].Children))
	}
	// depth-3 hallucination removed
	for _, ch := range info.TableOfContents[0].Children {
		if len(ch.Children) != 0 {
			t.Fatalf("lecture children should have no depth-3 children, got %d for %q",
				len(ch.Children), ch.Title.Original)
		}
	}
}

func TestMergeBookKeepsDepth3(t *testing.T) {
	q := BookQuery{Title: "Clean Code"}
	parts := CollectedParts{
		Metadata: BookMetadata{
			Title:     LocalizedTitle{Original: "Clean Code"},
			Author:    "Robert C. Martin",
			Publisher: "Addison-Wesley",
			ISBN13:    "978-0132350884", // has ISBN → book
			Pages:     464,
		},
		TOC: []TOCNode{
			{Title: LocalizedTitle{Original: "Chapter 1"}, Depth: 1, Children: []TOCNode{
				{Title: LocalizedTitle{Original: "Section 1.1"}, Depth: 2, Children: []TOCNode{
					{Title: LocalizedTitle{Original: "Subsection 1.1.1"}, Depth: 3},
				}},
			}},
		},
	}
	info, err := MergeBookInfo(q, parts, time.Now())
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	// Books should keep depth-3
	if len(info.TableOfContents[0].Children[0].Children) != 1 {
		t.Fatalf("book should keep depth-3 children")
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
