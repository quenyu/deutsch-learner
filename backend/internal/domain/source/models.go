package source

import "time"

type ProviderSlug string

const (
	ProviderManual     ProviderSlug = "manual"
	ProviderYouTube    ProviderSlug = "youtube"
	ProviderStepik     ProviderSlug = "stepik"
	ProviderEnrichment ProviderSlug = "enrichment"
)

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusFailed  SyncStatus = "failed"
)

type SourceKind string

const (
	SourceKindVideo            SourceKind = "video"
	SourceKindPlaylist         SourceKind = "playlist"
	SourceKindChannel          SourceKind = "channel"
	SourceKindCourse           SourceKind = "course"
	SourceKindLesson           SourceKind = "lesson"
	SourceKindArticle          SourceKind = "article"
	SourceKindPodcast          SourceKind = "podcast"
	SourceKindGrammarReference SourceKind = "grammar_reference"
	SourceKindExercise         SourceKind = "exercise"
	SourceKindExternalLink     SourceKind = "external_link"
	SourceKindVocabulary       SourceKind = "vocabulary"
	SourceKindGrammar          SourceKind = "grammar"
	SourceKindSentence         SourceKind = "sentence"
)

type IngestionOrigin string

const (
	IngestionOriginManual   IngestionOrigin = "manual"
	IngestionOriginImported IngestionOrigin = "imported"
)

type ImportJobStatus string

const (
	ImportJobPending   ImportJobStatus = "pending"
	ImportJobRunning   ImportJobStatus = "running"
	ImportJobCompleted ImportJobStatus = "completed"
	ImportJobFailed    ImportJobStatus = "failed"
)

type ImportResultStatus string

const (
	ImportResultImported ImportResultStatus = "imported"
	ImportResultUpdated  ImportResultStatus = "updated"
	ImportResultSkipped  ImportResultStatus = "skipped"
	ImportResultFailed   ImportResultStatus = "failed"
)

type Provider struct {
	ID          string       `json:"id"`
	Slug        ProviderSlug `json:"slug"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsEnabled   bool         `json:"isEnabled"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type Record struct {
	ID           string       `json:"id"`
	ProviderID   string       `json:"providerId"`
	ProviderSlug ProviderSlug `json:"providerSlug"`
	ExternalID   string       `json:"externalId"`
	ExternalURL  string       `json:"externalUrl"`
	SourceKind   SourceKind   `json:"sourceKind"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	AuthorName   string       `json:"authorName"`
	LanguageCode string       `json:"languageCode"`
	RawPayload   []byte       `json:"rawPayload"`
	SyncStatus   SyncStatus   `json:"syncStatus"`
	LastSyncedAt *time.Time   `json:"lastSyncedAt,omitempty"`
	PublishedAt  *time.Time   `json:"publishedAt,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type ImportJob struct {
	ID         string          `json:"id"`
	ProviderID string          `json:"providerId"`
	Mode       string          `json:"mode"`
	Status     ImportJobStatus `json:"status"`
	Query      string          `json:"query"`
	PlaylistID string          `json:"playlistId"`
	ChannelID  string          `json:"channelId"`
	FilePath   string          `json:"filePath"`
	LimitCount int             `json:"limitCount"`
	CEFRHint   string          `json:"cefrHint"`
	SkillsHint []string        `json:"skillsHint"`
	TopicsHint []string        `json:"topicsHint"`
	IsFreeHint *bool           `json:"isFreeHint,omitempty"`
	StartedAt  *time.Time      `json:"startedAt,omitempty"`
	EndedAt    *time.Time      `json:"endedAt,omitempty"`
	Error      string          `json:"error"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

type ImportResult struct {
	ID                string             `json:"id"`
	ImportJobID       string             `json:"importJobId"`
	SourceRecordID    *string            `json:"sourceRecordId,omitempty"`
	CatalogResourceID *string            `json:"catalogResourceId,omitempty"`
	Status            ImportResultStatus `json:"status"`
	Message           string             `json:"message"`
	CreatedAt         time.Time          `json:"createdAt"`
}
