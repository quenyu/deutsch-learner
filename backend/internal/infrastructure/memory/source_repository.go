package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	domainsource "deutsch-learner/backend/internal/domain/source"
)

var ErrProviderNotFound = errors.New("source provider not found")

type SourceRepository struct {
	mu        sync.RWMutex
	providers []domainsource.Provider
}

func NewSourceRepository() *SourceRepository {
	now := time.Now().UTC()
	return &SourceRepository{
		providers: []domainsource.Provider{
			{
				ID:          "provider-manual",
				Slug:        domainsource.ProviderManual,
				Name:        "Manual Curation",
				Description: "Curated links entered by editors.",
				IsEnabled:   true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "provider-youtube",
				Slug:        domainsource.ProviderYouTube,
				Name:        "YouTube Data API",
				Description: "Imported metadata from YouTube videos, playlists, and channels.",
				IsEnabled:   true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "provider-stepik",
				Slug:        domainsource.ProviderStepik,
				Name:        "Stepik API",
				Description: "Imported metadata from Stepik courses and lessons.",
				IsEnabled:   true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "provider-enrichment",
				Slug:        domainsource.ProviderEnrichment,
				Name:        "German Enrichment",
				Description: "Optional metadata enrichment hooks.",
				IsEnabled:   true,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
}

func (r *SourceRepository) ListProviders(_ context.Context) ([]domainsource.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domainsource.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		if !provider.IsEnabled {
			continue
		}
		result = append(result, provider)
	}

	return result, nil
}

func (r *SourceRepository) GetProviderBySlug(_ context.Context, slug string) (domainsource.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		if string(provider.Slug) == slug {
			return provider, nil
		}
	}

	return domainsource.Provider{}, ErrProviderNotFound
}
