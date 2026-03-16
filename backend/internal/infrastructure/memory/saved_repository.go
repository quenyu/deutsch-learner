package memory

import (
	"context"
	"sort"
	"sync"
	"time"
)

type SavedRepository struct {
	mu   sync.RWMutex
	data map[string]map[string]time.Time
}

func NewSavedRepository() *SavedRepository {
	return &SavedRepository{
		data: make(map[string]map[string]time.Time),
	}
}

func (r *SavedRepository) Save(_ context.Context, userID, resourceID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	userSaved, ok := r.data[userID]
	if !ok {
		userSaved = make(map[string]time.Time)
		r.data[userID] = userSaved
	}

	if _, exists := userSaved[resourceID]; exists {
		return false, nil
	}

	userSaved[resourceID] = time.Now().UTC()
	return true, nil
}

func (r *SavedRepository) Remove(_ context.Context, userID, resourceID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	userSaved, ok := r.data[userID]
	if !ok {
		return false, nil
	}

	if _, exists := userSaved[resourceID]; !exists {
		return false, nil
	}

	delete(userSaved, resourceID)
	return true, nil
}

func (r *SavedRepository) ListResourceIDs(_ context.Context, userID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userSaved, ok := r.data[userID]
	if !ok {
		return []string{}, nil
	}

	type item struct {
		resourceID string
		savedAt    time.Time
	}

	items := make([]item, 0, len(userSaved))
	for resourceID, savedAt := range userSaved {
		items = append(items, item{
			resourceID: resourceID,
			savedAt:    savedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].savedAt.After(items[j].savedAt)
	})

	ids := make([]string, 0, len(items))
	for _, row := range items {
		ids = append(ids, row.resourceID)
	}

	return ids, nil
}

func (r *SavedRepository) IsSaved(_ context.Context, userID, resourceID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userSaved, ok := r.data[userID]
	if !ok {
		return false, nil
	}

	_, exists := userSaved[resourceID]
	return exists, nil
}
