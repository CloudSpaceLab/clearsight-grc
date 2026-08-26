package evidence

import (
	"context"
	"strings"
	"time"
)

// ActiveSessionMetadata is the safe requester-facing view of one usable
// external capture session. Capability material and raw audience data are
// deliberately excluded.
type ActiveSessionMetadata struct {
	ID           string    `json:"id"`
	AudienceHint string    `json:"audience_hint"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type ActiveSessionMetadataPage struct {
	Items   []ActiveSessionMetadata `json:"items"`
	HasMore bool                    `json:"has_more"`
}

type ManageSessionsInput struct {
	TenantID         string `json:"tenant_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	RequestID        string `json:"request_id"`
	ActorPrincipalID string `json:"actor_principal_id,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

type activeSessionAdministrationStore interface {
	ListActiveSessionMetadata(context.Context, string, string, time.Time, int) ([]ActiveSessionMetadata, error)
}

func (s *Service) ListActiveSessionMetadata(ctx context.Context, input ManageSessionsInput) (ActiveSessionMetadataPage, error) {
	request, err := s.authorizeRequesterAdministration(ctx, input.TenantID, input.LegalEntityID, input.RequestID, input.ActorPrincipalID)
	if err != nil {
		return ActiveSessionMetadataPage{}, err
	}
	now := s.now().UTC()
	if !requestOpenAt(request, now) || !externalRecipientRequest(request) {
		return ActiveSessionMetadataPage{Items: []ActiveSessionMetadata{}}, nil
	}
	store, ok := s.repo.(activeSessionAdministrationStore)
	if !ok {
		return ActiveSessionMetadataPage{}, ErrRecipientInvalid
	}
	limit := boundedActiveSessionLimit(input.Limit)
	values, err := store.ListActiveSessionMetadata(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.RequestID), now, limit+1)
	if err != nil {
		return ActiveSessionMetadataPage{}, err
	}
	page := ActiveSessionMetadataPage{Items: values}
	if len(values) > limit {
		page.HasMore = true
		page.Items = values[:limit]
	}
	return page, nil
}

func boundedActiveSessionLimit(limit int) int {
	if limit < 1 || limit > 50 {
		return 50
	}
	return limit
}
