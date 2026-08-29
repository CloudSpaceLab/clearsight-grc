package evidence

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo             Repository
	store            ObjectStore
	now              func() time.Time
	sessionTTL       time.Duration
	maxArtifactBytes int64
	bindings         BindingReader
	legalEntities    LegalEntityResolver
}

type LegalEntityResolver interface {
	ResolveLegalEntity(context.Context, string, string) (string, error)
}

func NewService(repo Repository, store ObjectStore) *Service {
	return &Service{repo: repo, store: store, now: time.Now, sessionTTL: 20 * time.Minute, maxArtifactBytes: 20 << 20}
}

func (s *Service) Configure(sessionTTL time.Duration, maxArtifactBytes int64) {
	if sessionTTL > 0 {
		s.sessionTTL = sessionTTL
	}
	if maxArtifactBytes > 0 {
		s.maxArtifactBytes = maxArtifactBytes
	}
}

func (s *Service) ConfigureLegalEntityResolver(resolver LegalEntityResolver) {
	s.legalEntities = resolver
}

func (s *Service) CreateSource(ctx context.Context, input CreateSourceInput) (Source, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || !validSourceType(input.Type) || strings.TrimSpace(input.AuthorityClass) == "" {
		return Source{}, fmt.Errorf("tenant, code, name, source type and authority class are required")
	}
	if input.ExpectedFreshnessMinutes < 1 || input.ExpectedFreshnessMinutes > 525600 {
		return Source{}, fmt.Errorf("expected_freshness_minutes must be between 1 and 525600")
	}
	if strings.TrimSpace(input.LegalEntityID) != "" && s.legalEntities != nil {
		resolved, err := s.legalEntities.ResolveLegalEntity(ctx, input.TenantID, input.LegalEntityID)
		if err != nil {
			return Source{}, fmt.Errorf("legal entity: %w", err)
		}
		input.LegalEntityID = resolved
	}
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	if len(input.Endpoint) > 32<<10 || strings.IndexFunc(input.Endpoint, unicode.IsControl) >= 0 {
		return Source{}, fmt.Errorf("endpoint must be no more than 32768 bytes and contain no control characters")
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Source{}, err
	}
	now := s.now().UTC()
	return s.repo.CreateSource(ctx, Source{ID: valueID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, Code: input.Code, Name: input.Name, Type: input.Type, AuthorityClass: input.AuthorityClass, OwnerPrincipalID: input.OwnerPrincipalID, Endpoint: input.Endpoint, ExpectedFreshnessMinutes: input.ExpectedFreshnessMinutes, Health: HealthUnknown, Status: SourceActive, Version: 1, CreatedAt: now, UpdatedAt: now})
}

func (s *Service) ListSources(ctx context.Context, tenant string, limit int) ([]Source, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return s.repo.ListSources(ctx, tenant, bounded(limit))
}

func (s *Service) RecordSourceObservation(ctx context.Context, observation SourceObservation) (Source, error) {
	if strings.TrimSpace(observation.TenantID) == "" || strings.TrimSpace(observation.SourceID) == "" {
		return Source{}, fmt.Errorf("tenant and source are required")
	}
	var err error
	observation, err = normalizeSourceObservationScope(observation)
	if err != nil {
		return Source{}, err
	}
	evaluatedAt := s.now().UTC()
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = evaluatedAt
	} else {
		observation.ObservedAt = observation.ObservedAt.UTC()
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Source{}, err
	}
	observation.ID = valueID
	if scoped, ok := s.repo.(ScopedSourceHealthRepository); ok {
		return scoped.RecordScopedSourceObservation(ctx, observation, evaluatedAt)
	}
	health := HealthDegraded
	if observation.Unavailable {
		health = HealthUnavailable
	} else if observation.Success {
		health = HealthCurrent
	}
	return s.repo.RecordSourceObservation(ctx, observation, health)
}

func (s *Service) ListSourceScopeHealth(ctx context.Context, tenant, sourceID string, limit int) ([]SourceScopeHealth, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(sourceID) == "" {
		return nil, fmt.Errorf("tenant and source are required")
	}
	scoped, ok := s.repo.(ScopedSourceHealthRepository)
	if !ok {
		return nil, fmt.Errorf("scoped source health is unavailable")
	}
	return scoped.ListSourceScopeHealth(ctx, tenant, sourceID, s.now().UTC(), healthLimit(limit))
}

