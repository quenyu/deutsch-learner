package postgres

import "testing"

func TestParseJSONStringArray(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		values, err := parseJSONStringArray("")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(values) != 0 {
			t.Fatalf("expected empty array, got %+v", values)
		}
	})

	t.Run("valid", func(t *testing.T) {
		values, err := parseJSONStringArray(`["grammar","listening"]`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(values) != 2 {
			t.Fatalf("expected 2 values, got %+v", values)
		}
		if values[0] != "grammar" || values[1] != "listening" {
			t.Fatalf("unexpected values %+v", values)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseJSONStringArray(`{}`)
		if err == nil {
			t.Fatalf("expected error for non-array json input")
		}
	})
}
