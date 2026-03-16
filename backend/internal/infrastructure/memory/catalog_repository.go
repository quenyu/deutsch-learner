package memory

import (
	"context"
	"strings"
	"time"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
)

type CatalogRepository struct {
	resources []domaincatalog.Resource
}

func NewCatalogRepository(resources []domaincatalog.Resource) *CatalogRepository {
	copied := make([]domaincatalog.Resource, 0, len(resources))
	for _, resource := range resources {
		copied = append(copied, cloneResource(resource))
	}

	return &CatalogRepository{
		resources: copied,
	}
}

func DefaultResources() []domaincatalog.Resource {
	now := time.Now().UTC()

	return []domaincatalog.Resource{
		{
			ID:              "c01f9204-5f38-47e4-b3ec-c580691ff44f",
			Slug:            "dw-nicos-weg-a1-overview",
			Title:           "Nicos Weg A1 Overview",
			Summary:         "Structured beginner-friendly entry into practical German for daily communication.",
			SourceName:      "Deutsche Welle",
			SourceType:      domaincatalog.ResourceTypeCourse,
			ExternalURL:     "https://learngerman.dw.com/en/nicos-weg/c-36519687",
			CEFRLevel:       domaincatalog.CEFRLevelA1,
			Format:          "course",
			DurationMinutes: 40,
			IsFree:          true,
			SkillTags:       []string{"listening", "speaking"},
			TopicTags:       []string{"daily-life", "introductions"},
			ProviderSlug:    "manual",
			ProviderName:    "Manual Curation",
			IngestionOrigin: "manual",
			SourceKind:      "course",
			CreatedAt:       now.Add(-72 * time.Hour),
			UpdatedAt:       now.Add(-24 * time.Hour),
		},
		{
			ID:              "4c55f202-c4c5-43c8-98ac-3ccf490f4b7f",
			Slug:            "easy-german-street-interviews-a2",
			Title:           "Easy German Street Interviews",
			Summary:         "Natural spoken German with subtitles and context from real-world conversations.",
			SourceName:      "Easy German",
			SourceType:      domaincatalog.ResourceTypeYouTube,
			ExternalURL:     "https://www.youtube.com/@EasyGerman",
			CEFRLevel:       domaincatalog.CEFRLevelA2,
			Format:          "video",
			DurationMinutes: 18,
			IsFree:          true,
			SkillTags:       []string{"listening", "vocabulary"},
			TopicTags:       []string{"culture", "daily-life"},
			ProviderSlug:    "manual",
			ProviderName:    "Manual Curation",
			IngestionOrigin: "manual",
			SourceKind:      "video",
			CreatedAt:       now.Add(-96 * time.Hour),
			UpdatedAt:       now.Add(-36 * time.Hour),
		},
		{
			ID:              "ad7a4a03-3ee8-4ec1-a21a-b9ca3ee2676b",
			Slug:            "yourdailygerman-case-system-guide",
			Title:           "German Case System Guide",
			Summary:         "Clear explanation of nominative, accusative, dative, and genitive with examples.",
			SourceName:      "Your Daily German",
			SourceType:      domaincatalog.ResourceTypeArticle,
			ExternalURL:     "https://yourdailygerman.com/german-case-system/",
			CEFRLevel:       domaincatalog.CEFRLevelB1,
			Format:          "article",
			DurationMinutes: 25,
			IsFree:          true,
			SkillTags:       []string{"grammar", "reading"},
			TopicTags:       []string{"grammar", "cases"},
			ProviderSlug:    "manual",
			ProviderName:    "Manual Curation",
			IngestionOrigin: "manual",
			SourceKind:      "article",
			CreatedAt:       now.Add(-120 * time.Hour),
			UpdatedAt:       now.Add(-48 * time.Hour),
		},
		{
			ID:              "c58d0f53-fc1f-41d4-a8a0-91f77f74895d",
			Slug:            "langsam-gesprochene-nachrichten-b2",
			Title:           "Langsam Gesprochene Nachrichten",
			Summary:         "Slow-paced news audio that helps intermediate learners build listening confidence.",
			SourceName:      "Deutsche Welle",
			SourceType:      domaincatalog.ResourceTypePodcast,
			ExternalURL:     "https://www.dw.com/de/deutsch-lernen/nachrichten/s-8030",
			CEFRLevel:       domaincatalog.CEFRLevelB2,
			Format:          "podcast",
			DurationMinutes: 12,
			IsFree:          true,
			SkillTags:       []string{"listening", "news"},
			TopicTags:       []string{"current-events"},
			ProviderSlug:    "manual",
			ProviderName:    "Manual Curation",
			IngestionOrigin: "manual",
			SourceKind:      "podcast",
			CreatedAt:       now.Add(-48 * time.Hour),
			UpdatedAt:       now.Add(-12 * time.Hour),
		},
		{
			ID:              "2828d81f-45bb-4410-8653-df2a1d4e657d",
			Slug:            "goethe-online-live-group-course",
			Title:           "Goethe Online Live Group Course",
			Summary:         "Instructor-led classes with structured progression and collaborative exercises.",
			SourceName:      "Goethe-Institut",
			SourceType:      domaincatalog.ResourceTypeCourse,
			ExternalURL:     "https://www.goethe.de/en/spr/kur/typ.html",
			CEFRLevel:       domaincatalog.CEFRLevelB1,
			Format:          "course",
			DurationMinutes: 90,
			IsFree:          false,
			PriceCents:      intPtr(24900),
			SkillTags:       []string{"speaking", "grammar"},
			TopicTags:       []string{"exam-prep", "conversation"},
			ProviderSlug:    "manual",
			ProviderName:    "Manual Curation",
			IngestionOrigin: "manual",
			SourceKind:      "course",
			CreatedAt:       now.Add(-24 * time.Hour),
			UpdatedAt:       now.Add(-6 * time.Hour),
		},
	}
}

