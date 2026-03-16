package savedapp

import (
	"context"
	"testing"
)

type fakeRepository struct {
	data map[string]map[string]bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		data: make(map[string]map[string]bool),
	}
}

func (r *fakeRepository) Save(_ context.Context, userID, resourceID string) (bool, error) {
	userSaved, ok := r.data[userID]
	if !ok {
		userSaved = make(map[string]bool)
		r.data[userID] = userSaved
	}

	if userSaved[resourceID] {
		return false, nil
	}

	userSaved[resourceID] = true
	return true, nil
}

func (r *fakeRepository) Remove(_ context.Context, userID, resourceID string) (bool, error) {
	userSaved, ok := r.data[userID]
	if !ok {
		return false, nil
	}

	if !userSaved[resourceID] {
		return false, nil
	}

	delete(userSaved, resourceID)
	return true, nil
}

func (r *fakeRepository) ListResourceIDs(_ context.Context, userID string) ([]string, error) {
	userSaved, ok := r.data[userID]
	if !ok {
		return []string{}, nil
	}

	ids := make([]string, 0, len(userSaved))
	for resourceID := range userSaved {
		ids = append(ids, resourceID)
	}

	return ids, nil
}

func (r *fakeRepository) IsSaved(_ context.Context, userID, resourceID string) (bool, error) {
	userSaved, ok := r.data[userID]
	if !ok {
		return false, nil
	}
	return userSaved[resourceID], nil
}

func TestSaveIsIdempotent(t *testing.T) {
	svc := NewService(newFakeRepository())

	created, err := svc.Save(context.Background(), "user-1", "resource-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !created {
		t.Fatalf("expected first save to create entry")
	}

	created, err = svc.Save(context.Background(), "user-1", "resource-1")
	if err != nil {
		t.Fatalf("expected no error on second save, got %v", err)
	}
	if created {
		t.Fatalf("expected second save to be idempotent")
	}
}

func TestRemoveHandlesMissingEntry(t *testing.T) {
	svc := NewService(newFakeRepository())

	removed, err := svc.Remove(context.Background(), "user-1", "resource-unknown")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if removed {
		t.Fatalf("expected no removal for unknown resource")
	}
}

func TestValidationErrors(t *testing.T) {
	svc := NewService(newFakeRepository())

	if _, err := svc.Save(context.Background(), "", "resource-1"); err == nil {
		t.Fatalf("expected validation error for empty user id")
	}

	if _, err := svc.Save(context.Background(), "user-1", ""); err == nil {
		t.Fatalf("expected validation error for empty resource id")
	}

	if _, err := svc.ListResourceIDs(context.Background(), " "); err == nil {
		t.Fatalf("expected validation error for blank user id")
	}
}
