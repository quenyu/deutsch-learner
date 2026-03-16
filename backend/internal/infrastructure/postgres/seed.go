package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type SeedOptions struct {
	Enabled       bool
	DemoUserID    string
	DemoUserEmail string
	DemoUserName  string
}

type seededResource struct {
	ID              string
	Slug            string
	Title           string
	Summary         string
	SourceName      string
	SourceType      string
	ExternalURL     string
	CEFRLevel       string
	Format          string
	DurationMinutes int
	IsFree          bool
	PriceCents      *int
	SkillTags       []string
	TopicTags       []string
}

var curatedSkills = []struct {
	Slug  string
	Label string
}{
	{Slug: "listening", Label: "Listening"},
	{Slug: "speaking", Label: "Speaking"},
	{Slug: "vocabulary", Label: "Vocabulary"},
	{Slug: "grammar", Label: "Grammar"},
	{Slug: "reading", Label: "Reading"},
	{Slug: "news", Label: "News Comprehension"},
}

var curatedTopics = []struct {
	Slug  string
	Label string
}{
	{Slug: "daily-life", Label: "Daily Life"},
	{Slug: "introductions", Label: "Introductions"},
	{Slug: "culture", Label: "Culture"},
	{Slug: "grammar", Label: "Grammar"},
	{Slug: "cases", Label: "Cases"},
	{Slug: "current-events", Label: "Current Events"},
	{Slug: "exam-prep", Label: "Exam Preparation"},
	{Slug: "conversation", Label: "Conversation"},
	{Slug: "basics", Label: "Basics"},
}

var curatedSourceProviders = []struct {
	Slug        string
	Name        string
	Description string
}{
	{
		Slug:        "manual",
		Name:        "Manual Curation",
		Description: "Curated links entered by editors.",
	},
	{
		Slug:        "youtube",
		Name:        "YouTube Data API",
		Description: "Imported metadata from YouTube videos, playlists, and channels.",
	},
	{
		Slug:        "stepik",
		Name:        "Stepik API",
		Description: "Imported metadata from Stepik courses and lessons.",
	},
	{
		Slug:        "enrichment",
		Name:        "German Enrichment",
		Description: "Optional metadata enrichment hooks.",
	},
}

var curatedResources = []seededResource{
	{
		ID:              "c01f9204-5f38-47e4-b3ec-c580691ff44f",
		Slug:            "dw-nicos-weg-a1-overview",
		Title:           "Nicos Weg A1 Overview",
		Summary:         "Structured beginner-friendly entry into practical German for daily communication.",
		SourceName:      "Deutsche Welle",
		SourceType:      "course",
		ExternalURL:     "https://learngerman.dw.com/en/nicos-weg/c-36519687",
		CEFRLevel:       "A1",
		Format:          "course",
		DurationMinutes: 40,
		IsFree:          true,
		SkillTags:       []string{"listening", "speaking"},
		TopicTags:       []string{"daily-life", "introductions"},
	},
	{
		ID:              "4c55f202-c4c5-43c8-98ac-3ccf490f4b7f",
		Slug:            "easy-german-street-interviews-a2",
		Title:           "Easy German Street Interviews",
		Summary:         "Natural spoken German with subtitles and context from real-world conversations.",
		SourceName:      "Easy German",
		SourceType:      "youtube",
		ExternalURL:     "https://www.youtube.com/@EasyGerman",
		CEFRLevel:       "A2",
		Format:          "video",
		DurationMinutes: 18,
		IsFree:          true,
		SkillTags:       []string{"listening", "vocabulary"},
		TopicTags:       []string{"culture", "daily-life"},
	},
	{
		ID:              "ad7a4a03-3ee8-4ec1-a21a-b9ca3ee2676b",
		Slug:            "yourdailygerman-case-system-guide",
		Title:           "German Case System Guide",
		Summary:         "Clear explanation of nominative, accusative, dative, and genitive with examples.",
		SourceName:      "Your Daily German",
		SourceType:      "article",
		ExternalURL:     "https://yourdailygerman.com/german-case-system/",
		CEFRLevel:       "B1",
		Format:          "article",
		DurationMinutes: 25,
		IsFree:          true,
		SkillTags:       []string{"grammar", "reading"},
		TopicTags:       []string{"grammar", "cases"},
	},
	{
		ID:              "c58d0f53-fc1f-41d4-a8a0-91f77f74895d",
		Slug:            "langsam-gesprochene-nachrichten-b2",
		Title:           "Langsam Gesprochene Nachrichten",
		Summary:         "Slow-paced news audio that helps intermediate learners build listening confidence.",
		SourceName:      "Deutsche Welle",
		SourceType:      "podcast",
		ExternalURL:     "https://www.dw.com/de/deutsch-lernen/nachrichten/s-8030",
		CEFRLevel:       "B2",
		Format:          "podcast",
		DurationMinutes: 12,
		IsFree:          true,
		SkillTags:       []string{"listening", "news"},
		TopicTags:       []string{"current-events"},
	},
	{
		ID:              "2828d81f-45bb-4410-8653-df2a1d4e657d",
		Slug:            "goethe-online-live-group-course",
		Title:           "Goethe Online Live Group Course",
		Summary:         "Instructor-led classes with structured progression and collaborative exercises.",
		SourceName:      "Goethe-Institut",
		SourceType:      "course",
		ExternalURL:     "https://www.goethe.de/en/spr/kur/typ.html",
		CEFRLevel:       "B1",
		Format:          "course",
		DurationMinutes: 90,
		IsFree:          false,
		PriceCents:      intPtr(24900),
		SkillTags:       []string{"speaking", "grammar"},
		TopicTags:       []string{"exam-prep", "conversation"},
	},
	{
		ID:              "622671c2-1095-44fd-a4e1-325d18f82eb9",
		Slug:            "deutschtrainer-grammar-reference-a2",
		Title:           "Deutschtrainer Grammar Reference",
		Summary:         "Compact grammar explanations for common beginner and lower-intermediate mistakes.",
		SourceName:      "DW Deutschtrainer",
		SourceType:      "grammar_reference",
		ExternalURL:     "https://learngerman.dw.com/en/deutschtrainer/c-56169568",
		CEFRLevel:       "A2",
		Format:          "reference",
		DurationMinutes: 15,
		IsFree:          true,
		SkillTags:       []string{"grammar", "vocabulary"},
		TopicTags:       []string{"daily-life", "basics"},
	},
}

