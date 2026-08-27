package bankverticals

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type referenceEvidenceRepository struct {
	*evidence.MemoryRepository
	tenant        string
	legalEntityID string
}

func (r *referenceEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	if tenant != r.tenant || (subjectType != "PROGRAM" && subjectType != "MATTER") || strings.TrimSpace(subjectID) == "" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: r.legalEntityID, SubjectType: subjectType, SubjectID: subjectID}, nil
}

func newReferenceEvidenceService(now time.Time, legalEntityID string) *evidence.Service {
	repository := &referenceEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil), tenant: "bank-demo", legalEntityID: legalEntityID}
	return evidence.NewServiceWithClock(repository, evidence.NewMemoryObjectStore(), func() time.Time { return now })
}
