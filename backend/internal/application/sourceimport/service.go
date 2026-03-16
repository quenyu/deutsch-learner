package sourceimportapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

var (
	ErrProviderRequired    = errors.New("provider is required")
	ErrModeRequired        = errors.New("mode is required")
	ErrProviderUnsupported = errors.New("provider is not supported")
	ErrImportFailed        = errors.New("import failed")
)

type Repository interface {
	GetProviderBySlug(ctx context.Context, slug string) (domainsource.Provider, error)
	CreateImportJob(ctx context.Context, input CreateImportJobInput) (domainsource.ImportJob, error)
	CompleteImportJob(ctx context.Context, jobID string, status domainsource.ImportJobStatus, errMsg string) error
	UpsertSourceRecord(ctx context.Context, input UpsertSourceRecordInput) (domainsource.Record, bool, error)
	UpsertCatalogResource(ctx context.Context, input UpsertCatalogResourceInput) (domaincatalog.Resource, bool, error)
	CreateImportResult(ctx context.Context, input CreateImportResultInput) error
}

type ProviderFetcher interface {
	ProviderSlug() domainsource.ProviderSlug
	Fetch(ctx context.Context, request FetchRequest) ([]FetchedResource, error)
}

type Service struct {
	repo     Repository
	fetchers map[domainsource.ProviderSlug]ProviderFetcher
}

type ImportRequest struct {
	Provider   string
	Mode       string
	Query      string
	PlaylistID string
	ChannelID  string
	FilePath   string
	Limit      int

	CEFRHint  string
	Skills    []string
	Topics    []string
	IsFree    *bool
	Language  string
	Initiator string
}

type ImportSummary struct {
	JobID         string `json:"jobId"`
	Provider      string `json:"provider"`
	Mode          string `json:"mode"`
	ImportedCount int    `json:"importedCount"`
	UpdatedCount  int    `json:"updatedCount"`
	SkippedCount  int    `json:"skippedCount"`
	FailedCount   int    `json:"failedCount"`
}

type FetchRequest struct {
	Mode       string
	Query      string
	PlaylistID string
	ChannelID  string
	FilePath   string
	Limit      int
	Language   string
}

type FetchedResource struct {
	ExternalID      string
	ExternalURL     string
	SourceKind      domainsource.SourceKind
	SourceType      domaincatalog.ResourceType
	Title           string
	Summary         string
	SourceName      string
	AuthorName      string
	Format          string
	DurationMinutes int
	IsFree          bool
	PublishedAt     *time.Time
	RawPayload      []byte
	SkillTags       []string
	TopicTags       []string
	LanguageCode    string
}

type CreateImportJobInput struct {
	ProviderID string
	Mode       string
	Query      string
	PlaylistID string
	ChannelID  string
	FilePath   string
	Limit      int
	CEFRHint   string
	SkillsHint []string
	TopicsHint []string
	IsFreeHint *bool
}

type UpsertSourceRecordInput struct {
	ProviderID   string
	ProviderSlug domainsource.ProviderSlug
	ExternalID   string
	ExternalURL  string
	SourceKind   domainsource.SourceKind
	Title        string
	Summary      string
	AuthorName   string
	LanguageCode string
	RawPayload   []byte
	PublishedAt  *time.Time
}

type UpsertCatalogResourceInput struct {
	SourceRecordID  string
	SourceType      domaincatalog.ResourceType
	SourceName      string
	Title           string
	Summary         string
	ExternalURL     string
	Level           string
	Format          string
	DurationMinutes int
	IsFree          bool
	SkillTags       []string
	TopicTags       []string
	LanguageCode    string
	IngestionOrigin domainsource.IngestionOrigin
}

type CreateImportResultInput struct {
	ImportJobID       string
	SourceRecordID    *string
	CatalogResourceID *string
	Status            domainsource.ImportResultStatus
	Message           string
}

func NewService(repo Repository, fetchers []ProviderFetcher) *Service {
	fetcherMap := make(map[domainsource.ProviderSlug]ProviderFetcher, len(fetchers))
	for _, fetcher := range fetchers {
		if fetcher == nil {
			continue
		}
		fetcherMap[fetcher.ProviderSlug()] = fetcher
	}

	return &Service{
		repo:     repo,
		fetchers: fetcherMap,
	}
}