func (s *Service) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if now.IsZero() {
		now = s.now().UTC()
	}
	limit = bounded(limit)
	expired, err := s.repo.ExpireRequests(ctx, now, limit)
	if err != nil {
		return expired, err
	}
	if scoped, ok := s.repo.(ScopedSourceHealthRepository); ok {
		stale, scopedErr := scoped.EvaluateScopedSourceHealth(ctx, now, limit)
		return expired + stale, scopedErr
	}
	stale, err := s.repo.EvaluateSourceHealth(ctx, now, limit)
	return expired + stale, err
}

func (s *Service) CreateRequest(ctx context.Context, input CreateRequestInput) (Request, error) {
	if err := validateRequestInput(input); err != nil {
		return Request{}, err
	}
	if !requestOriginAllowed(ctx, input.Origin) {
		return Request{}, fmt.Errorf("request origin is reserved for its owning workflow")
	}
	scope, err := resolveCreateSubjectScope(ctx, s.repo, input)
	if err != nil {
		return Request{}, err
	}
	if strings.TrimSpace(input.CreatedBy) != "" {
		access, ok := s.repo.(SubjectAccessChecker)
		if !ok {
			return Request{}, ErrSubjectAccessDenied
		}
		allowed, accessErr := access.CanReadSubject(ctx, input.TenantID, input.CreatedBy, input.SubjectType, input.SubjectID)
		if accessErr != nil {
			return Request{}, accessErr
		}
		if !allowed {
			return Request{}, ErrSubjectAccessDenied
		}
	} else if _, exact := s.repo.(SubjectScopeResolver); exact {
		return Request{}, ErrSubjectAccessDenied
	}
	fields, sourceBindings, err := s.prepareRequestBindings(ctx, input)
	if err != nil {
		return Request{}, err
	}
	fields, err = normalizeFieldContracts(input.Presentation, input.Sections, fields)
	if err != nil {
		return Request{}, err
	}
	if strings.TrimSpace(input.CreatedBy) != "" {
		access, ok := s.repo.(SubjectAccessChecker)
		if !ok {
			return Request{}, fmt.Errorf("requester subject access validation is unavailable")
		}
		allowed, accessErr := access.CanReadSubject(ctx, input.TenantID, input.CreatedBy, input.SubjectType, input.SubjectID)
		if accessErr != nil {
			return Request{}, accessErr
		}
		if !allowed {
			return Request{}, ErrRecipientInvalid
		}
	}
	recipient, err := buildRecipient(ctx, s.repo, input.TenantID, input.AudienceType, input.Recipient)
	if err != nil {
		return Request{}, err
	}
	if recipient.Type == RecipientInternalPrincipal {
		access, ok := s.repo.(SubjectAccessChecker)
		if !ok {
			return Request{}, fmt.Errorf("internal recipient access validation is unavailable")
		}
		allowed, err := access.CanReadSubject(ctx, input.TenantID, recipient.PrincipalID, input.SubjectType, input.SubjectID)
		if err != nil {
			return Request{}, err
		}
		if !allowed {
			return Request{}, ErrRecipientInvalid
		}
	}
	store, err := recipientPersistence(s.repo)
	if err != nil {
		return Request{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Request{}, err
	}
	now := s.now().UTC()
	deadline := input.Deadline.UTC()
	if !deadline.After(now) {
		return Request{}, fmt.Errorf("deadline must be in the future")
	}
	contract, _ := formContract(input.Presentation, input.Sections, fields)
	request := Request{ID: valueID, TenantID: input.TenantID, LegalEntityID: scope.LegalEntityID, SubjectType: input.SubjectType, SubjectID: input.SubjectID, Title: input.Title, Purpose: input.Purpose, WhyYou: input.WhyYou, Sensitivity: input.Sensitivity, AudienceType: input.AudienceType, Recipient: recipient, EstimatedMinutes: input.EstimatedMinutes, Deadline: deadline, KnownFacts: cloneMap(input.KnownFacts), Presentation: contract.Presentation, Sections: contract.Sections, Fields: fields, SourceBindings: sourceBindings, FormTemplateID: strings.TrimSpace(input.FormTemplateID), FormTemplateVersion: input.FormTemplateVersion, CollectionPeriodStart: cloneTimePointer(input.CollectionPeriodStart), CollectionPeriodEnd: cloneTimePointer(input.CollectionPeriodEnd), Origin: input.Origin.normalized(), Status: RequestReady, CreatedBy: input.CreatedBy, Version: 1, CreatedAt: now, UpdatedAt: now}
	return store.CreateRequestWithRecipient(ctx, request)
}

func (s *Service) ListRequests(ctx context.Context, tenant string, limit int) ([]Request, error) {
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	values, err := s.repo.ListRequests(ctx, tenant, bounded(limit))
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	for index := range values {
		values[index] = effectiveRequest(values[index], now)
		withRecipient, hydrateErr := hydrateRequestRecipient(ctx, s.repo, values[index])
		if hydrateErr != nil {
			return nil, hydrateErr
		}
		values[index] = withRecipient
	}
	return values, nil
}

func (s *Service) GetRequest(ctx context.Context, tenant, requestID string) (Request, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(requestID) == "" {
		return Request{}, fmt.Errorf("tenant and request are required")
	}
	value, err := s.repo.GetRequest(ctx, tenant, requestID)
	if err != nil {
		return Request{}, err
	}
	value, err = hydrateRequestRecipient(ctx, s.repo, value)
	if err != nil {
		return Request{}, err
	}
	return effectiveRequest(value, s.now().UTC()), nil
}

func (s *Service) GetRequestForEntity(ctx context.Context, tenant, legalEntityID, requestID string) (Request, error) {
	value, err := s.GetRequest(ctx, tenant, requestID)
	if err != nil {
		return Request{}, err
	}
	if err := validateCurrentRequestScope(ctx, s.repo, value, legalEntityID); err != nil {
		return Request{}, err
	}
	return value, nil
}

func (s *Service) GetSubmission(ctx context.Context, tenant, submissionID string) (Submission, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(submissionID) == "" {
		return Submission{}, fmt.Errorf("tenant and submission are required")
	}
	reader, ok := s.repo.(SubmissionReader)
	if !ok {
		return Submission{}, fmt.Errorf("submission reads are unavailable")
	}
	return reader.GetSubmission(ctx, tenant, submissionID)
}

func (s *Service) Submit(ctx context.Context, submission Submission) (SubmissionReceipt, error) {
	request, err := s.GetRequest(ctx, submission.TenantID, submission.RequestID)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(submission.Channel), "MAGIC_LINK") {
		if err := validateCurrentRequestScope(ctx, s.repo, request, submission.LegalEntityID); err != nil {
			return SubmissionReceipt{}, err
		}
	}
	now := s.now().UTC()
	if !requestOpenAt(request, now) {
		return SubmissionReceipt{}, ErrRequestClosed
	}
	if strings.EqualFold(strings.TrimSpace(submission.Channel), "MAGIC_LINK") {
		if !externalRecipientRequest(request) || strings.TrimSpace(submission.SessionID) == "" {
			return SubmissionReceipt{}, ErrRecipientMismatch
		}
	} else {
		if strings.TrimSpace(submission.Channel) == "" {
			submission.Channel = "INTERNAL"
		}
		if !internalSubmissionAllowed(request, submission) {
			return SubmissionReceipt{}, ErrRecipientMismatch
		}
	}
	if err := s.validateAnswers(ctx, request, submission.Answers); err != nil {
		return SubmissionReceipt{}, err
	}
	if submission.ExpectedVersion < 1 {
		return SubmissionReceipt{}, fmt.Errorf("expected_version is required")
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return SubmissionReceipt{}, err
	}
	submission.ID = valueID
	submission.Answers = cloneAnswerValues(submission.Answers)
	submission.AnswerProvenance = s.deriveAnswerProvenance(ctx, request, submission.Answers)
	submission.SubmittedAt = now
	return s.repo.Submit(ctx, submission)
}

