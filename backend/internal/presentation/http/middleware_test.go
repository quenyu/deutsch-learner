package httpapi

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		withCORS([]string{"http://localhost:5173"}),
	)

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/resources", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid preflight, got %d", recorder.Code)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
}

func TestCORSRejectsUnknownOriginOnPreflight(t *testing.T) {
	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		withCORS([]string{"http://localhost:5173"}),
	)

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/resources", nil)
	request.Header.Set("Origin", "https://malicious.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid preflight origin, got %d", recorder.Code)
	}
}

func TestRequestIDHeaderIsAlwaysSet(t *testing.T) {
	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		withRequestID(),
	)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected request id header to be set")
	}
}

func TestHandlerTimeoutReturnsServiceUnavailable(t *testing.T) {
	handler := chain(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			time.Sleep(40 * time.Millisecond)
		}),
		withHandlerTimeout(10*time.Millisecond),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from timeout handler, got %d", recorder.Code)
	}
}

func TestConcurrencyLimitRejectsExcessRequests(t *testing.T) {
	block := make(chan struct{})
	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			<-block
			w.WriteHeader(http.StatusOK)
		}),
		withConcurrencyLimit(1),
	)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	firstRecorder := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.ServeHTTP(firstRecorder, firstReq)
	}()

	time.Sleep(15 * time.Millisecond)

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, secondReq)

	if secondRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected second request to be rejected while first is in-flight, got %d", secondRecorder.Code)
	}

	close(block)
	wg.Wait()
}