func (s *Service) RunImport(ctx context.Context, req ImportRequest) (ImportSummary, error) {
	providerSlug := domainsource.ProviderSlug(strings.ToLower(strings.TrimSpace(req.Provider)))
	if providerSlug == "" {
		return ImportSummary{}, ErrProviderRequired
	}

	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		return ImportSummary{}, ErrModeRequired
	}

	fetcher, ok := s.fetchers[providerSlug]
	if !ok {
		return ImportSummary{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, providerSlug)
	}

	provider, err := s.repo.GetProviderBySlug(ctx, string(providerSlug))
	if err != nil {
		return ImportSummary{}, err
	}
	if !provider.IsEnabled {
		return ImportSummary{}, fmt.Errorf("%w: %s is disabled", ErrProviderUnsupported, providerSlug)
	}

	normalizedLevel := normalizeLevel(req.CEFRHint)
	normalizedSkills := normalizeTags(req.Skills)
	normalizedTopics := normalizeTags(req.Topics)
	languageCode := normalizeLanguage(req.Language)

	job, err := s.repo.CreateImportJob(ctx, CreateImportJobInput{
		ProviderID: provider.ID,
		Mode:       mode,
		Query:      strings.TrimSpace(req.Query),
		PlaylistID: strings.TrimSpace(req.PlaylistID),
		ChannelID:  strings.TrimSpace(req.ChannelID),
		FilePath:   strings.TrimSpace(req.FilePath),
		Limit:      req.Limit,
		CEFRHint:   normalizedLevel,
		SkillsHint: normalizedSkills,
		TopicsHint: normalizedTopics,
		IsFreeHint: req.IsFree,
	})
	if err != nil {
		return ImportSummary{}, err
	}

	summary := ImportSummary{
		JobID:    job.ID,
		Provider: string(providerSlug),
		Mode:     mode,
	}

	resources, err := fetcher.Fetch(ctx, FetchRequest{
		Mode:       mode,
		Query:      strings.TrimSpace(req.Query),
		PlaylistID: strings.TrimSpace(req.PlaylistID),
		ChannelID:  strings.TrimSpace(req.ChannelID),
		FilePath:   strings.TrimSpace(req.FilePath),
		Limit:      req.Limit,
		Language:   languageCode,
	})
	if err != nil {
		_ = s.repo.CompleteImportJob(ctx, job.ID, domainsource.ImportJobFailed, err.Error())
		return summary, fmt.Errorf("%w: %v", ErrImportFailed, err)
	}

	for _, resource := range resources {
		working := normalizeFetchedResource(resource, provider, normalizedLevel, normalizedSkills, normalizedTopics, req.IsFree, languageCode)
		if strings.TrimSpace(working.ExternalID) == "" || strings.TrimSpace(working.ExternalURL) == "" {
			summary.FailedCount++
			_ = s.repo.CreateImportResult(ctx, CreateImportResultInput{
				ImportJobID: job.ID,
				Status:      domainsource.ImportResultFailed,
				Message:     "external id and external url are required",
			})
			continue
		}

		sourceRecord, _, err := s.repo.UpsertSourceRecord(ctx, UpsertSourceRecordInput{
			ProviderID:   provider.ID,
			ProviderSlug: providerSlug,
			ExternalID:   working.ExternalID,
			ExternalURL:  working.ExternalURL,
			SourceKind:   working.SourceKind,
			Title:        working.Title,
			Summary:      working.Summary,
			AuthorName:   working.AuthorName,
			LanguageCode: working.LanguageCode,
			RawPayload:   working.RawPayload,
			PublishedAt:  working.PublishedAt,
		})
		if err != nil {
			summary.FailedCount++
			_ = s.repo.CreateImportResult(ctx, CreateImportResultInput{
				ImportJobID: job.ID,
				Status:      domainsource.ImportResultFailed,
				Message:     err.Error(),
			})
			continue
		}

		catalogResource, created, err := s.repo.UpsertCatalogResource(ctx, UpsertCatalogResourceInput{
			SourceRecordID:  sourceRecord.ID,
			SourceType:      working.SourceType,
			SourceName:      working.SourceName,
			Title:           working.Title,
			Summary:         working.Summary,
			ExternalURL:     working.ExternalURL,
			Level:           normalizedLevel,
			Format:          working.Format,
			DurationMinutes: working.DurationMinutes,
			IsFree:          working.IsFree,
			SkillTags:       working.SkillTags,
			TopicTags:       working.TopicTags,
			LanguageCode:    working.LanguageCode,
			IngestionOrigin: domainsource.IngestionOriginImported,
		})
		if err != nil {
			summary.FailedCount++
			sourceRecordID := sourceRecord.ID
			_ = s.repo.CreateImportResult(ctx, CreateImportResultInput{
				ImportJobID:    job.ID,
				SourceRecordID: &sourceRecordID,
				Status:         domainsource.ImportResultFailed,
				Message:        err.Error(),
			})
			continue
		}

		status := domainsource.ImportResultUpdated
		if created {
			status = domainsource.ImportResultImported
			summary.ImportedCount++
		} else {
			summary.UpdatedCount++
		}

		sourceRecordID := sourceRecord.ID
		catalogResourceID := catalogResource.ID
		_ = s.repo.CreateImportResult(ctx, CreateImportResultInput{
			ImportJobID:       job.ID,
			SourceRecordID:    &sourceRecordID,
			CatalogResourceID: &catalogResourceID,
			Status:            status,
		})
	}

	if err := s.repo.CompleteImportJob(ctx, job.ID, domainsource.ImportJobCompleted, ""); err != nil {
		return summary, err
	}

	return summary, nil
}

