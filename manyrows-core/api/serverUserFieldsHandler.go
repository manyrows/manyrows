package api

import (
	"net/http"

	"manyrows-core/core"
	"manyrows-core/utils"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

// GET /x/{workspaceSlug}/api/apps/{appId}/user-fields
func (handler *RequestHandler) HandleServerGetUserFields(w http.ResponseWriter, r *http.Request) {
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

	fields, err := handler.repo.GetUserFieldsByUserPoolID(ctx, app.UserPoolID)
	if err != nil {
		log.Err(err).Msg("HandleServerGetUserFields: failed")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	utils.WriteJsonWithStatusCode(w, UserFieldsResponse{UserFields: fields}, http.StatusOK)
}

// GET /x/{workspaceSlug}/api/apps/{appId}/user-fields/users/{userId}
func (handler *RequestHandler) HandleServerGetUserFieldValues(w http.ResponseWriter, r *http.Request) {
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

	userID, err := uuid.FromString(chi.URLParam(r, "userId"))
	if err != nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	// Scope check: the user must belong to this app's pool, otherwise
	// the server SDK could read values from a foreign pool by guessing
	// IDs.
	target, err := handler.repo.GetUserByID(ctx, userID)
	if err != nil || target == nil || target.UserPoolID != app.UserPoolID {
		WriteError(w, r, "error.notFound", http.StatusNotFound)
		return
	}

	values, err := handler.repo.GetUserFieldValuesByUser(ctx, userID)
	if err != nil {
		log.Err(err).Msg("HandleServerGetUserFieldValues: failed")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	utils.WriteJsonWithStatusCode(w, UserFieldValuesResponse{Values: values}, http.StatusOK)
}