func (r *CatalogRepository) ListResources(_ context.Context, filter domaincatalog.ListFilter) ([]domaincatalog.Resource, error) {
	filter = filter.WithDefaults()
	filtered := make([]domaincatalog.Resource, 0, len(r.resources))
	query := strings.ToLower(filter.Query)

	for _, resource := range r.resources {
		if filter.Level != "" && string(resource.CEFRLevel) != filter.Level {
			continue
		}
		if filter.Skill != "" && !hasTag(resource.SkillTags, filter.Skill) {
			continue
		}
		if filter.Topic != "" && !hasTag(resource.TopicTags, filter.Topic) {
			continue
		}
		if filter.Provider != "" && !strings.EqualFold(resource.ProviderSlug, filter.Provider) {
			continue
		}
		if filter.Type != "" && !strings.EqualFold(string(resource.SourceType), filter.Type) {
			continue
		}
		if filter.OnlyFree != nil && resource.IsFree != *filter.OnlyFree {
			continue
		}
		if query != "" {
			searchable := strings.ToLower(resource.Title + " " + resource.Summary)
			if !strings.Contains(searchable, query) {
				continue
			}
		}

		filtered = append(filtered, cloneResource(resource))
	}

	start := filter.Offset
	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

func (r *CatalogRepository) GetResourceBySlug(_ context.Context, slug string) (domaincatalog.Resource, error) {
	for _, resource := range r.resources {
		if resource.Slug == slug {
			return cloneResource(resource), nil
		}
	}

	return domaincatalog.Resource{}, domaincatalog.ErrResourceNotFound
}

func (r *CatalogRepository) GetResourceByID(_ context.Context, id string) (domaincatalog.Resource, error) {
	for _, resource := range r.resources {
		if resource.ID == id {
			return cloneResource(resource), nil
		}
	}

	return domaincatalog.Resource{}, domaincatalog.ErrResourceNotFound
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}

	return false
}

func cloneResource(resource domaincatalog.Resource) domaincatalog.Resource {
	cloned := resource
	cloned.SkillTags = append([]string(nil), resource.SkillTags...)
	cloned.TopicTags = append([]string(nil), resource.TopicTags...)
	return cloned
}

func intPtr(v int) *int {
	return &v
}
