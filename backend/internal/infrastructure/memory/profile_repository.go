package memory

import (
	"context"
	"sync"
	"time"

	domainprofile "deutsch-learner/backend/internal/domain/profile"
)

type ProfileRepository struct {
	mu      sync.RWMutex
	profile map[string]domainprofile.UserProfile
}

func NewProfileRepository() *ProfileRepository {
	return &ProfileRepository{
		profile: make(map[string]domainprofile.UserProfile),
	}
}

func (r *ProfileRepository) GetByUserID(_ context.Context, userID string) (domainprofile.UserProfile, error) {
	r.mu.RLock()
	existing, ok := r.profile[userID]
	r.mu.RUnlock()
	if ok {
		return cloneProfile(existing), nil
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

func (r *ProfileRepository) Upsert(_ context.Context, profile domainprofile.UserProfile) (domainprofile.UserProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	existing, ok := r.profile[profile.UserID]
	if ok {
		profile.CreatedAt = existing.CreatedAt
	} else {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	r.profile[profile.UserID] = cloneProfile(profile)
	return cloneProfile(profile), nil
}

func cloneProfile(profile domainprofile.UserProfile) domainprofile.UserProfile {
	cloned := profile
	cloned.PreferredResourceTypes = append([]string(nil), profile.PreferredResourceTypes...)
	cloned.PreferredSkills = append([]string(nil), profile.PreferredSkills...)
	cloned.PreferredSourceProviders = append([]string(nil), profile.PreferredSourceProviders...)
	if profile.TargetLevel != nil {
		target := *profile.TargetLevel
		cloned.TargetLevel = &target
	}
	return cloned
}
