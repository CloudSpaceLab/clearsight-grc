package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// ResponseDiscoveryAuthorizer checks the current response Reviewer route. Field
// reviewer roles remain a separate write permission enforced by assessment.
type ResponseDiscoveryAuthorizer func(context.Context, identity.Actor, CompletedResponseSummary) error

func (s *DistributionService) WithResponseDiscoveryAuthorizer(a ResponseDiscoveryAuthorizer) *DistributionService {
	s.responseDiscoveryAuthorizer = a
	return s
}

type responseDiscoveryContextKey struct{}
type responseDiscoveryCheck func(CompletedResponseSummary) bool

func (s *DistributionService) withResponseDiscovery(ctx context.Context, tenant, entity, principal string) context.Context {
	check := responseDiscoveryCheck(func(summary CompletedResponseSummary) bool {
		actor, ok := identity.FromContext(ctx)
		if !ok || actor.Valid(s.currentTime()) != nil || actor.TenantID != tenant || actor.PrincipalID != principal || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") || summary.TenantID != tenant || summary.LegalEntityID != entity || s.responseDiscoveryAuthorizer == nil {
			return false
		}
		return s.responseDiscoveryAuthorizer(ctx, actor, summary) == nil
	})
	return context.WithValue(ctx, responseDiscoveryContextKey{}, check)
}
func (s *MemoryDistributionStore) responseDiscoveryVisible(ctx context.Context, principal string, d FormDistribution, revision ResponseRevision) bool {
	if d.SubjectType != "VENDOR_RELATIONSHIP" || revision.ID == "" || (revision.State != ResponseRevisionFinal && revision.State != ResponseRevisionProvisional) || revision.TenantID != d.TenantID || revision.LegalEntityID != d.LegalEntityID || revision.SubmissionID == "" || s.repo == nil {
		return false
	}
	sub, err := s.repo.GetSubmission(ctx, d.TenantID, revision.SubmissionID)
	if err != nil || sub.SubmittedBy == principal {
		return false
	}
	req, err := s.repo.GetRequest(ctx, d.TenantID, sub.RequestID)
	if err != nil || req.Origin.Type == "THIRD_PARTY_WORK" || req.Origin.Type == "THIRD_PARTY_ASSESSMENT" || req.LegalEntityID != d.LegalEntityID || s.requestDistribution[req.ID] != d.ID || req.SubjectType != d.SubjectType || req.SubjectID != d.SubjectID || req.FormTemplateID != d.FormTemplateID || req.FormTemplateVersion != d.FormTemplateVersion {
		return false
	}
	reviewable := false
	for _, f := range req.Fields {
		if f.Assessment.NeedsReview() {
			reviewable = true
			break
		}
	}
	if !reviewable {
		return false
	}
	check, ok := ctx.Value(responseDiscoveryContextKey{}).(responseDiscoveryCheck)
	return ok && check(completedResponseSummary(d, revision))
}
