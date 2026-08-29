package thirdparty

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type MemoryAssessmentRepository struct {
	*MemoryRepository
	relationshipLinkRepo *MemoryRelationshipLinkRepository
	assessmentMu         sync.RWMutex
	assessments          map[string]Assessment
	episodes             map[string]string
	requestLinks         map[string][]AssessmentRequestLink
	reactions            map[string]assessmentReactionReceipt
	setupJobs            map[string]AssessmentSetupJob
	matterLinks          map[string][]AssessmentMatterLink
	assessmentEvents     []memoryAssessmentAudit
	assessmentOutbox     []memoryAssessmentAudit
	assessmentDocuments  map[string]map[string]AssessmentDocument
	applicationReceipts  map[string]ResponseApplicationReceipt
}

type assessmentReactionReceipt struct {
	record     AssessmentReactionRecord
	assessment Assessment
}

type memoryAssessmentAudit struct {
	ActorPrincipalID  string            `json:"actor_principal_id,omitempty"`
	AssessmentID      string            `json:"assessment_id"`
	AssessmentVersion int64             `json:"assessment_version"`
	Type              string            `json:"type"`
	Payload           map[string]string `json:"payload"`
	OccurredAt        time.Time         `json:"occurred_at"`
}

func NewMemoryAssessmentRepository() *MemoryAssessmentRepository {
	return &MemoryAssessmentRepository{
		MemoryRepository:     NewMemoryRepository(),
		relationshipLinkRepo: NewMemoryRelationshipLinkRepository(),
		assessments:          map[string]Assessment{},
		episodes:             map[string]string{},
		requestLinks:         map[string][]AssessmentRequestLink{},
		reactions:            map[string]assessmentReactionReceipt{},
		setupJobs:            map[string]AssessmentSetupJob{},
		matterLinks:          map[string][]AssessmentMatterLink{},
		assessmentDocuments:  map[string]map[string]AssessmentDocument{},
		applicationReceipts:  map[string]ResponseApplicationReceipt{},
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
	for _, existing := range r.assessments {
		if existing.TenantID == record.TenantID && existing.LegalEntityID == record.LegalEntityID && existing.RelationshipID == record.RelationshipID && assessmentEpisodeActive(existing.Status) {
			return Assessment{}, ErrVersionConflict
		}
	}
	assessment := record.Assessment
	assessment.SelectedFieldIDs = append([]string(nil), record.Assessment.SelectedFieldIDs...)
	assessment.TenantID = record.TenantID
	assessment.LegalEntityID = record.LegalEntityID
	assessment.RelationshipID = record.RelationshipID
	r.assessments[assessment.ID] = assessment
	r.episodes[episodeIndex] = assessment.ID
	job, err := newMemoryAssessmentSetupJob(assessment)
	if err != nil {
		delete(r.assessments, assessment.ID)
		delete(r.episodes, episodeIndex)
		return Assessment{}, err
	}
	r.setupJobs[job.ID] = job
	assessment.SelectedFieldIDs = append([]string(nil), assessment.SelectedFieldIDs...)
	return assessment, nil
}

func (r *MemoryAssessmentRepository) GetAssessment(_ context.Context, scope Scope, assessmentID string) (Assessment, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return Assessment{}, ErrNotFound
	}
	assessment.SelectedFieldIDs = append([]string(nil), assessment.SelectedFieldIDs...)
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
		if !found || (assessmentEpisodeActive(assessment.Status) && !assessmentEpisodeActive(current.Status)) ||
			(assessmentEpisodeActive(assessment.Status) == assessmentEpisodeActive(current.Status) && (assessment.UpdatedAt.After(current.UpdatedAt) || (assessment.UpdatedAt.Equal(current.UpdatedAt) && assessment.ID > current.ID))) {
			current, found = assessment, true
		}
	}
	if !found {
		return Assessment{}, ErrNotFound
	}
	current.SelectedFieldIDs = append([]string(nil), current.SelectedFieldIDs...)
	return current, nil
}