func (s *Service) IssueInvitation(ctx context.Context, input IssueInvitationInput) (IssuedInvitation, error) {
	request, err := s.GetRequest(ctx, input.TenantID, input.RequestID)
	if err != nil {
		return IssuedInvitation{}, err
	}
	now := s.now().UTC()
	if !requestOpenAt(request, now) {
		return IssuedInvitation{}, ErrRequestClosed
	}
	if !RequestManageableBy(request, input.CreatedBy) || !externalRecipientRequest(request) {
		return IssuedInvitation{}, ErrRecipientMismatch
	}
	if err := validateCurrentRequestScope(ctx, s.repo, request, input.LegalEntityID); err != nil {
		return IssuedInvitation{}, err
	}
	access, ok := s.repo.(SubjectAccessChecker)
	if !ok {
		return IssuedInvitation{}, fmt.Errorf("request manager access validation is unavailable")
	}
	allowed, err := access.CanReadSubject(ctx, request.TenantID, input.CreatedBy, request.SubjectType, request.SubjectID)
	if err != nil {
		return IssuedInvitation{}, err
	}
	if !allowed {
		return IssuedInvitation{}, ErrRecipientMismatch
	}
	audience := normalizeAudience(input.Audience)
	if audience == "" || strings.TrimSpace(input.Purpose) == "" {
		return IssuedInvitation{}, fmt.Errorf("audience and purpose are required")
	}
	if !externalAudienceMatches(request, audience) {
		return IssuedInvitation{}, ErrRecipientMismatch
	}
	ttl := input.TTL
	if ttl <= 0 && input.TTLMinutes > 0 {
		ttl = time.Duration(input.TTLMinutes) * time.Minute
	}
	if ttl < 5*time.Minute || ttl > 30*24*time.Hour {
		return IssuedInvitation{}, fmt.Errorf("invitation ttl must be between 5 minutes and 30 days")
	}
	token, tokenHash, err := tokenPair()
	if err != nil {
		return IssuedInvitation{}, err
	}
	audienceDigest := sha256.Sum256([]byte(audience))
	valueID := strings.TrimSpace(input.InvitationID)
	if valueID == "" {
		valueID, err = id.NewUUIDv7()
		if err != nil {
			return IssuedInvitation{}, err
		}
	}
	invitation := Invitation{ID: valueID, TenantID: input.TenantID, RequestID: input.RequestID, TokenHash: tokenHash, AudienceHash: audienceDigest[:], AudienceHint: request.Recipient.AudienceHint, Purpose: input.Purpose, ExpiresAt: now.Add(ttl), MaxRedemptions: 1, CreatedBy: input.CreatedBy, CreatedAt: now}
	if invitation.ExpiresAt.After(request.Deadline) {
		invitation.ExpiresAt = request.Deadline
	}
	if !invitation.ExpiresAt.After(now) {
		return IssuedInvitation{}, ErrRequestClosed
	}
	if err := s.repo.CreateInvitation(ctx, invitation); err != nil {
		return IssuedInvitation{}, err
	}
	return IssuedInvitation{InvitationID: valueID, Token: token, AudienceHint: invitation.AudienceHint, ExpiresAt: invitation.ExpiresAt}, nil
}

