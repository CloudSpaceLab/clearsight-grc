package thirdparty

import (
	"strings"
	"time"
)

// ValidateProgramVendorExpectationLinks verifies the complete, scoped Program
// relationship population supplied by the canonical relationship-link reader.
// A vendor's assessment is not eligible merely because it shares the Program's
// tenant or legal entity. The caller must verify link-read completeness first.
func ValidateProgramVendorExpectationLinks(
	tenantID, legalEntityID, programID string,
	expected []ProgramVendorAssessmentExpectation,
	links []RelationshipLink,
	linksComplete bool,
	asOf time.Time,
) error {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(legalEntityID) == "" ||
		strings.TrimSpace(programID) == "" || asOf.IsZero() {
		return ErrInvalid
	}
	if len(expected) == 0 {
		return nil
	}
	if !linksComplete || len(expected) > 10000 || len(links) > 10000 {
		return ErrInvalid
	}
	linked := make(map[string]struct{}, len(links))
	for _, link := range links {
		if link.TenantID != tenantID || link.LegalEntityID != legalEntityID ||
			(link.State != RelationshipLinkActive && link.State != RelationshipLinkEnded) {
			return ErrInvalid
		}
		if link.TargetType != LinkTargetProgram || link.TargetID != programID ||
			link.CreatedAt.After(asOf) || link.EndedAt != nil && !asOf.Before(*link.EndedAt) {
			continue
		}
		linked[link.RelationshipID] = struct{}{}
	}
	for _, obligation := range expected {
		if obligation.RelationshipID == "" {
			return ErrInvalid
		}
		if _, exists := linked[obligation.RelationshipID]; !exists {
			return ErrNotFound // No eligible scoped relationship; never score another Program's vendor.
		}
	}
	return nil
}
