package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domainprofile "deutsch-learner/backend/internal/domain/profile"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, userID string) (domainprofile.UserProfile, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
SELECT
  u.id::text,
  u.display_name,
  p.target_level,
  COALESCE(p.learning_goals, ''),
  COALESCE(p.preferred_resource_types, '{}'::text[]),
  COALESCE(p.preferred_skills, '{}'::text[]),
  COALESCE(p.preferred_source_providers, '{}'::text[]),
  COALESCE(p.created_at, u.created_at),
  COALESCE(p.updated_at, u.created_at)
FROM app_users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1::uuid;
`,
		userID,
	)

	profile, err := scanUserProfile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainprofile.UserProfile{}, domainprofile.ErrProfileNotFound
		}
		return domainprofile.UserProfile{}, err
	}

	return profile, nil
}

func (r *ProfileRepository) Upsert(ctx context.Context, profile domainprofile.UserProfile) (domainprofile.UserProfile, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domainprofile.UserProfile{}, fmt.Errorf("begin profile transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(
		ctx,
		`
UPDATE app_users
SET display_name = $2
WHERE id = $1::uuid;
`,
		profile.UserID,
		profile.DisplayName,
	)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		return domainprofile.UserProfile{}, err
	}
	if updatedRows == 0 {
		return domainprofile.UserProfile{}, domainprofile.ErrProfileNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`
INSERT INTO user_profiles (
  user_id,
  target_level,
  learning_goals,
  preferred_resource_types,
  preferred_skills,
  preferred_source_providers,
  created_at,
  updated_at
)
VALUES (
  $1::uuid,
  NULLIF($2, ''),
  $3,
  $4::text[],
  $5::text[],
  $6::text[],
  now(),
  now()
)
ON CONFLICT (user_id) DO UPDATE
SET target_level = EXCLUDED.target_level,
    learning_goals = EXCLUDED.learning_goals,
    preferred_resource_types = EXCLUDED.preferred_resource_types,
    preferred_skills = EXCLUDED.preferred_skills,
    preferred_source_providers = EXCLUDED.preferred_source_providers,
    updated_at = now();
`,
		profile.UserID,
		levelOrEmpty(profile.TargetLevel),
		profile.LearningGoals,
		profile.PreferredResourceTypes,
		profile.PreferredSkills,
		profile.PreferredSourceProviders,
	)
	if err != nil {
		return domainprofile.UserProfile{}, err
	}

	if err := tx.Commit(); err != nil {
		return domainprofile.UserProfile{}, err
	}

	return r.GetByUserID(ctx, profile.UserID)
}

type profileRowScanner interface {
	Scan(dest ...any) error
}

func scanUserProfile(scanner profileRowScanner) (domainprofile.UserProfile, error) {
	var (
		profile     domainprofile.UserProfile
		targetLevel sql.NullString
	)

	if err := scanner.Scan(
		&profile.UserID,
		&profile.DisplayName,
		&targetLevel,
		&profile.LearningGoals,
		&profile.PreferredResourceTypes,
		&profile.PreferredSkills,
		&profile.PreferredSourceProviders,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		return domainprofile.UserProfile{}, err
	}

	if targetLevel.Valid {
		value := targetLevel.String
		profile.TargetLevel = &value
	}

	if profile.PreferredResourceTypes == nil {
		profile.PreferredResourceTypes = []string{}
	}
	if profile.PreferredSkills == nil {
		profile.PreferredSkills = []string{}
	}
	if profile.PreferredSourceProviders == nil {
		profile.PreferredSourceProviders = []string{}
	}

	return profile, nil
}

func levelOrEmpty(level *string) string {
	if level == nil {
		return ""
	}
	return *level
}
