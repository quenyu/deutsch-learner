package sourceimportapp

import (
	"context"
	"fmt"
	"testing"
	"time"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

type fakeImportRepository struct {
	providers       map[string]domainsource.Provider
	sourceRecords   map[string]domainsource.Record
	catalogBySource map[string]domaincatalog.Resource
	jobStatus       domainsource.ImportJobStatus
	resultsCount    int
}

func newFakeImportRepository() *fakeImportRepository {
	return &fakeImportRepository{
		providers: map[string]domainsource.Provider{
			"youtube": {
				ID:        "provider-youtube",
				Slug:      domainsource.ProviderYouTube,
				Name:      "YouTube Data API",
				IsEnabled: true,
			},
		},
		sourceRecords:   make(map[string]domainsource.Record),
		catalogBySource: make(map[string]domaincatalog.Resource),
	}
}

func (r *fakeImportRepository) GetProviderBySlug(_ context.Context, slug string) (domainsource.Provider, error) {
	provider, ok := r.providers[slug]
	if !ok {
		return domainsource.Provider{}, fmt.Errorf("provider not found: %s", slug)
	}
	return provider, nil
}

func (r *fakeImportRepository) CreateImportJob(_ context.Context, input CreateImportJobInput) (domainsource.ImportJob, error) {
	return domainsource.ImportJob{
		ID:         "import-job-1",
		ProviderID: input.ProviderID,
		Mode:       input.Mode,
		Status:     domainsource.ImportJobRunning,
	}, nil
}

func (r *fakeImportRepository) CompleteImportJob(_ context.Context, _ string, status domainsource.ImportJobStatus, _ string) error {
	r.jobStatus = status
	return nil
}

func (r *fakeImportRepository) UpsertSourceRecord(_ context.Context, input UpsertSourceRecordInput) (domainsource.Record, bool, error) {
	key := fmt.Sprintf("%s|%s|%s", input.ProviderSlug, input.ExternalID, input.SourceKind)
	if existing, ok := r.sourceRecords[key]; ok {
		existing.Title = input.Title
		existing.Summary = input.Summary
		r.sourceRecords[key] = existing
		return existing, false, nil
	}

	record := domainsource.Record{
		ID:           "source-" + input.ExternalID,
		ProviderID:   input.ProviderID,
		ProviderSlug: input.ProviderSlug,
		ExternalID:   input.ExternalID,
		ExternalURL:  input.ExternalURL,
		SourceKind:   input.SourceKind,
		Title:        input.Title,
		Summary:      input.Summary,
		AuthorName:   input.AuthorName,
		LanguageCode: input.LanguageCode,
		LastSyncedAt: timePtr(time.Now().UTC()),
	}
	r.sourceRecords[key] = record
	return record, true, nil
}

func (r *fakeImportRepository) UpsertCatalogResource(_ context.Context, input UpsertCatalogResourceInput) (domaincatalog.Resource, bool, error) {
	if existing, ok := r.catalogBySource[input.SourceRecordID]; ok {
		existing.Title = input.Title
		existing.Summary = input.Summary
		existing.CEFRLevel = domaincatalog.CEFRLevel(input.Level)
		existing.SourceKind = string(domainsource.SourceKindVideo)
		r.catalogBySource[input.SourceRecordID] = existing
		return existing, false, nil
	}

	resource := domaincatalog.Resource{
		ID:              "resource-" + input.SourceRecordID,
		Slug:            "slug-" + input.SourceRecordID,
		Title:           input.Title,
		Summary:         input.Summary,
		SourceName:      input.SourceName,
		SourceType:      input.SourceType,
		ExternalURL:     input.ExternalURL,
		CEFRLevel:       domaincatalog.CEFRLevel(input.Level),
		Format:          input.Format,
		DurationMinutes: input.DurationMinutes,
		IsFree:          input.IsFree,
		SkillTags:       append([]string(nil), input.SkillTags...),
		TopicTags:       append([]string(nil), input.TopicTags...),
		IngestionOrigin: string(input.IngestionOrigin),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	r.catalogBySource[input.SourceRecordID] = resource
	return resource, true, nil
}

func (r *fakeImportRepository) CreateImportResult(_ context.Context, _ CreateImportResultInput) error {
	r.resultsCount++
	return nil
}

type fakeFetcher struct {
	slug      domainsource.ProviderSlug
	resources []FetchedResource
	err       error
}

func (f fakeFetcher) ProviderSlug() domainsource.ProviderSlug {
	return f.slug
}

func (f fakeFetcher) Fetch(_ context.Context, _ FetchRequest) ([]FetchedResource, error) {
	return f.resources, f.err
}

func TestRunImportCreatesImportedRows(t *testing.T) {
	repo := newFakeImportRepository()
	service := NewService(repo, []ProviderFetcher{
		fakeFetcher{
			slug: domainsource.ProviderYouTube,
			resources: []FetchedResource{
				{
					ExternalID:      "yt-1",
					ExternalURL:     "https://www.youtube.com/watch?v=yt-1",
					SourceKind:      domainsource.SourceKindVideo,
					SourceType:      domaincatalog.ResourceTypeYouTube,
					Title:           "German Listening Drill",
					Summary:         "A short A2 listening exercise",
					SourceName:      "Easy German",
					Format:          "video",
					DurationMinutes: 12,
					IsFree:          true,
				},
			},
		},
	})

	summary, err := service.RunImport(context.Background(), ImportRequest{
		Provider: "youtube",
		Mode:     "video-search",
		Query:    "german listening",
		Limit:    1,
		CEFRHint: "A2",
		Skills:   []string{"listening"},
		Topics:   []string{"daily-life"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if summary.ImportedCount != 1 || summary.UpdatedCount != 0 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if repo.jobStatus != domainsource.ImportJobCompleted {
		t.Fatalf("expected completed import job status, got %s", repo.jobStatus)
	}
	if repo.resultsCount != 1 {
		t.Fatalf("expected 1 import result row, got %d", repo.resultsCount)
	}
}

func TestRunImportFailsForUnsupportedProvider(t *testing.T) {
	repo := newFakeImportRepository()
	service := NewService(repo, []ProviderFetcher{})

	_, err := service.RunImport(context.Background(), ImportRequest{
		Provider: "stepik",
		Mode:     "course-search",
	})
	if err == nil {
		t.Fatalf("expected unsupported provider error")
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
