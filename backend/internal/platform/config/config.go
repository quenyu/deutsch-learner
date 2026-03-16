package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	DataBackend string
	PostgresDSN string
	RedisAddr   string

	ReadTimeout           time.Duration
	ReadHeaderTimeout     time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	ShutdownTimeout       time.Duration
	HandlerTimeout        time.Duration
	SlowRequestThreshold  time.Duration
	ReadinessTimeout      time.Duration
	MaxHeaderBytes        int
	MaxBodyBytes          int64
	MaxConcurrentRequests int

	CORSAllowedOrigins []string

	RateLimitEnabled           bool
	RateLimitRequestsPerWindow int
	RateLimitWindow            time.Duration

	SeedEnabled       bool
	SeedDemoUserID    string
	SeedDemoUserEmail string
	SeedDemoUserName  string
}

func Load() Config {
	return Config{
		Port:        getEnv("APP_PORT", "8080"),
		DataBackend: getEnvOneOf("DATA_BACKEND", "postgres", map[string]struct{}{"memory": {}, "postgres": {}}),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/deutsch_learner?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),

		ReadTimeout:           getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		ReadHeaderTimeout:     getEnvDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		WriteTimeout:          getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:           getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:       getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		HandlerTimeout:        getEnvDuration("HTTP_HANDLER_TIMEOUT", 8*time.Second),
		SlowRequestThreshold:  getEnvDuration("HTTP_SLOW_REQUEST_THRESHOLD", 600*time.Millisecond),
		ReadinessTimeout:      getEnvDuration("HTTP_READINESS_TIMEOUT", 2*time.Second),
		MaxHeaderBytes:        getEnvInt("HTTP_MAX_HEADER_BYTES", 1<<20),
		MaxBodyBytes:          getEnvInt64("HTTP_MAX_BODY_BYTES", 1<<20),
		MaxConcurrentRequests: getEnvInt("HTTP_MAX_CONCURRENT_REQUESTS", 200),

		CORSAllowedOrigins: getEnvCSV(
			"CORS_ALLOWED_ORIGINS",
			[]string{
				"http://localhost:5173",
				"http://127.0.0.1:5173",
				"http://localhost:4173",
				"http://127.0.0.1:4173",
			},
		),

		RateLimitEnabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequestsPerWindow: getEnvInt("RATE_LIMIT_REQUESTS_PER_WINDOW", 120),
		RateLimitWindow:            getEnvDuration("RATE_LIMIT_WINDOW", 1*time.Minute),

		SeedEnabled:       getEnvBool("SEED_ENABLED", true),
		SeedDemoUserID:    getEnv("SEED_DEMO_USER_ID", "11111111-1111-1111-1111-111111111111"),
		SeedDemoUserEmail: getEnv("SEED_DEMO_USER_EMAIL", "demo@deutschlearner.local"),
		SeedDemoUserName:  getEnv("SEED_DEMO_USER_NAME", "Demo Learner"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvCSV(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return fallback
	}
	return result
}

func getEnvOneOf(key, fallback string, allowed map[string]struct{}) string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	if _, ok := allowed[value]; !ok {
		return fallback
	}

	return value
}
