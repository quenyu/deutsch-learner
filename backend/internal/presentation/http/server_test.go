package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	profileapp "deutsch-learner/backend/internal/application/profile"
	progressapp "deutsch-learner/backend/internal/application/progress"
	savedapp "deutsch-learner/backend/internal/application/saved"
	sourceapp "deutsch-learner/backend/internal/application/source"
	userstate "deutsch-learner/backend/internal/domain/userstate"
	"deutsch-learner/backend/internal/infrastructure/memory"
)

const (
	testUserID       = "11111111-1111-1111-1111-111111111111"
	testResourceID   = "c01f9204-5f38-47e4-b3ec-c580691ff44f"
	testResourceSlug = "dw-nicos-weg-a1-overview"
)

func TestMeSavedEndpointsRequireUserHeader(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/saved-resources", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when X-User-ID is missing, got %d", recorder.Code)
	}
}

func TestMeProgressEndpointsRequireUserHeader(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/progress", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when X-User-ID is missing for progress, got %d", recorder.Code)
	}
}

func TestMeProfileEndpointRequiresUserHeader(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when X-User-ID is missing for profile, got %d", recorder.Code)
	}
}

func TestGetAndPutProfileFlow(t *testing.T) {
	handler := newServerHandler(t)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile", nil)
	getRequest.Header.Set("X-User-ID", testUserID)
	getRecorder := httptest.NewRecorder()
	handler.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for profile get, got %d", getRecorder.Code)
	}

	var initial userProfileResponse
	if err := json.NewDecoder(getRecorder.Body).Decode(&initial); err != nil {
		t.Fatalf("decode initial profile: %v", err)
	}
	if initial.UserID != testUserID {
		t.Fatalf("expected profile user id %s, got %s", testUserID, initial.UserID)
	}

	body := bytes.NewBufferString(`{
		"displayName":"Max Mustermann",
		"targetLevel":"B1",
		"learningGoals":"Build confidence in daily conversations",
		"preferredResourceTypes":["youtube","course"],
		"preferredSkills":["listening","vocab"],
		"preferredSourceProviders":["youtube","manual"]
	}`)
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/me/profile", body)
	putRequest.Header.Set("X-User-ID", testUserID)
	putRequest.Header.Set("Content-Type", "application/json")
	putRecorder := httptest.NewRecorder()
	handler.ServeHTTP(putRecorder, putRequest)

	if putRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for profile put, got %d", putRecorder.Code)
	}

	var updated userProfileResponse
	if err := json.NewDecoder(putRecorder.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	if updated.DisplayName != "Max Mustermann" {
		t.Fatalf("expected updated display name, got %q", updated.DisplayName)
	}
	if updated.TargetLevel == nil || *updated.TargetLevel != "B1" {
		t.Fatalf("expected updated target level B1, got %+v", updated.TargetLevel)
	}
	if len(updated.PreferredSourceProviders) != 2 {
		t.Fatalf("expected preferred providers to be persisted, got %+v", updated.PreferredSourceProviders)
	}
	if len(updated.PreferredSkills) != 2 || updated.PreferredSkills[1] != "vocabulary" {
		t.Fatalf("expected preferred skills normalization, got %+v", updated.PreferredSkills)
	}
}

