package thirdparty

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (r *MemoryAssessmentRepository) LinkAssessmentDeficiency(_ context.Context, record LinkAssessmentDeficiencyRecord) (AssessmentMatterLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID, record.MatterID, record.MatterTriggerKey) || record.ExpectedVersion < 1 || record.LinkedAt.IsZero() {
		return AssessmentMatterLink{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentMatterLink{}, Assessment{}, ErrNotFound
	}
	for _, link := range r.matterLinks[current.ID] {
		if link.MatterID == record.MatterID {
			if current.Version == record.ExpectedVersion+1 && assessmentDeficiencyRecorded(r.assessmentEvents, current.ID, current.Version, record.MatterID) {
				return link, current, nil
			}
			return AssessmentMatterLink{}, Assessment{}, ErrVersionConflict
		}
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentMatterLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentUnderReview {
		return AssessmentMatterLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	canonical, err := r.ensureMemoryAssessmentMatterRelationshipLink(current, record.MatterID, AssessmentMatterDeficiency, record.ActorPrincipalID, record.LinkedAt)
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	link := AssessmentMatterLink{Scope: record.Scope, AssessmentID: current.ID, MatterID: record.MatterID, RelationshipLinkID: canonical.ID, Kind: AssessmentMatterDeficiency, CreatedAt: record.LinkedAt.UTC()}
	r.matterLinks[current.ID] = append(r.matterLinks[current.ID], link)
	current.Version++
	current.UpdatedAt = record.LinkedAt.UTC()
	r.assessments[current.ID] = current
	r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentDeficiencyLinked")
	payload := r.assessmentEvents[len(r.assessmentEvents)-1].Payload
	payload["deficiency_matter_id"] = record.MatterID
	payload["deficiency_trigger_key"] = record.MatterTriggerKey
	return link, current, nil
}

func (r *MemoryAssessmentRepository) ensureMemoryAssessmentMatterRelationshipLink(assessment Assessment, matterID string, kind AssessmentMatterLinkKind, actorID string, at time.Time) (RelationshipLink, error) {
	r.relationshipLinkRepo.mu.Lock()
	defer r.relationshipLinkRepo.mu.Unlock()
	for _, existing := range r.relationshipLinkRepo.links {
		if existing.TenantID == assessment.TenantID && existing.LegalEntityID == assessment.LegalEntityID && existing.RelationshipID == assessment.RelationshipID && existing.TargetType == LinkTargetMatter && existing.TargetID == matterID && existing.State == RelationshipLinkActive {
			return existing, nil
		}
	}
	linkID, err := id.NewUUIDv7()
	if err != nil {
		return RelationshipLink{}, err
	}
	purposeCode, purposeLabel := assessmentMatterRelationshipPurpose(kind)
	value := RelationshipLink{ID: linkID, TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, RelationshipID: assessment.RelationshipID, TargetType: LinkTargetMatter, TargetID: matterID, PurposeCode: purposeCode, PurposeLabel: purposeLabel, State: RelationshipLinkActive, CreatedBy: actorID, Version: 1, CreatedAt: at.UTC(), UpdatedAt: at.UTC()}
	r.relationshipLinkRepo.links[value.ID] = value
	return value, nil
}

func assessmentDeficiencyRecorded(events []memoryAssessmentAudit, assessmentID string, version int64, matterID string) bool {
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.AssessmentID == assessmentID && event.AssessmentVersion == version {
			return event.Type == "AssessmentDeficiencyLinked" && event.Payload["deficiency_matter_id"] == matterID
		}
	}
	return false
}
