package thirdparty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo          Repository
	now           func() time.Time
	newID         func() (string, error)
	identityGuard AssessmentCommandGuard
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, newID: id.NewUUIDv7}
}

func (s *Service) ConfigureIdentityAuthority(guard AssessmentCommandGuard) {
	if s != nil {
		s.identityGuard = guard
	}
}

func (s *Service) CreateRelationship(ctx context.Context, actor Actor, input CreateRelationshipInput) (Aggregate, error) {
	input.ExistingRelationshipID = strings.TrimSpace(input.ExistingRelationshipID)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	websiteDomain, err := normalizeOptionalWebsiteDomain(input.WebsiteDomain)
	if err != nil {
		return Aggregate{}, err
	}
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
			WebsiteDomain: websiteDomain,
		}
	}
	relationshipID, err := s.newID()
	if err != nil {
		return Aggregate{}, err
	}
	now := s.now().UTC()
	var brandJob *VendorBrandJob
	if !reuseVendor && websiteDomain != "" {
		jobID, err := s.newID()
		if err != nil {
			return Aggregate{}, err
		}
		brandJob = &VendorBrandJob{
			ID: jobID, TenantID: vendor.TenantID, VendorID: vendor.ID, VendorVersion: vendor.Version,
			JobType: VendorBrandDiscoveryJobType, WebsiteDomain: websiteDomain, State: VendorBrandJobReady,
			AvailableAt: now, CreatedAt: now, UpdatedAt: now, Version: 1,
		}
	}
	record := CreateRecord{
		Vendor: vendor,
		Relationship: Relationship{
			ID: relationshipID, TenantID: strings.TrimSpace(actor.TenantID), LegalEntityID: strings.TrimSpace(actor.LegalEntityID),
			VendorID: vendor.ID, ServiceName: input.ServiceName, BusinessOwnerPrincipalID: strings.TrimSpace(actor.PrincipalID),
			Criticality: input.Criticality, PrivacyRole: input.PrivacyRole, Status: RelationshipProposed,
			EffectiveFrom: input.EffectiveFrom, RenewalAt: input.RenewalAt, SourceID: strings.TrimSpace(input.SourceID),
			ExternalRef: strings.TrimSpace(input.ExternalRef), CreatedAt: now, UpdatedAt: now, Version: 1,
		}, ReuseVendor: reuseVendor, ActorID: strings.TrimSpace(actor.PrincipalID), BrandJob: brandJob,
	}
	return s.repo.CreateRelationship(ctx, record)
}

func (s *Service) UpdateVendorIdentity(ctx context.Context, _ Actor, vendorID string, input UpdateVendorIdentityInput) (Vendor, error) {
	vendorID = strings.TrimSpace(vendorID)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.TradingName = strings.TrimSpace(input.TradingName)
	input.RegistrationRef = strings.TrimSpace(input.RegistrationRef)
	input.Jurisdiction = strings.TrimSpace(input.Jurisdiction)
	websiteDomain, err := normalizeOptionalWebsiteDomain(input.WebsiteDomain)
	if err != nil || vendorID == "" || input.ExpectedVersion < 1 || input.LegalName == "" {
		return Vendor{}, ErrInvalid
	}
	actor, err := s.authorizeVendorIdentity(ctx, vendorID)
	if err != nil {
		return Vendor{}, err
	}
	current, err := s.repo.GetVendor(ctx, scopeFrom(actor), vendorID)
	if err != nil {
		return Vendor{}, err
	}
	if current.Version != input.ExpectedVersion {
		return Vendor{}, ErrVersionConflict
	}
	if current.LegalName == input.LegalName && current.TradingName == input.TradingName && current.RegistrationRef == input.RegistrationRef && current.Jurisdiction == input.Jurisdiction && current.WebsiteDomain == websiteDomain {
		return current, nil
	}
	now := s.now().UTC()
	updated := current
	updated.LegalName = input.LegalName
	updated.TradingName = input.TradingName
	updated.RegistrationRef = input.RegistrationRef
	updated.Jurisdiction = input.Jurisdiction
	updated.WebsiteDomain = websiteDomain
	updated.UpdatedAt = now
	var brandJob *VendorBrandJob
	if current.WebsiteDomain != websiteDomain {
		jobID, idErr := s.newID()
		if idErr != nil {
			return Vendor{}, idErr
		}
		state := VendorBrandJobReady
		if websiteDomain == "" {
			state = VendorBrandJobCancelled
		}
		brandJob = &VendorBrandJob{
			ID: jobID, TenantID: current.TenantID, VendorID: current.ID, VendorVersion: current.Version + 1,
			JobType: VendorBrandDiscoveryJobType, WebsiteDomain: websiteDomain, State: state,
			AvailableAt: now, CreatedAt: now, UpdatedAt: now, Version: 1,
		}
	}
	return s.repo.UpdateVendorIdentity(ctx, UpdateVendorIdentityRecord{
		Scope: scopeFrom(actor), ID: current.ID, ExpectedVersion: input.ExpectedVersion,
		Vendor: updated, ActorID: actor.PrincipalID, BrandJob: brandJob,
	})
}

func (s *Service) authorizeVendorIdentity(ctx context.Context, vendorID string) (Actor, error) {
	contextActor, err := identity.Require(ctx)
	if err != nil {
		return Actor{}, err
	}
	if err := contextActor.Valid(s.now().UTC()); err != nil || contextActor.LegalEntityID == "*" {
		if err != nil {
			return Actor{}, err
		}
		return Actor{}, identity.ErrInvalidIdentity
	}
	if s.identityGuard == nil {
		return Actor{}, errors.Join(ErrVendorIdentityAuthorityUnavailable, commandauth.ErrGuardUnavailable)
	}
	decision, err := s.identityGuard.Authorize(ctx, commandauth.Request{
		TenantID: contextActor.TenantID, LegalEntityID: contextActor.LegalEntityID,
		ObjectType: VendorIdentityObjectType, ObjectID: vendorID,
		Responsibility: authority.ResponsibilityOwner, DecisionType: VendorIdentityUpdateCommand, Materiality: 2,
	})
	if err != nil {
		return Actor{}, err
	}
	if !decision.Allowed {
		return Actor{}, commandauth.ErrNotAuthorized
	}
	if err := decision.Actor.Valid(s.now().UTC()); err != nil || !sameAssessmentIdentity(contextActor, decision.Actor) {
		return Actor{}, ErrVendorIdentityMismatch
	}
	return Actor{TenantID: decision.Actor.TenantID, LegalEntityID: decision.Actor.LegalEntityID, PrincipalID: decision.Actor.PrincipalID}, nil
}

func normalizeOptionalWebsiteDomain(value string) (WebsiteDomain, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return NormalizeWebsiteDomain(value)
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
