package validator

import "testing"

func TestValidatorSchemaChecks(t *testing.T) {
	v := NewJSONValidator()

	if _, err := v.ValidateTOCJSON([]byte(`"bad"`)); err == nil {
		t.Fatalf("toc invalid structure must fail")
	}
	if _, err := v.ValidateMetadataJSON([]byte(`{"author":"x"}`)); err == nil {
		t.Fatalf("metadata missing required fields must fail")
	}
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
