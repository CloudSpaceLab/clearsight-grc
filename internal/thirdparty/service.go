package thirdparty

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() (string, error)
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, newID: id.NewUUIDv7}
}

func (s *Service) CreateRelationship(ctx context.Context, actor Actor, input CreateRelationshipInput) (Aggregate, error) {
	input.ExistingRelationshipID = strings.TrimSpace(input.ExistingRelationshipID)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || strings.TrimSpace(actor.PrincipalID) == "" || (input.ExistingRelationshipID == "" && input.LegalName == "") || input.ServiceName == "" || !validCriticality(input.Criticality) || !validPrivacyRole(input.PrivacyRole) {
		return Aggregate{}, ErrInvalid
	}
	var vendor Vendor
	reuseVendor := input.ExistingRelationshipID != ""
	if reuseVendor {
		existing, err := s.repo.GetRelationship(ctx, scopeFrom(actor), input.ExistingRelationshipID)
		if err != nil {
			return Aggregate{}, err
		}
		vendor = existing.Vendor
	} else {
		vendorID, err := s.newID()
		if err != nil {
			return Aggregate{}, err
		}
		now := s.now().UTC()
		vendor = Vendor{
			ID: vendorID, TenantID: strings.TrimSpace(actor.TenantID), LegalName: input.LegalName,
			TradingName: strings.TrimSpace(input.TradingName), RegistrationRef: strings.TrimSpace(input.RegistrationRef),
			Jurisdiction: strings.TrimSpace(input.Jurisdiction), SourceID: strings.TrimSpace(input.SourceID),
			ExternalRef: strings.TrimSpace(input.ExternalRef), Status: VendorActive, CreatedAt: now, UpdatedAt: now, Version: 1,
		}
	}
	relationshipID, err := s.newID()
	if err != nil {
		return Aggregate{}, err
	}
	now := s.now().UTC()
	record := CreateRecord{
		Vendor: vendor,
		Relationship: Relationship{
			ID: relationshipID, TenantID: strings.TrimSpace(actor.TenantID), LegalEntityID: strings.TrimSpace(actor.LegalEntityID),
			VendorID: vendor.ID, ServiceName: input.ServiceName, BusinessOwnerPrincipalID: strings.TrimSpace(actor.PrincipalID),
			Criticality: input.Criticality, PrivacyRole: input.PrivacyRole, Status: RelationshipProposed,
			EffectiveFrom: input.EffectiveFrom, RenewalAt: input.RenewalAt, SourceID: strings.TrimSpace(input.SourceID),
			ExternalRef: strings.TrimSpace(input.ExternalRef), CreatedAt: now, UpdatedAt: now, Version: 1,
		}, ReuseVendor: reuseVendor,
	}
	return s.repo.CreateRelationship(ctx, record)
}

func (s *Service) GetRelationship(ctx context.Context, actor Actor, relationshipID string) (Aggregate, error) {
	if !validActor(actor) || strings.TrimSpace(relationshipID) == "" {
		return Aggregate{}, ErrInvalid
	}
	return s.repo.GetRelationship(ctx, scopeFrom(actor), strings.TrimSpace(relationshipID))
}

func (s *Service) ListRelationships(ctx context.Context, actor Actor, input ListInput) (RelationshipPage, error) {
	if !validActor(actor) || input.Limit < 0 || input.Limit > 100 {
		return RelationshipPage{}, ErrInvalid
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	return s.repo.ListRelationships(ctx, ListFilter{Scope: scopeFrom(actor), Search: strings.TrimSpace(input.Search), Cursor: strings.TrimSpace(input.Cursor), Limit: input.Limit})
}

func (s *Service) UpdateRelationship(ctx context.Context, actor Actor, relationshipID string, input UpdateRelationshipInput) (Aggregate, error) {
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	if !validActor(actor) || strings.TrimSpace(relationshipID) == "" || input.ExpectedVersion < 1 || input.ServiceName == "" || !validCriticality(input.Criticality) || !validPrivacyRole(input.PrivacyRole) {
		return Aggregate{}, ErrInvalid
	}
	current, err := s.repo.GetRelationship(ctx, scopeFrom(actor), strings.TrimSpace(relationshipID))
	if err != nil {
		return Aggregate{}, err
	}
	if current.Relationship.Version != input.ExpectedVersion {
		return Aggregate{}, ErrVersionConflict
	}
	relationship := current.Relationship
	relationship.ServiceName = input.ServiceName
	relationship.Criticality = input.Criticality
	relationship.PrivacyRole = input.PrivacyRole
	relationship.EffectiveFrom = input.EffectiveFrom
	relationship.RenewalAt = input.RenewalAt
	now := s.now().UTC()
	relationship.UpdatedAt = now
	return s.repo.UpdateRelationship(ctx, UpdateRecord{
		Scope: scopeFrom(actor), ID: strings.TrimSpace(relationshipID), ExpectedVersion: input.ExpectedVersion,
		Relationship: relationship, ActorID: strings.TrimSpace(actor.PrincipalID),
	})
}

func validActor(actor Actor) bool {
	return strings.TrimSpace(actor.TenantID) != "" && strings.TrimSpace(actor.LegalEntityID) != "" && strings.TrimSpace(actor.PrincipalID) != ""
}

func scopeFrom(actor Actor) Scope {
	return Scope{TenantID: strings.TrimSpace(actor.TenantID), LegalEntityID: strings.TrimSpace(actor.LegalEntityID)}
}

func validCriticality(value Criticality) bool {
	return value == CriticalityStandard || value == CriticalityImportant || value == CriticalityCritical
}

func validPrivacyRole(value PrivacyRole) bool {
	return value == PrivacyNone || value == PrivacyProcessor || value == PrivacyJointController
}
