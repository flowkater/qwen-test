package app

import (
	"fmt"
	"strings"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

const SystemPromptBookInfo = `You are a precise book/lecture metadata and table of contents extraction engine.

CRITICAL RULES:
1. Return ONLY valid JSON. No markdown, no explanation, no code blocks.
2. Follow the EXACT JSON schema requested in the user prompt.
3. Keep all extracted text in the resource's ORIGINAL language unless explicitly asked to translate.
4. NEVER translate chapter/section titles, author names, or publisher names by default.
5. Preserve complete table-of-contents hierarchy with correct parent-child relationships and depth.
6. Include appendices/annexes when available.
7. If a value is unknown, keep schema shape and use "" for strings, 0 for numbers, [] for arrays.
8. Prefer canonical data from the ORIGINAL edition unless the query explicitly requests another language/edition.`

func BuildTOCPrompt(q domain.BookQuery) string {
	return commonLead(q) + `

Return ONLY valid JSON array (no markdown, no explanation):
[{"title":{"original":"","korean":""},"depth":1,"children":[]}]

Rules:
- Keep "title" compatible with LocalizedTitle:
  - title.original: canonical title in ORIGINAL language.
  - title.korean: Korean translation only if confidently known, otherwise "".
- Include ALL main parts/chapters and appendices/annexes when available.
- Preserve full hierarchy as recursive children nodes. Do NOT flatten.
- Depth guidance:
  - depth 1: top-level part/chapter
  - depth 2: section
  - depth 3: subsection
  - depth 4: sub-subsection (if present)
- Do NOT translate title.original.
- Return ONLY the JSON array.`
}

func BuildMetadataPrompt(q domain.BookQuery) string {
	return commonLead(q) + `

Return ONLY valid JSON object (no markdown, no explanation):
{"title":{"original":"","korean":""},"author":"","publisher":"","published_date":"","isbn13":"","pages":0,"language":"","edition":"","selection_note":""}

Rules:
- title.original: canonical title in ORIGINAL language.
- title.korean: Korean translation only if confidently known, otherwise "".
- author: ORIGINAL-language author name only (no translation/transliteration).
- publisher: ORIGINAL-language publisher name from the original edition.
- isbn13: 13-digit ISBN string with hyphens when known (example: "978-0132350884").
- Prefer original-edition metadata unless query explicitly requests another language/edition.
- If unknown, use "" for text fields and 0 for pages.
- Return ONLY the JSON object.`
}

func BuildReviewPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn ONLY valid JSON object (no markdown, no explanation): " +
		`{"rating":0,"summary":{"pros":[],"cons":[]},"recommended_level":"","prerequisites":[]}` +
		"\nReturn review with rating, summary.pros, summary.cons, recommended_level, prerequisites."
}

func BuildCoursesPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn ONLY valid JSON array (no markdown, no explanation): " +
		`[{"title":"","platform":"","instructor":"","curriculum":[],"rating":0,"price":"","url":""}]` +
		"\nReturn related courses with title, platform, instructor, curriculum(array), rating, price, url."
}

func BuildSimilarPrompt(q domain.BookQuery) string {
	return commonLead(q) + "\nReturn ONLY valid JSON array (no markdown, no explanation): " +
		`[{"title":{"original":"","korean":""},"author":"","brief_description":"","difficulty_comparison":""}]` +
		"\nReturn 3-5 similar books including title(original/korean), author, brief_description, difficulty_comparison."
}



func BuildUnifiedPrompt(q domain.BookQuery) string {
	return commonLead(q) + `

Return ONLY valid JSON object (no markdown, no explanation) with this exact structure:
{
  "metadata": {
    "title": {"original": "", "korean": ""},
    "author": "",
    "publisher": "",
    "published_date": "",
    "isbn13": "",
    "pages": 0,
    "language": "",
    "edition": "",
    "selection_note": ""
  },
  "toc": [
    {"title": {"original": "", "korean": ""}, "depth": 1, "children": []}
  ]
}

Rules:
- metadata.title.original: canonical title in ORIGINAL language.
- metadata.title.korean: Korean translation only if confidently known, otherwise "".
- metadata.author: ORIGINAL-language author name only.
- metadata.publisher: ORIGINAL-language publisher name from the original edition.
- metadata.isbn13: 13-digit ISBN string with hyphens when known.
- toc: Include ALL main parts/chapters and appendices/annexes.
- Preserve full hierarchy as recursive children nodes. Do NOT flatten.
- Depth: 1=chapter, 2=section, 3=subsection, 4=sub-subsection.
- Do NOT translate title.original.
- If unknown, use "" for text fields and 0 for numbers.
- Return ONLY the JSON object.`
}

func GetSystemPrompt() string {
	return SystemPromptBookInfo
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
		builder.WriteString(fmt.Sprintf(
			"\nIMPORTANT LANGUAGE RULE: This is a %s-language resource. "+
				"Keep title.original, chapter/section titles, author, and publisher in %s. "+
				"Do NOT translate to other languages unless explicitly requested.",
			q.Lang, q.Lang,
		))
	}
	builder.WriteString(". If duplicate titles exist choose most popular; if same author/title choose latest edition.")
	builder.WriteString(" Keep title shape as {\"original\":\"\",\"korean\":\"\"}; original is mandatory and korean is optional.")
	return builder.String()
}