func (s *Service) RedeemInvitation(ctx context.Context, token, audience string) (RedeemedSession, error) {
	token = strings.TrimSpace(token)
	audience = normalizeAudience(audience)
	if token == "" || audience == "" {
		return RedeemedSession{}, ErrInvitationInvalid
	}
	sessionToken, sessionHash, err := tokenPair()
	if err != nil {
		return RedeemedSession{}, err
	}
	sessionID, err := id.NewUUIDv7()
	if err != nil {
		return RedeemedSession{}, err
	}
	audienceDigest := sha256.Sum256([]byte(audience))
	now := s.now().UTC()
	session, err := s.repo.RedeemInvitation(ctx, RedeemInput{InvitationTokenHash: hashToken(token), AudienceHash: audienceDigest[:], SessionTokenHash: sessionHash, SessionID: sessionID, Now: now, SessionExpiresAt: now.Add(s.sessionTTL)})
	if err != nil {
		return RedeemedSession{}, ErrInvitationInvalid
	}
	request, err := s.GetRequest(ctx, session.TenantID, session.RequestID)
	if err != nil || !externalAudienceMatches(request, audience) {
		_ = s.repo.RevokeSession(ctx, session.TenantID, session.ID, now)
		return RedeemedSession{}, ErrInvitationInvalid
	}
	return RedeemedSession{SessionID: session.ID, SessionToken: sessionToken, RequestID: session.RequestID, AudienceHint: session.AudienceHint, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) SessionRequest(ctx context.Context, sessionToken string) (Session, Request, error) {
	now := s.now().UTC()
	session, err := s.repo.SessionByTokenHash(ctx, hashToken(sessionToken), now)
	if err != nil {
		return Session{}, Request{}, ErrSessionInvalid
	}
	request, err := s.GetRequest(ctx, session.TenantID, session.RequestID)
	if err != nil {
		return Session{}, Request{}, err
	}
	if !requestOpenAt(request, now) || !externalRecipientRequest(request) || request.Recipient.AudienceHint != session.AudienceHint {
		return Session{}, Request{}, ErrSessionInvalid
	}
	return session, RespondentRequest(request), nil
}

func (s *Service) SubmitSession(ctx context.Context, sessionToken string, answers map[string]formcontract.AnswerValue, expectedVersion int64) (SubmissionReceipt, error) {
	session, request, err := s.SessionRequest(ctx, sessionToken)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	return s.Submit(ctx, Submission{TenantID: session.TenantID, RequestID: request.ID, SessionID: session.ID, Channel: "MAGIC_LINK", Answers: answers, ExpectedVersion: expectedVersion})
}

func (s *Service) RevokeInvitation(ctx context.Context, tenant, id string) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return fmt.Errorf("tenant and invitation are required")
	}
	return ErrRecipientManagerRequired
}

