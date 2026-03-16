package httpapi

import "context"

type ReadinessCheck struct {
	Name  string
	Check func(context.Context) error
}

func runReadinessChecks(ctx context.Context, checks []ReadinessCheck) map[string]string {
	if len(checks) == 0 {
		return nil
	}

	failures := make(map[string]string)
	for _, check := range checks {
		if check.Check == nil {
			continue
		}

		if err := check.Check(ctx); err != nil {
			name := check.Name
			if name == "" {
				name = "unknown"
			}
			failures[name] = err.Error()
		}
	}

	if len(failures) == 0 {
		return nil
	}
	return failures
}
