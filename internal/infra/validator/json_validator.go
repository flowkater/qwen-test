package validator

import (
	"encoding/json"
	"fmt"

	"github.com/flowkater/qwen/bookinfo/internal/domain"
)

type JSONValidator struct{}

func NewJSONValidator() *JSONValidator { return &JSONValidator{} }

func (v *JSONValidator) ValidateMetadataJSON(raw []byte) (domain.BookMetadata, error) {
	var payload domain.BookMetadata
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.BookMetadata{}, fmt.Errorf("schema error: metadata json parse: %w", err)
	}
	if payload.ISBN13 == "" || payload.Author == "" || payload.Publisher == "" {
		return domain.BookMetadata{}, fmt.Errorf("schema error: metadata required fields missing")
	}
	return payload, nil
}

func (v *JSONValidator) ValidateTOCJSON(raw []byte) ([]domain.TOCNode, error) {
	var arr []domain.TOCNode
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var obj struct {
		TableOfContents []domain.TOCNode `json:"table_of_contents"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("schema error: toc structure")
	}
	return obj.TableOfContents, nil
}

func (v *JSONValidator) ValidateReviewJSON(raw []byte) (domain.ReviewInfo, error) {
	var item domain.ReviewInfo
	if err := json.Unmarshal(raw, &item); err != nil {
		return domain.ReviewInfo{}, fmt.Errorf("schema error: review parse")
	}
	if item.Rating < 0 || item.Rating > 5 {
		return domain.ReviewInfo{}, fmt.Errorf("schema error: review rating out of range")
	}
	if item.Summary.Pros == nil || item.Summary.Cons == nil {
		return domain.ReviewInfo{}, fmt.Errorf("schema error: review pros/cons must be array")
	}
	return item, nil
}

func (v *JSONValidator) ValidateCoursesJSON(raw []byte) ([]domain.RelatedCourse, error) {
	var items []domain.RelatedCourse
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("schema error: courses parse")
	}
	for _, item := range items {
		if item.Curriculum == nil {
			return nil, fmt.Errorf("schema error: curriculum must be array")
		}
	}
	return items, nil
}

func (v *JSONValidator) ValidateSimilarBooksJSON(raw []byte) ([]domain.SimilarBook, error) {
	var items []domain.SimilarBook
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("schema error: similar parse")
	}
	if len(items) < 3 || len(items) > 5 {
		return nil, fmt.Errorf("schema error: similar items count must be 3..5")
	}
	return items, nil
}