func (s *Service) RevokeRequestCapabilities(ctx context.Context, tenant, requestID string) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("tenant and request are required")
	}
	return s.repo.RevokeRequestCapabilities(ctx, tenant, requestID, s.now().UTC())
}

func (s *Service) RevokeSession(ctx context.Context, tenant, id string) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(id) == "" {
		return fmt.Errorf("tenant and session are required")
	}
	return ErrRecipientManagerRequired
}

func (s *Service) StoreArtifact(ctx context.Context, input ArtifactInput, reader io.Reader) (Artifact, error) {
	if s.store == nil {
		return Artifact{}, fmt.Errorf("object store is unavailable")
	}
	now := s.now().UTC()
	request, err := s.GetRequest(ctx, input.TenantID, input.RequestID)
	if err != nil {
		return Artifact{}, err
	}
	if !requestOpenAt(request, now) {
		return Artifact{}, ErrRequestClosed
	}
	if err := s.authorizeArtifactUpload(ctx, request, input); err != nil {
		return Artifact{}, err
	}
	fileName := strings.TrimSpace(input.FileName)
	maximum := s.maxArtifactBytes
	var requestField *Field
	if strings.TrimSpace(input.FieldID) != "" {
		for index := range request.Fields {
			if request.Fields[index].ID == strings.TrimSpace(input.FieldID) {
				requestField = &request.Fields[index]
				break
			}
		}
		if requestField == nil || !isFileFieldType(requestField.Type) {
			return Artifact{}, ErrFieldInvalid
		}
		if requestField.Constraints.MaxFileBytes != nil && *requestField.Constraints.MaxFileBytes < maximum {
			maximum = *requestField.Constraints.MaxFileBytes
		}
	}
	data, mediaType, err := inspectArtifact(fileName, input.MediaType, reader, maximum)
	if err != nil {
		return Artifact{}, err
	}
	if !allowedMediaType(mediaType) {
		return Artifact{}, ErrMediaType
	}
	if requestField != nil {
		candidate := Artifact{MediaType: mediaType, SizeBytes: int64(len(data)), Status: ArtifactStoredUnscanned}
		if err := validateArtifactForField(*requestField, candidate); err != nil {
			return Artifact{}, fmt.Errorf("%w: %v", ErrFieldInvalid, err)
		}
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Artifact{}, err
	}
	key := strings.Join([]string{input.TenantID, "requests", input.RequestID, valueID}, "/")
	object, err := s.store.Put(ctx, key, bytes.NewReader(data), maximum)
	if err != nil {
		return Artifact{}, err
	}
	artifact := Artifact{ID: valueID, TenantID: input.TenantID, RequestID: input.RequestID, SubmissionID: input.SubmissionID, FileName: fileName, MediaType: mediaType, SizeBytes: object.SizeBytes, SHA256: object.SHA256, StorageKey: object.Key, Status: ArtifactStoredUnscanned, CreatedBy: input.CreatedBy, CreatedAt: now}
	created, err := s.repo.CreateArtifact(ctx, artifact)
	if err != nil {
		_ = s.store.Delete(ctx, object.Key)
		return Artifact{}, err
	}
	return created, nil
}

