package postgres

import (
	"encoding/json"
	"strings"
)

func parseJSONStringArray(raw string) ([]string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return []string{}, nil
	}

	var parsed []string
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, err
	}
	if parsed == nil {
		return []string{}, nil
	}
	return parsed, nil
}
