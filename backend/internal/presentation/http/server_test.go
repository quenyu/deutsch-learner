package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	savedapp "deutsch-learner/backend/internal/application/saved"
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

type serverOption func(*memory.SavedRepository)

func seedSavedForUser(userID, resourceID string) serverOption {
	return func(repo *memory.SavedRepository) {
		_, _ = repo.Save(context.Background(), userID, resourceID)
	}
}

func newServerHandler(t *testing.T, options ...serverOption) http.Handler {
	t.Helper()

	catalogRepo := memory.NewCatalogRepository(memory.DefaultResources())
	savedRepo := memory.NewSavedRepository()
	for _, option := range options {
		option(savedRepo)
	}

	server := NewServer(
		catalogapp.NewService(catalogRepo),
		savedapp.NewService(savedRepo),
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
