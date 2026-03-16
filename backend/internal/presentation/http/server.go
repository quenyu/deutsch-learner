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
	savedapp "deutsch-learner/backend/internal/application/saved"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
)

const defaultDemoUserID = "11111111-1111-1111-1111-111111111111"

type Server struct {
	catalogService *catalogapp.Service
	savedService   *savedapp.Service
	options        Options
}

type Options struct {
	CORSAllowedOrigins          []string
	MaxBodyBytes                int64
	MaxConcurrentRequests       int
	RateLimitEnabled            bool
	RateLimitRequestsPerWindow  int
	RateLimitWindow             time.Duration
	HandlerTimeout              time.Duration
	SlowRequestThreshold        time.Duration
	ReadinessChecks             []ReadinessCheck
	ReadinessTimeout            time.Duration
}

func NewServer(catalogService *catalogapp.Service, savedService *savedapp.Service, options Options) *Server {
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
		catalogService: catalogService,
		savedService:   savedService,
		options:        options,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/api/v1/resources", s.handleResources)
	mux.HandleFunc("/api/v1/resources/", s.handleResources)
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

	payload := make([]resourceResponse, 0, len(resources))
	for _, resource := range resources {
		payload = append(payload, mapResource(resource, false))
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

	userID := currentUserID(r)
	isSaved, err := s.savedService.IsSaved(r.Context(), userID, resource.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Message: "could not fetch saved state"})
		return
	}

	writeJSON(w, http.StatusOK, mapResource(resource, isSaved))
}

func (s *Server) listSavedResources(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
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

	userID := currentUserID(r)
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

	userID := currentUserID(r)
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

type apiError struct {
	Message string `json:"message"`
}

type resourceResponse struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	SourceName      string   `json:"sourceName"`
	SourceType      string   `json:"sourceType"`
	ExternalURL     string   `json:"externalUrl"`
	CEFRLevel       string   `json:"cefrLevel"`
	Format          string   `json:"format"`
	DurationMinutes int      `json:"durationMinutes"`
	IsFree          bool     `json:"isFree"`
	PriceCents      *int     `json:"priceCents,omitempty"`
	SkillTags       []string `json:"skillTags"`
	TopicTags       []string `json:"topicTags"`
	IsSaved         bool     `json:"isSaved"`
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
	}
}

func currentUserID(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if value == "" {
		return defaultDemoUserID
	}
	return value
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

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, apiError{Message: "method not allowed"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
