package api

import (
	"encoding/json"
	"net/http"

	"manyrows-core/utils"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
)

// UserTagsResponse is the wire shape for both the list-user-tags and
// replace-user-tags endpoints. Single field keeps the JSON narrow so the
// admin UI can ignore versioning concerns.
type UserTagsResponse struct {
	Tags []string `json:"tags"`
}

type ReplaceUserTagsRequest struct {
	Tags []string `json:"tags"`
}

// HandleListUserTags — GET /admin/.../apps/{appId}/users/{userId}/tags
func (handler *RequestHandler) HandleListUserTags(w http.ResponseWriter, r *http.Request) {
	_, _, appID, ok := handler.parseAppContext(w, r)
	if !ok {
		return
	}
	userID, err := utils.GetPathUUID("userId", r)
	if err != nil || userID == uuid.Nil {
		WriteError(w, r, "error.invalidUserId", http.StatusBadRequest)
		return
	}

	tags, err := handler.repo.ListUserTags(r.Context(), appID, userID)
	if err != nil {
		log.Err(err).Msg("failed to list user tags")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	utils.WriteJsonWithStatusCode(w, UserTagsResponse{Tags: tags}, http.StatusOK)
}

// HandleReplaceUserTags — PUT /admin/.../apps/{appId}/users/{userId}/tags
//
// Replaces the entire tag set for a user. Tags are normalized server-side
// (trimmed, lowercased, deduplicated, max 40 chars each); invalid entries
// are dropped silently. The response echoes the cleaned set so the UI can
// reconcile.
func (handler *RequestHandler) HandleReplaceUserTags(w http.ResponseWriter, r *http.Request) {
	_, _, appID, ok := handler.parseAppContext(w, r)
	if !ok {
		return
	}
	userID, err := utils.GetPathUUID("userId", r)
	if err != nil || userID == uuid.Nil {
		WriteError(w, r, "error.invalidUserId", http.StatusBadRequest)
		return
	}

	var req ReplaceUserTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "error.invalidJson", http.StatusBadRequest)
		return
	}
	// Cap absurd inputs — 50 tags per user is more than enough.
	if len(req.Tags) > 50 {
		req.Tags = req.Tags[:50]
	}

	tags, err := handler.repo.ReplaceUserTags(r.Context(), appID, userID, req.Tags)
	if err != nil {
		log.Err(err).Msg("failed to replace user tags")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	utils.WriteJsonWithStatusCode(w, UserTagsResponse{Tags: tags}, http.StatusOK)
}

// HandleListAppTags — GET /admin/.../apps/{appId}/tags
//
// Distinct tag names in use anywhere in this app. Powers the autocomplete
// on the edit dialog so admins reuse existing names rather than typo
// variants ("VIP" vs "vip" vs "v.i.p.").
func (handler *RequestHandler) HandleListAppTags(w http.ResponseWriter, r *http.Request) {
	_, _, appID, ok := handler.parseAppContext(w, r)
	if !ok {
		return
	}
	tags, err := handler.repo.ListAppDistinctTags(r.Context(), appID)
	if err != nil {
		log.Err(err).Msg("failed to list app tags")
		WriteError(w, r, "error.internalError", http.StatusInternalServerError)
		return
	}
	utils.WriteJsonWithStatusCode(w, UserTagsResponse{Tags: tags}, http.StatusOK)
}
