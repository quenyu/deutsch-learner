package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

var (
	ErrSourceProviderNotFound = errors.New("source provider not found")
	slugSanitizer             = regexp.MustCompile(`[^a-z0-9]+`)
)

type SourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) ListProviders(ctx context.Context) ([]domainsource.Provider, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
SELECT id::text, slug, name, description, is_enabled, created_at, updated_at
FROM source_providers
WHERE is_enabled = true
ORDER BY name ASC;
`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]domainsource.Provider, 0, 4)
	for rows.Next() {
		var provider domainsource.Provider
		var slug string
		if err := rows.Scan(
			&provider.ID,
			&slug,
			&provider.Name,
			&provider.Description,
			&provider.IsEnabled,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		); err != nil {
			return nil, err
		}
		provider.Slug = domainsource.ProviderSlug(slug)
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *SourceRepository) GetProviderBySlug(ctx context.Context, slug string) (domainsource.Provider, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
SELECT id::text, slug, name, description, is_enabled, created_at, updated_at
FROM source_providers
WHERE slug = $1;
`,
		strings.TrimSpace(strings.ToLower(slug)),
	)

	var provider domainsource.Provider
	var providerSlug string
	if err := row.Scan(
		&provider.ID,
		&providerSlug,
		&provider.Name,
		&provider.Description,
		&provider.IsEnabled,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainsource.Provider{}, ErrSourceProviderNotFound
		}
		return domainsource.Provider{}, err
	}
	provider.Slug = domainsource.ProviderSlug(providerSlug)
	return provider, nil
}

func (r *SourceRepository) CreateImportJob(ctx context.Context, input sourceimportapp.CreateImportJobInput) (domainsource.ImportJob, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
INSERT INTO import_jobs (
  provider_id,
  mode,
  status,
  query,
  playlist_id,
  channel_id,
  file_path,
  limit_count,
  cefr_hint,
  skills_hint,
  topics_hint,
  is_free_hint,
  started_at,
  created_at,
  updated_at
)
VALUES (
  $1::uuid,
  $2,
  'running',
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9::text[],
  $10::text[],
  $11,
  now(),
  now(),
  now()
)
RETURNING id::text, provider_id::text, mode, status, query, playlist_id, channel_id, file_path, limit_count, cefr_hint, skills_hint, topics_hint, is_free_hint, started_at, created_at, updated_at;
`,
		input.ProviderID,
		input.Mode,
		input.Query,
		input.PlaylistID,
		input.ChannelID,
		input.FilePath,
		input.Limit,
		input.CEFRHint,
		input.SkillsHint,
		input.TopicsHint,
		input.IsFreeHint,
	)

	var (
		job       domainsource.ImportJob
		status    string
		startedAt sql.NullTime
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)
	if err := row.Scan(
		&job.ID,
		&job.ProviderID,
		&job.Mode,
		&status,
		&job.Query,
		&job.PlaylistID,
		&job.ChannelID,
		&job.FilePath,
		&job.LimitCount,
		&job.CEFRHint,
		&job.SkillsHint,
		&job.TopicsHint,
		&job.IsFreeHint,
		&startedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domainsource.ImportJob{}, err
	}

	job.Status = domainsource.ImportJobStatus(status)
	if startedAt.Valid {
		value := startedAt.Time
		job.StartedAt = &value
	}
	if createdAt.Valid {
		job.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		job.UpdatedAt = updatedAt.Time
	}

	return job, nil
}

func (r *SourceRepository) CompleteImportJob(ctx context.Context, jobID string, status domainsource.ImportJobStatus, errMsg string) error {
	_, err := r.db.ExecContext(
		ctx,
		`
UPDATE import_jobs
SET status = $2,
    error_message = $3,
    ended_at = now(),
    updated_at = now()
WHERE id = $1::uuid;
`,
		jobID,
		string(status),
		strings.TrimSpace(errMsg),
	)
	return err
}

func (r *SourceRepository) UpsertSourceRecord(ctx context.Context, input sourceimportapp.UpsertSourceRecordInput) (domainsource.Record, bool, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
INSERT INTO source_records (
  provider_id,
  external_id,
  external_url,
  source_kind,
  title,
  summary,
  author_name,
  language_code,
  raw_payload,
  sync_status,
  last_synced_at,
  published_at,
  created_at,
  updated_at
)
VALUES (
  $1::uuid,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9::jsonb,
  'synced',
  now(),
  $10,
  now(),
  now()
)
ON CONFLICT (provider_id, external_id, source_kind) DO UPDATE
SET external_url = EXCLUDED.external_url,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    author_name = EXCLUDED.author_name,
    language_code = EXCLUDED.language_code,
    raw_payload = EXCLUDED.raw_payload,
    sync_status = 'synced',
    last_synced_at = now(),
    published_at = EXCLUDED.published_at,
    updated_at = now()
RETURNING
  id::text,
  provider_id::text,
  external_id,
  external_url,
  source_kind,
  title,
  summary,
  author_name,
  language_code,
  raw_payload,
  sync_status,
  last_synced_at,
  published_at,
  created_at,
  updated_at,
  (xmax = 0) AS created;
`,
		input.ProviderID,
		input.ExternalID,
		input.ExternalURL,
		string(input.SourceKind),
		input.Title,
		input.Summary,
		input.AuthorName,
		input.LanguageCode,
		normalizeJSON(input.RawPayload),
		input.PublishedAt,
	)

	var (
		record       domainsource.Record
		sourceKind   string
		syncStatus   string
		lastSyncedAt sql.NullTime
		publishedAt  sql.NullTime
		created      bool
	)
	if err := row.Scan(
		&record.ID,
		&record.ProviderID,
		&record.ExternalID,
		&record.ExternalURL,
		&sourceKind,
		&record.Title,
		&record.Summary,
		&record.AuthorName,
		&record.LanguageCode,
		&record.RawPayload,
		&syncStatus,
		&lastSyncedAt,
		&publishedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&created,
	); err != nil {
		return domainsource.Record{}, false, err
	}

	record.ProviderSlug = input.ProviderSlug
	record.SourceKind = domainsource.SourceKind(sourceKind)
	record.SyncStatus = domainsource.SyncStatus(syncStatus)
	if lastSyncedAt.Valid {
		value := lastSyncedAt.Time
		record.LastSyncedAt = &value
	}
	if publishedAt.Valid {
		value := publishedAt.Time
		record.PublishedAt = &value
	}

	return record, created, nil
}

