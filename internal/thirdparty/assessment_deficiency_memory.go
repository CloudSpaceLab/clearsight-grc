package thirdparty

import "context"

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
	link := AssessmentMatterLink{Scope: record.Scope, AssessmentID: current.ID, MatterID: record.MatterID, Kind: AssessmentMatterDeficiency, CreatedAt: record.LinkedAt.UTC()}
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

func assessmentDeficiencyRecorded(events []memoryAssessmentAudit, assessmentID string, version int64, matterID string) bool {
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.AssessmentID == assessmentID && event.AssessmentVersion == version {
			return event.Type == "AssessmentDeficiencyLinked" && event.Payload["deficiency_matter_id"] == matterID
		}
	}
	return false
}
