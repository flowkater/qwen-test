package domain

import (
	"fmt"
	"time"
)

type CollectedParts struct {
	Metadata      BookMetadata
	TOC           []TOCNode
	Review        *ReviewInfo
	RelatedCourse []RelatedCourse
	SimilarBooks  []SimilarBook
}

func MergeBookInfo(q BookQuery, parts CollectedParts, now time.Time) (BookInfo, error) {
	result := BookInfo{
		Book:            parts.Metadata,
		TableOfContents: normalizeTOC(parts.TOC),
		Metadata: CollectMetadata{
			CollectedAt: now.UTC(),
			Model:       q.Model,
			Source:      "openrouter",
			Mode:        q.Mode(),
			Query: CollectQuery{
				Title:  q.Title,
				ISBN13: q.ISBN13,
				Author: q.Author,
				Lang:   q.Lang,
			},
		},
	}
	if result.Metadata.Model == "" {
		result.Metadata.Model = DefaultModel
	}

	if parts.Review != nil {
		result.Review = *parts.Review
	}
	if parts.RelatedCourse != nil {
		result.RelatedCourses = parts.RelatedCourse
	}
	if parts.SimilarBooks != nil {
		result.SimilarBooks = parts.SimilarBooks
	}

	result.DataAvailability = DataAvailability{
		TableOfContents: len(result.TableOfContents) > 0,
		Review:          parts.Review != nil && parts.Review.Rating > 0,
		RelatedCourses:  len(result.RelatedCourses) > 0,
		SimilarBooks:    len(result.SimilarBooks) > 0,
	}

	if !q.FullMode {
		result.Review = ReviewInfo{
			Summary: ReviewSummary{
				Pros: []string{},
				Cons: []string{},
			},
			Prerequisites: []string{},
		}
		result.RelatedCourses = []RelatedCourse{}
		result.SimilarBooks = []SimilarBook{}
		result.DataAvailability.Review = false
		result.DataAvailability.RelatedCourses = false
		result.DataAvailability.SimilarBooks = false
	}

	if q.FullMode && len(result.SimilarBooks) > 0 {
		if len(result.SimilarBooks) < 3 || len(result.SimilarBooks) > 5 {
			return BookInfo{}, fmt.Errorf("similar books count must be 3..5")
		}
	}
	if q.FullMode && parts.Review != nil {
		if parts.Review.Rating < 0 || parts.Review.Rating > 5 {
			return BookInfo{}, fmt.Errorf("rating out of range")
		}
	}

	return result, validateTOCTree(result.TableOfContents)
}

func normalizeTOC(nodes []TOCNode) []TOCNode {
	if nodes == nil {
		return []TOCNode{}
	}
	out := make([]TOCNode, len(nodes))
	for i, n := range nodes {
		out[i] = n
		if out[i].Children == nil {
			out[i].Children = []TOCNode{}
		} else {
			out[i].Children = normalizeTOC(out[i].Children)
		}
	}
	return out
}

func validateTOCTree(nodes []TOCNode) error {
	for _, n := range nodes {
		if n.Title.Original == "" || n.Title.Korean == "" {
			return fmt.Errorf("toc title must include original and korean")
		}
		for _, c := range n.Children {
			if c.Depth <= n.Depth {
				return fmt.Errorf("toc depth must increase by child")
			}
		}
		if err := validateTOCTree(n.Children); err != nil {
			return err
		}
	}
	return nil
}
