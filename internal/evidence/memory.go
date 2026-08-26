package evidence

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type MemoryRepository struct {
	mu           sync.RWMutex
	sources      map[string]Source
	observations map[string]SourceObservation
	requests     map[string]Request
	submissions  map[string]Submission
	invitations  map[string]Invitation
	sessions     map[string]Session
	drafts       map[string]ResponseDraft
	artifacts    map[string]Artifact
}

func NewMemoryRepository(sources []Source, requests []Request) *MemoryRepository {
	repo := &MemoryRepository{sources: map[string]Source{}, observations: map[string]SourceObservation{}, requests: map[string]Request{}, submissions: map[string]Submission{}, invitations: map[string]Invitation{}, sessions: map[string]Session{}, drafts: map[string]ResponseDraft{}, artifacts: map[string]Artifact{}}
	for _, source := range sources {
		repo.sources[source.ID] = source
	}
	for _, request := range requests {
		request.KnownFacts = cloneMap(request.KnownFacts)
		request.Sections = cloneSections(request.Sections)
		request.Fields = cloneFields(request.Fields)
		request.SourceBindings = cloneRequestBindings(request.SourceBindings)
		repo.requests[request.ID] = request
	}
	return repo
}

func (r *MemoryRepository) CreateSource(_ context.Context, value Source) (Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.sources {
		if existing.TenantID == value.TenantID && existing.Code == value.Code && existing.Status != SourceRetired {
			return Source{}, ErrVersionConflict
		}
	}
	r.sources[value.ID] = value
	return value, nil
}

func (r *MemoryRepository) ListSources(_ context.Context, tenant string, limit int) ([]Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []Source{}
	for _, value := range r.sources {
		if value.TenantID == tenant {
			values = append(values, value)
		}
	}
	sortSources(values)
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) RecordSourceObservation(_ context.Context, observation SourceObservation, health SourceHealth) (Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	source, ok := r.sources[observation.SourceID]
	if !ok || source.TenantID != observation.TenantID {
		return Source{}, ErrNotFound
	}
	source.LastObservedAt = pointerTime(observation.ObservedAt)
	if observation.Success {
		source.LastSuccessAt = pointerTime(observation.ObservedAt)
	}
	source.Health = health
	source.Version++
	source.UpdatedAt = observation.ObservedAt
	r.sources[source.ID] = source
	r.observations[observation.ID] = observation
	return source, nil
}

func (r *MemoryRepository) EvaluateSourceHealth(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.sources))
	for id := range r.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	changed := 0
	for _, sourceID := range ids {
		if changed >= limit {
			break
		}
		source := r.sources[sourceID]
		if source.Status != SourceActive || source.LastSuccessAt == nil || source.Health == HealthUnavailable {
			continue
		}
		staleAt := source.LastSuccessAt.Add(time.Duration(source.ExpectedFreshnessMinutes) * time.Minute)
		if !now.Before(staleAt) && source.Health != HealthStale {
			source.Health = HealthStale
			source.Version++
			source.UpdatedAt = now
			r.sources[source.ID] = source
			changed++
		}
	}
	return changed, nil
}

func (r *MemoryRepository) CreateRequest(_ context.Context, value Request) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !value.Deadline.After(value.CreatedAt) {
		return Request{}, ErrRequestClosed
	}
	value.Origin = value.Origin.normalized()
	if err := value.Origin.validate(); err != nil {
		return Request{}, err
	}
	if !value.Origin.empty() {
		for _, existing := range r.requests {
			if existing.TenantID == value.TenantID && existing.Origin == value.Origin {
				return Request{}, ErrVersionConflict
			}
		}
	}
	value.KnownFacts = cloneMap(value.KnownFacts)
	value.Sections = cloneSections(value.Sections)
	value.Fields = cloneFields(value.Fields)
	value.SourceBindings = cloneRequestBindings(value.SourceBindings)
	r.requests[value.ID] = value
	return cloneRequest(value), nil
}

