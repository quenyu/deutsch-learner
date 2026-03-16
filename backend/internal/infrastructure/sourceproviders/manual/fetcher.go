package manual

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

type Fetcher struct{}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

func (f *Fetcher) ProviderSlug() domainsource.ProviderSlug {
	return domainsource.ProviderManual
}

func (f *Fetcher) Fetch(_ context.Context, request sourceimportapp.FetchRequest) ([]sourceimportapp.FetchedResource, error) {
	if strings.TrimSpace(request.Mode) != "file" {
		return nil, fmt.Errorf("manual provider supports only mode=file")
	}
	if strings.TrimSpace(request.FilePath) == "" {
		return nil, fmt.Errorf("file path is required for manual import")
	}

	content, err := os.ReadFile(request.FilePath)
	if err != nil {
		return nil, err
	}

	var entries []manualResource
	if err := json.Unmarshal(content, &entries); err != nil {
		var wrapped struct {
			Items []manualResource `json:"items"`
		}
		if wrapErr := json.Unmarshal(content, &wrapped); wrapErr != nil {
			return nil, err
		}
		entries = wrapped.Items
	}

	result := make([]sourceimportapp.FetchedResource, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.ExternalID) == "" {
			entry.ExternalID = entry.Slug
		}
		if strings.TrimSpace(entry.ExternalID) == "" {
			entry.ExternalID = entry.ExternalURL
		}

		sourceType := parseSourceType(entry.SourceType)
		sourceKind := parseSourceKind(entry.SourceKind, sourceType)
		payload, _ := json.Marshal(entry)

		resource := sourceimportapp.FetchedResource{
			ExternalID:      strings.TrimSpace(entry.ExternalID),
			ExternalURL:     strings.TrimSpace(entry.ExternalURL),
			SourceKind:      sourceKind,
			SourceType:      sourceType,
			Title:           strings.TrimSpace(entry.Title),
			Summary:         strings.TrimSpace(entry.Summary),
			SourceName:      strings.TrimSpace(entry.SourceName),
			AuthorName:      strings.TrimSpace(entry.AuthorName),
			Format:          strings.TrimSpace(entry.Format),
			DurationMinutes: entry.DurationMinutes,
			IsFree:          entry.IsFree,
			SkillTags:       entry.SkillTags,
			TopicTags:       entry.TopicTags,
			LanguageCode:    strings.TrimSpace(entry.LanguageCode),
			RawPayload:      payload,
		}

		if resource.DurationMinutes < 0 {
			resource.DurationMinutes = 0
		}

		result = append(result, resource)
	}

	return result, nil
}

type manualResource struct {
	ExternalID      string   `json:"externalId"`
	Slug            string   `json:"slug"`
	ExternalURL     string   `json:"externalUrl"`
	SourceType      string   `json:"sourceType"`
	SourceKind      string   `json:"sourceKind"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	SourceName      string   `json:"sourceName"`
	AuthorName      string   `json:"authorName"`
	Format          string   `json:"format"`
	DurationMinutes int      `json:"durationMinutes"`
	IsFree          bool     `json:"isFree"`
	SkillTags       []string `json:"skillTags"`
	TopicTags       []string `json:"topicTags"`
	LanguageCode    string   `json:"languageCode"`
}

func parseSourceType(raw string) domaincatalog.ResourceType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domaincatalog.ResourceTypeArticle):
		return domaincatalog.ResourceTypeArticle
	case string(domaincatalog.ResourceTypePlaylist):
		return domaincatalog.ResourceTypePlaylist
	case string(domaincatalog.ResourceTypeCourse):
		return domaincatalog.ResourceTypeCourse
	case string(domaincatalog.ResourceTypePodcast):
		return domaincatalog.ResourceTypePodcast
	case string(domaincatalog.ResourceTypeGrammarReference):
		return domaincatalog.ResourceTypeGrammarReference
	case string(domaincatalog.ResourceTypeExercise):
		return domaincatalog.ResourceTypeExercise
	default:
		return domaincatalog.ResourceTypeYouTube
	}
}

func parseSourceKind(raw string, sourceType domaincatalog.ResourceType) domainsource.SourceKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domainsource.SourceKindVideo):
		return domainsource.SourceKindVideo
	case string(domainsource.SourceKindPlaylist):
		return domainsource.SourceKindPlaylist
	case string(domainsource.SourceKindChannel):
		return domainsource.SourceKindChannel
	case string(domainsource.SourceKindCourse):
		return domainsource.SourceKindCourse
	case string(domainsource.SourceKindLesson):
		return domainsource.SourceKindLesson
	case string(domainsource.SourceKindArticle):
		return domainsource.SourceKindArticle
	case string(domainsource.SourceKindPodcast):
		return domainsource.SourceKindPodcast
	case string(domainsource.SourceKindGrammarReference):
		return domainsource.SourceKindGrammarReference
	case string(domainsource.SourceKindExercise):
		return domainsource.SourceKindExercise
	case string(domainsource.SourceKindExternalLink):
		return domainsource.SourceKindExternalLink
	default:
		switch sourceType {
		case domaincatalog.ResourceTypePlaylist:
			return domainsource.SourceKindPlaylist
		case domaincatalog.ResourceTypeCourse:
			return domainsource.SourceKindCourse
		case domaincatalog.ResourceTypeArticle:
			return domainsource.SourceKindArticle
		case domaincatalog.ResourceTypePodcast:
			return domainsource.SourceKindPodcast
		case domaincatalog.ResourceTypeGrammarReference:
			return domainsource.SourceKindGrammarReference
		case domaincatalog.ResourceTypeExercise:
			return domainsource.SourceKindExercise
		default:
			return domainsource.SourceKindVideo
		}
	}
}
