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

type ServerConfigResponse struct {
	Config []ServerConfigKeyEntry `json:"config"`
}

type ServerConfigKeyEntry struct {
	Key         string                 `json:"key"`
	Description *string                `json:"description,omitempty"`
	Exposure    string                 `json:"exposure"`
	ValueType   core.ConfigValueType   `json:"valueType"`
	Values      []ServerConfigAppValue `json:"values,omitempty"`
}

type ServerConfigAppValue struct {
	AppID uuid.UUID       `json:"appId"`
	Value json.RawMessage `json:"value,omitempty"`
}

type ServerSetConfigValueRequest struct {
	Value json.RawMessage `json:"value"`
}

// GET /x/{workspaceSlug}/api/products/{productId}/config
func (handler *RequestHandler) HandleServerGetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusNotFound)
		return
	}

	keys, err := handler.repo.GetConfigKeysByProductID(ctx, project.ID)
	if err != nil {
		log.Err(err).Msg("failed to get config keys")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	values, err := handler.repo.GetConfigValuesByProductID(ctx, project.ID)
	if err != nil {
		log.Err(err).Msg("failed to get config values")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	apps, err := handler.repo.GetAppsByWorkspaceAndProductID(ctx, ws.ID, project.ID)
	if err != nil {
		log.Err(err).Msg("failed to get apps")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	validAppIDs := make(map[uuid.UUID]struct{}, len(apps))
	for _, a := range apps {
		validAppIDs[a.ID] = struct{}{}
	}

	valuesByKey := make(map[uuid.UUID][]core.ConfigValue)
	for _, v := range values {
		valuesByKey[v.ConfigKeyID] = append(valuesByKey[v.ConfigKeyID], v)
	}

	entries := make([]ServerConfigKeyEntry, 0, len(keys))
	for _, ck := range keys {
		if ck.Status != "active" {
			continue
		}

		entry := ServerConfigKeyEntry{
			Key:         ck.Key,
			Description: ck.Description,
			Exposure:    ck.Exposure,
			ValueType:   ck.ValueType,
		}

		if ck.Exposure != core.ConfigExposureSecret {
			cvs := valuesByKey[ck.ID]
			appVals := make([]ServerConfigAppValue, 0, len(cvs))
			for _, cv := range cvs {
				if _, ok := validAppIDs[cv.AppID]; !ok {
					continue
				}
				appVals = append(appVals, ServerConfigAppValue{
					AppID: cv.AppID,
					Value: cv.ValueJSON,
				})
			}
			entry.Values = appVals
		}

		entries = append(entries, entry)
	}

	utils.WriteJsonWithStatusCode(w, ServerConfigResponse{Config: entries}, http.StatusOK)
}

// PUT /x/{workspaceSlug}/api/products/{productId}/config/{configKey}/apps/{appId}
func (handler *RequestHandler) HandleServerSetConfigValue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey, ok := core.APIKeyFromContext(ctx)
	if !ok || apiKey == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusNotFound)
		return
	}

	configKeyStr := chi.URLParam(r, "configKey")
	appIDStr := chi.URLParam(r, "appId")

	if configKeyStr == "" || appIDStr == "" {
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

	ck, err := handler.repo.GetConfigKeyByProductIDAndKey(ctx, project.ID, configKeyStr)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.configKeyNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("failed to get config key")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	if ck.Exposure == core.ConfigExposureSecret {
		WriteError(w, r, "error.secretsNotSupportedViaAPI", http.StatusBadRequest)
		return
	}

	var req ServerSetConfigValueRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if !isNonEmptyJSON(req.Value) {
		WriteError(w, r, "error.badRequest", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()

	cv := core.ConfigValue{
		ID:            utils.NewUUID(),
		ProductID:     project.ID,
		ConfigKeyID:   ck.ID,
		UpdatedAt:     now,
		UpdatedBy:     apiKey.CreatedBy,
	}

	out, err := handler.repo.UpsertConfigValueJSON(ws.ID, ctx, cv, req.Value, nil)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.configKeyNotFound", http.StatusNotFound)
			return
		} else if errors.Is(err, repo.ErrConflict) {
			WriteError(w, r, "error.conflict", http.StatusConflict)
			return
		} else if errors.Is(err, repo.ErrBadRequest) {
			WriteError(w, r, "error.badRequest", http.StatusBadRequest)
			return
		} else {
			log.Err(err).Msg("failed to upsert config value")
			WriteError(w, r, "error.internalError", http.StatusInternalServerError)
			return
		}
	}

	utils.WriteJsonWithStatusCode(w, ConfigValueResponse{ConfigValue: out}, http.StatusOK)
}

// DELETE /x/{workspaceSlug}/api/products/{productId}/config/{configKey}/apps/{appId}
func (handler *RequestHandler) HandleServerDeleteConfigValue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ws, ok := core.WorkspaceFromContext(ctx)
	if !ok || ws == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey, ok := core.APIKeyFromContext(ctx)
	if !ok || apiKey == nil {
		WriteError(w, r, "error.unauthorized", http.StatusUnauthorized)
		return
	}

	project, ok := core.ProductFromContext(ctx)
	if !ok || project == nil {
		WriteError(w, r, "error.projectNotFound", http.StatusNotFound)
		return
	}

	configKeyStr := chi.URLParam(r, "configKey")
	appIDStr := chi.URLParam(r, "appId")

	if configKeyStr == "" || appIDStr == "" {
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

	ck, err := handler.repo.GetConfigKeyByProductIDAndKey(ctx, project.ID, configKeyStr)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.configKeyNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("failed to get config key")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	err = handler.repo.DeleteConfigValue(ctx, project.ID, ck.ID, app.ID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			WriteError(w, r, "error.configKeyNotFound", http.StatusNotFound)
			return
		}
		log.Err(err).Msg("failed to delete config value")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
