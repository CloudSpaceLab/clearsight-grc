package evidence

import (
	"context"
	"encoding/json"
)

func cloneAssessmentSnapshot(v assessmentSnapshot) assessmentSnapshot {
	b, _ := json.Marshal(v)
	var copy assessmentSnapshot
	_ = json.Unmarshal(b, &copy)
	return copy
}
func (s *MemoryDistributionStore) assessmentMaterialLocked(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer) (assessmentMaterial, error) {
	for distributionID, revisions := range s.responseRevisions {
		d := s.distributions[distributionID]
		if d.TenantID != tenant || d.LegalEntityID != entity {
			continue
		}
		for _, revision := range revisions {
			if revision.ID != response {
				continue
			}
			if revision.State != ResponseRevisionFinal && revision.State != ResponseRevisionProvisional {
				return assessmentMaterial{}, ErrNotFound
			}
			allowed, err := s.completedResponseVisible(ctx, principal, d, revision)
			if err != nil {
				return assessmentMaterial{}, err
			}

			submission, err := s.repo.GetSubmission(ctx, tenant, revision.SubmissionID)
			if err != nil {
				return assessmentMaterial{}, err
			}
			request, err := s.repo.GetRequest(ctx, tenant, submission.RequestID)
			if err != nil {
				return assessmentMaterial{}, err
			}
			if request.LegalEntityID != entity || request.FormTemplateID != d.FormTemplateID || request.FormTemplateVersion != d.FormTemplateVersion || request.SubjectType != d.SubjectType || request.SubjectID != d.SubjectID {
				return assessmentMaterial{}, ErrNotFound
			}
			m := assessmentMaterial{Summary: completedResponseSummary(d, revision), Revision: cloneResponseRevision(revision), Request: request, Submission: submission}
			if d.Status == DistributionRevoked || d.Status == DistributionSuperseded {
				m.Revision.Current = false
				m.Summary.Current = false
			}
			if !allowed && (d.SubjectType != "VENDOR_RELATIONSHIP" || request.Origin.Type == "THIRD_PARTY_WORK" || request.Origin.Type == "THIRD_PARTY_ASSESSMENT" || authorize == nil || !authorize(ctx, m)) {
				return assessmentMaterial{}, ErrNotFound
			}
			versions := s.assessments[response]
			if len(versions) > 0 {
				m.Snapshot = cloneAssessmentSnapshot(versions[len(versions)-1])
			}
			return m, nil
		}
	}
	return assessmentMaterial{}, ErrNotFound
}
func (s *MemoryDistributionStore) ReadResponseAssessment(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer) (assessmentMaterial, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.assessmentMaterialLocked(ctx, tenant, entity, principal, response, authorize)
}
func (s *MemoryDistributionStore) WriteResponseAssessment(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer, apply func(context.Context, assessmentMaterial) (assessmentSnapshot, []FieldAssessmentDecision, error)) (assessmentMaterial, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.assessmentMaterialLocked(ctx, tenant, entity, principal, response, authorize)
	if err != nil {
		return assessmentMaterial{}, err
	}
	next, decisions, err := apply(ctx, m)
	if err != nil {
		return assessmentMaterial{}, err
	}
	m.Snapshot = cloneAssessmentSnapshot(next)
	event := assessmentEvent(m, principal, s.now().UTC())
	if s.assessments == nil {
		s.assessments = map[string][]assessmentSnapshot{}
	}
	s.assessments[response] = append(s.assessments[response], cloneAssessmentSnapshot(next))
	s.assessmentDecisions = append(s.assessmentDecisions, decisions...)
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return m, nil
}
func (s *MemoryDistributionStore) ReadAssessmentSummary(_ context.Context, tenant, response string) (*ResponseAssessmentSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, revisions := range s.responseRevisions {
		for _, r := range revisions {
			if r.ID == response && r.TenantID == tenant {
				v := s.assessments[response]
				if len(v) == 0 {
					return nil, nil
				}
				copy := cloneAssessmentSnapshot(v[len(v)-1])
				return &copy.Result, nil
			}
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryDistributionStore) WithAssessedResponseForExecution(ctx context.Context, tenant, response string, version int64, apply func() error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for distributionID, revisions := range s.responseRevisions {
		for _, r := range revisions {
			if r.ID != response || r.TenantID != tenant {
				continue
			}
			versions := s.assessments[response]
			d := s.distributions[distributionID]
			if !r.Current || d.Status == DistributionRevoked || d.Status == DistributionSuperseded || len(versions) == 0 {
				return ErrAssessmentConflict
			}
			latest := versions[len(versions)-1]
			if latest.Version != version || latest.Result.State != "ASSESSED" || latest.Result.Score == nil || !latest.Result.Score.Final || latest.Result.Score.State != ResponseScoreFinal {
				return ErrAssessmentConflict
			}
			return apply()
		}
	}
	return ErrAssessmentConflict
}
