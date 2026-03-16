package profile

import (
	"errors"
	"time"
)

var ErrProfileNotFound = errors.New("profile not found")

type UserProfile struct {
	UserID                   string    `json:"userId"`
	DisplayName              string    `json:"displayName"`
	TargetLevel              *string   `json:"targetLevel,omitempty"`
	LearningGoals            string    `json:"learningGoals"`
	PreferredResourceTypes   []string  `json:"preferredResourceTypes"`
	PreferredSkills          []string  `json:"preferredSkills"`
	PreferredSourceProviders []string  `json:"preferredSourceProviders"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}
