package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"manyrows-core/api"
	"manyrows-core/auth"
	"manyrows-core/auth/client"
	"manyrows-core/core"
	"manyrows-core/email"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
)

// setupProductsTestRouter creates a minimal router for testing project endpoints
func setupProductsTestRouter(t *testing.T) *chi.Mux {
	t.Helper()

	conf := GetTestConfig()

	adminAuthService, err := auth.NewAuthService(conf, testEnv.Repo)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	clientAuthService, err := client.NewAuthService(conf, testEnv.Repo, nil)
	if err != nil {
		t.Fatalf("failed to create client auth service: %v", err)
	}

	emailService := email.NewEmailService(true, nil)

	requestHandler := api.NewRequestHandler(
		testEnv.Repo,
		adminAuthService,
		clientAuthService,
		emailService,
		conf,
		nil,
		nil,
	)

	r := chi.NewRouter()

	// Auth middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acc, _, err := adminAuthService.GetLoggedInAccount(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if acc == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			ctx := core.WithAdminAccount(r.Context(), acc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Workspace routes
	r.Route("/admin/workspace/{workspaceId}", func(r chi.Router) {
		// Workspace middleware
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				acc, ok := core.AdminAccountFromContext(ctx)
				if !ok || acc == nil {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}

				wsIDStr := chi.URLParam(r, "workspaceId")
				if wsIDStr == "" {
					http.Error(w, "missing workspace id", http.StatusBadRequest)
					return
				}

				wsID, err := uuid.FromString(wsIDStr)
				if err != nil {
					http.Error(w, "invalid workspace id", http.StatusBadRequest)
					return
				}

				// Check membership with admin/owner role
				isMember, memberErr := testEnv.Repo.IsWorkspaceOwner(ctx, wsID, acc.ID)
				if memberErr != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if !isMember {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}

				ws, found, wsErr := testEnv.Repo.GetWorkspaceByID(ctx, wsID)
				if wsErr != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if !found {
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
					return
				}

				ctx = core.WithWorkspace(ctx, ws)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		r.Get("/products", requestHandler.GetProducts)
		r.Post("/products", requestHandler.CreateProduct)
		r.Get("/products/{productId}", requestHandler.GetProduct)
		r.Put("/products/{productId}", requestHandler.UpdateProduct)
		r.Delete("/products/{productId}", requestHandler.DeleteProduct)
	})

	return r
}

