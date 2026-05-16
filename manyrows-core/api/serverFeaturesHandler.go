package api

import (
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

// ----------------------------------------------------------
// Server API — Feature Flag endpoints (API-key auth)
// ----------------------------------------------------------

// serverFeatureFlagOverrideItem is one per-app override in the GET response.
type serverFeatureFlagOverrideItem struct {
	AppID   uuid.UUID `json:"appId"`
	Enabled bool      `json:"enabled"`
}

// serverFeatureFlagItem is a single flag in the GET /features response.
type serverFeatureFlagItem struct {
	Key            string                          `json:"key"`
	Description    *string                         `json:"description,omitempty"`
	Scope          core.FeatureFlagScope           `json:"scope"`
	DefaultEnabled bool                            `json:"defaultEnabled"`
	Apps           []serverFeatureFlagOverrideItem `json:"apps"`
}

// serverGetFeaturesResponse is the response for GET /features.
type serverGetFeaturesResponse struct {
	Features []serverFeatureFlagItem `json:"features"`
}

// serverSetFeatureFlagRequest is the body for PUT .../apps/{appId}.
type serverSetFeatureFlagRequest struct {
	Enabled *bool `json:"enabled"`
}

// HandleServerGetFeatures lists all feature flags with per-app overrides.
// Route: GET /x/{workspaceSlug}/api/products/{productId}/features
func (handler *RequestHandler) HandleServerGetFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusBadRequest)
		return
	}

	flags, err := handler.repo.GetFeatureFlagsByProductID(ctx, project.ID)
	if err != nil {
		log.Err(err).Msg("HandleServerGetFeatures: could not get feature flags")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	overrides, err := handler.repo.GetFeatureFlagOverridesByProductID(ctx, project.ID)
	if err != nil {
		log.Err(err).Msg("HandleServerGetFeatures: could not get feature flag overrides")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	apps, err := handler.repo.GetAppsByWorkspaceAndProductID(ctx, ws.ID, project.ID)
	if err != nil {
		log.Err(err).Msg("HandleServerGetFeatures: could not get apps")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	validAppIDs := make(map[uuid.UUID]struct{}, len(apps))
	for _, a := range apps {
		validAppIDs[a.ID] = struct{}{}
	}

	// Build flag ID -> overrides map. Skip overrides whose app no longer
	// belongs to this project (defense against stale rows).
	overridesByFlagID := make(map[uuid.UUID][]serverFeatureFlagOverrideItem)
	for _, o := range overrides {
		if _, ok := validAppIDs[o.AppID]; !ok {
			continue
		}
		overridesByFlagID[o.FeatureFlagID] = append(overridesByFlagID[o.FeatureFlagID], serverFeatureFlagOverrideItem{
			AppID:   o.AppID,
			Enabled: o.Enabled,
		})
	}

	// Assemble response
	items := make([]serverFeatureFlagItem, 0, len(flags))
	for _, f := range flags {
		appOverrides := overridesByFlagID[f.ID]
		if appOverrides == nil {
			appOverrides = []serverFeatureFlagOverrideItem{}
		}
		items = append(items, serverFeatureFlagItem{
			Key:            f.Key,
			Description:    f.Description,
			Scope:          f.Scope,
			DefaultEnabled: f.DefaultEnabled,
			Apps:           appOverrides,
		})
	}

	utils.WriteJsonWithStatusCode(w, serverGetFeaturesResponse{Features: items}, http.StatusOK)
}

// HandleServerSetFeatureFlag upserts a feature flag override for an app.
// Route: PUT /x/{workspaceSlug}/api/products/{productId}/features/{flagKey}/apps/{appId}
func (handler *RequestHandler) HandleServerSetFeatureFlag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusBadRequest)
		return
	}

	apiKey, ok := core.APIKeyFromContext(ctx)
	if !ok || apiKey == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	flagKey := chi.URLParam(r, "flagKey")
	appIDStr := chi.URLParam(r, "appId")

	if flagKey == "" || appIDStr == "" {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	appID, err := uuid.FromString(appIDStr)
	if err != nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	app, err := handler.repo.GetAppByID(ctx, appID)
	if err != nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}
	if app.ProductID != project.ID {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	// Look up feature flag by key
	flag, err := handler.repo.GetFeatureFlagByProductIDAndKey(ctx, project.ID, flagKey)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.featureFlagNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("HandleServerSetFeatureFlag: could not get feature flag")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	var req serverSetFeatureFlagRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Enabled == nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()

	override := core.FeatureFlagOverride{
		ID:            utils.NewUUID(),
		ProductID:     project.ID,
		FeatureFlagID: flag.ID,
		Enabled:       *req.Enabled,
		Status:        "active",
		UpdatedAt:     now,
		UpdatedBy:     apiKey.CreatedBy,
	}

	saved, err := handler.repo.UpsertFeatureFlagOverride(ctx, override)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.featureFlagNotFound", http.StatusNotFound)
			return
		}
		if errors.Is(err, repo.ErrConflict) || repo.IsUniqueViolation(err) {
			WriteError(w, r, "error.conflict", http.StatusConflict)
			return
		}
		log.Err(err).Msg("HandleServerSetFeatureFlag: could not upsert feature flag")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	utils.WriteJsonWithStatusCode(w, saved, http.StatusOK)
}

// HandleServerDeleteFeatureFlag removes a feature flag override for an app.
// Route: DELETE /x/{workspaceSlug}/api/products/{productId}/features/{flagKey}/apps/{appId}
func (handler *RequestHandler) HandleServerDeleteFeatureFlag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusBadRequest)
		return
	}

	apiKey, ok := core.APIKeyFromContext(ctx)
	if !ok || apiKey == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	flagKey := chi.URLParam(r, "flagKey")
	appIDStr := chi.URLParam(r, "appId")

	if flagKey == "" || appIDStr == "" {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	appID, err := uuid.FromString(appIDStr)
	if err != nil {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	app, err := handler.repo.GetAppByID(ctx, appID)
	if err != nil {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}
	if app.ProductID != project.ID {
		WriteError(w, r, "error.appNotFound", http.StatusNotFound)
		return
	}

	// Look up feature flag by key
	flag, err := handler.repo.GetFeatureFlagByProductIDAndKey(ctx, project.ID, flagKey)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.featureFlagNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("HandleServerDeleteFeatureFlag: could not get feature flag")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	err = handler.repo.DeleteFeatureFlagOverride(ctx, project.ID, flag.ID, app.ID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.featureFlagNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("HandleServerDeleteFeatureFlag: could not delete feature flag")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
