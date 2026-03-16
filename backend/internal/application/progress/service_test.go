package progressapp

import (
	"context"
	"testing"

	userstate "deutsch-learner/backend/internal/domain/userstate"
)

type fakeProgressRepository struct {
	data map[string]map[string]userstate.ResourceProgress
}

func newFakeProgressRepository() *fakeProgressRepository {
	return &fakeProgressRepository{
		data: make(map[string]map[string]userstate.ResourceProgress),
	}
}

func (r *fakeProgressRepository) ListByUser(_ context.Context, userID string) ([]userstate.ResourceProgress, error) {
	userData, ok := r.data[userID]
	if !ok {
		return []userstate.ResourceProgress{}, nil
	}

	rows := make([]userstate.ResourceProgress, 0, len(userData))
	for _, progress := range userData {
		rows = append(rows, progress)
	}

	return rows, nil
}

func (r *fakeProgressRepository) GetByUserAndResource(_ context.Context, userID, resourceID string) (userstate.ResourceProgress, bool, error) {
	userData, ok := r.data[userID]
	if !ok {
		return userstate.ResourceProgress{}, false, nil
	}

	progress, found := userData[resourceID]
	if !found {
		return userstate.ResourceProgress{}, false, nil
	}

	return progress, true, nil
}

func (r *fakeProgressRepository) UpsertStatus(_ context.Context, userID, resourceID string, status userstate.ProgressStatus) (userstate.ResourceProgress, error) {
	userData, ok := r.data[userID]
	if !ok {
		userData = make(map[string]userstate.ResourceProgress)
		r.data[userID] = userData
	}

	progress := userstate.ResourceProgress{
		UserID:          userID,
		ResourceID:      resourceID,
		Status:          status,
		ProgressPercent: userstate.ProgressPercentForStatus(status),
	}
	userData[resourceID] = progress

	return progress, nil
}

func TestSetStatusStoresValidatedProgress(t *testing.T) {
	service := NewService(newFakeProgressRepository())

	progress, err := service.SetStatus(context.Background(), "user-1", "resource-1", userstate.ProgressInProgress)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if progress.Status != userstate.ProgressInProgress {
		t.Fatalf("expected status %q, got %q", userstate.ProgressInProgress, progress.Status)
	}
	if progress.ProgressPercent != 50 {
		t.Fatalf("expected in_progress to map to 50%%, got %v", progress.ProgressPercent)
	}
}

func TestGetByUserAndResourceHandlesMissingEntry(t *testing.T) {
	service := NewService(newFakeProgressRepository())

	_, found, err := service.GetByUserAndResource(context.Background(), "user-1", "resource-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found {
		t.Fatalf("expected missing progress entry")
	}
}

func TestValidationErrors(t *testing.T) {
	service := NewService(newFakeProgressRepository())

	if _, err := service.SetStatus(context.Background(), "", "resource-1", userstate.ProgressInProgress); err == nil {
		t.Fatalf("expected validation error for empty user id")
	}

	if _, err := service.SetStatus(context.Background(), "user-1", "", userstate.ProgressInProgress); err == nil {
		t.Fatalf("expected validation error for empty resource id")
	}

	if _, err := service.SetStatus(context.Background(), "user-1", "resource-1", userstate.ProgressStatus("invalid")); err == nil {
		t.Fatalf("expected validation error for invalid progress status")
	}

	if _, err := service.ListByUser(context.Background(), " "); err == nil {
		t.Fatalf("expected validation error for blank user id")
	}
}
