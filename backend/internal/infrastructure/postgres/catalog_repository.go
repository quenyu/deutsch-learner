package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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
  COALESCE(sp.slug, 'manual') AS provider_slug,
  COALESCE(sp.name, 'Manual Curation') AS provider_name,
  r.ingestion_origin,
  COALESCE(sr.source_kind, CASE
    WHEN r.source_type = 'youtube' THEN 'video'
    WHEN r.source_type = 'playlist' THEN 'playlist'
    WHEN r.source_type = 'course' THEN 'course'
    WHEN r.source_type = 'article' THEN 'article'
    WHEN r.source_type = 'podcast' THEN 'podcast'
    WHEN r.source_type = 'grammar_reference' THEN 'grammar_reference'
    WHEN r.source_type = 'exercise' THEN 'exercise'
    ELSE 'external_link'
  END) AS source_kind,
  sr.last_synced_at,
  r.created_at,
  r.updated_at
FROM catalog_resources r
LEFT JOIN source_records sr ON sr.id = r.source_record_id
LEFT JOIN source_providers sp ON sp.id = sr.provider_id
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
  AND ($5 = '' OR COALESCE(sp.slug, 'manual') = $5)
  AND ($6 = '' OR r.source_type = $6)
  AND ($7::boolean IS NULL OR r.is_free = $7::boolean)
ORDER BY r.created_at DESC
LIMIT $8 OFFSET $9;
`,
		filter.Level,
		filter.Skill,
		filter.Topic,
		filter.Query,
		filter.Provider,
		filter.Type,
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
  COALESCE(sp.slug, 'manual') AS provider_slug,
  COALESCE(sp.name, 'Manual Curation') AS provider_name,
  r.ingestion_origin,
  COALESCE(sr.source_kind, CASE
    WHEN r.source_type = 'youtube' THEN 'video'
    WHEN r.source_type = 'playlist' THEN 'playlist'
    WHEN r.source_type = 'course' THEN 'course'
    WHEN r.source_type = 'article' THEN 'article'
    WHEN r.source_type = 'podcast' THEN 'podcast'
    WHEN r.source_type = 'grammar_reference' THEN 'grammar_reference'
    WHEN r.source_type = 'exercise' THEN 'exercise'
    ELSE 'external_link'
  END) AS source_kind,
  sr.last_synced_at,
  r.created_at,
  r.updated_at
FROM catalog_resources r
LEFT JOIN source_records sr ON sr.id = r.source_record_id
LEFT JOIN source_providers sp ON sp.id = sr.provider_id
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
  COALESCE(sp.slug, 'manual') AS provider_slug,
  COALESCE(sp.name, 'Manual Curation') AS provider_name,
  r.ingestion_origin,
  COALESCE(sr.source_kind, CASE
    WHEN r.source_type = 'youtube' THEN 'video'
    WHEN r.source_type = 'playlist' THEN 'playlist'
    WHEN r.source_type = 'course' THEN 'course'
    WHEN r.source_type = 'article' THEN 'article'
    WHEN r.source_type = 'podcast' THEN 'podcast'
    WHEN r.source_type = 'grammar_reference' THEN 'grammar_reference'
    WHEN r.source_type = 'exercise' THEN 'exercise'
    ELSE 'external_link'
  END) AS source_kind,
  sr.last_synced_at,
  r.created_at,
  r.updated_at
FROM catalog_resources r
LEFT JOIN source_records sr ON sr.id = r.source_record_id
LEFT JOIN source_providers sp ON sp.id = sr.provider_id
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
		resource        domaincatalog.Resource
		sourceType      string
		cefrLevel       string
		priceCents      sql.NullInt64
		providerSlug    sql.NullString
		providerName    sql.NullString
		ingestionOrigin sql.NullString
		sourceKind      sql.NullString
		lastSyncedAt    sql.NullTime
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
		&providerSlug,
		&providerName,
		&ingestionOrigin,
		&sourceKind,
		&lastSyncedAt,
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
	resource.ProviderSlug = "manual"
	if providerSlug.Valid && strings.TrimSpace(providerSlug.String) != "" {
		resource.ProviderSlug = providerSlug.String
	}
	resource.ProviderName = "Manual Curation"
	if providerName.Valid && strings.TrimSpace(providerName.String) != "" {
		resource.ProviderName = providerName.String
	}
	resource.IngestionOrigin = "manual"
	if ingestionOrigin.Valid && strings.TrimSpace(ingestionOrigin.String) != "" {
		resource.IngestionOrigin = ingestionOrigin.String
	}
	resource.SourceKind = fallbackSourceKind(resource.SourceType)
	if sourceKind.Valid && strings.TrimSpace(sourceKind.String) != "" {
		resource.SourceKind = sourceKind.String
	}
	if lastSyncedAt.Valid {
		value := lastSyncedAt.Time
		resource.LastSyncedAt = &value
	}

	return resource, nil
}

func fallbackSourceKind(sourceType domaincatalog.ResourceType) string {
	switch sourceType {
	case domaincatalog.ResourceTypePlaylist:
		return "playlist"
	case domaincatalog.ResourceTypeCourse:
		return "course"
	case domaincatalog.ResourceTypeArticle:
		return "article"
	case domaincatalog.ResourceTypePodcast:
		return "podcast"
	case domaincatalog.ResourceTypeGrammarReference:
		return "grammar_reference"
	case domaincatalog.ResourceTypeExercise:
		return "exercise"
	default:
		return "video"
	}
}
