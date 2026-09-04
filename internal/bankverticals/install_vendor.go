package bankverticals

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

const (
	referenceVendorSourceID    = "reference_data"
	referenceVendorExternalRef = "vendor:managed-infrastructure"
	referenceVendorLegalName   = "Northstar Infrastructure Services Limited"
	referenceVendorTradingName = "Northstar Infrastructure"
	referenceVendorServiceName = "Managed infrastructure and recovery services"
)

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