func (r *MemoryAssessmentRepository) CurrentRelationshipDocuments(_ context.Context, scope Scope, relationshipID, documentType string) ([]AssessmentDocument, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	items := []AssessmentDocument{}
	for _, byArtifact := range r.assessmentDocuments {
		for _, document := range byArtifact {
			if document.TenantID == scope.TenantID && document.LegalEntityID == scope.LegalEntityID && document.RelationshipID == relationshipID && strings.EqualFold(document.DocumentType, documentType) && (document.Status == AssessmentDocumentValidated || document.Status == AssessmentDocumentExpired) {
				items = append(items, document)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
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
	eventType := ""
	switch record.To {
	case AssessmentUnderReview:
		eventType = "AssessmentReviewStarted"
	case AssessmentCompleted:
		eventType = "AssessmentCompleted"
	case AssessmentCancelled:
		eventType = "AssessmentCancelled"
	}
	r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, eventType)
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
		canonical, err := r.ensureMemoryAssessmentMatterRelationshipLink(current, record.MatterID, AssessmentMatterReview, current.StartedByPrincipalID, record.At)
		if err != nil {
			return Assessment{}, err
		}
		r.matterLinks[current.ID] = append(r.matterLinks[current.ID], AssessmentMatterLink{Scope: record.Scope, AssessmentID: current.ID, MatterID: record.MatterID, RelationshipLinkID: canonical.ID, Kind: AssessmentMatterReview, CreatedAt: record.At.UTC()})
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

func (r *MemoryAssessmentRepository) PrepareAssessmentRequest(_ context.Context, record PrepareAssessmentRequestRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifier(record.ActorPrincipalID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	links := r.requestLinks[current.ID]
	for _, link := range links {
		if link.OriginType == record.OriginType && link.OriginID == record.OriginID && link.OriginSequence == record.OriginSequence {
			if link.RequestID == record.RequestID && link.Purpose == record.Purpose {
				if current.Version != record.ExpectedVersion {
					return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
				}
				if (link.Sequence == 1 && current.Status != AssessmentReadyToSend) || (link.Sequence > 1 && current.Status != AssessmentUnderReview) {
					return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
				}
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
	sequence := len(links) + 1
	if record.OriginType != AssessmentRequestOrigin || record.OriginID != current.ID || record.OriginSequence != sequence || !validAssessmentRequestPurpose(record.Purpose) {
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
		CreatedAt: record.PreparedAt.UTC(),
	}
	r.requestLinks[current.ID] = append(links, link)
	current.CurrentRequestID = record.RequestID
	current.UpdatedAt = record.PreparedAt.UTC()
	current.Version++
	r.assessments[current.ID] = current
	r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentRequestPrepared")
	payload := r.assessmentEvents[len(r.assessmentEvents)-1].Payload
	payload["request_purpose"] = string(record.Purpose)
	payload["origin_sequence"] = fmt.Sprintf("%d", record.OriginSequence)
	return link, current, nil
}

func (r *MemoryAssessmentRepository) RecordRequestIssued(_ context.Context, record RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifier(record.ActorPrincipalID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	links := r.requestLinks[current.ID]
	for index, link := range links {
		if link.OriginType == record.OriginType && link.OriginID == record.OriginID && link.OriginSequence == record.OriginSequence {
			if link.RequestID == record.RequestID && link.Purpose == record.Purpose && link.InvitationID == record.InvitationID {
				return link, current, nil
			}
			if link.RequestID == record.RequestID && link.Purpose == record.Purpose && link.InvitationID == "" {
				if current.Version != record.ExpectedVersion {
					return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
				}
				if (link.Sequence == 1 && current.Status != AssessmentReadyToSend) || (link.Sequence > 1 && current.Status != AssessmentUnderReview) {
					return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
				}
				link.InvitationID = record.InvitationID
				r.requestLinks[current.ID][index] = link
				current.CurrentRequestID = record.RequestID
				current.SubmissionID = ""
				current.Status = AssessmentCollecting
				current.UpdatedAt = record.IssuedAt.UTC()
				current.Version++
				r.assessments[current.ID] = current
				r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentRequestIssued")
				payload := r.assessmentEvents[len(r.assessmentEvents)-1].Payload
				payload["request_purpose"] = string(record.Purpose)
				payload["origin_sequence"] = fmt.Sprintf("%d", record.OriginSequence)
				return link, current, nil
			}
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
		if link.RequestID == record.RequestID {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
	}
	return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
}

func (r *MemoryAssessmentRepository) GetCurrentAssessmentRequestLink(_ context.Context, scope Scope, assessmentID string) (AssessmentRequestLink, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return AssessmentRequestLink{}, ErrNotFound
	}
	for _, link := range r.requestLinks[assessmentID] {
		if link.RequestID == assessment.CurrentRequestID {
			return link, nil
		}
	}
	return AssessmentRequestLink{}, ErrNotFound
}

func (r *MemoryAssessmentRepository) PrepareRequestReissue(_ context.Context, record PrepareRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.ActorPrincipalID, record.RequestID) || (record.ExpectedInvitationID != "" && !validAssessmentIdentifier(record.ExpectedInvitationID)) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	for index, link := range r.requestLinks[current.ID] {
		if link.RequestID != current.CurrentRequestID {
			continue
		}
		if link.InvitationID != record.ExpectedInvitationID {
			return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
		}
		link.InvitationID = ""
		r.requestLinks[current.ID][index] = link
		current.Version++
		current.UpdatedAt = record.PreparedAt.UTC()
		r.assessments[current.ID] = current
		r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentRequestReissuePrepared")
		return link, current, nil
	}
	return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
}

func (r *MemoryAssessmentRepository) FinalizeRequestReissue(_ context.Context, record FinalizeRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.ActorPrincipalID, record.RequestID, record.InvitationID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	for index, link := range r.requestLinks[current.ID] {
		if link.RequestID != current.CurrentRequestID {
			continue
		}
		if link.InvitationID != "" {
			return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
		}
		link.InvitationID = record.InvitationID
		r.requestLinks[current.ID][index] = link
		current.Version++
		current.UpdatedAt = record.ReissuedAt.UTC()
		r.assessments[current.ID] = current
		r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentRequestReissued")
		return link, current, nil
	}
	return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
}

func (r *MemoryAssessmentRepository) appendMemoryAssessmentAudit(assessment Assessment, actorID, eventType string) {
	payload := map[string]string{
		"status": string(assessment.Status), "relationship_id": assessment.RelationshipID,
		"request_id": assessment.CurrentRequestID, "matter_id": assessment.ReviewMatterID,
	}
	event := memoryAssessmentAudit{ActorPrincipalID: actorID, AssessmentID: assessment.ID, AssessmentVersion: assessment.Version, Type: eventType, Payload: payload, OccurredAt: assessment.UpdatedAt}
	outbox := event
	outbox.ActorPrincipalID = ""
	r.assessmentEvents = append(r.assessmentEvents, event)
	r.assessmentOutbox = append(r.assessmentOutbox, outbox)
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

func (r *MemoryAssessmentRepository) ListAssessmentMatterLinks(_ context.Context, scope Scope, assessmentID string, limit int) ([]AssessmentMatterLink, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentScope(scope) || !validAssessmentIdentifier(assessmentID) || limit < 1 || limit > assessmentReviewMaxMatters+1 {
		return nil, ErrInvalid
	}
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return nil, ErrNotFound
	}
	values := append([]AssessmentMatterLink(nil), r.matterLinks[assessment.ID]...)
	r.relationshipLinkRepo.mu.RLock()
	defer r.relationshipLinkRepo.mu.RUnlock()
	for _, link := range values {
		canonical, exists := r.relationshipLinkRepo.links[link.RelationshipLinkID]
		if !exists || canonical.TenantID != scope.TenantID || canonical.LegalEntityID != scope.LegalEntityID || canonical.RelationshipID != assessment.RelationshipID || canonical.TargetType != LinkTargetMatter || canonical.TargetID != link.MatterID {
			return nil, ErrNotFound
		}
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryAssessmentRepository) GetAssessmentMatterLink(_ context.Context, scope Scope, assessmentID, matterID string) (AssessmentMatterLink, error) {
	assessmentID, matterID = strings.TrimSpace(assessmentID), strings.TrimSpace(matterID)
	if !validAssessmentScope(scope) || !validAssessmentIdentifiers(assessmentID, matterID) {
		return AssessmentMatterLink{}, ErrInvalid
	}
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return AssessmentMatterLink{}, ErrNotFound
	}
	for _, link := range r.matterLinks[assessmentID] {
		if link.MatterID != matterID {
			continue
		}
		r.relationshipLinkRepo.mu.RLock()
		canonical, exists := r.relationshipLinkRepo.links[link.RelationshipLinkID]
		r.relationshipLinkRepo.mu.RUnlock()
		if !exists || canonical.TenantID != scope.TenantID || canonical.LegalEntityID != scope.LegalEntityID || canonical.RelationshipID != assessment.RelationshipID || canonical.TargetType != LinkTargetMatter || canonical.TargetID != matterID {
			return AssessmentMatterLink{}, ErrNotFound
		}
		return link, nil
	}
	return AssessmentMatterLink{}, ErrNotFound
}

func (r *MemoryAssessmentRepository) ReviewAssessmentDocument(_ context.Context, record AssessmentDocumentReviewRecord) (AssessmentDocument, Assessment, error) {
	if !validAssessmentScope(record.Scope) || !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID, record.Artifact.ID, record.Artifact.RequestID, record.Artifact.SubmissionID) || record.ExpectedVersion < 1 || !validAssessmentDocumentDecision(record.Decision) || !validAssessmentDocumentEvidenceClass(record.EvidenceClass) {
		return AssessmentDocument{}, Assessment{}, ErrInvalid
	}
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID || current.RelationshipID == "" {
		return AssessmentDocument{}, Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentDocument{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentUnderReview || current.CurrentRequestID != record.Artifact.RequestID || current.SubmissionID != record.Artifact.SubmissionID || record.Artifact.TenantID != record.TenantID || record.Document.ArtifactID != record.Artifact.ID {
		return AssessmentDocument{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	if record.Decision == AssessmentDocumentValidate && record.Artifact.Status != evidence.ArtifactAvailable {
		return AssessmentDocument{}, Assessment{}, ErrAssessmentCompletionBlocked
	}
	status := AssessmentDocumentValidated
	eventType := "AssessmentDocumentValidated"
	if record.Decision == AssessmentDocumentReject {
		status = AssessmentDocumentRejected
		eventType = "AssessmentDocumentRejected"
	}
	if r.assessmentDocuments[current.ID] == nil {
		r.assessmentDocuments[current.ID] = map[string]AssessmentDocument{}
	}
	document, exists := r.assessmentDocuments[current.ID][record.Artifact.ID]
	if !exists {
		document = AssessmentDocument{
			ID: "document-" + record.Artifact.ID, Scope: record.Scope, RelationshipID: current.RelationshipID, AssessmentID: current.ID,
			RequestID: current.CurrentRequestID, ArtifactID: record.Artifact.ID, CreatedAt: record.At.UTC(), Version: 1,
		}
	} else {
		document.Version++
	}
	issuedOn, err := assessmentDocumentDate(record.Document.IssuedOn)
	if err != nil {
		return AssessmentDocument{}, Assessment{}, ErrInvalid
	}
	document.DocumentType = record.DocumentType
	document.Reference = strings.TrimSpace(record.Document.Reference)
	document.IssuedBy = strings.TrimSpace(record.Document.IssuedBy)
	document.IssuedOn = issuedOn
	document.ExpiresOn = cloneAssessmentTime(record.ExpiresOn)
	document.EvidenceClass = record.EvidenceClass
	document.Status = status
	document.ValidatedByPrincipalID = record.ActorPrincipalID
	document.ValidatedAt = record.At.UTC()
	document.UpdatedAt = record.At.UTC()
	r.assessmentDocuments[current.ID][record.Artifact.ID] = document
	current.Version++
	current.UpdatedAt = record.At.UTC()
	current.ReviewerPrincipalID = record.ActorPrincipalID
	r.assessments[current.ID] = current
	r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, eventType)
	return document, current, nil
}

func (r *MemoryAssessmentRepository) ListAssessmentDocuments(_ context.Context, scope Scope, assessmentID string, limit int) ([]AssessmentDocument, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentScope(scope) || !validAssessmentIdentifier(assessmentID) || limit < 1 || limit > assessmentReviewMaxArtifacts+1 {
		return nil, ErrInvalid
	}
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[assessmentID]
	if !ok || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return nil, ErrNotFound
	}
	values := make([]AssessmentDocument, 0, len(r.assessmentDocuments[assessmentID]))
	for _, document := range r.assessmentDocuments[assessmentID] {
		values = append(values, document)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].UpdatedAt.Before(values[j].UpdatedAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryAssessmentRepository) ResolveAssessmentRequest(_ context.Context, tenantID string, origin evidence.RequestOrigin, requestID string) (AssessmentSubmissionTarget, error) {
	tenantID, requestID = strings.TrimSpace(tenantID), strings.TrimSpace(requestID)
	origin.Type, origin.ID = strings.ToUpper(strings.TrimSpace(origin.Type)), strings.TrimSpace(origin.ID)
	if tenantID == "" || requestID == "" || origin.Type != AssessmentRequestOrigin || !validAssessmentIdentifier(origin.ID) || origin.Version < 1 {
		return AssessmentSubmissionTarget{}, ErrNotFound
	}
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	assessment, ok := r.assessments[origin.ID]
	if !ok || assessment.TenantID != tenantID || assessment.Status != AssessmentCollecting || assessment.CurrentRequestID != requestID {
		return AssessmentSubmissionTarget{}, ErrNotFound
	}
	for _, link := range r.requestLinks[assessment.ID] {
		if link.TenantID == tenantID && link.LegalEntityID == assessment.LegalEntityID && link.AssessmentID == assessment.ID &&
			link.RequestID == requestID && link.OriginType == origin.Type && link.OriginID == origin.ID && int64(link.OriginSequence) == origin.Version {
			return AssessmentSubmissionTarget{
				Scope:        Scope{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID},
				AssessmentID: assessment.ID, AssessmentVersion: assessment.Version, RequestID: requestID,
			}, nil
		}
	}
	return AssessmentSubmissionTarget{}, ErrNotFound
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
