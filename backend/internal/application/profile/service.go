package profileapp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainprofile "deutsch-learner/backend/internal/domain/profile"
)

var (
	ErrInvalidUserID         = errors.New("user id is required")
	ErrInvalidDisplayName    = errors.New("display name is required")
	ErrInvalidTargetLevel    = errors.New("target level is invalid")
	ErrInvalidResourceType   = errors.New("preferred resource type is invalid")
	ErrInvalidSkill          = errors.New("preferred skill is invalid")
	ErrInvalidSourceProvider = errors.New("preferred source provider is invalid")
)

type Repository interface {
	GetByUserID(ctx context.Context, userID string) (domainprofile.UserProfile, error)
	Upsert(ctx context.Context, profile domainprofile.UserProfile) (domainprofile.UserProfile, error)
}

type Service struct {
	repo Repository
}

type UpsertInput struct {
	UserID                   string
	DisplayName              string
	TargetLevel              *string
	LearningGoals            string
	PreferredResourceTypes   []string
	PreferredSkills          []string
	PreferredSourceProviders []string
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByUserID(ctx context.Context, userID string) (domainprofile.UserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return domainprofile.UserProfile{}, ErrInvalidUserID
	}

	return s.repo.GetByUserID(ctx, userID)
}

func (s *Service) Upsert(ctx context.Context, input UpsertInput) (domainprofile.UserProfile, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return domainprofile.UserProfile{}, ErrInvalidUserID
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return domainprofile.UserProfile{}, ErrInvalidDisplayName
	}

	targetLevel, err := normalizeLevel(input.TargetLevel)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	resourceTypes, err := normalizeResourceTypes(input.PreferredResourceTypes)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	skills, err := normalizeSkills(input.PreferredSkills)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	sourceProviders, err := normalizeSourceProviders(input.PreferredSourceProviders)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	profile := domainprofile.UserProfile{
		UserID:                   userID,
		DisplayName:              displayName,
		TargetLevel:              targetLevel,
		LearningGoals:            strings.TrimSpace(input.LearningGoals),
		PreferredResourceTypes:   resourceTypes,
		PreferredSkills:          skills,
		PreferredSourceProviders: sourceProviders,
	}

	return s.repo.Upsert(ctx, profile)
}

func normalizeLevel(target *string) (*string, error) {
	if target == nil {
		return nil, nil
	}

	value := strings.ToUpper(strings.TrimSpace(*target))
	if value == "" {
		return nil, nil
	}

	switch domaincatalog.CEFRLevel(value) {
	case domaincatalog.CEFRLevelA1, domaincatalog.CEFRLevelA2, domaincatalog.CEFRLevelB1, domaincatalog.CEFRLevelB2, domaincatalog.CEFRLevelC1, domaincatalog.CEFRLevelC2:
		return &value, nil
	default:
		return nil, ErrInvalidTargetLevel
	}
}

func normalizeResourceTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	allowed := map[string]struct{}{
		string(domaincatalog.ResourceTypeYouTube):          {},
		string(domaincatalog.ResourceTypeArticle):          {},
		string(domaincatalog.ResourceTypePlaylist):         {},
		string(domaincatalog.ResourceTypeCourse):           {},
		string(domaincatalog.ResourceTypePodcast):          {},
		string(domaincatalog.ResourceTypeGrammarReference): {},
		string(domaincatalog.ResourceTypeExercise):         {},
	}

	return normalizeWithAllowed(values, allowed, ErrInvalidResourceType)
}

func normalizeSkills(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	allowed := map[string]struct{}{
		"grammar":    {},
		"vocabulary": {},
		"vocab":      {},
		"listening":  {},
		"reading":    {},
		"speaking":   {},
	}

	normalized, err := normalizeWithAllowed(values, allowed, ErrInvalidSkill)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(normalized))
	seen := make(map[string]struct{}, len(normalized))
	for _, skill := range normalized {
		if skill == "vocab" {
			skill = "vocabulary"
		}
		if _, ok := seen[skill]; ok {
			continue
		}
		seen[skill] = struct{}{}
		result = append(result, skill)
	}

	return result, nil
}

func normalizeSourceProviders(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	allowed := map[string]struct{}{
		"manual":     {},
		"youtube":    {},
		"stepik":     {},
		"enrichment": {},
	}

	return normalizeWithAllowed(values, allowed, ErrInvalidSourceProvider)
}

func normalizeWithAllowed(values []string, allowed map[string]struct{}, errType error) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}

		if _, ok := allowed[value]; !ok {
			return nil, fmt.Errorf("%w: %s", errType, value)
		}

		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result, nil
}
