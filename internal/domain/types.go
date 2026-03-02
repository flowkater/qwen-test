package domain

import (
	"encoding/json"
	"time"
)

const DefaultModel = "qwen/qwen3.5-flash-02-23"
const DefaultOutputFormat = "json"

type BookQuery struct {
	Title     string
	ISBN13    string
	Author    string
	Lang      string
	Format    string
	Model     string
	Output    string
	NoCache   bool
	FullMode  bool
	BatchPath string
	APIKey    string
}

func (q BookQuery) Mode() string {
	if q.FullMode {
		return "full"
	}
	return "basic"
}

func (q BookQuery) PreferredIdentifier() string {
	if q.ISBN13 != "" {
		return q.ISBN13
	}
	return q.Title
}

type BookInfo struct {
	Book             BookMetadata     `json:"book"`
	TableOfContents  []TOCNode        `json:"table_of_contents"`
	Review           ReviewInfo       `json:"review"`
	RelatedCourses   []RelatedCourse  `json:"related_courses"`
	SimilarBooks     []SimilarBook    `json:"similar_books"`
	DataAvailability DataAvailability `json:"data_availability"`
	Metadata         CollectMetadata  `json:"metadata"`
}

func (b BookInfo) MarshalJSON() ([]byte, error) {
	type alias BookInfo
	out := alias(b)
	if out.TableOfContents == nil {
		out.TableOfContents = []TOCNode{}
	}
	if out.RelatedCourses == nil {
		out.RelatedCourses = []RelatedCourse{}
	}
	if out.SimilarBooks == nil {
		out.SimilarBooks = []SimilarBook{}
	}
	if out.Review.Prerequisites == nil {
		out.Review.Prerequisites = []string{}
	}
	if out.Review.Summary.Pros == nil {
		out.Review.Summary.Pros = []string{}
	}
	if out.Review.Summary.Cons == nil {
		out.Review.Summary.Cons = []string{}
	}
	return json.Marshal(out)
}

type BookMetadata struct {
	Title         LocalizedTitle `json:"title"`
	Author        string         `json:"author"`
	Publisher     string         `json:"publisher"`
	PublishedDate string         `json:"published_date"`
	ISBN13        string         `json:"isbn13"`
	Pages         int            `json:"pages"`
	Language      string         `json:"language"`
	Edition       string         `json:"edition"`
	SelectionNote string         `json:"selection_note"`
}

type LocalizedTitle struct {
	Original string `json:"original"`
	Korean   string `json:"korean"`
}

type TOCNode struct {
	Title    LocalizedTitle `json:"title"`
	Depth    int            `json:"depth"`
	Children []TOCNode      `json:"children"`
}

func (n TOCNode) MarshalJSON() ([]byte, error) {
	type alias TOCNode
	out := alias(n)
	if out.Children == nil {
		out.Children = []TOCNode{}
	}
	return json.Marshal(out)
}

type ReviewInfo struct {
	Rating           float64       `json:"rating"`
	Summary          ReviewSummary `json:"summary"`
	RecommendedLevel string        `json:"recommended_level"`
	Prerequisites    []string      `json:"prerequisites"`
}

type ReviewSummary struct {
	Pros []string `json:"pros"`
	Cons []string `json:"cons"`
}

type RelatedCourse struct {
	Title      string   `json:"title"`
	Platform   string   `json:"platform"`
	Instructor string   `json:"instructor"`
	Curriculum []string `json:"curriculum"`
	Rating     float64  `json:"rating"`
	Price      string   `json:"price"`
	URL        string   `json:"url"`
}

type SimilarBook struct {
	Title                LocalizedTitle `json:"title"`
	Author               string         `json:"author"`
	BriefDescription     string         `json:"brief_description"`
	DifficultyComparison string         `json:"difficulty_comparison"`
}

type DataAvailability struct {
	TableOfContents bool `json:"table_of_contents"`
	Review          bool `json:"review"`
	RelatedCourses  bool `json:"related_courses"`
	SimilarBooks    bool `json:"similar_books"`
}

type CollectQuery struct {
	Title  string `json:"title"`
	ISBN13 string `json:"isbn13"`
	Author string `json:"author"`
	Lang   string `json:"lang"`
}

type CollectMetadata struct {
	CollectedAt time.Time    `json:"-"`
	Model       string       `json:"model"`
	Source      string       `json:"source"`
	Mode        string       `json:"mode"`
	Query       CollectQuery `json:"query"`
}

func (m CollectMetadata) MarshalJSON() ([]byte, error) {
	type alias CollectMetadata
	out := struct {
		CollectedAt string `json:"collected_at"`
		alias
	}{
		CollectedAt: m.CollectedAt.UTC().Format(time.RFC3339),
		alias:       alias(m),
	}
	return json.Marshal(out)
}

type CacheEntry struct {
	Key       string    `json:"key"`
	Value     BookInfo  `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type BatchItemResult struct {
	Query      string `json:"query"`
	Success    bool   `json:"success"`
	OutputPath string `json:"output_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

type BackoffConfig struct {
	InitialDelay time.Duration
	Multiplier   int
	MaxJitter    time.Duration
	MaxDelay     time.Duration
	MaxRetries   int
}

func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		InitialDelay: 2 * time.Second,
		Multiplier:   2,
		MaxJitter:    time.Second,
		MaxDelay:     30 * time.Second,
		MaxRetries:   3,
	}
}
