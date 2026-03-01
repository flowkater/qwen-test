//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "verified_source", "json", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return obj
}

func TestVerifiedSourceCleanCodeFixture(t *testing.T) {
	obj := loadFixture(t, "cleancode-en-book-toc.json")
	meta := obj["metadata"].(map[string]any)
	if meta["author"] != "Robert C. Martin" {
		t.Fatalf("author mismatch")
	}
	if meta["publisher"] != "Addison-Wesley Professional" {
		t.Fatalf("publisher mismatch")
	}
	if meta["isbn13"] != "978-0132350884" {
		t.Fatalf("isbn mismatch")
	}
	toc := obj["table_of_contents"].(map[string]any)
	if toc["total_top_level_items"] != float64(10) {
		t.Fatalf("toc top-level count mismatch: %v", toc["total_top_level_items"])
	}
	rules := obj["verification_rules"].(map[string]any)
	tocRule := rules["toc_structure_check"].(map[string]any)
	required := tocRule["chapter_titles_must_contain"].([]any)
	items := toc["items"].([]any)
	flat := flattenTitles(items)
	for _, must := range required {
		if !containsSubstring(flat, must.(string)) {
			t.Fatalf("required chapter title missing: %s", must.(string))
		}
	}
	depthRule := rules["toc_depth_check"].(map[string]any)
	if depthRule["max_depth_observed"] != float64(3) {
		t.Fatalf("depth mismatch: %v", depthRule["max_depth_observed"])
	}
}

func TestVerifiedSourceMCATFixture(t *testing.T) {
	obj := loadFixture(t, "mcat-en-book-toc.json")
	meta := obj["metadata"].(map[string]any)
	if meta["publisher"] != "Kaplan Test Prep" {
		t.Fatalf("publisher mismatch")
	}
	if meta["isbn13_biology"] != "978-1506297408" {
		t.Fatalf("biology isbn mismatch")
	}
	if meta["isbn13_biochemistry"] != "978-1506297385" {
		t.Fatalf("biochemistry isbn mismatch")
	}
	toc := obj["table_of_contents"].(map[string]any)
	if toc["total_books"] != float64(2) {
		t.Fatalf("expected two books in toc")
	}
	books := toc["books"].([]any)
	if len(books) != 2 {
		t.Fatalf("books length mismatch")
	}
	bio := books[0].(map[string]any)
	chapters := bio["chapters"].([]any)
	if len(chapters) != 8 {
		t.Fatalf("biology chapter count mismatch: %d", len(chapters))
	}
	ch7 := chapters[6].(map[string]any)
	if !strings.Contains(ch7["title"].(string), "Human Anatomy and Physiology") {
		t.Fatalf("chapter 7 title mismatch: %s", ch7["title"])
	}
}

func TestVerifiedSourceInflearnFixture(t *testing.T) {
	obj := loadFixture(t, "inflearn-system-kr.json")
	meta := obj["metadata"].(map[string]any)
	if meta["instructor"] != "mindlantern" {
		t.Fatalf("instructor mismatch")
	}
	if meta["rating"] != 4.9 {
		t.Fatalf("rating mismatch")
	}
	if meta["total_lectures"] != float64(24) {
		t.Fatalf("total lectures mismatch")
	}
	curr := obj["curriculum"].(map[string]any)
	if curr["total_sections"] != float64(4) {
		t.Fatalf("section count mismatch")
	}
	sections := curr["sections"].([]any)
	wantCounts := []float64{4, 6, 9, 5}
	for i, wc := range wantCounts {
		sec := sections[i].(map[string]any)
		if sec["lecture_count"] != wc {
			t.Fatalf("section %d lecture_count mismatch: %v", i+1, sec["lecture_count"])
		}
	}
	third := sections[2].(map[string]any)
	lectures := third["lectures"].([]any)
	found37, found33 := false, false
	for _, l := range lectures {
		lec := l.(map[string]any)
		if lec["number"] == "3.7" {
			found37 = lec["title"] == "Message Queue & Event Broker" && lec["duration"] == "32:12"
		}
		if lec["number"] == "3.3" {
			found33 = lec["title"] == "API Gateway & Load Balancer & Service Discovery" && lec["duration"] == "16:01"
		}
	}
	if !found37 || !found33 {
		t.Fatalf("expected key lectures 3.3/3.7 exact match")
	}
}

func TestVerifiedSourceRealDealFixture(t *testing.T) {
	obj := loadFixture(t, "realdealclass-kr-lecture.json")
	meta := obj["metadata"].(map[string]any)
	if meta["platform"] != "RealDealClass" {
		t.Fatalf("platform mismatch")
	}
	if meta["difficulty"] != "(왕)초급 - 초중급" {
		t.Fatalf("difficulty mismatch")
	}
	curr := obj["curriculum"].(map[string]any)
	if curr["has_special_lessons"] != true {
		t.Fatalf("expected special lessons")
	}
	sections := curr["sections"].([]any)
	requiredTitles := []string{"문장의 형태", "동사의 종류", "동사의 시제", "수동태", "준동사 & 가짜 동사", "긴 문장 만드는 원리", "풍부한 문장 완성하기"}
	flatTitles := []string{}
	for _, sec := range sections {
		flatTitles = append(flatTitles, sec.(map[string]any)["title"].(string))
	}
	for _, rt := range requiredTitles {
		if !containsSubstring(flatTitles, rt) {
			t.Fatalf("missing chapter title keyword %q", rt)
		}
	}
	lec1, lec27 := false, false
	for _, sec := range sections {
		s := sec.(map[string]any)
		if lecs, ok := s["lectures"].([]any); ok {
			for _, li := range lecs {
				l := li.(map[string]any)
				if l["number"] == float64(1) {
					lec1 = l["title"] == "문법의 개념" && l["duration"] == "37:23"
				}
				if l["number"] == float64(27) {
					lec27 = l["title"] == "to-V 1" && l["duration"] == "42:20"
				}
			}
		}
	}
	if !lec1 || !lec27 {
		t.Fatalf("lecture 1/27 exact checks failed")
	}
}

func flattenTitles(items []any) []string {
	var out []string
	var walk func([]any)
	walk = func(nodes []any) {
		for _, n := range nodes {
			m := n.(map[string]any)
			if v, ok := m["title"].(string); ok {
				out = append(out, v)
			}
			if children, ok := m["children"].([]any); ok {
				walk(children)
			}
			if titles, ok := m["children_titles"].([]any); ok {
				for _, t := range titles {
					out = append(out, t.(string))
				}
			}
		}
	}
	walk(items)
	return out
}

func containsSubstring(hay []string, needle string) bool {
	for _, h := range hay {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
