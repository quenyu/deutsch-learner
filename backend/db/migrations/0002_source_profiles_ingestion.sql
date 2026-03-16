CREATE TABLE IF NOT EXISTS source_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS source_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_id UUID NOT NULL REFERENCES source_providers(id) ON DELETE RESTRICT,
  external_id TEXT NOT NULL,
  external_url TEXT NOT NULL,
  source_kind TEXT NOT NULL CHECK (
    source_kind IN (
      'video',
      'playlist',
      'channel',
      'course',
      'lesson',
      'article',
      'podcast',
      'grammar_reference',
      'exercise',
      'external_link',
      'vocabulary',
      'grammar',
      'sentence'
    )
  ),
  title TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  author_name TEXT NOT NULL DEFAULT '',
  language_code TEXT NOT NULL DEFAULT 'de',
  raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  sync_status TEXT NOT NULL DEFAULT 'synced' CHECK (sync_status IN ('pending', 'synced', 'failed')),
  last_synced_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (provider_id, external_id, source_kind)
);

CREATE TABLE IF NOT EXISTS import_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_id UUID NOT NULL REFERENCES source_providers(id) ON DELETE RESTRICT,
  mode TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
  query TEXT NOT NULL DEFAULT '',
  playlist_id TEXT NOT NULL DEFAULT '',
  channel_id TEXT NOT NULL DEFAULT '',
  file_path TEXT NOT NULL DEFAULT '',
  limit_count INTEGER NOT NULL DEFAULT 0,
  cefr_hint TEXT NOT NULL DEFAULT '',
  skills_hint TEXT[] NOT NULL DEFAULT '{}',
  topics_hint TEXT[] NOT NULL DEFAULT '{}',
  is_free_hint BOOLEAN,
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS import_job_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  import_job_id UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
  source_record_id UUID REFERENCES source_records(id) ON DELETE SET NULL,
  catalog_resource_id UUID REFERENCES catalog_resources(id) ON DELETE SET NULL,
  status TEXT NOT NULL CHECK (status IN ('imported', 'updated', 'skipped', 'failed')),
  message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_profiles (
  user_id UUID PRIMARY KEY REFERENCES app_users(id) ON DELETE CASCADE,
  target_level TEXT CHECK (target_level IS NULL OR target_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2')),
  learning_goals TEXT NOT NULL DEFAULT '',
  preferred_resource_types TEXT[] NOT NULL DEFAULT '{}',
  preferred_skills TEXT[] NOT NULL DEFAULT '{}',
  preferred_source_providers TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE catalog_resources
  ADD COLUMN IF NOT EXISTS source_record_id UUID;

ALTER TABLE catalog_resources
  ADD COLUMN IF NOT EXISTS ingestion_origin TEXT;

UPDATE catalog_resources
SET ingestion_origin = 'manual'
WHERE ingestion_origin IS NULL OR ingestion_origin = '';

ALTER TABLE catalog_resources
  ALTER COLUMN ingestion_origin SET DEFAULT 'manual';

ALTER TABLE catalog_resources
  ALTER COLUMN ingestion_origin SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'catalog_resources_source_record_id_fkey'
  ) THEN
    ALTER TABLE catalog_resources
      ADD CONSTRAINT catalog_resources_source_record_id_fkey
      FOREIGN KEY (source_record_id)
      REFERENCES source_records(id)
      ON DELETE SET NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'catalog_resources_source_record_id_key'
  ) THEN
    ALTER TABLE catalog_resources
      ADD CONSTRAINT catalog_resources_source_record_id_key UNIQUE (source_record_id);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'catalog_resources_ingestion_origin_check'
  ) THEN
    ALTER TABLE catalog_resources
      ADD CONSTRAINT catalog_resources_ingestion_origin_check
      CHECK (ingestion_origin IN ('manual', 'imported'));
  END IF;
END $$;

