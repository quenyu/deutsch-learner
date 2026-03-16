//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	savedapp "deutsch-learner/backend/internal/application/saved"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	httpapi "deutsch-learner/backend/internal/presentation/http"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	integrationDemoUserID   = "11111111-1111-1111-1111-111111111111"
	integrationResourceID   = "c01f9204-5f38-47e4-b3ec-c580691ff44f"
	integrationResourceSlug = "dw-nicos-weg-a1-overview"
)

func TestCatalogAndSavedRepositoriesIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()

	catalogRepo := NewCatalogRepository(db)
	resources, err := catalogRepo.ListResources(ctx, domaincatalog.ListFilter{Limit: 25, Offset: 0})
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(resources) == 0 {
		t.Fatalf("expected non-empty catalog")
	}

	resource, err := catalogRepo.GetResourceBySlug(ctx, integrationResourceSlug)
	if err != nil {
		t.Fatalf("get resource by slug: %v", err)
	}
	if resource.ID != integrationResourceID {
		t.Fatalf("unexpected resource id: got %s", resource.ID)
	}

	savedRepo := NewSavedRepository(db)
	cleanupSavedState(t, ctx, db)

	created, err := savedRepo.Save(ctx, integrationDemoUserID, integrationResourceID)
	if err != nil {
		t.Fatalf("save resource: %v", err)
	}
	if !created {
		t.Fatalf("expected first save to create state")
	}

	created, err = savedRepo.Save(ctx, integrationDemoUserID, integrationResourceID)
	if err != nil {
		t.Fatalf("save resource (idempotent): %v", err)
	}
	if created {
		t.Fatalf("expected second save to be idempotent")
	}

	ids, err := savedRepo.ListResourceIDs(ctx, integrationDemoUserID)
	if err != nil {
		t.Fatalf("list saved resource ids: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("expected saved resources to include seeded resource id")
	}

	isSaved, err := savedRepo.IsSaved(ctx, integrationDemoUserID, integrationResourceID)
	if err != nil {
		t.Fatalf("is saved: %v", err)
	}
	if !isSaved {
		t.Fatalf("expected IsSaved to return true")
	}

	removed, err := savedRepo.Remove(ctx, integrationDemoUserID, integrationResourceID)
	if err != nil {
		t.Fatalf("remove saved resource: %v", err)
	}
	if !removed {
		t.Fatalf("expected resource to be removed")
	}
}

func TestPersistedHTTPFlowIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()
	catalogRepo := NewCatalogRepository(db)
	savedRepo := NewSavedRepository(db)

	cleanupSavedState(t, ctx, db)
	_, _ = savedRepo.Save(ctx, integrationDemoUserID, integrationResourceID)
	defer cleanupSavedState(t, ctx, db)

	server := httpapi.NewServer(
		catalogapp.NewService(catalogRepo),
		savedapp.NewService(savedRepo),
		httpapi.Options{RateLimitEnabled: false},
	).Routes()

	resourceListReq := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	resourceListReq.Header.Set("X-User-ID", integrationDemoUserID)
	resourceListResp := httptest.NewRecorder()
	server.ServeHTTP(resourceListResp, resourceListReq)
	if resourceListResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/v1/resources, got %d", resourceListResp.Code)
	}

	var listPayload struct {
		Items []struct {
			ID      string `json:"id"`
			IsSaved bool   `json:"isSaved"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resourceListResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}

	foundSaved := false
	for _, item := range listPayload.Items {
		if item.ID == integrationResourceID && item.IsSaved {
			foundSaved = true
		}
	}
	if !foundSaved {
		t.Fatalf("expected saved resource to be marked as saved in /api/v1/resources")
	}

	meMissingHeaderReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/saved-resources", nil)
	meMissingHeaderResp := httptest.NewRecorder()
	server.ServeHTTP(meMissingHeaderResp, meMissingHeaderReq)
	if meMissingHeaderResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for /api/v1/me/saved-resources without header, got %d", meMissingHeaderResp.Code)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/saved-resources", nil)
	meReq.Header.Set("X-User-ID", integrationDemoUserID)
	meResp := httptest.NewRecorder()
	server.ServeHTTP(meResp, meReq)
	if meResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/me/saved-resources, got %d", meResp.Code)
	}

	var mePayload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(meResp.Body).Decode(&mePayload); err != nil {
		t.Fatalf("decode /api/v1/me/saved-resources payload: %v", err)
	}
	if len(mePayload.Items) == 0 {
		t.Fatalf("expected non-empty saved resources payload")
	}
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping postgres: %v", err)
	}

	migrationsDir := filepath.Join("..", "..", "..", "db", "migrations")
	if _, err := ApplyMigrations(ctx, db, migrationsDir); err != nil {
		_ = db.Close()
		t.Fatalf("apply migrations: %v", err)
	}

	if err := SeedCuratedCatalog(ctx, db, SeedOptions{
		Enabled:       true,
		DemoUserID:    integrationDemoUserID,
		DemoUserEmail: "demo@deutschlearner.local",
		DemoUserName:  "Demo Learner",
	}); err != nil {
		_ = db.Close()
		t.Fatalf("seed curated catalog: %v", err)
	}

	return db
}

func cleanupSavedState(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(
		ctx,
		`
DELETE FROM user_saved_resources
WHERE user_id = $1::uuid
  AND resource_id = $2::uuid;
`,
		integrationDemoUserID,
		integrationResourceID,
	); err != nil {
		t.Fatalf("cleanup saved state: %v", err)
	}
}
