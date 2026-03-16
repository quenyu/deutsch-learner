package catalogapp

import (
	"context"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
)

type Repository interface {
	ListResources(ctx context.Context, filter domaincatalog.ListFilter) ([]domaincatalog.Resource, error)
	GetResourceBySlug(ctx context.Context, slug string) (domaincatalog.Resource, error)
	GetResourceByID(ctx context.Context, id string) (domaincatalog.Resource, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListResources(ctx context.Context, filter domaincatalog.ListFilter) ([]domaincatalog.Resource, error) {
	return s.repo.ListResources(ctx, filter.WithDefaults())
}

func (s *Service) GetResourceBySlug(ctx context.Context, slug string) (domaincatalog.Resource, error) {
	return s.repo.GetResourceBySlug(ctx, slug)
}

func (s *Service) GetResourceByID(ctx context.Context, id string) (domaincatalog.Resource, error) {
	return s.repo.GetResourceByID(ctx, id)
}
