package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	"deutsch-learner/backend/internal/infrastructure/postgres"
	"deutsch-learner/backend/internal/infrastructure/sourceproviders/manual"
	"deutsch-learner/backend/internal/infrastructure/sourceproviders/youtube"
	"deutsch-learner/backend/internal/platform/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()

	var (
		provider   string
		mode       string
		query      string
		playlistID string
		channelID  string
		filePath   string
		limit      int
		cefrHint   string
		skillsRaw  string
		topicsRaw  string
		isFreeRaw  string
		language   string
	)

	flag.StringVar(&provider, "provider", "", "source provider slug: youtube|manual")
	flag.StringVar(&mode, "mode", "", "import mode: video-search|playlist|channel|file")
	flag.StringVar(&query, "query", "", "query for search modes")
	flag.StringVar(&playlistID, "playlist-id", "", "youtube playlist id")
	flag.StringVar(&channelID, "channel-id", "", "youtube channel id")
	flag.StringVar(&filePath, "file", "", "manual import file path")
	flag.IntVar(&limit, "limit", 10, "max import records for search mode")
	flag.StringVar(&cefrHint, "cefr", "", "normalization hint (A1-A2-B1-B2-C1-C2)")
	flag.StringVar(&skillsRaw, "skills", "", "comma-separated skills hint")
	flag.StringVar(&topicsRaw, "topics", "", "comma-separated topics hint")
	flag.StringVar(&isFreeRaw, "is-free", "", "optional free flag hint: true|false")
	flag.StringVar(&language, "language", "de", "language hint")
	flag.Parse()

	isFreeHint, err := parseOptionalBool(isFreeRaw)
	if err != nil {
		log.Fatalf("parse --is-free: %v", err)
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open postgres connection: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.SourceImportTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("postgres connectivity check failed: %v", err)
	}

	repo := postgres.NewSourceRepository(db)
	service := sourceimportapp.NewService(repo, []sourceimportapp.ProviderFetcher{
		manual.NewFetcher(),
		youtube.NewFetcher(youtube.Config{
			APIKey:  cfg.YouTubeAPIKey,
			BaseURL: cfg.YouTubeAPIBaseURL,
			Timeout: cfg.SourceImportTimeout,
		}),
	})

	summary, err := service.RunImport(ctx, sourceimportapp.ImportRequest{
		Provider:   provider,
		Mode:       mode,
		Query:      query,
		PlaylistID: playlistID,
		ChannelID:  channelID,
		FilePath:   filePath,
		Limit:      limit,
		CEFRHint:   cefrHint,
		Skills:     csvList(skillsRaw),
		Topics:     csvList(topicsRaw),
		IsFree:     isFreeHint,
		Language:   language,
		Initiator:  "cli",
	})
	if err != nil {
		log.Fatalf("run import: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		log.Fatalf("print summary: %v", err)
	}
}

func csvList(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func parseOptionalBool(raw string) (*bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}
