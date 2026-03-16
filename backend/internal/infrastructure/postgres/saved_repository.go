package postgres

import (
	"context"
	"database/sql"
)

type SavedRepository struct {
	db *sql.DB
}

func NewSavedRepository(db *sql.DB) *SavedRepository {
	return &SavedRepository{db: db}
}

func (r *SavedRepository) Save(ctx context.Context, userID, resourceID string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`
INSERT INTO user_saved_resources (user_id, resource_id, saved_at)
VALUES ($1::uuid, $2::uuid, now())
ON CONFLICT (user_id, resource_id) DO NOTHING;
`,
		userID,
		resourceID,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func (r *SavedRepository) Remove(ctx context.Context, userID, resourceID string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`
DELETE FROM user_saved_resources
WHERE user_id = $1::uuid
  AND resource_id = $2::uuid;
`,
		userID,
		resourceID,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func (r *SavedRepository) ListResourceIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
SELECT resource_id::text
FROM user_saved_resources
WHERE user_id = $1::uuid
ORDER BY saved_at DESC;
`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resourceIDs := make([]string, 0, 16)
	for rows.Next() {
		var resourceID string
		if err := rows.Scan(&resourceID); err != nil {
			return nil, err
		}

		resourceIDs = append(resourceIDs, resourceID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resourceIDs, nil
}

func (r *SavedRepository) IsSaved(ctx context.Context, userID, resourceID string) (bool, error) {
	var isSaved bool
	err := r.db.QueryRowContext(
		ctx,
		`
SELECT EXISTS (
  SELECT 1
  FROM user_saved_resources
  WHERE user_id = $1::uuid
    AND resource_id = $2::uuid
);
`,
		userID,
		resourceID,
	).Scan(&isSaved)
	if err != nil {
		return false, err
	}

	return isSaved, nil
}
