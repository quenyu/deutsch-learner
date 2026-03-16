package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainsource "deutsch-learner/backend/internal/domain/source"
)

var youtubeDurationPattern = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

type Config struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Fetcher struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewFetcher(cfg Config) *Fetcher {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://www.googleapis.com/youtube/v3"
	}

	client := cfg.HTTPClient
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 20 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return &Fetcher{
		apiKey:     strings.TrimSpace(cfg.APIKey),
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: client,
	}
}

func (f *Fetcher) ProviderSlug() domainsource.ProviderSlug {
	return domainsource.ProviderYouTube
}

func (f *Fetcher) Fetch(ctx context.Context, request sourceimportapp.FetchRequest) ([]sourceimportapp.FetchedResource, error) {
	if f.apiKey == "" {
		return nil, fmt.Errorf("youtube api key is required")
	}

	mode := strings.ToLower(strings.TrimSpace(request.Mode))
	switch mode {
	case "video-search":
		query := strings.TrimSpace(request.Query)
		if query == "" {
			return nil, fmt.Errorf("query is required for youtube video-search mode")
		}
		return f.fetchVideoSearch(ctx, query, normalizeLimit(request.Limit), strings.TrimSpace(request.Language))
	case "playlist":
		playlistID := strings.TrimSpace(request.PlaylistID)
		if playlistID == "" {
			return nil, fmt.Errorf("playlist id is required for youtube playlist mode")
		}
		return f.fetchPlaylist(ctx, playlistID)
	case "channel":
		channelID := strings.TrimSpace(request.ChannelID)
		if channelID == "" {
			return nil, fmt.Errorf("channel id is required for youtube channel mode")
		}
		return f.fetchChannel(ctx, channelID)
	default:
		return nil, fmt.Errorf("unsupported youtube mode %q", request.Mode)
	}
}

func (f *Fetcher) fetchVideoSearch(
	ctx context.Context,
	query string,
	limit int,
	language string,
) ([]sourceimportapp.FetchedResource, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("maxResults", strconv.Itoa(limit))
	params.Set("q", query)
	if language != "" {
		params.Set("relevanceLanguage", strings.ToLower(language))
	}

	var searchResponse youtubeSearchResponse
	if err := f.getJSON(ctx, "/search", params, &searchResponse); err != nil {
		return nil, err
	}

	videoIDs := make([]string, 0, len(searchResponse.Items))
	searchByID := make(map[string]youtubeSearchItem, len(searchResponse.Items))
	for _, item := range searchResponse.Items {
		videoID := strings.TrimSpace(item.ID.VideoID)
		if videoID == "" {
			continue
		}
		videoIDs = append(videoIDs, videoID)
		searchByID[videoID] = item
	}
	if len(videoIDs) == 0 {
		return []sourceimportapp.FetchedResource{}, nil
	}

	params = url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("id", strings.Join(videoIDs, ","))
	params.Set("maxResults", strconv.Itoa(len(videoIDs)))

	var videosResponse youtubeVideosResponse
	if err := f.getJSON(ctx, "/videos", params, &videosResponse); err != nil {
		return nil, err
	}

	videoByID := make(map[string]youtubeVideoItem, len(videosResponse.Items))
	for _, item := range videosResponse.Items {
		videoByID[item.ID] = item
	}

	result := make([]sourceimportapp.FetchedResource, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		videoItem, ok := videoByID[videoID]
		if !ok {
			searchItem := searchByID[videoID]
			videoItem = youtubeVideoItem{
				ID:      videoID,
				Snippet: searchItem.Snippet,
			}
		}

		payload, _ := json.Marshal(videoItem)
		publishedAt := parseTimestamp(videoItem.Snippet.PublishedAt)
		durationMinutes := parseYouTubeDurationMinutes(videoItem.ContentDetails.Duration)

		resource := sourceimportapp.FetchedResource{
			ExternalID:      videoID,
			ExternalURL:     "https://www.youtube.com/watch?v=" + videoID,
			SourceKind:      domainsource.SourceKindVideo,
			SourceType:      domaincatalog.ResourceTypeYouTube,
			Title:           strings.TrimSpace(videoItem.Snippet.Title),
			Summary:         strings.TrimSpace(videoItem.Snippet.Description),
			SourceName:      strings.TrimSpace(videoItem.Snippet.ChannelTitle),
			AuthorName:      strings.TrimSpace(videoItem.Snippet.ChannelTitle),
			Format:          "video",
			DurationMinutes: durationMinutes,
			IsFree:          true,
			PublishedAt:     publishedAt,
			RawPayload:      payload,
			LanguageCode:    firstNonEmpty(videoItem.Snippet.DefaultLanguage, videoItem.Snippet.DefaultAudioLanguage),
		}

		result = append(result, resource)
	}

	return result, nil
}

