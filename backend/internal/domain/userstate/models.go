package userstate

import "time"

type SavedResource struct {
	UserID     string    `json:"userId"`
	ResourceID string    `json:"resourceId"`
	SavedAt    time.Time `json:"savedAt"`
}

type ProgressStatus string

const (
	ProgressNotStarted ProgressStatus = "not_started"
	ProgressInProgress ProgressStatus = "in_progress"
	ProgressCompleted  ProgressStatus = "completed"
)

type ResourceProgress struct {
	UserID          string         `json:"userId"`
	ResourceID      string         `json:"resourceId"`
	Status          ProgressStatus `json:"status"`
	ProgressPercent float64        `json:"progressPercent"`
	LastStudiedAt   *time.Time     `json:"lastStudiedAt,omitempty"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type ResourceNote struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	ResourceID string    `json:"resourceId"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ReviewItem struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	ResourceID  string    `json:"resourceId"`
	Prompt      string    `json:"prompt"`
	Response    string    `json:"response"`
	EaseFactor  float64   `json:"easeFactor"`
	IntervalDay int       `json:"intervalDay"`
	Repetitions int       `json:"repetitions"`
	DueAt       time.Time `json:"dueAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