func (r *MemoryRepository) GetRequestByOrigin(_ context.Context, tenant string, origin RequestOrigin) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	origin = origin.normalized()
	for _, value := range r.requests {
		if !origin.empty() && value.TenantID == tenant && value.Origin == origin {
			return cloneRequest(value), nil
		}
	}
	return Request{}, ErrNotFound
}

func (r *MemoryRepository) ListRequests(_ context.Context, tenant string, limit int) ([]Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []Request{}
	for _, value := range r.requests {
		if value.TenantID == tenant {
			values = append(values, cloneRequest(value))
		}
	}
	sortRequests(values)
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) GetRequest(_ context.Context, tenant, id string) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.requests[id]
	if !ok || value.TenantID != tenant {
		return Request{}, ErrNotFound
	}
	return cloneRequest(value), nil
}

func (r *MemoryRepository) Submit(_ context.Context, submission Submission) (SubmissionReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[submission.RequestID]
	if !ok || request.TenantID != submission.TenantID {
		return SubmissionReceipt{}, ErrNotFound
	}
	if !requestOpenAt(request, submission.SubmittedAt) {
		return SubmissionReceipt{}, ErrRequestClosed
	}
	if request.Version != submission.ExpectedVersion {
		return SubmissionReceipt{}, ErrVersionConflict
	}
	artifactIDs, err := submissionArtifactIDs(request, submission.Answers)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	for _, artifactID := range artifactIDs {
		artifact, exists := r.artifacts[artifactID]
		if !exists || artifact.TenantID != submission.TenantID || artifact.RequestID != submission.RequestID || artifact.SubmissionID != "" {
			return SubmissionReceipt{}, ErrNotFound
		}
	}
	request.Status = RequestSubmitted
	request.Version++
	request.UpdatedAt = submission.SubmittedAt
	r.requests[request.ID] = request
	submission.Answers = cloneAnswerValues(submission.Answers)
	submission.AnswerProvenance = cloneAnswerProvenance(submission.AnswerProvenance)
	r.submissions[submission.ID] = submission
	for _, artifactID := range artifactIDs {
		artifact := r.artifacts[artifactID]
		artifact.SubmissionID = submission.ID
		r.artifacts[artifactID] = artifact
	}
	if submission.SessionID != "" {
		delete(r.drafts, draftKey(submission.TenantID, submission.RequestID, submission.SessionID))
	}
	return SubmissionReceipt{SubmissionID: submission.ID, RequestID: request.ID, Status: request.Status, SubmittedAt: submission.SubmittedAt, Version: request.Version}, nil
}

func (r *MemoryRepository) GetDraft(_ context.Context, tenant, requestID, sessionID string) (ResponseDraft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.drafts[draftKey(tenant, requestID, sessionID)]
	if !ok {
		return ResponseDraft{}, ErrNotFound
	}
	return cloneResponseDraft(value), nil
}

func (r *MemoryRepository) SaveDraft(_ context.Context, record SaveDraftRecord) (ResponseDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if record.TenantID == "" || record.RequestID == "" || record.SessionID == "" || record.ID == "" || record.ExpectedVersion < 0 {
		return ResponseDraft{}, ErrNotFound
	}
	request, ok := r.requests[record.RequestID]
	if !ok || request.TenantID != record.TenantID {
		return ResponseDraft{}, ErrNotFound
	}
	sessionFound := false
	for _, session := range r.sessions {
		if session.ID == record.SessionID && session.TenantID == record.TenantID && session.RequestID == record.RequestID {
			sessionFound = true
			break
		}
	}
	if !sessionFound {
		return ResponseDraft{}, ErrNotFound
	}
	key := draftKey(record.TenantID, record.RequestID, record.SessionID)
	current, exists := r.drafts[key]
	if !exists {
		if record.ExpectedVersion != 0 {
			return ResponseDraft{}, ErrVersionConflict
		}
		value := ResponseDraft{
			ID: record.ID, TenantID: record.TenantID, RequestID: record.RequestID, SessionID: record.SessionID,
			Answers: cloneAnswerValues(record.Answers), PresentationMode: record.PresentationMode,
			Version: 1, CreatedAt: record.UpdatedAt.UTC(), UpdatedAt: record.UpdatedAt.UTC(),
		}
		r.drafts[key] = value
		return cloneResponseDraft(value), nil
	}
	if current.Version != record.ExpectedVersion {
		return ResponseDraft{}, ErrVersionConflict
	}
	current.Answers = cloneAnswerValues(record.Answers)
	current.PresentationMode = record.PresentationMode
	current.Version++
	current.UpdatedAt = record.UpdatedAt.UTC()
	r.drafts[key] = current
	return cloneResponseDraft(current), nil
}

