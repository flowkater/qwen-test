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
5. TABLE OF CONTENTS MUST BE DEEPLY HIERARCHICAL:
   - Every chapter MUST include its sections as children nodes.
   - Sections with subsections MUST include them as nested children.
   - A textbook typically has 3-4 depth levels. Do NOT return only top-level chapters.
   - Use web_search to find the actual detailed table of contents when available.
6. Include appendices/annexes when available.
7. If a value is unknown, keep schema shape and use "" for strings, 0 for numbers, [] for arrays.
8. Prefer canonical data from the ORIGINAL edition unless the query explicitly requests another language/edition.`

func BuildTOCPrompt(q domain.BookQuery) string {
	return commonLead(q) + `

Return ONLY valid JSON array (no markdown, no explanation):
[{"title":{"original":"Chapter 1: Example","korean":""},"depth":1,"children":[{"title":{"original":"1.1 Section","korean":""},"depth":2,"children":[]}]}]

Rules:
- Use web_search to find the book's ACTUAL detailed table of contents.
- title.original: canonical title in ORIGINAL language. Do NOT translate.
- title.korean: Korean translation only if confidently known, otherwise "".
- Include ALL chapters, sections, and subsections as a deeply nested tree.
- Each chapter (depth 1) MUST contain its sections (depth 2) as children.
- Sections with subsections MUST include them as children (depth 3+).
- Do NOT return only top-level chapter names with empty children.
- Depth: 1=part/chapter, 2=section, 3=subsection, 4=sub-subsection.
- A typical textbook has at least 2-3 levels. Include them all.
- Include appendices/annexes when available.
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
    {
      "title": {"original": "Chapter 1: Example", "korean": ""},
      "depth": 1,
      "children": [
        {
          "title": {"original": "1.1 Section Name", "korean": ""},
          "depth": 2,
          "children": [
            {"title": {"original": "1.1.1 Subsection", "korean": ""}, "depth": 3, "children": []}
          ]
        }
      ]
    }
  ]
}

Metadata rules:
- metadata.title.original: canonical title in ORIGINAL language.
- metadata.title.korean: Korean translation only if confidently known, otherwise "".
- metadata.author: ORIGINAL-language author name only.
- metadata.publisher: ORIGINAL-language publisher name from the original edition.
- metadata.isbn13: 13-digit ISBN string with hyphens when known.

TOC rules (CRITICAL — follow strictly):
- Use web_search to find the book's ACTUAL detailed table of contents.
- For multi-volume sets or compilations, search EACH volume/book individually to get full chapter details.
- Include ALL chapters, sections, and subsections as a deeply nested tree.
- Each chapter (depth 1) MUST contain its sections (depth 2) as children.
- Sections with subsections MUST include them as children (depth 3+).
- Do NOT return only top-level chapter names with empty children.
- Do NOT summarize or compress chapters. A 500+ page book typically has 8-15+ chapters, each with 5-15 sections.
- Depth: 1=part/chapter, 2=section, 3=subsection, 4=sub-subsection.
- A typical textbook has at least 2-3 levels. Include them all.
- MINIMUM DETAIL: Each chapter MUST have at least 3-5 section-level children. Never leave children empty for a real chapter.
- Include appendices/annexes when available.
- Do NOT translate title.original. Keep in original language.
- COMPLETENESS CHECK: Every part/volume MUST have a similar level of detail. Do NOT give detailed TOC for some parts and sparse TOC for others.

General:
- If unknown, use "" for text fields and 0 for numbers.
- Return ONLY the JSON object.`
}

// BuildPartTOCPrompt creates a prompt for enriching a specific part/volume's TOC
// with section-level detail, while providing parent book context for accurate search.
func BuildPartTOCPrompt(q domain.BookQuery, parentTitle string, partTitle string, existingChapters []string) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Find the DETAILED table of contents for: \"%s\"", partTitle))
	if parentTitle != "" {
		builder.WriteString(fmt.Sprintf(" (part of \"%s\")", parentTitle))
	}
	if q.ISBN13 != "" {
		builder.WriteString(fmt.Sprintf(", parent ISBN: %s", q.ISBN13))
	}
	if q.Author != "" {
		builder.WriteString(fmt.Sprintf(", author: %s", q.Author))
	}
	if q.Lang != "" {
		builder.WriteString(fmt.Sprintf(" lang=%s", q.Lang))
	}
	builder.WriteString("\n\nKnown chapters in this part:\n")
	for _, ch := range existingChapters {
		builder.WriteString(fmt.Sprintf("- %s\n", ch))
	}
	builder.WriteString(`

CRITICAL: Return ONLY the chapters listed above. Do NOT add new chapters.
Do NOT include front matter (Preface, Getting Started, About, How to Use, etc.).
Return ONLY valid JSON array:
[{"title":{"original":"Chapter Name","korean":""},"depth":1,"children":[{"title":{"original":"1.1 Section","korean":""},"depth":2,"children":[]}]}]

Rules:
- Use web_search to find actual section details for each chapter.
- Output MUST contain EXACTLY the same chapters listed above, in the same order.
- Do NOT add, remove, rename, or merge chapters. Only add children to them.
- Do NOT include introductory/meta chapters (Preface, About, Getting Started, How to Use This Book, etc.).
- Add 3-10 sections per chapter as children (depth 2).
- Add subsections as depth 3 children if available.
- title.original: keep in ORIGINAL language. Do NOT translate.
- title.korean: Korean translation only if confidently known, otherwise "".
- Return ONLY the JSON array.`)
	return builder.String()
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
