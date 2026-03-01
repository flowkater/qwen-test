package app

import (
	"fmt"
	"strings"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

func BuildTOCPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn detailed table_of_contents as recursive children nodes. " +
		"Do not impose artificial depth limits. Provide original + korean titles for every node."
}

func BuildMetadataPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn metadata fields: title(original/korean), author, publisher, published_date, isbn13, pages, language, edition, selection_note."
}

func BuildReviewPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn review with rating, summary.pros, summary.cons, recommended_level, prerequisites."
}

func BuildCoursesPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn related courses with title, platform, instructor, curriculum(array), rating, price, url."
}

func BuildSimilarPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn 3-5 similar books including title(original/korean), author, brief_description, difficulty_comparison."
}

func commonLead(q domain.BookQuery) string {
	identifier := q.Title
	if q.ISBN13 != "" {
		identifier = fmt.Sprintf("ISBN-13: %s", q.ISBN13)
	}
	builder := strings.Builder{}
	builder.WriteString("Find canonical book/lecture data for: ")
	builder.WriteString(identifier)
	if q.Author != "" {
		builder.WriteString(" author=" + q.Author)
	}
	if q.Lang != "" {
		builder.WriteString(" lang=" + q.Lang)
	}
	builder.WriteString(". If duplicate titles exist choose most popular; if same author/title choose latest edition.")
	builder.WriteString(" Always include original and korean title pairs.")
	return builder.String()
}
