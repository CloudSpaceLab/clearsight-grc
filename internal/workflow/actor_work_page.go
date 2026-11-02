package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidWorkCursor = errors.New("work cursor is invalid")

const (
	actorWorkMaxPageSize = 100
	actorWorkCursorTTL   = 15 * time.Minute
)

type ActorWorkPageQuery struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
	Search        string
	Status        Status
	DueFrom       *time.Time
	DueBefore     *time.Time
	Cursor        string
	Limit         int
}

type ActorWorkPage struct {
	Items      []Task    `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
	HasMore    bool      `json:"has_more"`
	AsOf       time.Time `json:"as_of"`
	Coverage   string    `json:"coverage"`
	CountBasis string    `json:"count_basis"`
}

type actorWorkCursor struct {
	Version  int          `json:"v"`
	Scope    string       `json:"scope"`
	AsOf     time.Time    `json:"as_of"`
	Position WorkPosition `json:"after"`
}

// ListActorWorkPage reads only current actor-visible, non-terminal assignments.
// Scope is checked in the source queries before keyset/limit, and again when
// projecting rows. The time fence limits newer revisions but is NOT a durable
// snapshot: revocation and mutations can remove rows between page requests.
func (s *Service) ListActorWorkPage(ctx context.Context, query ActorWorkPageQuery) (ActorWorkPage, error) {
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	query.PrincipalID = strings.TrimSpace(query.PrincipalID)
	query.Search = strings.ToLower(strings.TrimSpace(query.Search))
	if query.TenantID == "" || query.LegalEntityID == "" || query.PrincipalID == "" {
		return ActorWorkPage{}, fmt.Errorf("actor work requires verified tenant, entity and principal")
	}
	if query.Status == StatusCompleted || query.Status == StatusCancelled {
		return ActorWorkPage{}, fmt.Errorf("terminal work requires a separate receipt-backed query")
	}
	if query.Limit <= 0 || query.Limit > actorWorkMaxPageSize {
		query.Limit = 25
	}
	if len(query.Cursor) > 2048 {
		return ActorWorkPage{}, ErrInvalidWorkCursor
	}
	filter := ListFilter{
		TenantID: query.TenantID, LegalEntityID: query.LegalEntityID,
		PrincipalID: query.PrincipalID, Search: query.Search, Status: query.Status,
		DueFrom: query.DueFrom, DueBefore: query.DueBefore,
		ActiveOnly: true, VisibleActorWorkOnly: true, Limit: query.Limit + 1,
	}
	fingerprint := actorWorkScopeFingerprint(query)
	asOf := time.Now().UTC().Truncate(time.Microsecond)
	if query.Cursor != "" {
		decoded, err := decodeActorWorkCursor(query.Cursor, fingerprint, asOf)
		if err != nil {
			return ActorWorkPage{}, err
		}
		asOf = decoded.AsOf
		filter.PageAfter = &decoded.Position
	}
	filter.UpdatedThrough = &asOf
	items, err := s.List(ctx, filter)
	if err != nil {
		return ActorWorkPage{}, err
	}
	page := ActorWorkPage{
		Items: append([]Task{}, items...), AsOf: asOf,
		Coverage: "WORKFLOW_ASSIGNMENTS_ONLY", CountBasis: "NOT_AVAILABLE",
	}
	if len(page.Items) > query.Limit {
		page.HasMore = true
		page.Items = page.Items[:query.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeActorWorkCursor(actorWorkCursor{
			Version: 1, Scope: fingerprint, AsOf: asOf, Position: positionForTask(last),
		})
	}
	return page, nil
}

func actorWorkScopeFingerprint(query ActorWorkPageQuery) string {
	// Identity, selected scope, search and status must all match every page.
	// The fingerprint is a cursor consistency check, not an access grant.
	value, _ := json.Marshal(struct {
		Tenant, Entity, Principal, Search string
		Status                            Status
		DueFrom, DueBefore                *time.Time
	}{
		query.TenantID, query.LegalEntityID, query.PrincipalID,
		query.Search, query.Status, query.DueFrom, query.DueBefore,
	})
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func encodeActorWorkCursor(cursor actorWorkCursor) string {
	value, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeActorWorkCursor(encoded, scope string, now time.Time) (actorWorkCursor, error) {
	var cursor actorWorkCursor
	value, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || json.Unmarshal(value, &cursor) != nil ||
		cursor.Version != 1 || cursor.Scope != scope ||
		cursor.AsOf.IsZero() || cursor.AsOf.After(now.Add(time.Minute)) ||
		now.Sub(cursor.AsOf) > actorWorkCursorTTL ||
		cursor.Position.UpdatedAt.IsZero() || cursor.Position.UpdatedAt.After(cursor.AsOf) ||
		!validNotificationUUID(cursor.Position.ID) {
		return actorWorkCursor{}, ErrInvalidWorkCursor
	}
	return cursor, nil
}