func isFileFieldType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(formcontract.TypeFile), string(formcontract.TypePhoto), string(formcontract.TypeSignature), string(formcontract.TypeVendorDocument):
		return true
	default:
		return false
	}
}

func (s *Service) authorizeArtifactUpload(ctx context.Context, request Request, input ArtifactInput) error {
	switch request.Recipient.Type {
	case RecipientInternalPrincipal:
		if strings.TrimSpace(input.SessionToken) != "" || !RequestAssignedTo(request, input.CreatedBy) {
			return ErrRecipientMismatch
		}
		return nil
	case RecipientExternalAudience:
		if strings.TrimSpace(input.CreatedBy) != "" || strings.TrimSpace(input.SessionToken) == "" {
			return ErrRecipientMismatch
		}
		session, sessionRequest, err := s.SessionRequest(ctx, input.SessionToken)
		if err != nil {
			return ErrSessionInvalid
		}
		if session.TenantID != request.TenantID || session.RequestID != request.ID || sessionRequest.ID != request.ID {
			return ErrSessionInvalid
		}
		return nil
	default:
		return ErrRecipientMismatch
	}
}

func validateRequestInput(input CreateRequestInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.SubjectType) == "" || strings.TrimSpace(input.SubjectID) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Purpose) == "" || strings.TrimSpace(input.WhyYou) == "" || strings.TrimSpace(input.Sensitivity) == "" || strings.TrimSpace(input.AudienceType) == "" {
		return fmt.Errorf("tenant, subject, title, purpose, recipient context, sensitivity and audience type are required")
	}
	switch input.AudienceType {
	case "INTERNAL", "EXTERNAL", "CUSTOMER", "VENDOR", "AUTHORITY":
	default:
		return fmt.Errorf("audience_type is invalid")
	}
	if input.Recipient.Type == "" {
		return ErrRecipientRequired
	}
	if input.EstimatedMinutes < 1 || input.EstimatedMinutes > 60 || input.Deadline.IsZero() {
		return fmt.Errorf("estimated_minutes must be 1-60 and deadline is required")
	}
	if (strings.TrimSpace(input.FormTemplateID) == "") != (input.FormTemplateVersion == 0) || input.FormTemplateVersion < 0 {
		return fmt.Errorf("form template id and version must be provided together")
	}
	if (input.CollectionPeriodStart == nil) != (input.CollectionPeriodEnd == nil) {
		return fmt.Errorf("collection period start and end must be provided together")
	}
	if input.CollectionPeriodStart != nil && input.CollectionPeriodStart.After(*input.CollectionPeriodEnd) {
		return fmt.Errorf("collection period start must not be after the end")
	}
	if err := input.Origin.validate(); err != nil {
		return err
	}
	if len(input.Fields) == 0 || len(input.Fields) > 200 {
		return fmt.Errorf("request must contain 1-200 fields")
	}
	seen := map[string]struct{}{}
	for _, field := range input.Fields {
		if strings.TrimSpace(field.ID) == "" || strings.TrimSpace(field.Label) == "" || strings.TrimSpace(field.Type) == "" {
			return fmt.Errorf("every field requires id, label and type")
		}
		if _, exists := seen[field.ID]; exists {
			return fmt.Errorf("field ids must be unique")
		}
		seen[field.ID] = struct{}{}
	}
	return nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func requestOpenAt(request Request, at time.Time) bool {
	return (request.Status == RequestReady || request.Status == RequestInProgress) && at.Before(request.Deadline)
}

func effectiveRequest(request Request, at time.Time) Request {
	if (request.Status == RequestReady || request.Status == RequestInProgress) && !at.Before(request.Deadline) {
		request.Status = RequestExpired
	}
	return request
}

func normalizeAudience(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func tokenPair() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return digest[:]
}

func audienceHint(audience string) string {
	if at := strings.LastIndex(audience, "@"); at > 1 {
		return audience[:1] + "***" + audience[at:]
	}
	if len(audience) <= 4 {
		return "External respondent"
	}
	return audience[:2] + "***" + audience[len(audience)-2:]
}

func allowedMediaType(value string) bool {
	allowed := map[string]struct{}{"application/pdf": {}, "image/png": {}, "image/jpeg": {}, "text/plain": {}, "text/csv": {}, "application/vnd.openxmlformats-officedocument.wordprocessingml.document": {}, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {}}
	_, ok := allowed[value]
	return ok
}

