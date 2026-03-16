package catalog

import (
	"errors"
	"strings"
	"time"
)

var ErrResourceNotFound = errors.New("resource not found")

type ResourceType string

const (
	ResourceTypeYouTube          ResourceType = "youtube"
	ResourceTypeArticle          ResourceType = "article"
	ResourceTypePlaylist         ResourceType = "playlist"
	ResourceTypeCourse           ResourceType = "course"
	ResourceTypePodcast          ResourceType = "podcast"
	ResourceTypeGrammarReference ResourceType = "grammar_reference"
	ResourceTypeExercise         ResourceType = "exercise"
)

type CEFRLevel string

const (
	CEFRLevelA1 CEFRLevel = "A1"
	CEFRLevelA2 CEFRLevel = "A2"
	CEFRLevelB1 CEFRLevel = "B1"
	CEFRLevelB2 CEFRLevel = "B2"
	CEFRLevelC1 CEFRLevel = "C1"
	CEFRLevelC2 CEFRLevel = "C2"
)

type Resource struct {
	ID              string       `json:"id"`
	Slug            string       `json:"slug"`
	Title           string       `json:"title"`
	Summary         string       `json:"summary"`
	SourceName      string       `json:"sourceName"`
	SourceType      ResourceType `json:"sourceType"`
	ExternalURL     string       `json:"externalUrl"`
	CEFRLevel       CEFRLevel    `json:"cefrLevel"`
	Format          string       `json:"format"`
	DurationMinutes int          `json:"durationMinutes"`
	IsFree          bool         `json:"isFree"`
	PriceCents      *int         `json:"priceCents,omitempty"`
	SkillTags       []string     `json:"skillTags"`
	TopicTags       []string     `json:"topicTags"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

type ListFilter struct {
	Level    string
	Skill    string
	Topic    string
	Query    string
	OnlyFree *bool
	Limit    int
	Offset   int
}

func (f ListFilter) WithDefaults() ListFilter {
	f.Level = strings.ToUpper(strings.TrimSpace(f.Level))
	f.Skill = strings.ToLower(strings.TrimSpace(f.Skill))
	f.Topic = strings.ToLower(strings.TrimSpace(f.Topic))
	f.Query = strings.TrimSpace(f.Query)

	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 24
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	return f
}