func SeedCuratedCatalog(ctx context.Context, db *sql.DB, options SeedOptions) error {
	if !options.Enabled {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := upsertDemoUser(ctx, tx, options); err != nil {
		return err
	}

	for _, skill := range curatedSkills {
		if err := upsertSkill(ctx, tx, skill.Slug, skill.Label); err != nil {
			return err
		}
	}
	for _, topic := range curatedTopics {
		if err := upsertTopic(ctx, tx, topic.Slug, topic.Label); err != nil {
			return err
		}
	}
	for _, provider := range curatedSourceProviders {
		if err := upsertSourceProvider(ctx, tx, provider.Slug, provider.Name, provider.Description); err != nil {
			return err
		}
	}

	for _, resource := range curatedResources {
		if err := upsertResource(ctx, tx, resource); err != nil {
			return err
		}

		if err := syncResourceSkills(ctx, tx, resource.ID, resource.SkillTags); err != nil {
			return err
		}
		if err := syncResourceTopics(ctx, tx, resource.ID, resource.TopicTags); err != nil {
			return err
		}

		if err := upsertManualSourceRecordForResource(ctx, tx, resource); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func upsertDemoUser(ctx context.Context, tx *sql.Tx, options SeedOptions) error {
	if strings.TrimSpace(options.DemoUserID) == "" {
		return fmt.Errorf("seed demo user id is required")
	}
	if strings.TrimSpace(options.DemoUserEmail) == "" {
		return fmt.Errorf("seed demo user email is required")
	}
	if strings.TrimSpace(options.DemoUserName) == "" {
		return fmt.Errorf("seed demo user name is required")
	}

	_, err := tx.ExecContext(
		ctx,
		`
INSERT INTO app_users (id, email, display_name)
VALUES ($1::uuid, $2, $3)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name;
`,
		options.DemoUserID,
		options.DemoUserEmail,
		options.DemoUserName,
	)
	if err != nil {
		return fmt.Errorf("upsert seed demo user: %w", err)
	}

	return nil
}

func upsertSkill(ctx context.Context, tx *sql.Tx, slug, label string) error {
	_, err := tx.ExecContext(
		ctx,
		`
INSERT INTO catalog_skills (slug, label)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
SET label = EXCLUDED.label;
`,
		slug,
		label,
	)
	if err != nil {
		return fmt.Errorf("upsert skill %q: %w", slug, err)
	}
	return nil
}

func upsertTopic(ctx context.Context, tx *sql.Tx, slug, label string) error {
	_, err := tx.ExecContext(
		ctx,
		`
INSERT INTO catalog_topics (slug, label)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
SET label = EXCLUDED.label;
`,
		slug,
		label,
	)
	if err != nil {
		return fmt.Errorf("upsert topic %q: %w", slug, err)
	}
	return nil
}

func upsertSourceProvider(ctx context.Context, tx *sql.Tx, slug, name, description string) error {
	_, err := tx.ExecContext(
		ctx,
		`
INSERT INTO source_providers (slug, name, description, is_enabled, created_at, updated_at)
VALUES ($1, $2, $3, true, now(), now())
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_enabled = true,
    updated_at = now();
`,
		slug,
		name,
		description,
	)
	if err != nil {
		return fmt.Errorf("upsert source provider %q: %w", slug, err)
	}
	return nil
}

func upsertResource(ctx context.Context, tx *sql.Tx, resource seededResource) error {
	var priceCents any
	if resource.PriceCents != nil {
		priceCents = *resource.PriceCents
	}

	_, err := tx.ExecContext(
		ctx,
		`
INSERT INTO catalog_resources (
  id,
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
  created_at,
  updated_at
) VALUES (
  $1::uuid,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  'de',
  now(),
  now()
)
ON CONFLICT (id) DO UPDATE
SET slug = EXCLUDED.slug,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    source_name = EXCLUDED.source_name,
    source_type = EXCLUDED.source_type,
    external_url = EXCLUDED.external_url,
    cefr_level = EXCLUDED.cefr_level,
    format = EXCLUDED.format,
    duration_minutes = EXCLUDED.duration_minutes,
    is_free = EXCLUDED.is_free,
    price_cents = EXCLUDED.price_cents,
    updated_at = now();
`,
		resource.ID,
		resource.Slug,
		resource.Title,
		resource.Summary,
		resource.SourceName,
		resource.SourceType,
		resource.ExternalURL,
		resource.CEFRLevel,
		resource.Format,
		resource.DurationMinutes,
		resource.IsFree,
		priceCents,
	)
	if err != nil {
		return fmt.Errorf("upsert resource %q: %w", resource.Slug, err)
	}

	return nil
}

func upsertManualSourceRecordForResource(ctx context.Context, tx *sql.Tx, resource seededResource) error {
	payload, err := json.Marshal(map[string]any{
		"origin": "seed",
		"slug":   resource.Slug,
	})
	if err != nil {
		return fmt.Errorf("marshal source payload for %q: %w", resource.Slug, err)
	}

	var sourceRecordID string
	err = tx.QueryRowContext(
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
  created_at,
  updated_at
)
VALUES (
  (SELECT id FROM source_providers WHERE slug = 'manual'),
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  'de',
  $7::jsonb,
  'synced',
  now(),
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
    sync_status = EXCLUDED.sync_status,
    last_synced_at = EXCLUDED.last_synced_at,
    updated_at = now()
RETURNING id::text;
`,
		resource.Slug,
		resource.ExternalURL,
		sourceKindFromType(resource.SourceType),
		resource.Title,
		resource.Summary,
		resource.SourceName,
		string(payload),
	).Scan(&sourceRecordID)
	if err != nil {
		return fmt.Errorf("upsert source record for %q: %w", resource.Slug, err)
	}

	_, err = tx.ExecContext(
		ctx,
		`
UPDATE catalog_resources
SET source_record_id = $2::uuid,
    ingestion_origin = 'manual',
    updated_at = now()
WHERE id = $1::uuid;
`,
		resource.ID,
		sourceRecordID,
	)
	if err != nil {
		return fmt.Errorf("link source record for %q: %w", resource.Slug, err)
	}

	return nil
}

func syncResourceSkills(ctx context.Context, tx *sql.Tx, resourceID string, skillSlugs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_resource_skills WHERE resource_id = $1::uuid;`, resourceID); err != nil {
		return fmt.Errorf("clear skills for resource %q: %w", resourceID, err)
	}

	for _, skillSlug := range skillSlugs {
		result, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_resource_skills (resource_id, skill_id)
SELECT $1::uuid, s.id
FROM catalog_skills s
WHERE s.slug = $2
ON CONFLICT (resource_id, skill_id) DO NOTHING;
`,
			resourceID,
			skillSlug,
		)
		if err != nil {
			return fmt.Errorf("link skill %q to resource %q: %w", skillSlug, resourceID, err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check skill link result for resource %q: %w", resourceID, err)
		}
		if rows == 0 {
			return fmt.Errorf("skill slug %q not found while seeding resource %q", skillSlug, resourceID)
		}
	}

	return nil
}

func syncResourceTopics(ctx context.Context, tx *sql.Tx, resourceID string, topicSlugs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_resource_topics WHERE resource_id = $1::uuid;`, resourceID); err != nil {
		return fmt.Errorf("clear topics for resource %q: %w", resourceID, err)
	}

	for _, topicSlug := range topicSlugs {
		result, err := tx.ExecContext(
			ctx,
			`
INSERT INTO catalog_resource_topics (resource_id, topic_id)
SELECT $1::uuid, t.id
FROM catalog_topics t
WHERE t.slug = $2
ON CONFLICT (resource_id, topic_id) DO NOTHING;
`,
			resourceID,
			topicSlug,
		)
		if err != nil {
			return fmt.Errorf("link topic %q to resource %q: %w", topicSlug, resourceID, err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check topic link result for resource %q: %w", resourceID, err)
		}
		if rows == 0 {
			return fmt.Errorf("topic slug %q not found while seeding resource %q", topicSlug, resourceID)
		}
	}

	return nil
}

func intPtr(v int) *int {
	return &v
}

func sourceKindFromType(sourceType string) string {
	switch strings.TrimSpace(strings.ToLower(sourceType)) {
	case "playlist":
		return "playlist"
	case "course":
		return "course"
	case "article":
		return "article"
	case "podcast":
		return "podcast"
	case "grammar_reference":
		return "grammar_reference"
	case "exercise":
		return "exercise"
	default:
		return "video"
	}
}