func TestSourceProvidersEndpoint(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/source-providers", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for source providers, got %d", recorder.Code)
	}

	var payload struct {
		Items []sourceProviderResponse `json:"items"`
		Count int                      `json:"count"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode source providers payload: %v", err)
	}
	if payload.Count == 0 || len(payload.Items) == 0 {
		t.Fatalf("expected source providers payload to be non-empty")
	}
}

func TestResourcesFilterByProviderAndType(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources?provider=manual&type=course", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for filtered resources, got %d", recorder.Code)
	}

	payload := decodeListPayload(t, recorder)
	if payload.Count == 0 {
		t.Fatalf("expected filtered resources to include course records")
	}
	for _, item := range payload.Items {
		if item.ProviderSlug != "manual" {
			t.Fatalf("expected provider slug manual, got %q", item.ProviderSlug)
		}
		if item.SourceType != "course" {
			t.Fatalf("expected source type course, got %q", item.SourceType)
		}
	}
}

func TestResourcesRejectInvalidUserHeaderWhenProvided(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	request.Header.Set("X-User-ID", "not-a-uuid")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid optional X-User-ID header, got %d", recorder.Code)
	}
}

func TestResourcesListReturnsUnsavedWithoutHeader(t *testing.T) {
	handler := newServerHandler(t, seedSavedForUser(testUserID, testResourceID))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	payload := decodeListPayload(t, recorder)
	item := findResource(t, payload.Items, testResourceID)
	if item.IsSaved {
		t.Fatalf("expected resource to be unsaved without user header")
	}
}

func TestResourcesListComputesSavedStateFromHeader(t *testing.T) {
	handler := newServerHandler(t, seedSavedForUser(testUserID, testResourceID))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	request.Header.Set("X-User-ID", testUserID)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	payload := decodeListPayload(t, recorder)
	item := findResource(t, payload.Items, testResourceID)
	if !item.IsSaved {
		t.Fatalf("expected resource to be marked as saved for matching user")
	}
}

func TestResourceDetailRespectsOptionalUserHeader(t *testing.T) {
	handler := newServerHandler(t, seedSavedForUser(testUserID, testResourceID))

	withoutHeader := httptest.NewRequest(http.MethodGet, "/api/v1/resources/"+testResourceSlug, nil)
	withoutHeaderRecorder := httptest.NewRecorder()
	handler.ServeHTTP(withoutHeaderRecorder, withoutHeader)
	if withoutHeaderRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 without header, got %d", withoutHeaderRecorder.Code)
	}

	var withoutUser resourceResponse
	if err := json.NewDecoder(withoutHeaderRecorder.Body).Decode(&withoutUser); err != nil {
		t.Fatalf("decode detail payload: %v", err)
	}
	if withoutUser.IsSaved {
		t.Fatalf("expected detail response to be unsaved without X-User-ID")
	}

	withHeader := httptest.NewRequest(http.MethodGet, "/api/v1/resources/"+testResourceSlug, nil)
	withHeader.Header.Set("X-User-ID", testUserID)
	withHeaderRecorder := httptest.NewRecorder()
	handler.ServeHTTP(withHeaderRecorder, withHeader)
	if withHeaderRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 with header, got %d", withHeaderRecorder.Code)
	}

	var withUser resourceResponse
	if err := json.NewDecoder(withHeaderRecorder.Body).Decode(&withUser); err != nil {
		t.Fatalf("decode detail payload with user: %v", err)
	}
	if !withUser.IsSaved {
		t.Fatalf("expected detail response to be saved with valid user header")
	}
}

func TestProgressGetReturnsDefaultWhenMissing(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/progress/"+testResourceID, nil)
	request.Header.Set("X-User-ID", testUserID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for default progress state, got %d", recorder.Code)
	}

	payload := decodeProgressPayload(t, recorder)
	if payload.Status != string(userstate.ProgressNotStarted) {
		t.Fatalf("expected default status %q, got %q", userstate.ProgressNotStarted, payload.Status)
	}
	if payload.ProgressPercent != 0 {
		t.Fatalf("expected default progress percent to be 0, got %v", payload.ProgressPercent)
	}
}

func TestProgressPutAndListFlow(t *testing.T) {
	handler := newServerHandler(t)

	body := bytes.NewBufferString(`{"status":"in_progress"}`)
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v1/me/progress/"+testResourceID, body)
	updateRequest.Header.Set("X-User-ID", testUserID)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateRecorder := httptest.NewRecorder()
	handler.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 when updating progress, got %d", updateRecorder.Code)
	}

	updated := decodeProgressPayload(t, updateRecorder)
	if updated.Status != string(userstate.ProgressInProgress) {
		t.Fatalf("expected status %q, got %q", userstate.ProgressInProgress, updated.Status)
	}
	if updated.ProgressPercent != 50 {
		t.Fatalf("expected progress percent 50 for in_progress, got %v", updated.ProgressPercent)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me/progress", nil)
	listRequest.Header.Set("X-User-ID", testUserID)
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listRequest)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from progress list endpoint, got %d", listRecorder.Code)
	}

	var listPayload struct {
		Items []progressResponse `json:"items"`
		Count int                `json:"count"`
	}
	if err := json.NewDecoder(listRecorder.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode progress list payload: %v", err)
	}
	if listPayload.Count == 0 {
		t.Fatalf("expected progress list to include updated resource")
	}
	if listPayload.Items[0].ResourceID != testResourceID {
		t.Fatalf("expected first progress item to match resource %s, got %s", testResourceID, listPayload.Items[0].ResourceID)
	}
}

func TestProgressUpdateRejectsInvalidStatus(t *testing.T) {
	handler := newServerHandler(t)

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/me/progress/"+testResourceID,
		bytes.NewBufferString(`{"status":"paused"}`),
	)
	request.Header.Set("X-User-ID", testUserID)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid progress status, got %d", recorder.Code)
	}
}

type serverOption func(*serverFixture)

type serverFixture struct {
	savedRepo    *memory.SavedRepository
	progressRepo *memory.ProgressRepository
	profileRepo  *memory.ProfileRepository
	sourceRepo   *memory.SourceRepository
}

func seedSavedForUser(userID, resourceID string) serverOption {
	return func(fixture *serverFixture) {
		_, _ = fixture.savedRepo.Save(context.Background(), userID, resourceID)
	}
}

func seedProgressForUser(userID, resourceID string, status userstate.ProgressStatus) serverOption {
	return func(fixture *serverFixture) {
		_, _ = fixture.progressRepo.UpsertStatus(context.Background(), userID, resourceID, status)
	}
}

func newServerHandler(t *testing.T, options ...serverOption) http.Handler {
	t.Helper()

	catalogRepo := memory.NewCatalogRepository(memory.DefaultResources())
	fixture := &serverFixture{
		savedRepo:    memory.NewSavedRepository(),
		progressRepo: memory.NewProgressRepository(),
		profileRepo:  memory.NewProfileRepository(),
		sourceRepo:   memory.NewSourceRepository(),
	}
	for _, option := range options {
		option(fixture)
	}

	server := NewServer(
		catalogapp.NewService(catalogRepo),
		profileapp.NewService(fixture.profileRepo),
		progressapp.NewService(fixture.progressRepo),
		savedapp.NewService(fixture.savedRepo),
		sourceapp.NewService(fixture.sourceRepo),
		Options{
			RateLimitEnabled: false,
			HandlerTimeout:   2 * time.Second,
		},
	)

	return server.Routes()
}

type listResponsePayload struct {
	Items []resourceResponse `json:"items"`
	Count int                `json:"count"`
}

func decodeListPayload(t *testing.T, recorder *httptest.ResponseRecorder) listResponsePayload {
	t.Helper()
	var payload listResponsePayload
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}
	return payload
}

func findResource(t *testing.T, items []resourceResponse, resourceID string) resourceResponse {
	t.Helper()
	for _, item := range items {
		if item.ID == resourceID {
			return item
		}
	}
	t.Fatalf("resource %s not found in payload", resourceID)
	return resourceResponse{}
}

func decodeProgressPayload(t *testing.T, recorder *httptest.ResponseRecorder) progressResponse {
	t.Helper()
	var payload progressResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode progress payload: %v", err)
	}
	return payload
}