func (r *MemoryRepository) DeleteDraft(_ context.Context, tenant, requestID, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.drafts, draftKey(tenant, requestID, sessionID))
	return nil
}

func (r *MemoryRepository) GetSubmission(_ context.Context, tenant, id string) (Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.submissions[id]
	if !ok || value.TenantID != tenant {
		return Submission{}, ErrNotFound
	}
	value.Answers = cloneAnswerValues(value.Answers)
	value.AnswerProvenance = cloneAnswerProvenance(value.AnswerProvenance)
	return value, nil
}

func (r *MemoryRepository) ExpireRequests(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.requests))
	for requestID := range r.requests {
		ids = append(ids, requestID)
	}
	sort.Strings(ids)
	changed := 0
	for _, requestID := range ids {
		if changed >= limit {
			break
		}
		request := r.requests[requestID]
		if (request.Status != RequestReady && request.Status != RequestInProgress) || now.Before(request.Deadline) {
			continue
		}
		request.Status = RequestExpired
		request.Version++
		request.UpdatedAt = now
		r.requests[requestID] = request
		changed++
	}
	return changed, nil
}

func (r *MemoryRepository) CreateInvitation(_ context.Context, value Invitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[value.RequestID]
	if !ok || request.TenantID != value.TenantID {
		return ErrNotFound
	}
	if !requestOpenAt(request, value.CreatedAt) || !value.ExpiresAt.After(value.CreatedAt) {
		return ErrRequestClosed
	}
	key := hex.EncodeToString(value.TokenHash)
	if _, exists := r.invitations[key]; exists {
		return ErrVersionConflict
	}
	for existingKey, invitation := range r.invitations {
		if invitation.TenantID == value.TenantID && invitation.RequestID == value.RequestID && invitation.RevokedAt == nil {
			invitation.RevokedAt = pointerTime(value.CreatedAt)
			r.invitations[existingKey] = invitation
		}
	}
	for sessionKey, session := range r.sessions {
		if session.TenantID == value.TenantID && session.RequestID == value.RequestID && session.RevokedAt == nil {
			session.RevokedAt = pointerTime(value.CreatedAt)
			r.sessions[sessionKey] = session
		}
	}
	r.invitations[key] = value
	return nil
}

func (r *MemoryRepository) RedeemInvitation(_ context.Context, input RedeemInput) (Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := hex.EncodeToString(input.InvitationTokenHash)
	invitation, ok := r.invitations[key]
	if !ok || invitation.RevokedAt != nil || invitation.Redemptions >= invitation.MaxRedemptions || !input.Now.Before(invitation.ExpiresAt) || subtle.ConstantTimeCompare(invitation.AudienceHash, input.AudienceHash) != 1 {
		return Session{}, ErrInvitationInvalid
	}
	request, ok := r.requests[invitation.RequestID]
	if !ok || request.TenantID != invitation.TenantID || !requestOpenAt(request, input.Now) {
		return Session{}, ErrInvitationInvalid
	}
	expires := input.SessionExpiresAt
	if expires.After(invitation.ExpiresAt) {
		expires = invitation.ExpiresAt
	}
	if expires.After(request.Deadline) {
		expires = request.Deadline
	}
	if !expires.After(input.Now) {
		return Session{}, ErrInvitationInvalid
	}
	invitation.Redemptions++
	r.invitations[key] = invitation
	session := Session{ID: input.SessionID, TenantID: invitation.TenantID, RequestID: invitation.RequestID, AudienceHint: invitation.AudienceHint, TokenHash: append([]byte(nil), input.SessionTokenHash...), ExpiresAt: expires, CreatedAt: input.Now}
	r.sessions[hex.EncodeToString(session.TokenHash)] = session
	return session, nil
}

