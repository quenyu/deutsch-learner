package sourceapp

import (
	"context"

	domainsource "deutsch-learner/backend/internal/domain/source"
)

type Repository interface {
	ListProviders(ctx context.Context) ([]domainsource.Provider, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListProviders(ctx context.Context) ([]domainsource.Provider, error) {
	return s.repo.ListProviders(ctx)
}
