package postgres

import (
	"context"
	"database/sql"
	"errors"

	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
)

type CatalogRepository struct {
	db *sql.DB
}

func NewCatalogRepository(db *sql.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) ListResources(ctx context.Context, filter domaincatalog.ListFilter) ([]domaincatalog.Resource, error) {
	filter = filter.WithDefaults()

	var onlyFree any
	if filter.OnlyFree != nil {
		onlyFree = *filter.OnlyFree
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
SELECT
  r.id,
  r.slug,
  r.title,
  r.summary,
  r.source_name,
  r.source_type,
  r.external_url,
  r.cefr_level,
  r.format,
  r.duration_minutes,
  r.is_free,
  r.price_cents,
  r.created_at,
  r.updated_at
FROM catalog_resources r
WHERE ($1 = '' OR r.cefr_level = $1)
  AND ($2 = '' OR EXISTS (
    SELECT 1
    FROM catalog_resource_skills rs
    INNER JOIN catalog_skills s ON s.id = rs.skill_id
    WHERE rs.resource_id = r.id
      AND s.slug = $2
  ))
  AND ($3 = '' OR EXISTS (
    SELECT 1
    FROM catalog_resource_topics rt
    INNER JOIN catalog_topics t ON t.id = rt.topic_id
    WHERE rt.resource_id = r.id
      AND t.slug = $3
  ))
  AND ($4 = '' OR r.title ILIKE '%' || $4 || '%' OR r.summary ILIKE '%' || $4 || '%')
  AND ($5::boolean IS NULL OR r.is_free = $5::boolean)
ORDER BY r.created_at DESC
LIMIT $6 OFFSET $7;
`,
		filter.Level,
		filter.Skill,
		filter.Topic,
		filter.Query,
		onlyFree,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]domaincatalog.Resource, 0, filter.Limit)
	for rows.Next() {
		resource, scanErr := scanResource(rows)
		if scanErr != nil {
			return nil, scanErr
		}

		if err := r.attachTags(ctx, &resource); err != nil {
			return nil, err
		}

		resources = append(resources, resource)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resources, nil
}

func (r *CatalogRepository) GetResourceBySlug(ctx context.Context, slug string) (domaincatalog.Resource, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
SELECT
  r.id,
  r.slug,
  r.title,
  r.summary,
  r.source_name,
  r.source_type,
  r.external_url,
  r.cefr_level,
  r.format,
  r.duration_minutes,
  r.is_free,
  r.price_cents,
  r.created_at,
  r.updated_at
FROM catalog_resources r
WHERE r.slug = $1;
`,
		slug,
	)

	resource, err := scanResource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domaincatalog.Resource{}, domaincatalog.ErrResourceNotFound
		}
		return domaincatalog.Resource{}, err
	}

	if err := r.attachTags(ctx, &resource); err != nil {
		return domaincatalog.Resource{}, err
	}

	return resource, nil
}

func (r *CatalogRepository) GetResourceByID(ctx context.Context, id string) (domaincatalog.Resource, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
SELECT
  r.id,
  r.slug,
  r.title,
  r.summary,
  r.source_name,
  r.source_type,
  r.external_url,
  r.cefr_level,
  r.format,
  r.duration_minutes,
  r.is_free,
  r.price_cents,
  r.created_at,
  r.updated_at
FROM catalog_resources r
WHERE r.id = $1::uuid;
`,
		id,
	)

	resource, err := scanResource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domaincatalog.Resource{}, domaincatalog.ErrResourceNotFound
		}
		return domaincatalog.Resource{}, err
	}

	if err := r.attachTags(ctx, &resource); err != nil {
		return domaincatalog.Resource{}, err
	}

	return resource, nil
}

func (r *CatalogRepository) attachTags(ctx context.Context, resource *domaincatalog.Resource) error {
	skills, err := r.fetchTags(
		ctx,
		`
SELECT s.slug
FROM catalog_resource_skills rs
INNER JOIN catalog_skills s ON s.id = rs.skill_id
WHERE rs.resource_id = $1::uuid
ORDER BY s.slug;
`,
		resource.ID,
	)
	if err != nil {
		return err
	}

	topics, err := r.fetchTags(
		ctx,
		`
SELECT t.slug
FROM catalog_resource_topics rt
INNER JOIN catalog_topics t ON t.id = rt.topic_id
WHERE rt.resource_id = $1::uuid
ORDER BY t.slug;
`,
		resource.ID,
	)
	if err != nil {
		return err
	}

	resource.SkillTags = skills
	resource.TopicTags = topics
	return nil
}

func (r *CatalogRepository) fetchTags(ctx context.Context, query, resourceID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]string, 0, 4)
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		tags = append(tags, slug)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResource(scanner rowScanner) (domaincatalog.Resource, error) {
	var (
		resource   domaincatalog.Resource
		sourceType string
		cefrLevel  string
		priceCents sql.NullInt64
	)

	if err := scanner.Scan(
		&resource.ID,
		&resource.Slug,
		&resource.Title,
		&resource.Summary,
		&resource.SourceName,
		&sourceType,
		&resource.ExternalURL,
		&cefrLevel,
		&resource.Format,
		&resource.DurationMinutes,
		&resource.IsFree,
		&priceCents,
		&resource.CreatedAt,
		&resource.UpdatedAt,
	); err != nil {
		return domaincatalog.Resource{}, err
	}

	resource.SourceType = domaincatalog.ResourceType(sourceType)
	resource.CEFRLevel = domaincatalog.CEFRLevel(cefrLevel)
	if priceCents.Valid {
		value := int(priceCents.Int64)
		resource.PriceCents = &value
	}

	return resource, nil
}