func (r *MemoryRepository) SessionByTokenHash(_ context.Context, tokenHash []byte, now time.Time) (Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[hex.EncodeToString(tokenHash)]
	if !ok || session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
		return Session{}, ErrSessionInvalid
	}
	return session, nil
}

func (r *MemoryRepository) RevokeRequestCapabilities(_ context.Context, tenant, requestID string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[requestID]
	if !ok || request.TenantID != tenant {
		return ErrNotFound
	}
	for key, invitation := range r.invitations {
		if invitation.TenantID == tenant && invitation.RequestID == requestID && invitation.RevokedAt == nil {
			invitation.RevokedAt = pointerTime(now)
			r.invitations[key] = invitation
		}
	}
	for key, session := range r.sessions {
		if session.TenantID == tenant && session.RequestID == requestID && session.RevokedAt == nil {
			session.RevokedAt = pointerTime(now)
			r.sessions[key] = session
		}
	}
	return nil
}

func (r *MemoryRepository) RevokeInvitation(_ context.Context, tenant, id string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, invitation := range r.invitations {
		if invitation.ID == id && invitation.TenantID == tenant {
			if invitation.RevokedAt == nil {
				invitation.RevokedAt = pointerTime(now)
				r.invitations[key] = invitation
			}
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryRepository) RevokeSession(_ context.Context, tenant, id string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, session := range r.sessions {
		if session.ID == id && session.TenantID == tenant {
			if session.RevokedAt == nil {
				session.RevokedAt = pointerTime(now)
				r.sessions[key] = session
			}
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryRepository) CreateArtifact(_ context.Context, artifact Artifact) (Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	request, ok := r.requests[artifact.RequestID]
	if !ok || request.TenantID != artifact.TenantID {
		return Artifact{}, ErrNotFound
	}
	if !requestOpenAt(request, artifact.CreatedAt) {
		return Artifact{}, ErrRequestClosed
	}
	r.artifacts[artifact.ID] = artifact
	return artifact, nil
}

func cloneRequest(value Request) Request {
	value.KnownFacts = cloneMap(value.KnownFacts)
	value.Sections = cloneSections(value.Sections)
	value.Fields = cloneFields(value.Fields)
	value.SourceBindings = cloneRequestBindings(value.SourceBindings)
	value.CollectionPeriodStart = cloneTimePointer(value.CollectionPeriodStart)
	value.CollectionPeriodEnd = cloneTimePointer(value.CollectionPeriodEnd)
	return value
}

func cloneSections(input []formcontract.Section) []formcontract.Section {
	out := append([]formcontract.Section(nil), input...)
	for index := range out {
		if input[index].Condition != nil {
			condition := *input[index].Condition
			condition.Values = append([]string(nil), input[index].Condition.Values...)
			out[index].Condition = &condition
		}
	}
	return out
}

func cloneResponseDraft(value ResponseDraft) ResponseDraft {
	value.Answers = cloneAnswerValues(value.Answers)
	return value
}

func draftKey(tenant, requestID, sessionID string) string {
	return strings.Join([]string{tenant, requestID, sessionID}, "\x00")
}

func pointerTime(value time.Time) *time.Time {
	copy := value
	return &copy
}

var _ DraftStore = (*MemoryRepository)(nil)
var _ OriginRequestStore = (*MemoryRepository)(nil)