func TestGetProducts_Unauthenticated(t *testing.T) {
	// Create fixtures
	email := "test-unauth-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	// Make request without session cookie
	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGetProducts_NoWorkspaceAccess(t *testing.T) {
	// Create two accounts - one owns workspace, other doesn't have access
	ownerEmail := "test-owner-" + GenerateUniqueSlug("user") + "@test.com"
	otherEmail := "test-other-" + GenerateUniqueSlug("user") + "@test.com"

	owner := testEnv.CreateTestAccount(t, ownerEmail)
	other := testEnv.CreateTestAccount(t, otherEmail)

	ws := testEnv.CreateTestWorkspace(t, owner, "Test Workspace", GenerateUniqueSlug("ws"))

	// Create session for the "other" user who doesn't have access
	sess, claims := testEnv.CreateTestSession(t, other)

	fixtures := &TestFixtures{
		Account:   owner,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)
	defer func() {
		// Also cleanup the other account
		testEnv.DB.Pool().Exec(context.Background(), "DELETE FROM accounts WHERE id = $1", other.ID)
	}()

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products", nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGetProducts_EmptyWorkspace(t *testing.T) {
	email := "test-empty-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Empty Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products", nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var products []core.Product
	if err := json.NewDecoder(rec.Body).Decode(&products); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

func TestGetProducts_WithProducts(t *testing.T) {
	email := "test-products-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	// Create multiple products
	p1 := testEnv.CreateTestProduct(t, ws, acc, "Product One", GenerateUniqueSlug("proj1"))
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	p2 := testEnv.CreateTestProduct(t, ws, acc, "Product Two", GenerateUniqueSlug("proj2"))
	time.Sleep(10 * time.Millisecond)
	p3 := testEnv.CreateTestProduct(t, ws, acc, "Product Three", GenerateUniqueSlug("proj3"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		Products:  []core.Product{*p1, *p2, *p3},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products", nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var products []core.Product
	if err := json.NewDecoder(rec.Body).Decode(&products); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(products) != 3 {
		t.Errorf("expected 3 products, got %d", len(products))
	}

	// Products should be ordered by created_at desc (newest first)
	if len(products) >= 3 {
		if products[0].ID != p3.ID {
			t.Errorf("expected first project to be %s, got %s", p3.ID, products[0].ID)
		}
		if products[1].ID != p2.ID {
			t.Errorf("expected second project to be %s, got %s", p2.ID, products[1].ID)
		}
		if products[2].ID != p1.ID {
			t.Errorf("expected third project to be %s, got %s", p1.ID, products[2].ID)
		}
	}
}

func TestGetProducts_ProductFields(t *testing.T) {
	email := "test-fields-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	projectName := "Test Product Fields"
	projectSlug := GenerateUniqueSlug("proj")
	p := testEnv.CreateTestProduct(t, ws, acc, projectName, projectSlug)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		Products:  []core.Product{*p},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products", nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var products []core.Product
	if err := json.NewDecoder(rec.Body).Decode(&products); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 project, got %d", len(products))
	}

	project := products[0]

	// Verify all expected fields
	if project.ID != p.ID {
		t.Errorf("expected ID %s, got %s", p.ID, project.ID)
	}
	if project.WorkspaceID != ws.ID {
		t.Errorf("expected WorkspaceID %s, got %s", ws.ID, project.WorkspaceID)
	}
	if project.Name != projectName {
		t.Errorf("expected Name %s, got %s", projectName, project.Name)
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if project.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestGetProduct_ById(t *testing.T) {
	email := "test-getbyid-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	p := testEnv.CreateTestProduct(t, ws, acc, "Single Product", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		Products:  []core.Product{*p},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+p.ID.String(), nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var project core.Product
	if err := json.NewDecoder(rec.Body).Decode(&project); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if project.ID != p.ID {
		t.Errorf("expected ID %s, got %s", p.ID, project.ID)
	}
	if project.Name != "Single Product" {
		t.Errorf("expected Name 'Single Product', got %s", project.Name)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	email := "test-notfound-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	// Use a random UUID that doesn't exist (not all-zeros, which is treated as "missing")
	fakeID := "12345678-1234-1234-1234-123456789012"
	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+fakeID, nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestGetProducts_InvalidWorkspaceId(t *testing.T) {
	email := "test-invalid-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/invalid-uuid/products", nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

// ============================================================================
// POST /admin/workspace/{workspaceId}/products - Create Product
// ============================================================================

func TestCreateProduct_Success(t *testing.T) {
	email := "test-create-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	projectName := "New Product"
	body := map[string]string{
		"name": projectName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/workspace/"+ws.ID.String()+"/products", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var project core.Product
	if err := json.NewDecoder(rec.Body).Decode(&project); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Cleanup the created project
	defer testEnv.DB.Pool().Exec(context.Background(), "DELETE FROM products WHERE id = $1", project.ID)

	if project.Name != projectName {
		t.Errorf("expected Name %s, got %s", projectName, project.Name)
	}
	if project.WorkspaceID != ws.ID {
		t.Errorf("expected WorkspaceID %s, got %s", ws.ID, project.WorkspaceID)
	}
	if project.ID == uuid.Nil {
		t.Error("expected project ID to be set")
	}
}

func TestCreateProduct_Unauthenticated(t *testing.T) {
	email := "test-create-unauth-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	body := map[string]string{
		"name": "Test Product",
		"slug": "test-proj",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/workspace/"+ws.ID.String()+"/products", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateProduct_MissingName(t *testing.T) {
	email := "test-create-noname-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	body := map[string]string{
		"slug": "test-proj",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/workspace/"+ws.ID.String()+"/products", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateProduct_InvalidJSON(t *testing.T) {
	email := "test-create-badjson-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/workspace/"+ws.ID.String()+"/products", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

// ============================================================================
// PUT /admin/workspace/{workspaceId}/products/{productId} - Update Product
// ============================================================================

func TestUpdateProduct_Success(t *testing.T) {
	email := "test-update-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	p := testEnv.CreateTestProduct(t, ws, acc, "Original Name", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		Products:  []core.Product{*p},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	newName := "Updated Name"
	body := map[string]string{
		"name": newName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/workspace/"+ws.ID.String()+"/products/"+p.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}

	// Verify the update
	updated, err := testEnv.Repo.GetProduct(context.Background(), p.ID, ws.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected Name %s, got %s", newName, updated.Name)
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	email := "test-update-notfound-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	body := map[string]string{
		"name": "Updated Name",
	}
	bodyBytes, _ := json.Marshal(body)

	fakeID := "12345678-1234-1234-1234-123456789012"
	req := httptest.NewRequest(http.MethodPut, "/admin/workspace/"+ws.ID.String()+"/products/"+fakeID, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestUpdateProduct_EmptyName(t *testing.T) {
	email := "test-update-emptyname-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	p := testEnv.CreateTestProduct(t, ws, acc, "Original Name", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		Products:  []core.Product{*p},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	body := map[string]string{
		"name": "",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/workspace/"+ws.ID.String()+"/products/"+p.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

// ============================================================================
// DELETE /admin/workspace/{workspaceId}/products/{productId} - Delete Product
// ============================================================================

func TestDeleteProduct_Success(t *testing.T) {
	email := "test-delete-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	p := testEnv.CreateTestProduct(t, ws, acc, "To Be Deleted", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
		// Don't add project to fixtures since we're deleting it
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/admin/workspace/"+ws.ID.String()+"/products/"+p.ID.String(), nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}

	// Verify the project is deleted
	deleted, err := testEnv.Repo.GetProduct(context.Background(), p.ID, ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("expected project to be deleted, but it still exists")
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	email := "test-delete-notfound-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Session:   sess,
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	fakeID := "12345678-1234-1234-1234-123456789012"
	req := httptest.NewRequest(http.MethodDelete, "/admin/workspace/"+ws.ID.String()+"/products/"+fakeID, nil)
	testEnv.SetSessionCookie(t, req, claims)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestDeleteProduct_Unauthenticated(t *testing.T) {
	email := "test-delete-unauth-" + GenerateUniqueSlug("user") + "@test.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test Workspace", GenerateUniqueSlug("ws"))

	p := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{
		Account:   acc,
		Workspace: ws,
		Products:  []core.Product{*p},
	}
	defer testEnv.CleanupTestData(t, fixtures)

	router := setupProductsTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/admin/workspace/"+ws.ID.String()+"/products/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