func validSourceType(value SourceType) bool {
	switch value {
	case SourceRegulatory, SourceSystem, SourceDocument, SourceHuman, SourceVendor:
		return true
	default:
		return false
	}
}

func bounded(limit int) int {
	if limit < 1 || limit > 200 {
		return 50
	}
	return limit
}

func cloneMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneAnswerValues(input map[string]formcontract.AnswerValue) map[string]formcontract.AnswerValue {
	out := make(map[string]formcontract.AnswerValue, len(input))
	for key, value := range input {
		copy := value
		if value.Text != nil {
			text := *value.Text
			copy.Text = &text
		}
		copy.Values = append([]string(nil), value.Values...)
		copy.ArtifactIDs = append([]string(nil), value.ArtifactIDs...)
		if value.Document != nil {
			document := *value.Document
			copy.Document = &document
		}
		out[key] = copy
	}
	return out
}

func cloneFields(input []Field) []Field {
	out := append([]Field(nil), input...)
	for index := range out {
		out[index].Options = append([]string(nil), out[index].Options...)
		out[index].AcceptedFormats = append([]string(nil), out[index].AcceptedFormats...)
		out[index].Bindings = append([]FieldBindingReference(nil), out[index].Bindings...)
		for bindingIndex := range out[index].Bindings {
			if input[index].Bindings[bindingIndex].LookupValue != nil {
				lookup := *input[index].Bindings[bindingIndex].LookupValue
				out[index].Bindings[bindingIndex].LookupValue = &lookup
			}
		}
		out[index].SourceResolutions = cloneSourceResolutions(input[index].SourceResolutions)
		if input[index].RecordTarget != nil {
			target := *input[index].RecordTarget
			out[index].RecordTarget = &target
		}
		if input[index].RecordBaseline != nil {
			baseline := *input[index].RecordBaseline
			baseline.ExpiresAt = cloneTimePointer(input[index].RecordBaseline.ExpiresAt)
			out[index].RecordBaseline = &baseline
		}
	}
	return out
}

func DemoSources() []Source {
	now := time.Now().UTC()
	last := now.Add(-18 * time.Minute)
	return []Source{{ID: "019fd111-1111-7111-8111-111111111111", TenantID: "bank-demo", LegalEntityID: "bank-ng", Code: "CBN_CIRCULARS", Name: "CBN circulars", Type: SourceRegulatory, AuthorityClass: "OFFICIAL", ExpectedFreshnessMinutes: 60, LastObservedAt: &last, LastSuccessAt: &last, Health: HealthCurrent, Status: SourceActive, Version: 1, CreatedAt: now, UpdatedAt: now}, {ID: "019fd222-2222-7222-8222-222222222222", TenantID: "bank-demo", LegalEntityID: "bank-ng", Code: "IAM_DIRECTORY", Name: "Identity directory", Type: SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 15, LastObservedAt: &last, LastSuccessAt: &last, Health: HealthStale, Status: SourceActive, Version: 1, CreatedAt: now, UpdatedAt: now}}
}

func DemoRequests() []Request {
	now := time.Now().UTC()
	return []Request{{ID: "019fd333-3333-7333-8333-333333333333", TenantID: "bank-demo", LegalEntityID: "bank-ng", SubjectType: "CONTROL", SubjectID: "branch-backup-power", Title: "Confirm branch backup-power condition", Purpose: "Complete the August resilience review for Enugu Main Branch.", WhyYou: "You are the current branch operations manager.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL", Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "role-branch-manager", DisplayName: "Branch operations manager"}, EstimatedMinutes: 2, Deadline: now.Add(48 * time.Hour), KnownFacts: map[string]string{"Branch": "Enugu Main Branch", "Last service": "18 Jul 2026", "Maintenance firm": "Northstar Engineering"}, Fields: []Field{{ID: "condition", Label: "Current generator condition", Type: "single_select", Required: true, Options: []string{"Operational", "Operational with concern", "Unavailable"}}, {ID: "concern", Label: "Concern or supporting note", Type: "text"}}, Status: RequestReady, Version: 1, CreatedAt: now, UpdatedAt: now}}
}

func sortSources(values []Source) {
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
}

func sortRequests(values []Request) {
	sort.Slice(values, func(i, j int) bool { return values[i].Deadline.Before(values[j].Deadline) })
}
