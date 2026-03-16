package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	profileapp "deutsch-learner/backend/internal/application/profile"
	progressapp "deutsch-learner/backend/internal/application/progress"
	savedapp "deutsch-learner/backend/internal/application/saved"
	sourceapp "deutsch-learner/backend/internal/application/source"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainprofile "deutsch-learner/backend/internal/domain/profile"
	domainsource "deutsch-learner/backend/internal/domain/source"
	userstate "deutsch-learner/backend/internal/domain/userstate"
)

var (
	errMissingUserIDHeader = errors.New("X-User-ID header is required for /api/v1/me endpoints")
	errInvalidUserIDHeader = errors.New("X-User-ID must be a valid UUID")
)

type Server struct {
	catalogService  *catalogapp.Service
	profileService  *profileapp.Service
	progressService *progressapp.Service
	savedService    *savedapp.Service
	sourceService   *sourceapp.Service
	options         Options
}

type Options struct {
	CORSAllowedOrigins         []string
	MaxBodyBytes               int64
	MaxConcurrentRequests      int
	RateLimitEnabled           bool
	RateLimitRequestsPerWindow int
	RateLimitWindow            time.Duration
	HandlerTimeout             time.Duration
	SlowRequestThreshold       time.Duration
	ReadinessChecks            []ReadinessCheck
	ReadinessTimeout           time.Duration
}