func (r *SourceRepository) UpsertCatalogResource(ctx context.Context, input sourceimportapp.UpsertCatalogResourceInput) (domaincatalog.Resource, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domaincatalog.Resource{}, false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	row := tx.QueryRowContext(
		ctx,
		`
SELECT id::text
FROM catalog_resources
WHERE source_record_id = $1::uuid;
`,
		input.SourceRecordID,
	)

	var existingID string
	err = row.Scan(&existingID)
	created := false
	switch {
	case err == nil:
		_, err = tx.ExecContext(
			ctx,
			`
UPDATE catalog_resources
SET title = $2,
    summary = $3,
    source_name = $4,
    source_type = $5,
    external_url = $6,
    cefr_level = $7,
    format = $8,
    duration_minutes = $9,
    is_free = $10,
    price_cents = NULL,
    language_code = $11,
    ingestion_origin = $12,
    updated_at = now()
WHERE id = $1::uuid;
`,
			existingID,
			input.Title,
			input.Summary,
			input.SourceName,
			string(input.SourceType),
			input.ExternalURL,
			input.Level,
			input.Format,
			input.DurationMinutes,
			input.IsFree,
			input.LanguageCode,
			string(input.IngestionOrigin),
		)
		if err != nil {
			return domaincatalog.Resource{}, false, err
		}
	case errors.Is(err, sql.ErrNoRows):
		created = true
		slug := buildImportedSlug(input.Title, input.SourceRecordID)
		err = tx.QueryRowContext(
			ctx,
			`
INSERT INTO catalog_resources (
  slug,
  title,
  summary,
  source_name,
  source_type,
  external_url,
  cefr_level,
  format,
  duration_minutes,
  is_free,
  price_cents,
  language_code,
  source_record_id,
  ingestion_origin,
  created_at,
  updated_at
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  NULL,
  $11,
  $12::uuid,
  $13,
  now(),
  now()
)
RETURNING id::text;
`,
			slug,
			input.Title,
			input.Summary,
			input.SourceName,
			string(input.SourceType),
			input.ExternalURL,
			input.Level,
			input.Format,
			input.DurationMinutes,
			input.IsFree,
			input.LanguageCode,
			input.SourceRecordID,
			string(input.IngestionOrigin),
		).Scan(&existingID)
		if err != nil {
			return domaincatalog.Resource{}, false, err
		}
	default:
		return domaincatalog.Resource{}, false, err
	}

	if err := syncResourceSkillsInTx(ctx, tx, existingID, input.SkillTags); err != nil {
		return domaincatalog.Resource{}, false, err
	}
	if err := syncResourceTopicsInTx(ctx, tx, existingID, input.TopicTags); err != nil {
		return domaincatalog.Resource{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return domaincatalog.Resource{}, false, err
	}

	catalogRepo := NewCatalogRepository(r.db)
	resource, err := catalogRepo.GetResourceByID(ctx, existingID)
	if err != nil {
		return domaincatalog.Resource{}, false, err
	}

	return resource, created, nil
}

func (r *SourceRepository) CreateImportResult(ctx context.Context, input sourceimportapp.CreateImportResultInput) error {
	_, err := r.db.ExecContext(
		ctx,
		`
INSERT INTO import_job_results (
  import_job_id,
  source_record_id,
  catalog_resource_id,
  status,
  message,
  created_at
)
VALUES (
  $1::uuid,
  CASE WHEN $2 = '' THEN NULL ELSE $2::uuid END,
  CASE WHEN $3 = '' THEN NULL ELSE $3::uuid END,
  $4,
  $5,
  now()
);
`,
		input.ImportJobID,
		stringOrEmpty(input.SourceRecordID),
		stringOrEmpty(input.CatalogResourceID),
		string(input.Status),
		input.Message,
	)
	return err
}

func syncResourceSkillsInTx(ctx context.Context, tx *sql.Tx, resourceID string, skills []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_resource_skills WHERE resource_id = $1::uuid;`, resourceID); err != nil {
		return err
	}

	for _, skill := range normalizeTags(skills) {
		if _, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_skills (slug, label)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET label = EXCLUDED.label;
`,
			skill,
			toTitleLabel(skill),
		); err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_resource_skills (resource_id, skill_id)
SELECT $1::uuid, id
FROM catalog_skills
WHERE slug = $2
ON CONFLICT (resource_id, skill_id) DO NOTHING;
`,
			resourceID,
			skill,
		); err != nil {
			return err
		}
	}

	return nil
}

func syncResourceTopicsInTx(ctx context.Context, tx *sql.Tx, resourceID string, topics []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_resource_topics WHERE resource_id = $1::uuid;`, resourceID); err != nil {
		return err
	}

	for _, topic := range normalizeTags(topics) {
		if _, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_topics (slug, label)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET label = EXCLUDED.label;
`,
			topic,
			toTitleLabel(topic),
		); err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_resource_topics (resource_id, topic_id)
SELECT $1::uuid, id
FROM catalog_topics
WHERE slug = $2
ON CONFLICT (resource_id, topic_id) DO NOTHING;
`,
			resourceID,
			topic,
		); err != nil {
			return err
		}
	}

	return nil
}

func normalizeJSON(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "{}"
	}

	normalized, err := json.Marshal(parsed)
	if err != nil {
		return "{}"
	}

	return string(normalized)
}

func normalizeTags(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func buildImportedSlug(title, sourceRecordID string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	base = slugSanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "resource"
	}

	suffix := sourceRecordID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	return fmt.Sprintf("%s-%s", base, strings.ToLower(suffix))
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toTitleLabel(value string) string {
	parts := strings.Split(strings.ReplaceAll(value, "-", " "), " ")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
