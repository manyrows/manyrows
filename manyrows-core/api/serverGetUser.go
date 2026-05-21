package api

import (
	"errors"
	"net/http"
	"strings"

	"manyrows-core/core"
	"manyrows-core/core/repo"
	"manyrows-core/utils"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

type ServerUserResponse struct {
	User        *core.UserResource    `json:"user"`
	Roles       []string              `json:"roles"`
	Permissions []string              `json:"permissions"`
	Fields      []core.UserFieldValue `json:"fields,omitempty"`
}

// GET /x/{workspaceSlug}/api/v1/apps/{appId}/users?email=...&id=...
func (handler *RequestHandler) HandleServerGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, ok := core.WorkspaceFromContext(ctx)
	if !ok {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	app, ok := core.AppFromContext(ctx)
	if !ok || app == nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	email := strings.TrimSpace(strings.ToLower(q.Get("email")))
	idStr := strings.TrimSpace(q.Get("id"))

	if email == "" && idStr == "" {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	var user *core.User
	var err error

	if idStr != "" {
		// Lookup by ID
		userID, parseErr := uuid.FromString(idStr)
		if parseErr != nil {
			WriteError(w, r, "error.badRequest", http.StatusBadRequest)
			return
		}
		user, err = handler.repo.GetUserByID(ctx, userID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				WriteError(w, r, "error.notFound", http.StatusNotFound)
				return
			}
			log.Err(err).Msg("HandleServerGetUser: lookup by id failed")
			WriteError(w, r, "error.internalError", http.StatusInternalServerError)
			return
		}
	} else {
		// Lookup by email within scope
		user, err = handler.repo.GetUserByEmail(ctx, email, app)
		if err != nil {
			log.Err(err).Msg("HandleServerGetUser: lookup failed")
			WriteError(w, r, "error.internalError", http.StatusInternalServerError)
			return
		}
	}

	if user == nil {
		WriteError(w, r, "error.notFound", http.StatusNotFound)
		return
	}

	// Server API scopes to app membership: the pool only shares credentials,
	// so a user who exists in the pool but hasn't joined this app is not
	// visible here. This also closes the cross-pool lookup (a foreign user
	// has no app_users row for this app), so the by-id path needs no separate
	// pool check.
	if !handler.requireAppMember(w, r, app.ID, user.ID) {
		return
	}

	// Get roles and permissions (app-scoped now that env layer is gone)
	roles, permissions, _ := handler.resolveRolesAndPermissions(ctx, project.ID, user.ID, app.ID)

	// Get user field values (pool is implicit via user_id)
	fields, _ := handler.repo.GetUserFieldValuesByUser(ctx, user.ID)

	resp := ServerUserResponse{
		User:        core.ToUserResource(user),
		Roles:       roles,
		Permissions: permissions,
		Fields:      fields,
	}

	if resp.Roles == nil {
		resp.Roles = []string{}
	}
	if resp.Permissions == nil {
		resp.Permissions = []string{}
	}
	if resp.Fields == nil {
		resp.Fields = []core.UserFieldValue{}
	}

	utils.WriteJsonWithStatusCode(w, resp, http.StatusOK)
}