func normalizeFetchedResource(
	resource FetchedResource,
	provider domainsource.Provider,
	level string,
	skills []string,
	topics []string,
	isFreeHint *bool,
	languageCode string,
) FetchedResource {
	resource.Title = strings.TrimSpace(resource.Title)
	if resource.Title == "" {
		resource.Title = strings.TrimSpace(resource.ExternalID)
	}

	resource.Summary = strings.TrimSpace(resource.Summary)
	if resource.SourceName == "" {
		if strings.TrimSpace(resource.AuthorName) != "" {
			resource.SourceName = strings.TrimSpace(resource.AuthorName)
		} else {
			resource.SourceName = provider.Name
		}
	}

	if strings.TrimSpace(resource.Format) == "" {
		resource.Format = string(resource.SourceKind)
	}

	if resource.DurationMinutes < 0 {
		resource.DurationMinutes = 0
	}

	if len(resource.SkillTags) == 0 {
		resource.SkillTags = append([]string(nil), skills...)
	} else {
		resource.SkillTags = normalizeTags(resource.SkillTags)
	}

	if len(resource.TopicTags) == 0 {
		resource.TopicTags = append([]string(nil), topics...)
	} else {
		resource.TopicTags = normalizeTags(resource.TopicTags)
	}

	if isFreeHint != nil {
		resource.IsFree = *isFreeHint
	} else if !resource.IsFree {
		resource.IsFree = true
	}

	if strings.TrimSpace(resource.LanguageCode) == "" {
		resource.LanguageCode = languageCode
	}

	if resource.SourceType == "" {
		switch resource.SourceKind {
		case domainsource.SourceKindPlaylist:
			resource.SourceType = domaincatalog.ResourceTypePlaylist
		case domainsource.SourceKindCourse:
			resource.SourceType = domaincatalog.ResourceTypeCourse
		case domainsource.SourceKindArticle:
			resource.SourceType = domaincatalog.ResourceTypeArticle
		case domainsource.SourceKindPodcast:
			resource.SourceType = domaincatalog.ResourceTypePodcast
		case domainsource.SourceKindGrammarReference:
			resource.SourceType = domaincatalog.ResourceTypeGrammarReference
		case domainsource.SourceKindExercise:
			resource.SourceType = domaincatalog.ResourceTypeExercise
		default:
			resource.SourceType = domaincatalog.ResourceTypeYouTube
		}
	}

	_ = level
	return resource
}

func normalizeLevel(level string) string {
	value := strings.ToUpper(strings.TrimSpace(level))
	switch domaincatalog.CEFRLevel(value) {
	case domaincatalog.CEFRLevelA1, domaincatalog.CEFRLevelA2, domaincatalog.CEFRLevelB1, domaincatalog.CEFRLevelB2, domaincatalog.CEFRLevelC1, domaincatalog.CEFRLevelC2:
		return value
	default:
		return string(domaincatalog.CEFRLevelA1)
	}
}

func normalizeLanguage(language string) string {
	value := strings.ToLower(strings.TrimSpace(language))
	if value == "" {
		return "de"
	}
	return value
}

func normalizeTags(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
