package httpapi

import (
	"context"
	"errors"
	"testing"
)

func TestRunReadinessChecks(t *testing.T) {
	checks := []ReadinessCheck{
		{
			Name: "postgres",
			Check: func(context.Context) error {
				return nil
			},
		},
		{
			Name: "redis",
			Check: func(context.Context) error {
				return errors.New("dial timeout")
			},
		},
	}

	failures := runReadinessChecks(context.Background(), checks)
	if len(failures) != 1 {
		t.Fatalf("expected one failed readiness check, got %d", len(failures))
	}

	if failures["redis"] != "dial timeout" {
		t.Fatalf("unexpected readiness failure payload: %#v", failures)
	}
}