func NewServer(
	catalogService *catalogapp.Service,
	profileService *profileapp.Service,
	progressService *progressapp.Service,
	savedService *savedapp.Service,
	sourceService *sourceapp.Service,
	options Options,
) *Server {
	if len(options.CORSAllowedOrigins) == 0 {
		options.CORSAllowedOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	if options.MaxBodyBytes <= 0 {
		options.MaxBodyBytes = 1 << 20
	}
	if options.MaxConcurrentRequests <= 0 {
		options.MaxConcurrentRequests = 200
	}
	if options.RateLimitRequestsPerWindow <= 0 {
		options.RateLimitRequestsPerWindow = 120
	}
	if options.RateLimitWindow <= 0 {
		options.RateLimitWindow = time.Minute
	}
	if options.HandlerTimeout <= 0 {
		options.HandlerTimeout = 8 * time.Second
	}
	if options.SlowRequestThreshold <= 0 {
		options.SlowRequestThreshold = 600 * time.Millisecond
	}
	if options.ReadinessTimeout <= 0 {
		options.ReadinessTimeout = 2 * time.Second
	}

	return &Server{
		catalogService:  catalogService,
		profileService:  profileService,
		progressService: progressService,
		savedService:    savedService,
		sourceService:   sourceService,
		options:         options,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/api/v1/resources", s.handleResources)
	mux.HandleFunc("/api/v1/resources/", s.handleResources)
	mux.HandleFunc("/api/v1/source-providers", s.handleSourceProviders)
	mux.HandleFunc("/api/v1/me/profile", s.handleProfile)
	mux.HandleFunc("/api/v1/me/progress", s.handleProgress)
	mux.HandleFunc("/api/v1/me/progress/", s.handleProgress)
	mux.HandleFunc("/api/v1/me/saved-resources", s.handleSavedResources)
	mux.HandleFunc("/api/v1/me/saved-resources/", s.handleSavedResources)

	middlewares := []middleware{
		withRecovery(),
		withRequestID(),
		withLogging(s.options.SlowRequestThreshold),
		withSecurityHeaders(),
		withCORS(s.options.CORSAllowedOrigins),
		withBodyLimit(s.options.MaxBodyBytes),
		withHandlerTimeout(s.options.HandlerTimeout),
		withConcurrencyLimit(s.options.MaxConcurrentRequests),
	}

	if s.options.RateLimitEnabled {
		limiter := newFixedWindowLimiter(s.options.RateLimitRequestsPerWindow, s.options.RateLimitWindow)
		middlewares = append(middlewares, withRateLimit(limiter))
	}

	return chain(mux, middlewares...)
}

func (s *Server) handleSourceProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	providers, err := s.sourceService.ListProviders(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch source providers"})
		return
	}

	items := make([]sourceProviderResponse, 0, len(providers))
	for _, provider := range providers {
		items = append(items, mapSourceProvider(provider))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getProfile(w, r)
	case http.MethodPut:
		s.putProfile(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.options.ReadinessTimeout)
	defer cancel()

	failures := runReadinessChecks(ctx, s.options.ReadinessChecks)
	if len(failures) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":   "not_ready",
			"failures": failures,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resources":
		s.listResources(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/resources/"):
		s.getResourceBySlug(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) handleSavedResources(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me/saved-resources":
		s.listSavedResources(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/me/saved-resources":
		s.saveResource(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/me/saved-resources/"):
		s.removeSavedResource(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me/progress":
		s.listProgress(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/me/progress/"):
		s.getProgress(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/me/progress/"):
		s.updateProgress(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var onlyFree *bool
	if query.Has("free") {
		value, err := strconv.ParseBool(query.Get("free"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Message: "free must be true or false"})
			return
		}
		onlyFree = &value
	}

	limit := parseIntWithFallback(query.Get("limit"), 24)
	offset := parseIntWithFallback(query.Get("offset"), 0)

	filter := domaincatalog.ListFilter{
		Level:    query.Get("level"),
		Skill:    query.Get("skill"),
		Topic:    query.Get("topic"),
		Provider: query.Get("provider"),
		Type:     query.Get("type"),
		Query:    query.Get("q"),
		OnlyFree: onlyFree,
		Limit:    limit,
		Offset:   offset,
	}

	resources, err := s.catalogService.ListResources(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch resources"})
		return
	}

	userID, hasUserID, err := optionalUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	savedByResourceID := make(map[string]struct{})
	if hasUserID {
		savedResourceIDs, err := s.savedService.ListResourceIDs(r.Context(), userID)
		if err != nil {
			writeJSON(w, statusFromSavedError(err), apiError{Message: err.Error()})
			return
		}

		for _, savedResourceID := range savedResourceIDs {
			savedByResourceID[savedResourceID] = struct{}{}
		}
	}

	payload := make([]resourceResponse, 0, len(resources))
	for _, resource := range resources {
		_, isSaved := savedByResourceID[resource.ID]
		payload = append(payload, mapResource(resource, isSaved))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": payload,
		"count": len(payload),
	})
}

func (s *Server) getResourceBySlug(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/v1/resources/")
	if strings.TrimSpace(slug) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "resource slug is required"})
		return
	}

	resource, err := s.catalogService.GetResourceBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, domaincatalog.ErrResourceNotFound) {
			writeJSON(w, http.StatusNotFound, apiError{Message: "resource not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch resource"})
		return
	}

	userID, hasUserID, err := optionalUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	isSaved := false
	if hasUserID {
		isSaved, err = s.savedService.IsSaved(r.Context(), userID, resource.ID)
		if err != nil {
			writeJSON(w, statusFromSavedError(err), apiError{Message: err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, mapResource(resource, isSaved))
}

func (s *Server) listSavedResources(w http.ResponseWriter, r *http.Request) {
	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	resourceIDs, err := s.savedService.ListResourceIDs(r.Context(), userID)
	if err != nil {
		writeJSON(w, statusFromSavedError(err), apiError{Message: err.Error()})
		return
	}

	savedResources := make([]resourceResponse, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		resource, err := s.catalogService.GetResourceByID(r.Context(), resourceID)
		if err != nil {
			if errors.Is(err, domaincatalog.ErrResourceNotFound) {
				continue
			}
			writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch saved resources"})
			return
		}

		savedResources = append(savedResources, mapResource(resource, true))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": savedResources,
		"count": len(savedResources),
	})
}

func (s *Server) saveResource(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ResourceID string `json:"resourceId"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, apiError{Message: "request body is too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body is invalid"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body must contain a single JSON object"})
		return
	}

	request.ResourceID = strings.TrimSpace(request.ResourceID)
	if request.ResourceID == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "resource id is required"})
		return
	}

	if _, err := s.catalogService.GetResourceByID(r.Context(), request.ResourceID); err != nil {
		if errors.Is(err, domaincatalog.ErrResourceNotFound) {
			writeJSON(w, http.StatusNotFound, apiError{Message: "resource not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not save resource"})
		return
	}

	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	created, err := s.savedService.Save(r.Context(), userID, request.ResourceID)
	if err != nil {
		writeJSON(w, statusFromSavedError(err), apiError{Message: err.Error()})
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	writeJSON(w, status, map[string]any{
		"saved":   true,
		"created": created,
	})
}

func (s *Server) removeSavedResource(w http.ResponseWriter, r *http.Request) {
	resourceID := strings.TrimPrefix(r.URL.Path, "/api/v1/me/saved-resources/")
	if strings.TrimSpace(resourceID) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "resource id is required"})
		return
	}

	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	removed, err := s.savedService.Remove(r.Context(), userID, resourceID)
	if err != nil {
		writeJSON(w, statusFromSavedError(err), apiError{Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"saved":   false,
		"removed": removed,
	})
}

func (s *Server) listProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	progressRows, err := s.progressService.ListByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, statusFromProgressError(err), apiError{Message: err.Error()})
		return
	}

	items := make([]progressResponse, 0, len(progressRows))
	for _, progress := range progressRows {
		items = append(items, mapProgress(progress))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	resourceID := strings.TrimPrefix(r.URL.Path, "/api/v1/me/progress/")
	if strings.TrimSpace(resourceID) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "resource id is required"})
		return
	}

	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	if _, err := s.catalogService.GetResourceByID(r.Context(), resourceID); err != nil {
		if errors.Is(err, domaincatalog.ErrResourceNotFound) {
			writeJSON(w, http.StatusNotFound, apiError{Message: "resource not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch progress"})
		return
	}

	progress, found, err := s.progressService.GetByUserAndResource(r.Context(), userID, resourceID)
	if err != nil {
		writeJSON(w, statusFromProgressError(err), apiError{Message: err.Error()})
		return
	}

	if !found {
		writeJSON(w, http.StatusOK, progressResponse{
			UserID:          userID,
			ResourceID:      resourceID,
			Status:          string(userstate.ProgressNotStarted),
			ProgressPercent: userstate.ProgressPercentForStatus(userstate.ProgressNotStarted),
		})
		return
	}

	writeJSON(w, http.StatusOK, mapProgress(progress))
}

func (s *Server) updateProgress(w http.ResponseWriter, r *http.Request) {
	resourceID := strings.TrimPrefix(r.URL.Path, "/api/v1/me/progress/")
	if strings.TrimSpace(resourceID) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "resource id is required"})
		return
	}

	var request struct {
		Status string `json:"status"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, apiError{Message: "request body is too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body is invalid"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body must contain a single JSON object"})
		return
	}

	request.Status = strings.TrimSpace(request.Status)
	if request.Status == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "progress status is required"})
		return
	}

	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	if _, err := s.catalogService.GetResourceByID(r.Context(), resourceID); err != nil {
		if errors.Is(err, domaincatalog.ErrResourceNotFound) {
			writeJSON(w, http.StatusNotFound, apiError{Message: "resource not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not update progress"})
		return
	}

	progress, err := s.progressService.SetStatus(r.Context(), userID, resourceID, userstate.ProgressStatus(request.Status))
	if err != nil {
		writeJSON(w, statusFromProgressError(err), apiError{Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mapProgress(progress))
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	profile, err := s.profileService.GetByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, statusFromProfileError(err), apiError{Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mapUserProfile(profile))
}

func (s *Server) putProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := requiredUserID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}

	var request struct {
		DisplayName              string   `json:"displayName"`
		TargetLevel              *string  `json:"targetLevel"`
		LearningGoals            string   `json:"learningGoals"`
		PreferredResourceTypes   []string `json:"preferredResourceTypes"`
		PreferredSkills          []string `json:"preferredSkills"`
		PreferredSourceProviders []string `json:"preferredSourceProviders"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, apiError{Message: "request body is too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body is invalid"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, apiError{Message: "request body must contain a single JSON object"})
		return
	}

	profile, err := s.profileService.Upsert(r.Context(), profileapp.UpsertInput{
		UserID:                   userID,
		DisplayName:              request.DisplayName,
		TargetLevel:              request.TargetLevel,
		LearningGoals:            request.LearningGoals,
		PreferredResourceTypes:   request.PreferredResourceTypes,
		PreferredSkills:          request.PreferredSkills,
		PreferredSourceProviders: request.PreferredSourceProviders,
	})
	if err != nil {
		writeJSON(w, statusFromProfileError(err), apiError{Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mapUserProfile(profile))
}

type apiError struct {
	Message string `json:"message"`
}

type resourceResponse struct {
	ID              string     `json:"id"`
	Slug            string     `json:"slug"`
	Title           string     `json:"title"`
	Summary         string     `json:"summary"`
	SourceName      string     `json:"sourceName"`
	SourceType      string     `json:"sourceType"`
	ExternalURL     string     `json:"externalUrl"`
	CEFRLevel       string     `json:"cefrLevel"`
	Format          string     `json:"format"`
	DurationMinutes int        `json:"durationMinutes"`
	IsFree          bool       `json:"isFree"`
	PriceCents      *int       `json:"priceCents,omitempty"`
	SkillTags       []string   `json:"skillTags"`
	TopicTags       []string   `json:"topicTags"`
	IsSaved         bool       `json:"isSaved"`
	ProviderSlug    string     `json:"providerSlug"`
	ProviderName    string     `json:"providerName"`
	IngestionOrigin string     `json:"ingestionOrigin"`
	SourceKind      string     `json:"sourceKind"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt,omitempty"`
}

type progressResponse struct {
	UserID          string     `json:"userId"`
	ResourceID      string     `json:"resourceId"`
	Status          string     `json:"status"`
	ProgressPercent float64    `json:"progressPercent"`
	LastStudiedAt   *time.Time `json:"lastStudiedAt,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
}

type sourceProviderResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
}

type userProfileResponse struct {
	UserID                   string    `json:"userId"`
	DisplayName              string    `json:"displayName"`
	TargetLevel              *string   `json:"targetLevel,omitempty"`
	LearningGoals            string    `json:"learningGoals"`
	PreferredResourceTypes   []string  `json:"preferredResourceTypes"`
	PreferredSkills          []string  `json:"preferredSkills"`
	PreferredSourceProviders []string  `json:"preferredSourceProviders"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

func mapResource(resource domaincatalog.Resource, isSaved bool) resourceResponse {
	return resourceResponse{
		ID:              resource.ID,
		Slug:            resource.Slug,
		Title:           resource.Title,
		Summary:         resource.Summary,
		SourceName:      resource.SourceName,
		SourceType:      string(resource.SourceType),
		ExternalURL:     resource.ExternalURL,
		CEFRLevel:       string(resource.CEFRLevel),
		Format:          resource.Format,
		DurationMinutes: resource.DurationMinutes,
		IsFree:          resource.IsFree,
		PriceCents:      resource.PriceCents,
		SkillTags:       resource.SkillTags,
		TopicTags:       resource.TopicTags,
		IsSaved:         isSaved,
		ProviderSlug:    resource.ProviderSlug,
		ProviderName:    resource.ProviderName,
		IngestionOrigin: resource.IngestionOrigin,
		SourceKind:      resource.SourceKind,
		LastSyncedAt:    resource.LastSyncedAt,
	}
}

func mapProgress(progress userstate.ResourceProgress) progressResponse {
	response := progressResponse{
		UserID:          progress.UserID,
		ResourceID:      progress.ResourceID,
		Status:          string(progress.Status),
		ProgressPercent: progress.ProgressPercent,
		LastStudiedAt:   progress.LastStudiedAt,
	}

	if !progress.UpdatedAt.IsZero() {
		updatedAt := progress.UpdatedAt
		response.UpdatedAt = &updatedAt
	}

	return response
}

func mapSourceProvider(provider domainsource.Provider) sourceProviderResponse {
	return sourceProviderResponse{
		ID:          provider.ID,
		Slug:        string(provider.Slug),
		Name:        provider.Name,
		Description: provider.Description,
		IsEnabled:   provider.IsEnabled,
	}
}

func mapUserProfile(profile domainprofile.UserProfile) userProfileResponse {
	return userProfileResponse{
		UserID:                   profile.UserID,
		DisplayName:              profile.DisplayName,
		TargetLevel:              profile.TargetLevel,
		LearningGoals:            profile.LearningGoals,
		PreferredResourceTypes:   profile.PreferredResourceTypes,
		PreferredSkills:          profile.PreferredSkills,
		PreferredSourceProviders: profile.PreferredSourceProviders,
		UpdatedAt:                profile.UpdatedAt,
	}
}

func requiredUserID(r *http.Request) (string, error) {
	value, present, err := optionalUserID(r)
	if err != nil {
		return "", err
	}
	if !present {
		return "", errMissingUserIDHeader
	}
	return value, nil
}

func optionalUserID(r *http.Request) (userID string, present bool, err error) {
	value := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if value == "" {
		return "", false, nil
	}

	if !isUUID(value) {
		return "", false, errInvalidUserIDHeader
	}

	return value, true, nil
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !isHexChar(char) {
				return false
			}
		}
	}

	return true
}

func isHexChar(char rune) bool {
	return (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

func parseIntWithFallback(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func statusFromSavedError(err error) int {
	switch {
	case errors.Is(err, savedapp.ErrInvalidUserID):
		return http.StatusBadRequest
	case errors.Is(err, savedapp.ErrInvalidResourceID):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func statusFromProgressError(err error) int {
	switch {
	case errors.Is(err, progressapp.ErrInvalidUserID):
		return http.StatusBadRequest
	case errors.Is(err, progressapp.ErrInvalidResourceID):
		return http.StatusBadRequest
	case errors.Is(err, progressapp.ErrInvalidProgressState):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func statusFromProfileError(err error) int {
	switch {
	case errors.Is(err, profileapp.ErrInvalidUserID):
		return http.StatusBadRequest
	case errors.Is(err, profileapp.ErrInvalidDisplayName):
		return http.StatusBadRequest
	case errors.Is(err, profileapp.ErrInvalidTargetLevel):
		return http.StatusBadRequest
	case errors.Is(err, profileapp.ErrInvalidResourceType):
		return http.StatusBadRequest
	case errors.Is(err, profileapp.ErrInvalidSkill):
		return http.StatusBadRequest
	case errors.Is(err, profileapp.ErrInvalidSourceProvider):
		return http.StatusBadRequest
	case errors.Is(err, domainprofile.ErrProfileNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, apiError{Message: "method not allowed"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
