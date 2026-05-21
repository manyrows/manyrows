package api_test

import (
	"context"
	"errors"
	"testing"

	"manyrows-core/core"
	"manyrows-core/core/repo"

	"github.com/gofrs/uuid/v5"
)

// TestExternalIDPRepo_CRUD exercises the full lifecycle and, by running
// against the real test DB, validates that migration 00005 applies and
// the NULL/COALESCE round-trip + repo defaults behave.
func TestExternalIDPRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	acc := testEnv.CreateTestAccount(t, "extidp-"+GenerateUniqueSlug("u")+"@example.com")
	ws := testEnv.CreateTestWorkspace(t, acc, "ExtIDP WS", GenerateUniqueSlug("ws"))
	app := testEnv.CreateTestApp(t, ws, acc)

	e := &core.ExternalIDP{
		AppID:                 app.ID,
		Slug:                  "acme-okta",
		DisplayName:           "Acme Okta",
		Enabled:               true,
		Mode:                  core.ExternalIDPModeOIDC,
		IssuerURL:             "https://acme.okta.com",
		ClientID:              "client-123",
		ClientSecretEncrypted: []byte("ciphertext-bytes"),
		// Scopes / SubjectField / EmailField intentionally left empty —
		// the repo must fill the standard-claim defaults.
	}
	if err := testEnv.Repo.CreateExternalIDP(ctx, e); err != nil {
		t.Fatalf("create: %v", err)
	}
	if e.ID == uuid.Nil {
		t.Fatal("create must assign an ID")
	}
	if e.Scopes != "openid email profile" {
		t.Fatalf("default scopes not applied: %q", e.Scopes)
	}
	if e.SubjectField != "sub" || e.EmailField != "email" {
		t.Fatalf("default claim fields not applied: sub=%q email=%q", e.SubjectField, e.EmailField)
	}

	got, err := testEnv.Repo.GetExternalIDPByAppAndSlug(ctx, app.ID, "acme-okta")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if got.IssuerURL != "https://acme.okta.com" {
		t.Fatalf("issuer round-trip: %q", got.IssuerURL)
	}
	if string(got.ClientSecretEncrypted) != "ciphertext-bytes" {
		t.Fatalf("secret round-trip mismatch")
	}
	if got.Mode != core.ExternalIDPModeOIDC {
		t.Fatalf("mode round-trip: %q", got.Mode)
	}
	// OAuth2-only columns were never set → stored NULL → COALESCEd to "".
	if got.AuthorizeURL != "" || got.TokenURL != "" || got.UserinfoURL != "" {
		t.Fatalf("oidc row should have empty oauth2 endpoints, got authorize=%q token=%q userinfo=%q",
			got.AuthorizeURL, got.TokenURL, got.UserinfoURL)
	}
	if got.ProviderKey() != "idp:acme-okta" {
		t.Fatalf("provider key: %q", got.ProviderKey())
	}

	all, err := testEnv.Repo.ListExternalIDPsByApp(ctx, app.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("list-by-app: err=%v len=%d", err, len(all))
	}

	// Disable → drops out of the enabled-only list AppKit reads.
	got.DisplayName = "Acme Okta (prod)"
	got.Enabled = false
	if err := testEnv.Repo.UpdateExternalIDP(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	enabled, err := testEnv.Repo.ListEnabledExternalIDPsByApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("list-enabled: %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("expected 0 enabled after disable, got %d", len(enabled))
	}

	ok, err := testEnv.Repo.DeleteExternalIDP(ctx, app.ID, got.ID)
	if err != nil || !ok {
		t.Fatalf("delete: ok=%v err=%v", ok, err)
	}
	if _, err := testEnv.Repo.GetExternalIDPByAppAndSlug(ctx, app.ID, "acme-okta"); !errors.Is(err, repo.ErrExternalIDPNotFound) {
		t.Fatalf("expected ErrExternalIDPNotFound after delete, got %v", err)
	}
}

// TestExternalIDPRepo_ModeConstraints validates the per-mode endpoint
// CHECK and the slug-format CHECK from migration 00005.
func TestExternalIDPRepo_ModeConstraints(t *testing.T) {
	ctx := context.Background()
	acc := testEnv.CreateTestAccount(t, "extidp-c-"+GenerateUniqueSlug("u")+"@example.com")
	ws := testEnv.CreateTestWorkspace(t, acc, "ExtIDP Constraints WS", GenerateUniqueSlug("ws"))
	app := testEnv.CreateTestApp(t, ws, acc)

	// OIDC mode without an issuer_url must violate the per-mode CHECK.
	if err := testEnv.Repo.CreateExternalIDP(ctx, &core.ExternalIDP{
		AppID: app.ID, Slug: "no-issuer", DisplayName: "x", Mode: core.ExternalIDPModeOIDC,
		ClientID: "c", ClientSecretEncrypted: []byte("x"),
	}); err == nil {
		t.Fatal("expected CHECK violation: oidc mode requires issuer_url")
	}

	// OAuth2 mode with the three explicit endpoints must insert cleanly.
	if err := testEnv.Repo.CreateExternalIDP(ctx, &core.ExternalIDP{
		AppID: app.ID, Slug: "discord", DisplayName: "Discord", Mode: core.ExternalIDPModeOAuth2,
		AuthorizeURL: "https://discord.com/oauth2/authorize",
		TokenURL:     "https://discord.com/api/oauth2/token",
		UserinfoURL:  "https://discord.com/api/users/@me",
		ClientID:     "c", ClientSecretEncrypted: []byte("x"), Scopes: "identify email",
	}); err != nil {
		t.Fatalf("oauth2 with all endpoints should insert: %v", err)
	}

	// A non-DNS-label slug must violate the slug-format CHECK.
	if err := testEnv.Repo.CreateExternalIDP(ctx, &core.ExternalIDP{
		AppID: app.ID, Slug: "Bad Slug!", DisplayName: "x", Mode: core.ExternalIDPModeOIDC,
		IssuerURL: "https://x.example", ClientID: "c", ClientSecretEncrypted: []byte("x"),
	}); err == nil {
		t.Fatal("expected CHECK violation: bad slug format")
	}
}