func (f *Fetcher) fetchPlaylist(ctx context.Context, playlistID string) ([]sourceimportapp.FetchedResource, error) {
	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("id", playlistID)
	params.Set("maxResults", "1")

	var response youtubePlaylistsResponse
	if err := f.getJSON(ctx, "/playlists", params, &response); err != nil {
		return nil, err
	}
	if len(response.Items) == 0 {
		return []sourceimportapp.FetchedResource{}, nil
	}

	item := response.Items[0]
	payload, _ := json.Marshal(item)
	publishedAt := parseTimestamp(item.Snippet.PublishedAt)

	resource := sourceimportapp.FetchedResource{
		ExternalID:      item.ID,
		ExternalURL:     "https://www.youtube.com/playlist?list=" + item.ID,
		SourceKind:      domainsource.SourceKindPlaylist,
		SourceType:      domaincatalog.ResourceTypePlaylist,
		Title:           strings.TrimSpace(item.Snippet.Title),
		Summary:         strings.TrimSpace(item.Snippet.Description),
		SourceName:      strings.TrimSpace(item.Snippet.ChannelTitle),
		AuthorName:      strings.TrimSpace(item.Snippet.ChannelTitle),
		Format:          "playlist",
		DurationMinutes: 0,
		IsFree:          true,
		PublishedAt:     publishedAt,
		RawPayload:      payload,
		LanguageCode:    firstNonEmpty(item.Snippet.DefaultLanguage, item.Snippet.DefaultAudioLanguage),
	}

	return []sourceimportapp.FetchedResource{resource}, nil
}

func (f *Fetcher) fetchChannel(ctx context.Context, channelID string) ([]sourceimportapp.FetchedResource, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("id", channelID)
	params.Set("maxResults", "1")

	var response youtubeChannelsResponse
	if err := f.getJSON(ctx, "/channels", params, &response); err != nil {
		return nil, err
	}
	if len(response.Items) == 0 {
		return []sourceimportapp.FetchedResource{}, nil
	}

	item := response.Items[0]
	payload, _ := json.Marshal(item)
	publishedAt := parseTimestamp(item.Snippet.PublishedAt)

	resource := sourceimportapp.FetchedResource{
		ExternalID:      item.ID,
		ExternalURL:     "https://www.youtube.com/channel/" + item.ID,
		SourceKind:      domainsource.SourceKindChannel,
		SourceType:      domaincatalog.ResourceTypeYouTube,
		Title:           strings.TrimSpace(item.Snippet.Title),
		Summary:         strings.TrimSpace(item.Snippet.Description),
		SourceName:      strings.TrimSpace(item.Snippet.Title),
		AuthorName:      strings.TrimSpace(item.Snippet.Title),
		Format:          "channel",
		DurationMinutes: 0,
		IsFree:          true,
		PublishedAt:     publishedAt,
		RawPayload:      payload,
		LanguageCode:    firstNonEmpty(item.Snippet.DefaultLanguage, item.Snippet.DefaultAudioLanguage),
	}

	return []sourceimportapp.FetchedResource{resource}, nil
}

func (f *Fetcher) getJSON(ctx context.Context, path string, params url.Values, target any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", f.apiKey)

	endpoint := f.baseURL + path + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	response, err := f.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return fmt.Errorf("youtube api request failed: %s", strings.TrimSpace(string(body)))
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return 10
	case limit > 50:
		return 50
	default:
		return limit
	}
}

func parseYouTubeDurationMinutes(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}

	matches := youtubeDurationPattern.FindStringSubmatch(value)
	if len(matches) != 4 {
		return 0
	}

	hours := parseInt(matches[1])
	minutes := parseInt(matches[2])
	seconds := parseInt(matches[3])

	totalMinutes := hours*60 + minutes
	if seconds > 0 {
		totalMinutes++
	}
	return totalMinutes
}

func parseTimestamp(raw string) *time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}

	normalized := parsed.UTC()
	return &normalized
}

func parseInt(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type youtubeSnippet struct {
	Title                string `json:"title"`
	Description          string `json:"description"`
	ChannelTitle         string `json:"channelTitle"`
	PublishedAt          string `json:"publishedAt"`
	DefaultLanguage      string `json:"defaultLanguage"`
	DefaultAudioLanguage string `json:"defaultAudioLanguage"`
}

type youtubeSearchResponse struct {
	Items []youtubeSearchItem `json:"items"`
}

type youtubeSearchItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet youtubeSnippet `json:"snippet"`
}

type youtubeVideosResponse struct {
	Items []youtubeVideoItem `json:"items"`
}

type youtubeVideoItem struct {
	ID             string         `json:"id"`
	Snippet        youtubeSnippet `json:"snippet"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
}

type youtubePlaylistsResponse struct {
	Items []youtubePlaylistItem `json:"items"`
}

type youtubePlaylistItem struct {
	ID             string         `json:"id"`
	Snippet        youtubeSnippet `json:"snippet"`
	ContentDetails struct {
		ItemCount int `json:"itemCount"`
	} `json:"contentDetails"`
}

type youtubeChannelsResponse struct {
	Items []youtubeChannelItem `json:"items"`
}

type youtubeChannelItem struct {
	ID      string         `json:"id"`
	Snippet youtubeSnippet `json:"snippet"`
}
