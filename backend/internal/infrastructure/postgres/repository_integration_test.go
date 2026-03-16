//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogapp "deutsch-learner/backend/internal/application/catalog"
	profileapp "deutsch-learner/backend/internal/application/profile"
	progressapp "deutsch-learner/backend/internal/application/progress"
	savedapp "deutsch-learner/backend/internal/application/saved"
	sourceapp "deutsch-learner/backend/internal/application/source"
	sourceimportapp "deutsch-learner/backend/internal/application/sourceimport"
	domaincatalog "deutsch-learner/backend/internal/domain/catalog"
	domainprofile "deutsch-learner/backend/internal/domain/profile"
	domainsource "deutsch-learner/backend/internal/domain/source"
	userstate "deutsch-learner/backend/internal/domain/userstate"
	httpapi "deutsch-learner/backend/internal/presentation/http"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	integrationDemoUserID   = "11111111-1111-1111-1111-111111111111"
	integrationResourceID   = "c01f9204-5f38-47e4-b3ec-c580691ff44f"
	integrationResourceSlug = "dw-nicos-weg-a1-overview"
)

func TestCatalogSavedAndProgressRepositoriesIntegration(t *testing.T) {
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
	if resource.ProviderSlug == "" || resource.ProviderName == "" {
		t.Fatalf("expected source provider metadata on catalog resource")
	}
	if resource.IngestionOrigin == "" || resource.SourceKind == "" {
		t.Fatalf("expected ingestion metadata on catalog resource")
	}

	filtered, err := catalogRepo.ListResources(ctx, domaincatalog.ListFilter{
		Provider: "manual",
		Type:     "course",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("list resources with provider/type filter: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected provider/type filter to return rows")
	}
	for _, item := range filtered {
		if item.ProviderSlug != "manual" {
			t.Fatalf("expected provider slug manual, got %s", item.ProviderSlug)
		}
		if item.SourceType != domaincatalog.ResourceTypeCourse {
			t.Fatalf("expected course source type, got %s", item.SourceType)
		}
	}

	savedRepo := NewSavedRepository(db)
	progressRepo := NewProgressRepository(db)
	cleanupSavedState(t, ctx, db)
	cleanupProgressState(t, ctx, db)

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

	progress, err := progressRepo.UpsertStatus(ctx, integrationDemoUserID, integrationResourceID, userstate.ProgressInProgress)
	if err != nil {
		t.Fatalf("upsert in_progress: %v", err)
	}
	if progress.ProgressPercent != 50 {
		t.Fatalf("expected in_progress percent to be 50, got %v", progress.ProgressPercent)
	}

	progress, found, err := progressRepo.GetByUserAndResource(ctx, integrationDemoUserID, integrationResourceID)
	if err != nil {
		t.Fatalf("get progress by user and resource: %v", err)
	}
	if !found {
		t.Fatalf("expected persisted progress entry")
	}
	if progress.Status != userstate.ProgressInProgress {
		t.Fatalf("expected status %q, got %q", userstate.ProgressInProgress, progress.Status)
	}

	progressList, err := progressRepo.ListByUser(ctx, integrationDemoUserID)
	if err != nil {
		t.Fatalf("list progress by user: %v", err)
	}
	if len(progressList) == 0 {
		t.Fatalf("expected progress list to include updated resource")
	}

	progress, err = progressRepo.UpsertStatus(ctx, integrationDemoUserID, integrationResourceID, userstate.ProgressCompleted)
	if err != nil {
		t.Fatalf("upsert completed: %v", err)
	}
	if progress.ProgressPercent != 100 {
		t.Fatalf("expected completed percent to be 100, got %v", progress.ProgressPercent)
	}
}

func TestSourceImportAndProfileRepositoriesIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()
	sourceRepo := NewSourceRepository(db)
	catalogRepo := NewCatalogRepository(db)
	profileRepo := NewProfileRepository(db)

	provider, err := sourceRepo.GetProviderBySlug(ctx, "manual")
	if err != nil {
		t.Fatalf("get provider by slug: %v", err)
	}

	importJob, err := sourceRepo.CreateImportJob(ctx, sourceimportapp.CreateImportJobInput{
		ProviderID: provider.ID,
		Mode:       "file",
		FilePath:   "integration/resources.json",
		Limit:      1,
		CEFRHint:   "A2",
		SkillsHint: []string{"grammar"},
		TopicsHint: []string{"cases"},
	})
	if err != nil {
		t.Fatalf("create import job: %v", err)
	}

	uniqueSuffix := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	sourceRecord, createdRecord, err := sourceRepo.UpsertSourceRecord(ctx, sourceimportapp.UpsertSourceRecordInput{
		ProviderID:   provider.ID,
		ProviderSlug: domainsource.ProviderManual,
		ExternalID:   uniqueSuffix,
		ExternalURL:  "https://example.com/" + uniqueSuffix,
		SourceKind:   domainsource.SourceKindArticle,
		Title:        "Integration Source " + uniqueSuffix,
		Summary:      "Repository integration source record",
		AuthorName:   "Integration Bot",
		LanguageCode: "de",
		RawPayload:   []byte(`{"source":"integration-test"}`),
	})
	if err != nil {
		t.Fatalf("upsert source record: %v", err)
	}
	if !createdRecord {
		t.Fatalf("expected first source record upsert to create")
	}

	resource, createdResource, err := sourceRepo.UpsertCatalogResource(ctx, sourceimportapp.UpsertCatalogResourceInput{
		SourceRecordID:  sourceRecord.ID,
		SourceType:      domaincatalog.ResourceTypeArticle,
		SourceName:      "Integration Bot",
		Title:           "Integration Catalog " + uniqueSuffix,
		Summary:         "Repository integration catalog resource",
		ExternalURL:     "https://example.com/resource/" + uniqueSuffix,
		Level:           "A2",
		Format:          "article",
		DurationMinutes: 20,
		IsFree:          true,
		SkillTags:       []string{"grammar"},
		TopicTags:       []string{"cases"},
		LanguageCode:    "de",
		IngestionOrigin: domainsource.IngestionOriginImported,
	})
	if err != nil {
		t.Fatalf("upsert catalog resource: %v", err)
	}
	if !createdResource {
		t.Fatalf("expected first catalog upsert to create")
	}
	if resource.ProviderSlug != "manual" {
		t.Fatalf("expected provider slug manual, got %s", resource.ProviderSlug)
	}
	if resource.IngestionOrigin != "imported" {
		t.Fatalf("expected ingestion origin imported, got %s", resource.IngestionOrigin)
	}

	catalogByID, err := catalogRepo.GetResourceByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("get catalog by id: %v", err)
	}
	if catalogByID.ID != resource.ID {
		t.Fatalf("expected resource id %s, got %s", resource.ID, catalogByID.ID)
	}

	sourceRecordID := sourceRecord.ID
	catalogResourceID := resource.ID
	if err := sourceRepo.CreateImportResult(ctx, sourceimportapp.CreateImportResultInput{
		ImportJobID:       importJob.ID,
		SourceRecordID:    &sourceRecordID,
		CatalogResourceID: &catalogResourceID,
		Status:            domainsource.ImportResultImported,
		Message:           "imported in integration test",
	}); err != nil {
		t.Fatalf("create import result: %v", err)
	}

	if err := sourceRepo.CompleteImportJob(ctx, importJob.ID, domainsource.ImportJobCompleted, ""); err != nil {
		t.Fatalf("complete import job: %v", err)
	}

	profile, err := profileRepo.GetByUserID(ctx, integrationDemoUserID)
	if err != nil {
		t.Fatalf("get profile by user id: %v", err)
	}

	targetLevel := "B1"
	profile.DisplayName = "Integration Learner"
	profile.TargetLevel = &targetLevel
	profile.LearningGoals = "Finish more curated resources"
	profile.PreferredResourceTypes = []string{"course", "article"}
	profile.PreferredSkills = []string{"grammar", "listening"}
	profile.PreferredSourceProviders = []string{"manual", "youtube"}

	updatedProfile, err := profileRepo.Upsert(ctx, profile)
	if err != nil {
		t.Fatalf("upsert profile: %v", err)
	}
	if updatedProfile.DisplayName != "Integration Learner" {
		t.Fatalf("expected updated display name, got %s", updatedProfile.DisplayName)
	}
	if updatedProfile.TargetLevel == nil || *updatedProfile.TargetLevel != "B1" {
		t.Fatalf("expected updated target level B1, got %+v", updatedProfile.TargetLevel)
	}

	if _, err := profileRepo.Upsert(ctx, domainprofile.UserProfile{
		UserID:      "00000000-0000-0000-0000-000000000000",
		DisplayName: "Unknown",
	}); err == nil {
		t.Fatalf("expected upsert to fail for missing app user")
	}
}

func TestPersistedHTTPFlowIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()
	catalogRepo := NewCatalogRepository(db)
	profileRepo := NewProfileRepository(db)
	progressRepo := NewProgressRepository(db)
	savedRepo := NewSavedRepository(db)
	sourceRepo := NewSourceRepository(db)

	cleanupSavedState(t, ctx, db)
	cleanupProgressState(t, ctx, db)
	_, _ = savedRepo.Save(ctx, integrationDemoUserID, integrationResourceID)
	_, _ = progressRepo.UpsertStatus(ctx, integrationDemoUserID, integrationResourceID, userstate.ProgressInProgress)
	defer cleanupSavedState(t, ctx, db)
	defer cleanupProgressState(t, ctx, db)

	server := httpapi.NewServer(
		catalogapp.NewService(catalogRepo),
		profileapp.NewService(profileRepo),
		progressapp.NewService(progressRepo),
		savedapp.NewService(savedRepo),
		sourceapp.NewService(sourceRepo),
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
			ID              string `json:"id"`
			IsSaved         bool   `json:"isSaved"`
			ProviderSlug    string `json:"providerSlug"`
			ProviderName    string `json:"providerName"`
			IngestionOrigin string `json:"ingestionOrigin"`
			SourceKind      string `json:"sourceKind"`
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
		if item.ProviderSlug == "" || item.ProviderName == "" {
			t.Fatalf("expected provider metadata on list response")
		}
		if item.IngestionOrigin == "" || item.SourceKind == "" {
			t.Fatalf("expected ingestion metadata on list response")
		}
	}
	if !foundSaved {
		t.Fatalf("expected saved resource to be marked as saved in /api/v1/resources")
	}

	providersReq := httptest.NewRequest(http.MethodGet, "/api/v1/source-providers", nil)
	providersResp := httptest.NewRecorder()
	server.ServeHTTP(providersResp, providersReq)
	if providersResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/source-providers, got %d", providersResp.Code)
	}

	var providersPayload struct {
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	if err := json.NewDecoder(providersResp.Body).Decode(&providersPayload); err != nil {
		t.Fatalf("decode source providers payload: %v", err)
	}
	if len(providersPayload.Items) == 0 {
		t.Fatalf("expected source providers payload to be non-empty")
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

	progressMissingHeaderReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/progress", nil)
	progressMissingHeaderResp := httptest.NewRecorder()
	server.ServeHTTP(progressMissingHeaderResp, progressMissingHeaderReq)
	if progressMissingHeaderResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for /api/v1/me/progress without header, got %d", progressMissingHeaderResp.Code)
	}

	progressReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/progress", nil)
	progressReq.Header.Set("X-User-ID", integrationDemoUserID)
	progressResp := httptest.NewRecorder()
	server.ServeHTTP(progressResp, progressReq)
	if progressResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/me/progress, got %d", progressResp.Code)
	}

	var progressPayload struct {
		Items []struct {
			ResourceID string `json:"resourceId"`
			Status     string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(progressResp.Body).Decode(&progressPayload); err != nil {
		t.Fatalf("decode /api/v1/me/progress payload: %v", err)
	}
	if len(progressPayload.Items) == 0 {
		t.Fatalf("expected non-empty progress payload")
	}
	if progressPayload.Items[0].Status != string(userstate.ProgressInProgress) {
		t.Fatalf("expected progress status %q, got %q", userstate.ProgressInProgress, progressPayload.Items[0].Status)
	}

	updateProgressReq := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/me/progress/"+integrationResourceID,
		strings.NewReader(`{"status":"completed"}`),
	)
	updateProgressReq.Header.Set("X-User-ID", integrationDemoUserID)
	updateProgressReq.Header.Set("Content-Type", "application/json")
	updateProgressResp := httptest.NewRecorder()
	server.ServeHTTP(updateProgressResp, updateProgressReq)
	if updateProgressResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for progress update, got %d", updateProgressResp.Code)
	}

	var updatedProgress struct {
		Status          string  `json:"status"`
		ProgressPercent float64 `json:"progressPercent"`
	}
	if err := json.NewDecoder(updateProgressResp.Body).Decode(&updatedProgress); err != nil {
		t.Fatalf("decode progress update payload: %v", err)
	}
	if updatedProgress.Status != string(userstate.ProgressCompleted) {
		t.Fatalf("expected updated status %q, got %q", userstate.ProgressCompleted, updatedProgress.Status)
	}
	if updatedProgress.ProgressPercent != 100 {
		t.Fatalf("expected completed percent to be 100, got %v", updatedProgress.ProgressPercent)
	}

	profileMissingHeaderReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile", nil)
	profileMissingHeaderResp := httptest.NewRecorder()
	server.ServeHTTP(profileMissingHeaderResp, profileMissingHeaderReq)
	if profileMissingHeaderResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for /api/v1/me/profile without header, got %d", profileMissingHeaderResp.Code)
	}

	profileGetReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile", nil)
	profileGetReq.Header.Set("X-User-ID", integrationDemoUserID)
	profileGetResp := httptest.NewRecorder()
	server.ServeHTTP(profileGetResp, profileGetReq)
	if profileGetResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for profile get, got %d", profileGetResp.Code)
	}

	profileUpdateReq := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/me/profile",
		strings.NewReader(`{
			"displayName":"Demo Learner Updated",
			"targetLevel":"B2",
			"learningGoals":"Improve grammar consistency",
			"preferredResourceTypes":["youtube","article"],
			"preferredSkills":["grammar","listening"],
			"preferredSourceProviders":["manual","youtube"]
		}`),
	)
	profileUpdateReq.Header.Set("X-User-ID", integrationDemoUserID)
	profileUpdateReq.Header.Set("Content-Type", "application/json")
	profileUpdateResp := httptest.NewRecorder()
	server.ServeHTTP(profileUpdateResp, profileUpdateReq)
	if profileUpdateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for profile update, got %d", profileUpdateResp.Code)
	}

	var profilePayload struct {
		DisplayName string `json:"displayName"`
		TargetLevel string `json:"targetLevel"`
	}
	if err := json.NewDecoder(profileUpdateResp.Body).Decode(&profilePayload); err != nil {
		t.Fatalf("decode profile update payload: %v", err)
	}
	if profilePayload.DisplayName != "Demo Learner Updated" {
		t.Fatalf("expected updated display name, got %q", profilePayload.DisplayName)
	}
	if profilePayload.TargetLevel != "B2" {
		t.Fatalf("expected updated target level B2, got %q", profilePayload.TargetLevel)
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

func cleanupProgressState(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(
		ctx,
		`
DELETE FROM user_resource_progress
WHERE user_id = $1::uuid
  AND resource_id = $2::uuid;
`,
		integrationDemoUserID,
		integrationResourceID,
	); err != nil {
		t.Fatalf("cleanup progress state: %v", err)
	}
}
