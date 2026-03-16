package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	userstate "deutsch-learner/backend/internal/domain/userstate"
)

type ProgressRepository struct {
	mu   sync.RWMutex
	data map[string]map[string]userstate.ResourceProgress
}

func NewProgressRepository() *ProgressRepository {
	return &ProgressRepository{
		data: make(map[string]map[string]userstate.ResourceProgress),
	}
}

func (r *ProgressRepository) ListByUser(_ context.Context, userID string) ([]userstate.ResourceProgress, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userProgress, ok := r.data[userID]
	if !ok {
		return []userstate.ResourceProgress{}, nil
	}

	rows := make([]userstate.ResourceProgress, 0, len(userProgress))
	for _, progress := range userProgress {
		rows = append(rows, cloneProgress(progress))
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
	})

	return rows, nil
}

func (r *ProgressRepository) GetByUserAndResource(_ context.Context, userID, resourceID string) (userstate.ResourceProgress, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userProgress, ok := r.data[userID]
	if !ok {
		return userstate.ResourceProgress{}, false, nil
	}

	progress, exists := userProgress[resourceID]
	if !exists {
		return userstate.ResourceProgress{}, false, nil
	}

	return cloneProgress(progress), true, nil
}

func (r *ProgressRepository) UpsertStatus(_ context.Context, userID, resourceID string, status userstate.ProgressStatus) (userstate.ResourceProgress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()

	userProgress, ok := r.data[userID]
	if !ok {
		userProgress = make(map[string]userstate.ResourceProgress)
		r.data[userID] = userProgress
	}

	progress, exists := userProgress[resourceID]
	if !exists {
		progress = userstate.ResourceProgress{
			UserID:     userID,
			ResourceID: resourceID,
		}
	}

	progress.Status = status
	progress.ProgressPercent = userstate.ProgressPercentForStatus(status)
	progress.UpdatedAt = now
	if status == userstate.ProgressNotStarted {
		progress.LastStudiedAt = nil
	} else {
		lastStudiedAt := now
		progress.LastStudiedAt = &lastStudiedAt
	}

	userProgress[resourceID] = progress

	return cloneProgress(progress), nil
}

func cloneProgress(progress userstate.ResourceProgress) userstate.ResourceProgress {
	cloned := progress
	if progress.LastStudiedAt != nil {
		lastStudiedAt := *progress.LastStudiedAt
		cloned.LastStudiedAt = &lastStudiedAt
	}

	return cloned
}
