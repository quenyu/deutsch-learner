CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS app_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_resources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  source_name TEXT NOT NULL,
  source_type TEXT NOT NULL CHECK (
    source_type IN (
      'youtube',
      'article',
      'playlist',
      'course',
      'podcast',
      'grammar_reference',
      'exercise'
    )
  ),
  external_url TEXT NOT NULL,
  cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2')),
  format TEXT NOT NULL,
  duration_minutes INTEGER NOT NULL DEFAULT 0 CHECK (duration_minutes >= 0),
  is_free BOOLEAN NOT NULL DEFAULT true,
  price_cents INTEGER CHECK (price_cents IS NULL OR price_cents >= 0),
  language_code TEXT NOT NULL DEFAULT 'de',
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_skills (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_topics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_resource_skills (
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  skill_id UUID NOT NULL REFERENCES catalog_skills(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (resource_id, skill_id)
);

CREATE TABLE IF NOT EXISTS catalog_resource_topics (
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  topic_id UUID NOT NULL REFERENCES catalog_topics(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (resource_id, topic_id)
);

CREATE TABLE IF NOT EXISTS catalog_learning_paths (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  cefr_level TEXT CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2')),
  is_published BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_learning_path_items (
  learning_path_id UUID NOT NULL REFERENCES catalog_learning_paths(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  step_order INTEGER NOT NULL CHECK (step_order > 0),
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (learning_path_id, step_order),
  UNIQUE (learning_path_id, resource_id)
);

CREATE TABLE IF NOT EXISTS user_saved_resources (
  user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  saved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, resource_id)
);

CREATE TABLE IF NOT EXISTS user_resource_progress (
  user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('not_started', 'in_progress', 'completed')),
  progress_percent NUMERIC(5,2) NOT NULL DEFAULT 0.00 CHECK (progress_percent >= 0 AND progress_percent <= 100),
  last_studied_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, resource_id)
);

CREATE TABLE IF NOT EXISTS user_resource_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_review_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  prompt TEXT NOT NULL,
  response TEXT NOT NULL DEFAULT '',
  ease_factor NUMERIC(4,2) NOT NULL DEFAULT 2.50,
  interval_days INTEGER NOT NULL DEFAULT 1,
  repetitions INTEGER NOT NULL DEFAULT 0,
  due_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_recommendations (
  user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
  resource_id UUID NOT NULL REFERENCES catalog_resources(id) ON DELETE CASCADE,
  score NUMERIC(5,2) NOT NULL DEFAULT 0,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, resource_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_resources_level ON catalog_resources (cefr_level);
CREATE INDEX IF NOT EXISTS idx_catalog_resources_source_type ON catalog_resources (source_type);
CREATE INDEX IF NOT EXISTS idx_catalog_resources_is_free ON catalog_resources (is_free);
CREATE INDEX IF NOT EXISTS idx_catalog_resource_skills_skill_id ON catalog_resource_skills (skill_id);
CREATE INDEX IF NOT EXISTS idx_catalog_resource_topics_topic_id ON catalog_resource_topics (topic_id);
CREATE INDEX IF NOT EXISTS idx_user_saved_resources_saved_at ON user_saved_resources (user_id, saved_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_progress_updated_at ON user_resource_progress (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_review_due ON user_review_items (user_id, due_at);