INSERT INTO source_providers (slug, name, description, is_enabled)
VALUES
  ('manual', 'Manual Curation', 'Curated links entered by editors.', true),
  ('youtube', 'YouTube Data API', 'Imported metadata from YouTube videos, playlists, and channels.', true),
  ('stepik', 'Stepik API', 'Imported metadata from Stepik courses and lessons.', true),
  ('enrichment', 'German Enrichment', 'Optional metadata enrichment hooks for vocabulary and grammar.', true)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_enabled = EXCLUDED.is_enabled,
    updated_at = now();

WITH manual_provider AS (
  SELECT id
  FROM source_providers
  WHERE slug = 'manual'
),
backfill_records AS (
  SELECT
    r.id AS resource_id,
    mp.id AS provider_id,
    COALESCE(r.slug, r.id::text) AS external_id,
    r.external_url AS external_url,
    CASE
      WHEN r.source_type = 'youtube' THEN 'video'
      WHEN r.source_type = 'playlist' THEN 'playlist'
      WHEN r.source_type = 'course' THEN 'course'
      WHEN r.source_type = 'article' THEN 'article'
      WHEN r.source_type = 'podcast' THEN 'podcast'
      WHEN r.source_type = 'grammar_reference' THEN 'grammar_reference'
      WHEN r.source_type = 'exercise' THEN 'exercise'
      ELSE 'external_link'
    END AS source_kind,
    r.title AS title,
    r.summary AS summary,
    r.source_name AS author_name,
    r.language_code AS language_code,
    r.published_at AS published_at
  FROM catalog_resources r
  CROSS JOIN manual_provider mp
)
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
SELECT
  b.provider_id,
  b.external_id,
  b.external_url,
  b.source_kind,
  b.title,
  b.summary,
  b.author_name,
  b.language_code,
  jsonb_build_object(
    'origin', 'migration_0002',
    'catalog_resource_id', b.resource_id::text
  ),
  'synced',
  now(),
  b.published_at,
  now(),
  now()
FROM backfill_records b
ON CONFLICT (provider_id, external_id, source_kind) DO UPDATE
SET external_url = EXCLUDED.external_url,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    author_name = EXCLUDED.author_name,
    language_code = EXCLUDED.language_code,
    raw_payload = EXCLUDED.raw_payload,
    sync_status = EXCLUDED.sync_status,
    last_synced_at = EXCLUDED.last_synced_at,
    published_at = EXCLUDED.published_at,
    updated_at = now();

WITH manual_provider AS (
  SELECT id
  FROM source_providers
  WHERE slug = 'manual'
),
source_lookup AS (
  SELECT
    r.id AS resource_id,
    sr.id AS source_record_id
  FROM catalog_resources r
  INNER JOIN manual_provider mp ON true
  INNER JOIN source_records sr
    ON sr.provider_id = mp.id
   AND sr.external_id = COALESCE(r.slug, r.id::text)
   AND sr.source_kind = CASE
      WHEN r.source_type = 'youtube' THEN 'video'
      WHEN r.source_type = 'playlist' THEN 'playlist'
      WHEN r.source_type = 'course' THEN 'course'
      WHEN r.source_type = 'article' THEN 'article'
      WHEN r.source_type = 'podcast' THEN 'podcast'
      WHEN r.source_type = 'grammar_reference' THEN 'grammar_reference'
      WHEN r.source_type = 'exercise' THEN 'exercise'
      ELSE 'external_link'
   END
)
UPDATE catalog_resources r
SET source_record_id = sl.source_record_id,
    ingestion_origin = COALESCE(NULLIF(r.ingestion_origin, ''), 'manual'),
    updated_at = now()
FROM source_lookup sl
WHERE r.id = sl.resource_id
  AND (r.source_record_id IS DISTINCT FROM sl.source_record_id OR r.ingestion_origin IS DISTINCT FROM 'manual');

CREATE INDEX IF NOT EXISTS idx_source_providers_slug ON source_providers (slug);
CREATE INDEX IF NOT EXISTS idx_source_records_provider_kind ON source_records (provider_id, source_kind);
CREATE INDEX IF NOT EXISTS idx_source_records_last_synced ON source_records (provider_id, last_synced_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status_created ON import_jobs (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_job_results_job ON import_job_results (import_job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_profiles_target_level ON user_profiles (target_level);
