package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"manyrows-core/core"
	"manyrows-core/core/repo"
	"manyrows-core/utils"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

// Write side of the server-to-server API. Everything here is app-scoped
// (the app is resolved by middleware) and gated so a key for one app can
// only touch users who are MEMBERS of that app — see requireAppMember.

// requireAppMember writes a 404 (and returns false) unless userID has an
// app_users row for appID. The server API scopes to app membership: the user
// pool only shares credentials/identity across apps (SSO), it is NOT an
// access boundary, so a key for one app must not see or act on users who only
// belong to a sibling app in the same pool. A missing/cross-pool/never-joined
// user all collapse to the same 404, which also avoids leaking existence.
func (handler *RequestHandler) requireAppMember(w http.ResponseWriter, r *http.Request, appID, userID uuid.UUID) bool {
	member, err := handler.repo.GetAppUser(r.Context(), appID, userID)
	if err != nil {
		log.Err(err).Msg("requireAppMember: GetAppUser failed")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return false
	}
	if member == nil {
		WriteError(w, r, "error.notFound", http.StatusNotFound)
		return false
	}
	return true
}

// serverActorID returns the account to record as the actor for a write
// made via an API key. The key has no session/account of its own, so we
// attribute the change to whoever provisioned the key (a real account),
// which renders sensibly anywhere updated_by/created_by is shown.
func serverActorID(ctx context.Context) uuid.UUID {
	if key, ok := core.APIKeyFromContext(ctx); ok && key != nil {
		return key.CreatedBy
	}
	return uuid.Nil
}

type ServerRevokeSessionsResponse struct {
	Revoked int64 `json:"revoked"`
}

// ServerRevokeUserSessions force-logs-out a user from this app by deleting
// all of their client sessions for it.
// DELETE /x/{workspaceSlug}/api/v1/apps/{appId}/users/{userId}/sessions
func (handler *RequestHandler) ServerRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app, ok := core.AppFromContext(ctx)
	if !ok || app == nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	if !handler.requireAppMember(w, r, app.ID, userID) {
		return
	}

	revoked, err := handler.repo.DeleteClientSessionsByUserAndApp(ctx, userID, app.ID)
	if err != nil {
		log.Err(err).Msg("ServerRevokeUserSessions: delete failed")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	utils.WriteJson(w, ServerRevokeSessionsResponse{Revoked: revoked})
}

// ServerUpsertUserFieldValue sets a user's metadata field value.
// PUT /x/{workspaceSlug}/api/v1/apps/{appId}/user-fields/{userFieldId}/users/{userId}
func (handler *RequestHandler) ServerUpsertUserFieldValue(w http.ResponseWriter, r *http.Request) {
	app, ok := core.AppFromContext(r.Context())
	if !ok || app == nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	fieldID, err := uuid.FromString(chi.URLParam(r, "userFieldId"))
	if err != nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}
	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	if !handler.requireAppMember(w, r, app.ID, userID) {
		return
	}

	handler.upsertUserFieldValueScoped(w, r, app.UserPoolID, fieldID, userID, serverActorID(r.Context()))
}

// ServerDeleteUserFieldValue clears a user's metadata field value.
// DELETE /x/{workspaceSlug}/api/v1/apps/{appId}/user-fields/{userFieldId}/users/{userId}
func (handler *RequestHandler) ServerDeleteUserFieldValue(w http.ResponseWriter, r *http.Request) {
	app, ok := core.AppFromContext(r.Context())
	if !ok || app == nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	fieldID, err := uuid.FromString(chi.URLParam(r, "userFieldId"))
	if err != nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}
	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	if !handler.requireAppMember(w, r, app.ID, userID) {
		return
	}

	handler.deleteUserFieldValueScoped(w, r, app.UserPoolID, fieldID, userID)
}

type ServerReplaceRolesRequest struct {
	// Roles is the full set of role slugs the user should have in this
	// app (replace semantics, not merge). An empty array clears all roles.
	Roles []string `json:"roles"`
}

type ServerRolesResponse struct {
	Roles []string `json:"roles"`
}

// ServerReplaceUserRoles replaces a user's role assignments in this app.
// Accepts role slugs (consistent with the read API, which returns slugs).
// PUT /x/{workspaceSlug}/api/v1/apps/{appId}/users/{userId}/roles
func (handler *RequestHandler) ServerReplaceUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusNotFound)
		return
	}
	app, ok := core.AppFromContext(ctx)
	if !ok || app == nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	// Only assign roles to existing members of this app (pool ≠ access
	// boundary). Provisioning roles before a user joins is intentionally
	// not supported on the server API.
	if !handler.requireAppMember(w, r, app.ID, userID) {
		return
	}

	var req ServerReplaceRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "error.invalidJson", http.StatusBadRequest)
		return
	}

	// Resolve slugs to role IDs within the product, de-duplicating. Any
	// unknown slug is a 400 — silently dropping it would let a typo quietly
	// under-grant.
	roleIDs := []uuid.UUID{}
	slugs := []string{}
	if len(req.Roles) > 0 {
		productRoles, err := handler.repo.GetRolesByProductID(ctx, project.ID)
		if err != nil {
			log.Err(err).Msg("ServerReplaceUserRoles: GetRolesByProductID failed")
			WriteError(w, r, "error.internalError", http.StatusInternalServerError)
			return
		}
		bySlug := make(map[string]uuid.UUID, len(productRoles))
		for _, role := range productRoles {
			bySlug[role.Slug] = role.ID
		}
		seen := make(map[string]bool, len(req.Roles))
		for _, raw := range req.Roles {
			slug := strings.TrimSpace(raw)
			id, known := bySlug[slug]
			if !known {
				WriteError(w, r, "error.rolesInvalid", http.StatusBadRequest)
				return
			}
			if seen[slug] {
				continue
			}
			seen[slug] = true
			roleIDs = append(roleIDs, id)
			slugs = append(slugs, slug)
		}
	}

	if err := handler.repo.ReplaceUserRoles(ctx, repo.ReplaceUserRolesParams{
		ProductID: project.ID,
		AppID:     app.ID,
		UserID:    userID,
		RoleIDs:   roleIDs,
		Now:       time.Now().UTC(),
	}); err != nil {
		if errors.Is(err, repo.ErrBadRequest) {
			WriteError(w, r, "error.rolesInvalid", http.StatusBadRequest)
			return
		}
		log.Err(err).Msg("ServerReplaceUserRoles: ReplaceUserRoles failed")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	// Clearing all roles removes the user's access; revoke their live
	// sessions so the change takes effect immediately rather than at token
	// expiry. Mirrors the admin member-roles handler.
	if len(roleIDs) == 0 {
		if n, err := handler.repo.DeleteClientSessionsByUserAndApp(ctx, userID, app.ID); err != nil {
			log.Err(err).Msg("ServerReplaceUserRoles: failed to revoke sessions after clearing roles")
		} else if n > 0 {
			log.Info().Int64("deleted", n).Str("userId", userID.String()).Str("appId", app.ID.String()).
				Msg("Revoked sessions after clearing roles via server API")
		}
	}

	// Echo the assigned slugs: they are exactly what was just stored, so no
	// read-back query is needed.
	utils.WriteJson(w, ServerRolesResponse{Roles: slugs})
}
