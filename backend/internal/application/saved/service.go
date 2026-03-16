package savedapp

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidUserID     = errors.New("user id is required")
	ErrInvalidResourceID = errors.New("resource id is required")
)

type Repository interface {
	Save(ctx context.Context, userID, resourceID string) (created bool, err error)
	Remove(ctx context.Context, userID, resourceID string) (removed bool, err error)
	ListResourceIDs(ctx context.Context, userID string) ([]string, error)
	IsSaved(ctx context.Context, userID, resourceID string) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Save(ctx context.Context, userID, resourceID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, ErrInvalidUserID
	}
	if strings.TrimSpace(resourceID) == "" {
		return false, ErrInvalidResourceID
	}

	return s.repo.Save(ctx, userID, resourceID)
}

func (s *Service) Remove(ctx context.Context, userID, resourceID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, ErrInvalidUserID
	}
	if strings.TrimSpace(resourceID) == "" {
		return false, ErrInvalidResourceID
	}

	return s.repo.Remove(ctx, userID, resourceID)
}

func (s *Service) ListResourceIDs(ctx context.Context, userID string) ([]string, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUserID
	}

	return s.repo.ListResourceIDs(ctx, userID)
}

func (s *Service) IsSaved(ctx context.Context, userID, resourceID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, ErrInvalidUserID
	}
	if strings.TrimSpace(resourceID) == "" {
		return false, ErrInvalidResourceID
	}

	return s.repo.IsSaved(ctx, userID, resourceID)
}
