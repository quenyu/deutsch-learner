package profileapp

import (
	"context"
	"testing"
	"time"

	domainprofile "deutsch-learner/backend/internal/domain/profile"
)

type fakeProfileRepository struct {
	items map[string]domainprofile.UserProfile
}

func newFakeProfileRepository() *fakeProfileRepository {
	return &fakeProfileRepository{
		items: make(map[string]domainprofile.UserProfile),
	}
}

func (r *fakeProfileRepository) GetByUserID(_ context.Context, userID string) (domainprofile.UserProfile, error) {
	item, ok := r.items[userID]
	if ok {
		return item, nil
	}

	now := time.Now().UTC()
	return domainprofile.UserProfile{
		UserID:                   userID,
		DisplayName:              "Demo Learner",
		LearningGoals:            "",
		PreferredResourceTypes:   []string{},
		PreferredSkills:          []string{},
		PreferredSourceProviders: []string{},
		CreatedAt:                now,
		UpdatedAt:                now,
	}, nil
}

func (r *fakeProfileRepository) Upsert(_ context.Context, profile domainprofile.UserProfile) (domainprofile.UserProfile, error) {
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	r.items[profile.UserID] = profile
	return profile, nil
}

func TestUpsertNormalizesArraysAndTargetLevel(t *testing.T) {
	svc := NewService(newFakeProfileRepository())
	level := "b1"

	profile, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:                   "user-1",
		DisplayName:              "Max Mustermann",
		TargetLevel:              &level,
		LearningGoals:            "Improve listening",
		PreferredResourceTypes:   []string{"youtube", "course", "youtube"},
		PreferredSkills:          []string{"vocab", "listening"},
		PreferredSourceProviders: []string{"youtube", "manual", "manual"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if profile.TargetLevel == nil || *profile.TargetLevel != "B1" {
		t.Fatalf("expected target level B1, got %+v", profile.TargetLevel)
	}
	if len(profile.PreferredResourceTypes) != 2 {
		t.Fatalf("expected deduped resource types, got %+v", profile.PreferredResourceTypes)
	}
	if len(profile.PreferredSkills) != 2 || profile.PreferredSkills[0] != "vocabulary" {
		t.Fatalf("expected normalized skills, got %+v", profile.PreferredSkills)
	}
	if len(profile.PreferredSourceProviders) != 2 {
		t.Fatalf("expected deduped providers, got %+v", profile.PreferredSourceProviders)
	}
}

func TestValidationErrors(t *testing.T) {
	svc := NewService(newFakeProfileRepository())

	if _, err := svc.GetByUserID(context.Background(), " "); err == nil {
		t.Fatalf("expected invalid user id error")
	}

	if _, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:      "user-1",
		DisplayName: "",
	}); err == nil {
		t.Fatalf("expected invalid display name error")
	}

	level := "Z9"
	if _, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:      "user-1",
		DisplayName: "Max",
		TargetLevel: &level,
	}); err == nil {
		t.Fatalf("expected invalid target level error")
	}

	if _, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:                 "user-1",
		DisplayName:            "Max",
		PreferredResourceTypes: []string{"unsupported"},
	}); err == nil {
		t.Fatalf("expected invalid resource type error")
	}

	if _, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:          "user-1",
		DisplayName:     "Max",
		PreferredSkills: []string{"unsupported"},
	}); err == nil {
		t.Fatalf("expected invalid skill error")
	}

	if _, err := svc.Upsert(context.Background(), UpsertInput{
		UserID:                   "user-1",
		DisplayName:              "Max",
		PreferredSourceProviders: []string{"unsupported"},
	}); err == nil {
		t.Fatalf("expected invalid provider error")
	}
}
