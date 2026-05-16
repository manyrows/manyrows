package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"manyrows-core/api"
	"manyrows-core/auth"
	"manyrows-core/auth/client"
	"manyrows-core/core"
	"manyrows-core/core/repo"
	"manyrows-core/email"
	"manyrows-core/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
)

// setupMemberRolesRouter creates a router for member roles tests
func setupMemberRolesRouter(t *testing.T) *chi.Mux {
	t.Helper()

	cfg := GetTestConfig()
	adminAuthService, err := auth.NewAuthService(cfg, testEnv.Repo)
	if err != nil {
		t.Fatalf("failed to create auth service: %v", err)
	}

	clientAuthService, err := client.NewAuthService(cfg, testEnv.Repo, nil)
	if err != nil {
		t.Fatalf("failed to create client auth service: %v", err)
	}

	emailService := email.NewEmailService(true, nil)

	requestHandler := api.NewRequestHandler(
		testEnv.Repo,
		adminAuthService,
		clientAuthService,
		emailService,
		cfg,
		nil,
		nil,
	)

	r := chi.NewRouter()

	adminRouter := chi.NewRouter()
	adminRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acc, _, err := adminAuthService.GetLoggedInAccount(r)
			if err != nil || acc == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			ctx := core.WithAdminAccount(r.Context(), acc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	adminWorkspaceRouter := chi.NewRouter()
	adminWorkspaceRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			acc, ok := core.AdminAccountFromContext(ctx)
			if !ok || acc == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			wsIDStr := chi.URLParam(r, "workspaceId")
			wsID, err := uuid.FromString(wsIDStr)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			ok, err = testEnv.Repo.IsWorkspaceOwner(ctx, wsID, acc.ID)
			if err != nil || !ok {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			ws, ok, err := testEnv.Repo.GetWorkspaceByID(ctx, wsID)
			if err != nil || !ok {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			ctx = core.WithWorkspace(ctx, ws)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	adminWorkspaceRouter.Get("/products/{productId}/memberRoles", requestHandler.HandleGetMemberRoles)
	adminWorkspaceRouter.Put("/products/{productId}/memberRoles/{userId}", requestHandler.HandlerUpdateMemberRoles)
	adminWorkspaceRouter.Get("/products/{productId}/members", requestHandler.HandleGetProductMembers)
	adminWorkspaceRouter.Delete("/products/{productId}/members/{userId}", requestHandler.HandleRemoveProductMember)

	adminRouter.Mount("/workspace/{workspaceId}", adminWorkspaceRouter)
	r.Mount("/admin", adminRouter)

	return r
}

// createTestRole creates a role for testing
func createTestRole(t *testing.T, productID uuid.UUID) *core.Role {
	t.Helper()
	ctx := context.Background()

	slug := GenerateUniqueSlug("role")
	params := repo.CreateRoleParams{
		ProductID: productID,
		Name:      "test-role-" + slug,
		Slug:      slug,
		Now:       time.Now().UTC(),
	}

	role, err := testEnv.Repo.CreateRole(ctx, params)
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	return &role
}

// TestGetMemberRoles_Success tests getting member roles for a project
func TestGetMemberRoles_Success(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "member-roles-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}, Session: sess}
	defer testEnv.CleanupTestData(t, fixtures)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/memberRoles", nil)
	testEnv.SetSessionCookie(t, req, claims)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		return
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// TestGetMemberRoles_Unauthenticated tests without auth
func TestGetMemberRoles_Unauthenticated(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "member-roles-unauth-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}}
	defer testEnv.CleanupTestData(t, fixtures)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/memberRoles", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestGetMemberRoles_NotFound tests getting member roles for non-existent project
func TestGetMemberRoles_NotFound(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "member-roles-nf-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Session: sess}
	defer testEnv.CleanupTestData(t, fixtures)

	fakeProductID := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+fakeProductID.String()+"/memberRoles", nil)
	testEnv.SetSessionCookie(t, req, claims)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound && rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d or %d, got %d: %s", http.StatusNotFound, http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

// TestUpdateMemberRoles_Success tests updating member roles
func TestUpdateMemberRoles_Success(t *testing.T) {
	router := setupMemberRolesRouter(t)

	ownerEmail := "member-roles-owner-" + GenerateUniqueSlug("test") + "@example.com"
	owner := testEnv.CreateTestAccount(t, ownerEmail)
	ws := testEnv.CreateTestWorkspace(t, owner, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, owner, "Test Product", GenerateUniqueSlug("proj"))
	sess, claims := testEnv.CreateTestSession(t, owner)

	// Create an app so we can create a user with the new model
	appID := utils.NewUUID()
	ctx := context.Background()
	userPool, err := testEnv.Repo.CreateUserPool(ctx, ws.ID, "Pool "+GenerateUniqueSlug("p"))
	if err != nil {
		t.Fatalf("failed to create user pool: %v", err)
	}
	pool := testEnv.DB.Pool()
	_, err = pool.Exec(ctx, `
		INSERT INTO apps (id, workspace_id, product_id, user_pool_id, type, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'dev', true, NOW(), NOW())
	`, appID, ws.ID, project.ID, userPool.ID)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Create a user (client app user) to update roles for
	memberEmail := "member-target-" + GenerateUniqueSlug("test") + "@example.com"
	user, _, err := testEnv.GetOrCreateUserWithMembership(ctx, memberEmail, &core.App{ID: appID}, core.UserSourceInvited)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	role := createTestRole(t, project.ID)

	fixtures := &TestFixtures{Account: owner, Workspace: ws, Products: []core.Product{*project}, Session: sess}
	defer testEnv.CleanupTestData(t, fixtures)
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM user_roles WHERE user_id = $1", user.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM roles WHERE id = $1", role.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM apps WHERE id = $1", appID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	}()

	body := map[string]any{
		// Handler requires appId in the body now ("env-scoped, no 'all envs'").
		"appId":   appID.String(),
		"roleIds": []string{role.ID.String()},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/memberRoles/"+user.ID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	testEnv.SetSessionCookie(t, req, claims)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d or %d, got %d: %s", http.StatusOK, http.StatusNoContent, rr.Code, rr.Body.String())
	}
}

// TestGetProductMembers_Success tests getting project members
func TestGetProductMembers_Success(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "proj-members-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))
	appID := createTestApp(t, ws.ID, project.ID, uuid.Nil, "Members Test App")
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}, Session: sess}
	defer testEnv.CleanupTestData(t, fixtures)
	defer func() {
		_, _ = testEnv.DB.Pool().Exec(context.Background(), "DELETE FROM apps WHERE id = $1", appID)
	}()

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/members?appId="+appID.String(), nil)
	testEnv.SetSessionCookie(t, req, claims)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		return
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["members"] == nil {
		t.Error("expected members in response")
	}
}

// TestGetProductMembers_Unauthenticated tests without auth
func TestGetProductMembers_Unauthenticated(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "proj-members-unauth-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}}
	defer testEnv.CleanupTestData(t, fixtures)

	req := httptest.NewRequest(http.MethodGet, "/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/members", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestGetProductMembers_NoAppId_AppScope tests that app-scoped products require appId
func TestGetProductMembers_NoAppId_AppScope(t *testing.T) {
	router := setupMemberRolesRouter(t)

	email := "proj-members-noapp-app-" + GenerateUniqueSlug("test") + "@example.com"
	acc := testEnv.CreateTestAccount(t, email)
	ws := testEnv.CreateTestWorkspace(t, acc, "Test WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "Test Product", GenerateUniqueSlug("proj"))
	sess, claims := testEnv.CreateTestSession(t, acc)

	fixtures := &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}, Session: sess}
	defer testEnv.CleanupTestData(t, fixtures)

	// No appId — app scope should reject
	req := httptest.NewRequest(http.MethodGet,
		"/admin/workspace/"+ws.ID.String()+"/products/"+project.ID.String()+"/members",
		nil)
	testEnv.SetSessionCookie(t, req, claims)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestGetProductMembers_NoAppId_ProductScope tests that project-scoped products
// can list members without appId (for autocomplete across all apps).

// removeMemberFixture builds ws+product+pool+app and a user that is a
// member of the app, with one role assigned and one active client
// session. Returns the ids + cleanup.
func removeMemberFixture(t *testing.T, emailPrefix string) (claims core.TokenClaims, wsID, productID, appID, userID, roleID uuid.UUID, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	acc := testEnv.CreateTestAccount(t, emailPrefix+GenerateUniqueSlug("u")+"@example.com")
	ws := testEnv.CreateTestWorkspace(t, acc, "WS", GenerateUniqueSlug("ws"))
	project := testEnv.CreateTestProduct(t, ws, acc, "P", GenerateUniqueSlug("p"))
	_, claims = testEnv.CreateTestSession(t, acc)

	userPool, err := testEnv.Repo.CreateUserPool(ctx, ws.ID, "Pool "+GenerateUniqueSlug("p"))
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	appID = utils.NewUUID()
	db := testEnv.DB.Pool()
	if _, err := db.Exec(ctx, `
		INSERT INTO apps (id, workspace_id, product_id, user_pool_id, type, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'dev',true,NOW(),NOW())`, appID, ws.ID, project.ID, userPool.ID); err != nil {
		t.Fatalf("create app: %v", err)
	}
	user, _, err := testEnv.GetOrCreateUserWithMembership(ctx, acc.Email+".m", &core.App{ID: appID}, core.UserSourceInvited)
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	role := createTestRole(t, project.ID)
	if err := testEnv.Repo.ReplaceUserRoles(ctx, repo.ReplaceUserRolesParams{
		ProductID: project.ID, AppID: appID, UserID: user.ID,
		RoleIDs: []uuid.UUID{role.ID}, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("assign role: %v", err)
	}
	now := time.Now().UTC()
	if err := testEnv.Repo.InsertClientSession(ctx, &core.ClientSession{
		ID: utils.NewUUID(), UserID: user.ID, AppID: &appID,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(24 * time.Hour),
		UserAgent: "test", IP: "127.0.0.1",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cleanup = func() {
		c := context.Background()
		_, _ = db.Exec(c, "DELETE FROM client_sessions WHERE user_id = $1", user.ID)
		_, _ = db.Exec(c, "DELETE FROM apps WHERE id = $1", appID)
		_, _ = db.Exec(c, "DELETE FROM users WHERE user_pool_id = $1", userPool.ID)
		_, _ = db.Exec(c, "DELETE FROM user_pools WHERE id = $1", userPool.ID)
		testEnv.CleanupTestData(t, &TestFixtures{Account: acc, Workspace: ws, Products: []core.Product{*project}})
	}
	return claims, ws.ID, project.ID, appID, user.ID, role.ID, cleanup
}

func countRows(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := testEnv.DB.Pool().QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count (%s): %v", q, err)
	}
	return n
}

func TestRemoveProductMember_Success(t *testing.T) {
	router := setupMemberRolesRouter(t)
	claims, wsID, productID, appID, userID, _, cleanup := removeMemberFixture(t, "rpm-ok-")
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete,
		"/admin/workspace/"+wsID.String()+"/products/"+productID.String()+"/members/"+userID.String()+"?appId="+appID.String(), nil)
	testEnv.SetSessionCookie(t, req, claims)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if n := countRows(t, "SELECT count(*) FROM app_users WHERE app_id=$1 AND user_id=$2", appID, userID); n != 0 {
		t.Errorf("membership should be gone, count=%d", n)
	}
	if n := countRows(t, "SELECT count(*) FROM user_roles WHERE app_id=$1 AND user_id=$2", appID, userID); n != 0 {
		t.Errorf("roles should be cleared, count=%d", n)
	}
	if n := countRows(t, "SELECT count(*) FROM client_sessions WHERE user_id=$1 AND app_id=$2", userID, appID); n != 0 {
		t.Errorf("sessions should be revoked, count=%d", n)
	}
	if n := countRows(t, "SELECT count(*) FROM users WHERE id=$1", userID); n != 1 {
		t.Errorf("pool account must be preserved, count=%d", n)
	}
}

func TestRemoveProductMember_Idempotent(t *testing.T) {
	router := setupMemberRolesRouter(t)
	claims, wsID, productID, appID, userID, _, cleanup := removeMemberFixture(t, "rpm-idem-")
	defer cleanup()

	path := "/admin/workspace/" + wsID.String() + "/products/" + productID.String() + "/members/" + userID.String() + "?appId=" + appID.String()
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		testEnv.SetSessionCookie(t, req, claims)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200 (idempotent), got %d: %s", i, rr.Code, rr.Body.String())
		}
	}
}

func TestRemoveProductMember_MissingAppId(t *testing.T) {
	router := setupMemberRolesRouter(t)
	claims, wsID, productID, _, userID, _, cleanup := removeMemberFixture(t, "rpm-noapp-")
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete,
		"/admin/workspace/"+wsID.String()+"/products/"+productID.String()+"/members/"+userID.String(), nil)
	testEnv.SetSessionCookie(t, req, claims)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without appId, got %d: %s", rr.Code, rr.Body.String())
	}
}
