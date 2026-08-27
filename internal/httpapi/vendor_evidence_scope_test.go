package httpapi

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

// scopedVendorEvidenceRepository keeps memory-backed HTTP tests faithful to
// production request creation, where a vendor relationship is resolved to its
// authoritative legal-entity scope before any capture request is persisted.
type scopedVendorEvidenceRepository struct {
	*evidence.MemoryRepository
	tenant          string
	legalEntityID   string
	relationshipIDs map[string]struct{}
}

func newScopedVendorEvidenceRepository(tenant, legalEntityID string, relationshipIDs ...string) *scopedVendorEvidenceRepository {
	allowed := make(map[string]struct{}, len(relationshipIDs))
	for _, relationshipID := range relationshipIDs {
		if relationshipID = strings.TrimSpace(relationshipID); relationshipID != "" {
			allowed[relationshipID] = struct{}{}
		}
	}
	return &scopedVendorEvidenceRepository{
		MemoryRepository: evidence.NewMemoryRepository(nil, nil), tenant: tenant, legalEntityID: legalEntityID, relationshipIDs: allowed,
	}
}

func (r *scopedVendorEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	if tenant != r.tenant || subjectType != "VENDOR_RELATIONSHIP" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	if _, ok := r.relationshipIDs[strings.TrimSpace(subjectID)]; !ok {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: r.legalEntityID, SubjectType: subjectType, SubjectID: subjectID}, nil
}
