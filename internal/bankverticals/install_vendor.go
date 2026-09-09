package bankverticals

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

const (
	referenceVendorSourceID    = "reference_data"
	referenceVendorExternalRef = "vendor:managed-infrastructure"
	referenceVendorLegalName   = "Northstar Infrastructure Services Limited"
	referenceVendorTradingName = "Northstar Infrastructure"
	referenceVendorServiceName = "Managed infrastructure and recovery services"
)

type operatingVendorSpec struct {
	externalRef, legalName, tradingName, registrationRef, serviceName string
	criticality                                                       thirdparty.Criticality
	privacyRole                                                       thirdparty.PrivacyRole
}

// EnsureOperatingVendors adds fictional suppliers that represent common bank
// services. Managed source identifiers make reruns safe and preserve later edits.
func (s *Service) EnsureOperatingVendors(ctx context.Context, config SeedConfig, vendors *thirdparty.Service) ([]thirdparty.Aggregate, error) {
	first, err := s.EnsureReferenceVendor(ctx, config, vendors)
	if err != nil {
		return nil, err
	}
	result := []thirdparty.Aggregate{first}
	specs := []operatingVendorSpec{
		{externalRef: "vendor:payment-switching", legalName: "Paywave Transaction Services Limited", tradingName: "Paywave Transactions", registrationRef: "REF-NG-TP-002", serviceName: "Payment switching and terminal support", criticality: thirdparty.CriticalityCritical, privacyRole: thirdparty.PrivacyProcessor},
		{externalRef: "vendor:records-custody", legalName: "ArchiveGuard Records Limited", tradingName: "ArchiveGuard", registrationRef: "REF-NG-TP-003", serviceName: "Secure records storage and destruction", criticality: thirdparty.CriticalityImportant, privacyRole: thirdparty.PrivacyProcessor},
		{externalRef: "vendor:payroll-processing", legalName: "PeopleLink Payroll Services Limited", tradingName: "PeopleLink Payroll", registrationRef: "REF-NG-TP-004", serviceName: "Payroll processing", criticality: thirdparty.CriticalityImportant, privacyRole: thirdparty.PrivacyProcessor},
		{externalRef: "vendor:collections-platform", legalName: "Sentinel Collections Technology Limited", tradingName: "Sentinel Collections", registrationRef: "REF-NG-TP-005", serviceName: "Loan collections platform", criticality: thirdparty.CriticalityImportant, privacyRole: thirdparty.PrivacyProcessor},
	}
	for _, spec := range specs {
		item, ensureErr := s.ensureOperatingVendor(ctx, config, vendors, spec)
		if ensureErr != nil {
			return nil, ensureErr
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Service) ensureOperatingVendor(ctx context.Context, config SeedConfig, vendors *thirdparty.Service, spec operatingVendorSpec) (thirdparty.Aggregate, error) {
	actor := thirdparty.Actor{TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, PrincipalID: config.OwnerPrincipalID}
	page, err := vendors.ListRelationships(ctx, actor, thirdparty.ListInput{Search: spec.externalRef, Limit: 100})
	if err != nil {
		return thirdparty.Aggregate{}, fmt.Errorf("list sample vendor %s: %w", spec.externalRef, err)
	}
	for _, item := range page.Items {
		if strings.EqualFold(strings.TrimSpace(item.Relationship.SourceID), referenceVendorSourceID) && strings.EqualFold(strings.TrimSpace(item.Relationship.ExternalRef), spec.externalRef) {
			if !strings.EqualFold(strings.TrimSpace(item.Vendor.SourceID), referenceVendorSourceID) || !strings.EqualFold(strings.TrimSpace(item.Vendor.ExternalRef), spec.externalRef) {
				return thirdparty.Aggregate{}, fmt.Errorf("sample vendor %s is bound to a different vendor identity", spec.externalRef)
			}
			return item, nil
		}
	}
	created, err := vendors.CreateRelationship(ctx, actor, thirdparty.CreateRelationshipInput{
		LegalName: spec.legalName, TradingName: spec.tradingName, RegistrationRef: spec.registrationRef, Jurisdiction: "Nigeria",
		SourceID: referenceVendorSourceID, ExternalRef: spec.externalRef, RegisteredAddress: "Sample data — Lagos, Nigeria",
		ServiceName: spec.serviceName, Criticality: spec.criticality, PrivacyRole: spec.privacyRole,
	})
	if err != nil {
		return thirdparty.Aggregate{}, fmt.Errorf("create sample vendor %s: %w", spec.externalRef, err)
	}
	return created, nil
}

// EnsureReferenceVendor installs the persisted third-party record used by
// reference/demo journeys through the canonical third-party service. The
// source identity is the provenance and idempotency key: reruns reuse the
// existing relationship and deliberately preserve later governed edits.
func (s *Service) EnsureReferenceVendor(ctx context.Context, config SeedConfig, vendors *thirdparty.Service) (thirdparty.Aggregate, error) {
	if s == nil || vendors == nil {
		return thirdparty.Aggregate{}, fmt.Errorf("reference vendor installer is unavailable")
	}
	config = normalizeSeedConfig(config)
	if err := validateSeedConfig(config); err != nil {
		return thirdparty.Aggregate{}, err
	}
	if s.continuity != nil {
		entityCtx := continuity.WithTrustedSystemEntityScope(ctx, config.TenantID, config.LegalEntityID)
		canonicalEntityID, err := s.continuity.ResolveLegalEntity(entityCtx, config.TenantID, config.LegalEntityID)
		if err != nil {
			return thirdparty.Aggregate{}, fmt.Errorf("resolve reference vendor legal entity: %w", err)
		}
		config.LegalEntityID = canonicalEntityID
	}

	actor := thirdparty.Actor{
		TenantID:      config.TenantID,
		LegalEntityID: config.LegalEntityID,
		PrincipalID:   config.OwnerPrincipalID,
	}
	page, err := vendors.ListRelationships(ctx, actor, thirdparty.ListInput{Search: referenceVendorExternalRef, Limit: 100})
	if err != nil {
		return thirdparty.Aggregate{}, fmt.Errorf("list reference vendor relationships: %w", err)
	}
	matches := make([]thirdparty.Aggregate, 0, 1)
	for _, item := range page.Items {
		if strings.EqualFold(strings.TrimSpace(item.Relationship.SourceID), referenceVendorSourceID) &&
			strings.EqualFold(strings.TrimSpace(item.Relationship.ExternalRef), referenceVendorExternalRef) {
			matches = append(matches, item)
		}
	}
	if len(matches) > 1 {
		return thirdparty.Aggregate{}, fmt.Errorf("reference vendor source identity is duplicated")
	}
	if len(matches) == 1 {
		if !strings.EqualFold(strings.TrimSpace(matches[0].Vendor.SourceID), referenceVendorSourceID) ||
			!strings.EqualFold(strings.TrimSpace(matches[0].Vendor.ExternalRef), referenceVendorExternalRef) {
			return thirdparty.Aggregate{}, fmt.Errorf("reference vendor relationship is bound to a non-reference vendor identity")
		}
		return matches[0], nil
	}

	created, err := vendors.CreateRelationship(ctx, actor, thirdparty.CreateRelationshipInput{
		LegalName:         referenceVendorLegalName,
		TradingName:       referenceVendorTradingName,
		RegistrationRef:   "REF-NG-TP-001",
		Jurisdiction:      "Nigeria",
		SourceID:          referenceVendorSourceID,
		ExternalRef:       referenceVendorExternalRef,
		RegisteredAddress: "Reference data — Lagos, Nigeria",
		ServiceName:       referenceVendorServiceName,
		Criticality:       thirdparty.CriticalityImportant,
		PrivacyRole:       thirdparty.PrivacyProcessor,
	})
	if err != nil {
		return thirdparty.Aggregate{}, fmt.Errorf("create reference vendor relationship: %w", err)
	}
	if !strings.EqualFold(created.Vendor.SourceID, referenceVendorSourceID) ||
		!strings.EqualFold(created.Vendor.ExternalRef, referenceVendorExternalRef) ||
		!strings.EqualFold(created.Relationship.SourceID, referenceVendorSourceID) ||
		!strings.EqualFold(created.Relationship.ExternalRef, referenceVendorExternalRef) {
		return thirdparty.Aggregate{}, fmt.Errorf("reference vendor source identity was not preserved")
	}
	return created, nil
}
