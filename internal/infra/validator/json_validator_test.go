package validator

import (
	"strings"
	"testing"
)

func TestValidateMetadataJSON_AuthorRequired(t *testing.T) {
	v := NewJSONValidator()

	if _, err := v.ValidateMetadataJSON([]byte(`{"author":"x"}`)); err != nil {
		t.Fatalf("author-only metadata should pass: %v", err)
	}
	if _, err := v.ValidateMetadataJSON([]byte(`{"publisher":"p","isbn13":"i"}`)); err == nil {
		t.Fatalf("metadata without author must fail")
	} else if !strings.Contains(err.Error(), "author is required") {
		t.Fatalf("unexpected metadata error: %v", err)
	}
}

func TestValidateTOCJSON_AcceptsDirectArray(t *testing.T) {
	v := NewJSONValidator()

	nodes, err := v.ValidateTOCJSON([]byte(`[{"title":{"original":"Chapter 1"},"depth":1,"children":[]}]`))
	if err != nil {
		t.Fatalf("direct array toc should pass: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 toc node, got %d", len(nodes))
	}
}

func TestValidateTOCJSON_AcceptsWrapperKeys(t *testing.T) {
	v := NewJSONValidator()

	cases := []struct {
		name string
		raw  string
	}{
		{name: "table_of_contents", raw: `{"table_of_contents":[{"depth":1}]}`},
		{name: "chapters", raw: `{"chapters":[{"depth":1}]}`},
		{name: "toc", raw: `{"toc":[{"depth":1}]}`},
		{name: "items", raw: `{"items":[{"depth":1}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := v.ValidateTOCJSON([]byte(tc.raw))
			if err != nil {
				t.Fatalf("wrapper key %s should pass: %v", tc.name, err)
			}
			if len(nodes) != 1 {
				t.Fatalf("expected 1 toc node for key %s, got %d", tc.name, len(nodes))
			}
		})
	}
}

func TestValidateTOCJSON_ReturnsClearSchemaErrors(t *testing.T) {
	v := NewJSONValidator()

	if _, err := v.ValidateTOCJSON([]byte(`"bad"`)); err == nil {
		t.Fatalf("toc invalid json structure must fail")
	} else if !strings.Contains(err.Error(), "schema error: toc json parse") {
		t.Fatalf("expected parse schema error, got: %v", err)
	}

	if _, err := v.ValidateTOCJSON([]byte(`{"toc":{}}`)); err == nil {
		t.Fatalf("toc wrapper non-array must fail")
	} else if !strings.Contains(err.Error(), `schema error: toc "toc" must be array`) {
		t.Fatalf("expected wrapper schema error, got: %v", err)
	}

	if _, err := v.ValidateTOCJSON([]byte(`{"unknown":[]}`)); err == nil {
		t.Fatalf("toc missing known wrapper key must fail")
	} else if !strings.Contains(err.Error(), "schema error: toc wrapper key missing") {
		t.Fatalf("expected missing wrapper key schema error, got: %v", err)
	}
}

func TestValidatorSchemaChecks(t *testing.T) {
	v := NewJSONValidator()

	if _, err := v.ValidateReviewJSON([]byte(`{"rating":10,"summary":{"pros":[],"cons":[]}}`)); err == nil {
		t.Fatalf("review out of range must fail")
	}
	if _, err := v.ValidateCoursesJSON([]byte(`[{"title":"x","curriculum":null}]`)); err == nil {
		t.Fatalf("courses curriculum null must fail")
	}
	if _, err := v.ValidateSimilarBooksJSON([]byte(`[{"title":{"original":"a","korean":"b"}}]`)); err == nil {
		t.Fatalf("similar count outside 3..5 must fail")
	}
}
