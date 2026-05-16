package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"manyrows-core/core"
	"manyrows-core/core/repo"
	"manyrows-core/utils"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

// =====================
// DTOs
// =====================

// UserRolesResponse wraps the list of user_roles rows for an admin
// "member roles" view. The JSON key stays `memberRoles` so the admin
// UI keeps working without coordinated changes; rename the wire shape
// in a follow-on if/when the UI is ready to migrate.
type UserRolesResponse struct {
	UserRoles []core.UserRole `json:"memberRoles"`
}

type UpdateMemberRolesRequest struct {
	RoleIDs []uuid.UUID `json:"roleIds"`
	AppID   uuid.UUID   `json:"appId"` // REQUIRED: no project-wide roles allowed
}

// =====================
// Handlers
// =====================

func (handler *RequestHandler) HandleGetMemberRoles(w http.ResponseWriter, r *http.Request) {
	_, _, project, ok := handler.adminAndProduct(w, r)
	if !ok {
		return
	}

	roles, err := handler.repo.GetUserRolesByProductID(r.Context(), project.ID)
	if err != nil {
		log.Err(err).Msg("Could not get member roles")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	utils.WriteJsonWithStatusCode(w, UserRolesResponse{UserRoles: roles}, http.StatusOK)
}

func (handler *RequestHandler) HandlerUpdateMemberRoles(w http.ResponseWriter, r *http.Request) {
	_, _, project, ok := handler.adminAndProduct(w, r)
	if !ok {
		return
	}

	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	// Validate the user exists (users are global, not workspace-scoped).
	_, err := handler.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.userNotFound", http.StatusBadRequest)
			return
		}
		log.Err(err).Msg("Could not validate user")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	var req UpdateMemberRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "error.invalidJson", http.StatusBadRequest)
		return
	}

	// Enforce env-scoped access (no "all envs")
	if req.AppID == uuid.Nil {
		WriteError(w, r, "error.appIdRequired", http.StatusBadRequest)
		return
	}

	roleIDs := req.RoleIDs
	if roleIDs == nil {
		roleIDs = []uuid.UUID{}
	}

	appID := req.AppID
	now := time.Now().UTC()

	err = handler.repo.ReplaceUserRoles(r.Context(), repo.ReplaceUserRolesParams{
		ProductID: project.ID,
		AppID:     appID,
		UserID:    userID, // ALWAYS non-nil now
		RoleIDs:   roleIDs,
		Now:       now,
	})
	if err != nil {
		if errors.Is(err, repo.ErrBadRequest) {
			WriteError(w, r, "error.rolesInvalid", http.StatusBadRequest)
			return
		}
		log.Err(err).Msg("Could not replace project member roles")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	if len(roleIDs) == 0 {
		if n, err := handler.repo.DeleteClientSessionsByUserAndApp(r.Context(), userID, appID); err != nil {
			log.Err(err).Msg("Could not revoke sessions after removing member roles")
		} else if n > 0 {
			log.Info().Int64("deleted", n).Str("userId", userID.String()).Str("appId", appID.String()).Msg("Revoked sessions after removing member roles")
		}
	}

	// Return the current assignments for that user in this project (all scopes)
	rows, err := handler.repo.GetUserRolesByUserID(r.Context(), project.ID, userID)
	if err != nil {
		// Update succeeded; if read fails just return 204
		w.WriteHeader(http.StatusNoContent)
		return
	}

	utils.WriteJsonWithStatusCode(w, UserRolesResponse{UserRoles: rows}, http.StatusOK)
}

// HandleRemoveProductMember fully removes a user from an app: clears
// their role assignments for that app AND deletes the app_users
// membership row, then revokes their sessions for the app. Roles are
// optional, so clearing roles alone (the prior behaviour) left the
// user a member with zero roles — still listed, still counted. This
// actually removes them.
// DELETE /admin/workspace/{workspaceId}/products/{productId}/members/{userId}?appId=...
func (handler *RequestHandler) HandleRemoveProductMember(w http.ResponseWriter, r *http.Request) {
	_, _, project, ok := handler.adminAndProduct(w, r)
	if !ok {
		return
	}

	userID, ok := handler.userIDFromURL(w, r)
	if !ok {
		return
	}

	appID, err := uuid.FromString(r.URL.Query().Get("appId"))
	if err != nil || appID == uuid.Nil {
		WriteError(w, r, "error.appIdRequired", http.StatusBadRequest)
		return
	}

	// Clear the user's role assignments for this app first. user_roles
	// has no FK to app_users, so deleting the membership wouldn't
	// cascade them — they'd be orphaned and re-applied on re-add.
	if err := handler.repo.ReplaceUserRoles(r.Context(), repo.ReplaceUserRolesParams{
		ProductID: project.ID,
		AppID:     appID,
		UserID:    userID,
		RoleIDs:   []uuid.UUID{},
		Now:       time.Now().UTC(),
	}); err != nil {
		if errors.Is(err, repo.ErrBadRequest) {
			WriteError(w, r, "error.rolesInvalid", http.StatusBadRequest)
			return
		}
		log.Err(err).Msg("Could not clear roles while removing app member")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	// Clear direct permission overrides for this app too — same orphan
	// reasoning as roles (no FK from user_permissions to app_users).
	if err := handler.repo.SetDirectPermissions(r.Context(), project.ID, userID, appID, []uuid.UUID{}); err != nil {
		log.Err(err).Msg("Could not clear permission overrides while removing app member")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	// Delete the membership row. Idempotent: a missing row (already
	// removed / double-submit) is not an error.
	if err := handler.repo.DeleteAppMember(r.Context(), appID, userID); err != nil && !errors.Is(err, repo.ErrNotFound) {
		log.Err(err).Msg("Could not delete app membership")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	// No longer a member — revoke their sessions for this app (mirrors
	// the roles-emptied path in HandlerUpdateMemberRoles).
	if n, err := handler.repo.DeleteClientSessionsByUserAndApp(r.Context(), userID, appID); err != nil {
		log.Err(err).Msg("Could not revoke sessions after removing app member")
	} else if n > 0 {
		log.Info().Int64("deleted", n).Str("userId", userID.String()).Str("appId", appID.String()).Msg("Revoked sessions after removing app member")
	}

	utils.WriteJsonWithStatusCode(w, map[string]any{"removed": true}, http.StatusOK)
}

/* ===== URL helper ===== */

func (handler *RequestHandler) userIDFromURL(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "userId")
	id, err := uuid.FromString(raw)
	if err != nil {
		WriteError(w, r, "error.invalidUserId", http.StatusBadRequest)
		return uuid.UUID{}, false
	}
	return id, true
}
