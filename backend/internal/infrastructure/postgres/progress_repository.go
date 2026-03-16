package postgres

import (
	"context"
	"database/sql"

	userstate "deutsch-learner/backend/internal/domain/userstate"
)

type ProgressRepository struct {
	db *sql.DB
}

func NewProgressRepository(db *sql.DB) *ProgressRepository {
	return &ProgressRepository{db: db}
}

func (r *ProgressRepository) ListByUser(ctx context.Context, userID string) ([]userstate.ResourceProgress, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
SELECT
  user_id::text,
  resource_id::text,
  status,
  progress_percent::double precision,
  last_studied_at,
  updated_at
FROM user_resource_progress
WHERE user_id = $1::uuid
ORDER BY updated_at DESC;
`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progressRows := make([]userstate.ResourceProgress, 0, 16)
	for rows.Next() {
		progress, scanErr := scanProgress(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		progressRows = append(progressRows, progress)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return progressRows, nil
}

func (r *ProgressRepository) GetByUserAndResource(ctx context.Context, userID, resourceID string) (userstate.ResourceProgress, bool, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
SELECT
  user_id::text,
  resource_id::text,
  status,
  progress_percent::double precision,
  last_studied_at,
  updated_at
FROM user_resource_progress
WHERE user_id = $1::uuid
  AND resource_id = $2::uuid;
`,
		userID,
		resourceID,
	)

	progress, err := scanProgress(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return userstate.ResourceProgress{}, false, nil
		}
		return userstate.ResourceProgress{}, false, err
	}

	return progress, true, nil
}

func (r *ProgressRepository) UpsertStatus(ctx context.Context, userID, resourceID string, status userstate.ProgressStatus) (userstate.ResourceProgress, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
INSERT INTO user_resource_progress (
  user_id,
  resource_id,
  status,
  progress_percent,
  last_studied_at,
  updated_at
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3,
  CASE
    WHEN $3 = 'completed' THEN 100
    WHEN $3 = 'in_progress' THEN 50
    ELSE 0
  END,
  CASE
    WHEN $3 = 'not_started' THEN NULL
    ELSE now()
  END,
  now()
)
ON CONFLICT (user_id, resource_id) DO UPDATE
SET status = EXCLUDED.status,
    progress_percent = EXCLUDED.progress_percent,
    last_studied_at = EXCLUDED.last_studied_at,
    updated_at = now()
RETURNING
  user_id::text,
  resource_id::text,
  status,
  progress_percent::double precision,
  last_studied_at,
  updated_at;
`,
		userID,
		resourceID,
		string(status),
	)

	progress, err := scanProgress(row)
	if err != nil {
		return userstate.ResourceProgress{}, err
	}

	return progress, nil
}

type progressRowScanner interface {
	Scan(dest ...any) error
}

func scanProgress(scanner progressRowScanner) (userstate.ResourceProgress, error) {
	var (
		progress      userstate.ResourceProgress
		status        string
		lastStudiedAt sql.NullTime
		updatedAt     sql.NullTime
	)

	if err := scanner.Scan(
		&progress.UserID,
		&progress.ResourceID,
		&status,
		&progress.ProgressPercent,
		&lastStudiedAt,
		&updatedAt,
	); err != nil {
		return userstate.ResourceProgress{}, err
	}

	progress.Status = userstate.ProgressStatus(status)
	if lastStudiedAt.Valid {
		value := lastStudiedAt.Time
		progress.LastStudiedAt = &value
	}
	if updatedAt.Valid {
		progress.UpdatedAt = updatedAt.Time
	}

	return progress, nil
}
