package thirdparty

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type LinkTargetType string

const (
	LinkTargetProgram LinkTargetType = "PROGRAM"
	LinkTargetMatter  LinkTargetType = "MATTER"
)

type RelationshipLinkState string

const (
	RelationshipLinkActive RelationshipLinkState = "ACTIVE"
	RelationshipLinkEnded  RelationshipLinkState = "ENDED"
)

type RelationshipLink struct {
	ID             string                `json:"id"`
	TenantID       string                `json:"tenant_id"`
	LegalEntityID  string                `json:"legal_entity_id"`
	RelationshipID string                `json:"relationship_id"`
	TargetType     LinkTargetType        `json:"target_type"`
	TargetID       string                `json:"target_id"`
	PurposeCode    string                `json:"purpose_code"`
	PurposeLabel   string                `json:"purpose_label"`
	State          RelationshipLinkState `json:"state"`
	CreatedBy      string                `json:"created_by"`
	EndedBy        string                `json:"ended_by,omitempty"`
	EndReason      string                `json:"end_reason,omitempty"`
	Version        int64                 `json:"version"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	EndedAt        *time.Time            `json:"ended_at,omitempty"`
}

type LinkRelationshipInput struct {
	TargetType   LinkTargetType `json:"target_type"`
	TargetID     string         `json:"target_id"`
	PurposeCode  string         `json:"purpose_code"`
	PurposeLabel string         `json:"purpose_label"`
}

type EndRelationshipLinkInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason"`
}

type RelationshipLinkListInput struct {
	RelationshipID string         `json:"relationship_id,omitempty"`
	TargetType     LinkTargetType `json:"target_type,omitempty"`
	TargetID       string         `json:"target_id,omitempty"`
	IncludeEnded   bool           `json:"include_ended,omitempty"`
	Cursor         string         `json:"cursor,omitempty"`
	Limit          int            `json:"limit,omitempty"`
}

type RelationshipLinkPage struct {
	Items      []RelationshipLink `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type RelationshipLinkRepository interface {
	RelationshipExists(context.Context, Scope, string) (bool, error)
	TargetAvailable(context.Context, Scope, LinkTargetType, string) (bool, error)
	CreateRelationshipLink(context.Context, RelationshipLink) (RelationshipLink, error)
	GetRelationshipLink(context.Context, Scope, string) (RelationshipLink, error)
	EndRelationshipLink(context.Context, Scope, string, int64, string, string, time.Time) (RelationshipLink, error)
	ListRelationshipLinks(context.Context, Scope, RelationshipLinkListInput) (RelationshipLinkPage, error)
}

type RelationshipLinkService struct {
	repo  RelationshipLinkRepository
	now   func() time.Time
	newID func() (string, error)
}

func NewRelationshipLinkService(repo RelationshipLinkRepository) *RelationshipLinkService {
	return &RelationshipLinkService{repo: repo, now: time.Now, newID: id.NewUUIDv7}
}

var purposeCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

func (s *RelationshipLinkService) Link(ctx context.Context, actor Actor, relationshipID string, input LinkRelationshipInput) (RelationshipLink, error) {
	relationshipID = strings.TrimSpace(relationshipID)
	input.TargetID = strings.TrimSpace(input.TargetID)
	input.PurposeCode = strings.ToUpper(strings.TrimSpace(input.PurposeCode))
	input.PurposeLabel = strings.TrimSpace(input.PurposeLabel)
	if s == nil || s.repo == nil || !validActor(actor) || relationshipID == "" || input.TargetID == "" ||
		(input.TargetType != LinkTargetProgram && input.TargetType != LinkTargetMatter) || !purposeCodePattern.MatchString(input.PurposeCode) || len(input.PurposeLabel) == 0 || len(input.PurposeLabel) > 160 {
		return RelationshipLink{}, ErrInvalid
	}
	scope := scopeFrom(actor)
	exists, err := s.repo.RelationshipExists(ctx, scope, relationshipID)
	if err != nil {
		return RelationshipLink{}, err
	}
	if !exists {
		return RelationshipLink{}, ErrNotFound
	}
	available, err := s.repo.TargetAvailable(ctx, scope, input.TargetType, input.TargetID)
	if err != nil {
		return RelationshipLink{}, err
	}
	if !available {
		return RelationshipLink{}, ErrNotFound
	}
	linkID, err := s.newID()
	if err != nil {
		return RelationshipLink{}, err
	}
	now := s.now().UTC()
	return s.repo.CreateRelationshipLink(ctx, RelationshipLink{
		ID: linkID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: relationshipID,
		TargetType: input.TargetType, TargetID: input.TargetID, PurposeCode: input.PurposeCode, PurposeLabel: input.PurposeLabel,
		State: RelationshipLinkActive, CreatedBy: strings.TrimSpace(actor.PrincipalID), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *RelationshipLinkService) End(ctx context.Context, actor Actor, linkID string, input EndRelationshipLinkInput) (RelationshipLink, error) {
	linkID, input.Reason = strings.TrimSpace(linkID), strings.TrimSpace(input.Reason)
	if s == nil || s.repo == nil || !validActor(actor) || linkID == "" || input.ExpectedVersion < 1 || input.Reason == "" || len(input.Reason) > 1000 {
		return RelationshipLink{}, ErrInvalid
	}
	return s.repo.EndRelationshipLink(ctx, scopeFrom(actor), linkID, input.ExpectedVersion, strings.TrimSpace(actor.PrincipalID), input.Reason, s.now().UTC())
}

func (s *RelationshipLinkService) List(ctx context.Context, actor Actor, input RelationshipLinkListInput) (RelationshipLinkPage, error) {
	if s == nil || s.repo == nil || !validActor(actor) || input.Limit < 0 || input.Limit > 100 {
		return RelationshipLinkPage{}, ErrInvalid
	}
	input.RelationshipID, input.TargetID, input.Cursor = strings.TrimSpace(input.RelationshipID), strings.TrimSpace(input.TargetID), strings.TrimSpace(input.Cursor)
	if input.TargetType != "" && input.TargetType != LinkTargetProgram && input.TargetType != LinkTargetMatter {
		return RelationshipLinkPage{}, ErrInvalid
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	return s.repo.ListRelationshipLinks(ctx, scopeFrom(actor), input)
}

func relationshipLinkConflict(err error) error {
	if errors.Is(err, ErrVersionConflict) {
		return ErrVersionConflict
	}
	return err
}
