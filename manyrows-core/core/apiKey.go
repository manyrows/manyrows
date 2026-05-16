package core

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspaceId"`
	AppID       *uuid.UUID `json:"appId,omitempty"`

	Name   string `json:"name"`
	Prefix string `json:"prefix"` // e.g. first 8 chars for UI display
	Hash   string `json:"-"`      // stored hash only, never exposed

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy uuid.UUID `json:"createdBy"`
}
