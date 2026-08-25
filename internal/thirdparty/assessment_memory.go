package thirdparty

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryAssessmentRepository struct {
	*MemoryRepository
	assessmentMu sync.RWMutex
	assessments  map[string]Assessment
	episodes     map[string]string
	requestLinks map[string][]AssessmentRequestLink
	reactions    map[string]assessmentReactionReceipt
}

type assessmentReactionReceipt struct {
	record     AssessmentReactionRecord
	assessment Assessment
}

func NewMemoryAssessmentRepository() *MemoryAssessmentRepository {
	return &MemoryAssessmentRepository{
		MemoryRepository: NewMemoryRepository(),
		assessments:      map[string]Assessment{},
		episodes:         map[string]string{},
		requestLinks:     map[string][]AssessmentRequestLink{},
		reactions:        map[string]assessmentReactionReceipt{},
	}
}

func (r *MemoryAssessmentRepository) CreateAssessment(ctx context.Context, record CreateAssessmentRecord) (Assessment, error) {
	current, err := r.GetRelationship(ctx, record.Scope, record.RelationshipID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Relationship.Version != record.RelationshipVersion {
		return Assessment{}, ErrVersionConflict
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	episodeIndex := assessmentEpisodeIndex(record.TenantID, record.LegalEntityID, record.Assessment.StableEpisodeKey)
	if assessmentID, ok := r.episodes[episodeIndex]; ok {
		return r.assessments[assessmentID], nil
	}
	assessment := record.Assessment
	assessment.TenantID = record.TenantID
	assessment.LegalEntityID = record.LegalEntityID
	assessment.RelationshipID = record.RelationshipID
	r.assessments[assessment.ID] = assessment
	r.episodes[episodeIndex] = assessment.ID
	return assessment, nil
}

func (r *MemoryAssessmentRepository) GetAssessment(_ context.Context, scope Scope, assessmentID string) (Assessment, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return Assessment{}, ErrNotFound
	}
	return assessment, nil
}

func (r *MemoryAssessmentRepository) GetCurrentAssessment(_ context.Context, scope Scope, relationshipID string, kind AssessmentReviewKind) (Assessment, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	var current Assessment
	found := false
	for _, assessment := range r.assessments {
		if assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID || assessment.RelationshipID != relationshipID || assessment.ReviewKind != kind {
			continue
		}
		if !found || assessment.UpdatedAt.After(current.UpdatedAt) || (assessment.UpdatedAt.Equal(current.UpdatedAt) && assessment.ID > current.ID) {
			current, found = assessment, true
		}
	}
	if !found {
		return Assessment{}, ErrNotFound
	}
	return current, nil
}

func (r *MemoryAssessmentRepository) ListAssessments(_ context.Context, filter AssessmentListFilter) (AssessmentPage, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 50
	}
	var cursorTime time.Time
	var cursorID string
	if filter.Cursor != "" {
		var err error
		cursorTime, cursorID, err = decodeCursor(filter.Cursor)
		if err != nil {
			return AssessmentPage{}, ErrInvalid
		}
	}
	items := make([]Assessment, 0, len(r.assessments))
	for _, assessment := range r.assessments {
		if assessment.TenantID != filter.TenantID || assessment.LegalEntityID != filter.LegalEntityID || (filter.Status != "" && assessment.Status != filter.Status) {
			continue
		}
		if filter.Cursor != "" && (assessment.UpdatedAt.After(cursorTime) || (assessment.UpdatedAt.Equal(cursorTime) && assessment.ID >= cursorID)) {
			continue
		}
		items = append(items, assessment)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	page := AssessmentPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (r *MemoryAssessmentRepository) TransitionAssessment(_ context.Context, record AssessmentTransitionRecord) (Assessment, error) {
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.ID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if !containsAssessmentStatus(record.From, current.Status) {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	switch record.To {
	case AssessmentUnderReview:
		if current.Status != AssessmentSubmitted || !validAssessmentIdentifier(record.ActorPrincipalID) {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
	case AssessmentCompleted:
		if current.Status != AssessmentUnderReview || !validAssessmentIdentifier(record.ActorPrincipalID) || !validAssessmentConclusion(record.Conclusion) || strings.TrimSpace(record.ConclusionRationale) == "" {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
	case AssessmentCancelled:
		if current.Status == AssessmentCompleted || current.Status == AssessmentCancelled || strings.TrimSpace(record.CancellationReason) == "" {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
	default:
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	current.Status = record.To
	current.UpdatedAt = record.At.UTC()
	current.Version++
	switch record.To {
	case AssessmentUnderReview:
		at := record.At.UTC()
		current.ReviewStartedAt = &at
		current.ReviewerPrincipalID = record.ActorPrincipalID
	case AssessmentCompleted:
		at := record.At.UTC()
		current.CompletedAt = &at
		current.ReviewerPrincipalID = record.ActorPrincipalID
		current.Conclusion = record.Conclusion
		current.ConclusionUncertainty = record.ConclusionUncertainty
		current.ConclusionRationale = record.ConclusionRationale
		current.NextReviewRecommendedAt = cloneAssessmentTime(record.NextReviewRecommendedAt)
	case AssessmentCancelled:
		current.CancellationReason = record.CancellationReason
	}
	r.assessments[current.ID] = current
	return current, nil
}

func (r *MemoryAssessmentRepository) ApplyAssessmentReaction(_ context.Context, record AssessmentReactionRecord) (Assessment, error) {
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	key := assessmentReactionIndex(record)
	if receipt, ok := r.reactions[key]; ok {
		if !sameAssessmentReaction(receipt.record, record) {
			return Assessment{}, ErrInvalid
		}
		return receipt.assessment, nil
	}
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	switch record.Kind {
	case AssessmentReactionSetupCompleted:
		if current.Status != AssessmentSetupPending || !validAssessmentIdentifiers(record.CausationID, record.JobID, record.MatterID) {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		current.Status = AssessmentReadyToSend
		current.ReviewMatterID = record.MatterID
	case AssessmentReactionSubmitted:
		if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID || !validAssessmentIdentifiers(record.CausationID, record.EventID, record.RequestID, record.SubmissionID) {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		current.Status = AssessmentSubmitted
		current.SubmissionID = record.SubmissionID
		at := record.At.UTC()
		current.SubmittedAt = &at
	default:
		return Assessment{}, ErrInvalid
	}
	current.UpdatedAt = record.At.UTC()
	current.Version++
	r.assessments[current.ID] = current
	r.reactions[key] = assessmentReactionReceipt{record: record, assessment: current}
	return current, nil
}

func (r *MemoryAssessmentRepository) RecordRequestIssued(_ context.Context, record RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error) {
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	links := r.requestLinks[current.ID]
	for _, link := range links {
		if link.OriginType == record.OriginType && link.OriginID == record.OriginID && link.OriginSequence == record.OriginSequence {
			if link.RequestID == record.RequestID && link.Purpose == record.Purpose && link.InvitationID == record.InvitationID {
				return link, current, nil
			}
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
		if link.RequestID == record.RequestID {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if !validAssessmentRequestPurpose(record.Purpose) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	sequence := len(links) + 1
	if record.OriginType != AssessmentRequestOrigin || record.OriginID != current.ID || record.OriginSequence != sequence {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	if sequence == 1 {
		if record.Purpose != AssessmentRequestInitial || current.Status != AssessmentReadyToSend || current.CurrentRequestID != "" {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
		}
	} else if record.Purpose != AssessmentRequestClarification || current.Status != AssessmentUnderReview || current.CurrentRequestID == "" {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	link := AssessmentRequestLink{
		TenantID: current.TenantID, LegalEntityID: current.LegalEntityID, AssessmentID: current.ID,
		RequestID: record.RequestID, Purpose: record.Purpose, Sequence: sequence,
		OriginType: record.OriginType, OriginID: record.OriginID, OriginSequence: record.OriginSequence,
		InvitationID: record.InvitationID, CreatedAt: record.IssuedAt.UTC(),
	}
	r.requestLinks[current.ID] = append(links, link)
	current.CurrentRequestID = record.RequestID
	current.SubmissionID = ""
	current.Status = AssessmentCollecting
	current.UpdatedAt = record.IssuedAt.UTC()
	current.Version++
	r.assessments[current.ID] = current
	return link, current, nil
}

func (r *MemoryAssessmentRepository) ListAssessmentRequestLinks(_ context.Context, scope Scope, assessmentID string) ([]AssessmentRequestLink, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return nil, ErrNotFound
	}
	links := append([]AssessmentRequestLink(nil), r.requestLinks[assessmentID]...)
	sort.Slice(links, func(i, j int) bool { return links[i].Sequence < links[j].Sequence })
	return links, nil
}

func assessmentEpisodeIndex(tenantID, legalEntityID, stableKey string) string {
	return tenantID + "\x00" + legalEntityID + "\x00" + stableKey
}

func assessmentReactionIndex(record AssessmentReactionRecord) string {
	return record.TenantID + "\x00" + record.LegalEntityID + "\x00" + string(record.Kind) + "\x00" + record.CausationID
}

func sameAssessmentReaction(left, right AssessmentReactionRecord) bool {
	return left.TenantID == right.TenantID && left.LegalEntityID == right.LegalEntityID &&
		left.AssessmentID == right.AssessmentID && left.ExpectedVersion == right.ExpectedVersion && left.Kind == right.Kind &&
		left.CausationID == right.CausationID && left.JobID == right.JobID && left.EventID == right.EventID &&
		left.MatterID == right.MatterID && left.RequestID == right.RequestID && left.SubmissionID == right.SubmissionID
}

func cloneAssessmentTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
