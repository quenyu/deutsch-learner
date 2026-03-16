package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

func TestFetchVideoSearchMapsYouTubePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Fatalf("expected api key to be present")
		}

		switch r.URL.Path {
		case "/search":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": map[string]any{"videoId": "video-1"},
						"snippet": map[string]any{
							"title":        "German A1 Intro",
							"description":  "Start learning basic German.",
							"channelTitle": "Learn German",
							"publishedAt":  "2025-01-01T10:00:00Z",
						},
					},
					{
						"id": map[string]any{"videoId": "video-2"},
						"snippet": map[string]any{
							"title":        "German Cases",
							"description":  "Understand nominative and accusative.",
							"channelTitle": "Grammar Lab",
							"publishedAt":  "2025-01-10T10:00:00Z",
						},
					},
				},
			})
		case "/videos":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "video-1",
						"snippet": map[string]any{
							"title":        "German A1 Intro",
							"description":  "Start learning basic German.",
							"channelTitle": "Learn German",
							"publishedAt":  "2025-01-01T10:00:00Z",
						},
						"contentDetails": map[string]any{
							"duration": "PT5M30S",
						},
					},
					{
						"id": "video-2",
						"snippet": map[string]any{
							"title":        "German Cases",
							"description":  "Understand nominative and accusative.",
							"channelTitle": "Grammar Lab",
							"publishedAt":  "2025-01-10T10:00:00Z",
						},
						"contentDetails": map[string]any{
							"duration": "PT1H2M",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	items, err := fetcher.Fetch(context.Background(), sourceimportapp.FetchRequest{
		Mode:     "video-search",
		Query:    "german a1",
		Limit:    2,
		Language: "de",
	})
	if err != nil {
		t.Fatalf("fetch video-search: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].ExternalID != "video-1" {
		t.Fatalf("expected first external id video-1, got %s", items[0].ExternalID)
	}
	if items[0].SourceKind != domainsource.SourceKindVideo {
		t.Fatalf("expected video source kind, got %s", items[0].SourceKind)
	}
	if items[0].SourceType != domaincatalog.ResourceTypeYouTube {
		t.Fatalf("expected youtube source type, got %s", items[0].SourceType)
	}
	if items[0].DurationMinutes != 6 {
		t.Fatalf("expected rounded duration 6 minutes, got %d", items[0].DurationMinutes)
	}
	if items[1].DurationMinutes != 62 {
		t.Fatalf("expected duration 62 minutes, got %d", items[1].DurationMinutes)
	}
}

func TestFetchPlaylistAndChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playlists":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "playlist-1",
						"snippet": map[string]any{
							"title":        "A2 Listening Playlist",
							"description":  "Curated listening practice.",
							"channelTitle": "Learn German",
						},
					},
				},
			})
		case "/channels":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "channel-1",
						"snippet": map[string]any{
							"title":       "Easy German",
							"description": "Street interviews in German.",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fetcher := NewFetcher(Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	playlist, err := fetcher.Fetch(context.Background(), sourceimportapp.FetchRequest{
		Mode:       "playlist",
		PlaylistID: "playlist-1",
	})
	if err != nil {
		t.Fatalf("fetch playlist: %v", err)
	}
	if len(playlist) != 1 {
		t.Fatalf("expected single playlist resource, got %d", len(playlist))
	}
	if playlist[0].SourceKind != domainsource.SourceKindPlaylist {
		t.Fatalf("expected playlist source kind, got %s", playlist[0].SourceKind)
	}
	if playlist[0].ExternalURL != "https://www.youtube.com/playlist?list=playlist-1" {
		t.Fatalf("unexpected playlist url %s", playlist[0].ExternalURL)
	}

	channel, err := fetcher.Fetch(context.Background(), sourceimportapp.FetchRequest{
		Mode:      "channel",
		ChannelID: "channel-1",
	})
	if err != nil {
		t.Fatalf("fetch channel: %v", err)
	}
	if len(channel) != 1 {
		t.Fatalf("expected single channel resource, got %d", len(channel))
	}
	if channel[0].SourceKind != domainsource.SourceKindChannel {
		t.Fatalf("expected channel source kind, got %s", channel[0].SourceKind)
	}
	if channel[0].Format != "channel" {
		t.Fatalf("expected channel format, got %s", channel[0].Format)
	}
}
