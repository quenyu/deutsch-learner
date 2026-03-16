package progressapp

import (
	"context"
	"errors"
	"strings"

	userstate "deutsch-learner/backend/internal/domain/userstate"
)

var (
	ErrInvalidUserID        = errors.New("user id is required")
	ErrInvalidResourceID    = errors.New("resource id is required")
	ErrInvalidProgressState = errors.New("progress status is invalid")
)

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]userstate.ResourceProgress, error)
	GetByUserAndResource(ctx context.Context, userID, resourceID string) (userstate.ResourceProgress, bool, error)
	UpsertStatus(ctx context.Context, userID, resourceID string, status userstate.ProgressStatus) (userstate.ResourceProgress, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]userstate.ResourceProgress, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUserID
	}

	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) GetByUserAndResource(ctx context.Context, userID, resourceID string) (userstate.ResourceProgress, bool, error) {
	if strings.TrimSpace(userID) == "" {
		return userstate.ResourceProgress{}, false, ErrInvalidUserID
	}
	if strings.TrimSpace(resourceID) == "" {
		return userstate.ResourceProgress{}, false, ErrInvalidResourceID
	}

	return s.repo.GetByUserAndResource(ctx, userID, resourceID)
}

func (s *Service) SetStatus(ctx context.Context, userID, resourceID string, status userstate.ProgressStatus) (userstate.ResourceProgress, error) {
	if strings.TrimSpace(userID) == "" {
		return userstate.ResourceProgress{}, ErrInvalidUserID
	}
	if strings.TrimSpace(resourceID) == "" {
		return userstate.ResourceProgress{}, ErrInvalidResourceID
	}
	if !userstate.IsValidProgressStatus(status) {
		return userstate.ResourceProgress{}, ErrInvalidProgressState
	}

	return s.repo.UpsertStatus(ctx, userID, resourceID, status)
}
