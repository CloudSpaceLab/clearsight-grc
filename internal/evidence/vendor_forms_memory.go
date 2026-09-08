package evidence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// ConfigureVendorProgress shares the existing capture draft projection. Only
// field counts and missing labels leave the vendor read boundary.
func (s *DistributionService) ConfigureVendorProgress(access *MemoryDistributionAccessStore) {
	store, ok := s.store.(*MemoryDistributionStore)
	if !ok || access == nil {
		return
	}
	store.vendorProgress = func(distributionID string) (map[string]formcontract.AnswerValue, bool, time.Time) {
		access.workspaceMu.Lock()
		state := access.workspaceStates[distributionID]
		access.workspaceMu.Unlock()
		if state == nil {
			return nil, false, time.Time{}
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		return cloneAnswerValues(state.answers), true, state.workspace.UpdatedAt
	}
}

func (s *MemoryDistributionStore) vendorRows(ctx context.Context, q VendorFormsQuery) ([]VendorFormRow, error) {
	if err := normalizeVendorFormsQuery(&q); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, ErrNotFound
	}
	wanted := map[string]bool{}
	for _, id := range q.RelationshipIDs {
		wanted[id] = true
	}
	s.repo.mu.RLock()
	requests := []Request{}
	for _, r := range s.repo.requests {
		if r.TenantID == q.TenantID && r.LegalEntityID == q.LegalEntityID && r.SubjectType == "VENDOR_RELATIONSHIP" && wanted[r.SubjectID] && r.FormTemplateID != "" {
			requests = append(requests, r)
		}
	}
	latestSubmissions := map[string]Submission{}
	for _, sub := range s.repo.submissions {
		if sub.TenantID == q.TenantID {
			prior := latestSubmissions[sub.RequestID]
			if prior.ID == "" || sub.SubmittedAt.After(prior.SubmittedAt) || sub.SubmittedAt.Equal(prior.SubmittedAt) && sub.ID > prior.ID {
				latestSubmissions[sub.RequestID] = sub
			}
		}
	}
	s.repo.mu.RUnlock()
	requestSources := map[string]vendorRequestSource{}
	latestFields := map[string]documentFieldSource{}
	rows := []VendorFormRow{}
	now := s.now().UTC()
	for _, req := range requests {
		var dc DocumentContext
		var contextErr error
		sub := latestSubmissions[req.ID]
		ordinaryRead := true
		if req.Origin.Type == "THIRD_PARTY_WORK" || req.Origin.Type == "THIRD_PARTY_ASSESSMENT" {
			if s.documentContexts == nil {
				continue
			}
			dc, contextErr = s.documentContexts.ResolveDocumentContext(ctx, DocumentQuery{TenantID: q.TenantID, LegalEntityID: q.LegalEntityID, PrincipalID: q.PrincipalID}, req, Submission{TenantID: q.TenantID, RequestID: req.ID})
			err := contextErr
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
		} else {
			allowed, err := s.completedResponseSubjectVisible(ctx, q.TenantID, q.PrincipalID, req.SubjectType, req.SubjectID)
			if err != nil {
				return nil, err
			}
			ordinaryRead = allowed
		}
		s.mu.RLock()
		distributionID := s.requestDistribution[req.ID]
		dist := s.distributions[distributionID]
		var revision *ResponseRevision
		for _, r := range s.responseRevisions[distributionID] {
			if r.Current {
				s.repo.mu.RLock()
				sub, exists := s.repo.submissions[r.SubmissionID]
				s.repo.mu.RUnlock()
				if !exists || sub.TenantID != q.TenantID || sub.RequestID != req.ID {
					continue
				}
				v := cloneResponseRevision(r)
				revision = &v
				break
			}
		}
		if !ordinaryRead && (revision == nil || !s.responseDiscoveryVisible(ctx, q.PrincipalID, dist, *revision)) {
			s.mu.RUnlock()
			continue
		}
		delivery := ""
		for _, r := range s.recipients[distributionID] {
			if r.safe.RequestID == req.ID {
				delivery = string(r.safe.State)
				break
			}
		}
		s.mu.RUnlock()
		answers := map[string]formcontract.AnswerValue{}
		known := req.Status == RequestReady || req.Status == RequestDraft
		observed := req.UpdatedAt
		if revision != nil {
			s.repo.mu.RLock()
			sub, found := s.repo.submissions[revision.SubmissionID]
			s.repo.mu.RUnlock()
			if found && sub.TenantID == q.TenantID {
				answers = cloneAnswerValues(sub.Answers)
				known = true
			}
		}
		if revision == nil && distributionID != "" && s.vendorProgress != nil {
			if a, ok, at := s.vendorProgress(distributionID); ok {
				answers = a
				known = true
				observed = at
			}
		}
		if revision == nil && !known {
			s.repo.mu.RLock()
			var draft ResponseDraft
			for _, d := range s.repo.drafts {
				if d.TenantID == q.TenantID && d.RequestID == req.ID && d.UpdatedAt.After(draft.UpdatedAt) {
					draft = d
				}
			}
			s.repo.mu.RUnlock()
			if draft.ID != "" {
				answers = cloneAnswerValues(draft.Answers)
				known = true
				observed = draft.UpdatedAt
			}
		}
		row := vendorFormRow(req, answers, known, revision, now)
		row.ResponseCurrency = "CURRENT"
		if sub.ID != "" && (req.Origin.Type == "THIRD_PARTY_WORK" || req.Origin.Type == "THIRD_PARTY_ASSESSMENT") {
			if revision != nil && (dist.TenantID != req.TenantID || dist.LegalEntityID != req.LegalEntityID || dist.SubjectType != req.SubjectType || dist.SubjectID != req.SubjectID || dist.FormTemplateID != req.FormTemplateID || dist.FormTemplateVersion != req.FormTemplateVersion) {
				continue
			}
			source := documentFieldSource{SubmissionID: sub.ID, Sequence: req.Origin.Version, SubmittedAt: sub.SubmittedAt}
			if revision != nil {
				source.Revision = revision.Revision
			}
			key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d|%s|%s", req.TenantID, req.LegalEntityID, req.Origin.Type, req.Origin.ID, req.SubjectType, req.SubjectID, req.FormTemplateID, req.FormTemplateVersion, dc.WorkRequestID, dc.AssessmentID)
			requestSources[req.ID] = vendorRequestSource{Request: req, Source: source, Key: key}
			for _, field := range req.Fields {
				fieldKey := key + "|" + field.ID
				if prior, ok := latestFields[fieldKey]; !ok || source.after(prior) {
					latestFields[fieldKey] = source
				}
			}
		}

		if revision != nil {
			row.AssessmentState = "NOT_REQUIRED"
			if revision.Score != nil {
				row.RequiredReviews = revision.Score.AssessmentRequiredCount
				if revision.Score.AssessmentReviewCount > 0 {
					row.AssessmentState = "AWAITING_REVIEW"
				}
			}
			s.mu.RLock()
			snapshots := s.assessments[revision.ID]
			if len(snapshots) > 0 {
				latest := cloneAssessmentSnapshot(snapshots[len(snapshots)-1])
				row.AssessmentState = latest.Result.State
				row.AssessedScore = latest.Result.Score
				row.RequiredReviews = latest.Result.RequiredCount
				row.CompletedReviews = latest.Result.ReviewedRequiredCount
			}
			s.mu.RUnlock()
		}
		row.DistributionID = distributionID
		row.DeliveryState = delivery
		if !observed.IsZero() {
			row.UpdatedAt = observed
		}
		if dist.Status == DistributionRevoked || dist.Status == DistributionSuperseded {
			row.ResponseState = string(dist.Status)
			row.Current = false
		}
		rows = append(rows, row)
	}
	filtered := rows[:0]
	for _, row := range rows {
		if !row.Current {
			row.ResponseCurrency = "HISTORICAL"
		} else if source, ok := requestSources[row.RequestID]; ok {
			replaced := 0
			for _, field := range source.Request.Fields {
				if latestFields[source.Key+"|"+field.ID].after(source.Source) {
					replaced++
				}
			}
			if replaced > 0 {
				row.ResponseCurrency = "PARTIALLY_REPLACED"
				if replaced == len(source.Request.Fields) {
					row.ResponseCurrency = "HISTORICAL"
					row.Current = false
				}
			}
		}
		if vendorFormMatches(row, q, now) {
			filtered = append(filtered, row)
		}
	}
	rows = filtered
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].RequestID > rows[j].RequestID
		}
		return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
	})
	return rows, nil
}
func (s *MemoryDistributionStore) ListVendorForms(ctx context.Context, q VendorFormsQuery) (VendorFormsPage, error) {
	rows, err := s.vendorRows(ctx, q)
	if err != nil {
		return VendorFormsPage{}, err
	}
	c := decodeVendorFormsCursor(q.Cursor)
	visible := []VendorFormRow{}
	for _, row := range rows {
		if vendorFormsAfter(row, c) {
			visible = append(visible, row)
		}
	}
	return vendorFormsPage(visible, q.Limit, s.now().UTC()), nil
}
func (s *MemoryDistributionStore) VendorFormSummaries(ctx context.Context, q VendorFormsQuery) ([]VendorFormSummary, error) {
	rows, err := s.vendorRows(ctx, q)
	if err != nil {
		return nil, err
	}
	values := []VendorFormSummary{}
	for _, id := range q.RelationshipIDs {
		values = append(values, summarizeVendorForms(id, rows, s.now().UTC()))
	}
	return values, nil
}

type vendorRequestSource struct {
	Request Request
	Source  documentFieldSource
	Key     string
}
