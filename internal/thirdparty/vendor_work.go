package thirdparty

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const VendorWorkOrigin = "THIRD_PARTY_WORK"

var (
	ErrVendorWorkAuthorityUnavailable = errors.New("vendor work authority is unavailable")
	ErrVendorWorkIdentityMismatch     = errors.New("vendor work authority identity does not match the request identity")
	ErrVendorWorkAcceptanceBlocked    = errors.New("vendor work response contains an unavailable document")
)

type VendorWorkState string

const (
	VendorWorkPreparing        VendorWorkState = "PREPARING"
	VendorWorkAwaitingVendor   VendorWorkState = "AWAITING_VENDOR"
	VendorWorkResponseReceived VendorWorkState = "RESPONSE_RECEIVED"
	VendorWorkUnderReview      VendorWorkState = "UNDER_REVIEW"
	VendorWorkChangesRequested VendorWorkState = "CHANGES_REQUESTED"
	VendorWorkAccepted         VendorWorkState = "ACCEPTED"
	VendorWorkCancelled        VendorWorkState = "CANCELLED"
)

type VendorWorkDeliveryState string

const (
	VendorWorkDeliveryNotSent       VendorWorkDeliveryState = "NOT_SENT"
	VendorWorkDeliveryLinkAvailable VendorWorkDeliveryState = "LINK_CREATED_EMAIL_NOT_SENT"
	VendorWorkDeliveryDelivered     VendorWorkDeliveryState = "DELIVERED"
	VendorWorkDeliveryRetryRequired VendorWorkDeliveryState = "RETRY_REQUIRED"
)

type VendorWorkRequestKind string

const (
	VendorWorkGeneral              VendorWorkRequestKind = "GENERAL"
	VendorWorkAddressVerification  VendorWorkRequestKind = "ADDRESS_VERIFICATION"
	VendorWorkCertificationRefresh VendorWorkRequestKind = "CERTIFICATION_REFRESH"
)

type VendorWorkRequest struct {
	ID                         string                        `json:"id"`
	TenantID                   string                        `json:"tenant_id"`
	LegalEntityID              string                        `json:"legal_entity_id"`
	RelationshipID             string                        `json:"relationship_id"`
	RelationshipLinkID         string                        `json:"relationship_link_id"`
	TargetType                 LinkTargetType                `json:"target_type"`
	TargetID                   string                        `json:"target_id"`
	RequestKind                VendorWorkRequestKind         `json:"request_kind"`
	Purpose                    string                        `json:"purpose"`
	Instructions               string                        `json:"instructions"`
	OwnerPrincipalID           string                        `json:"owner_principal_id"`
	ReviewerPrincipalID        string                        `json:"reviewer_principal_id,omitempty"`
	FormTemplateID             string                        `json:"form_template_id"`
	FormTemplateVersion        int64                         `json:"form_template_version"`
	Presentation               formcontract.PresentationMode `json:"presentation"`
	CurrentRequestID           string                        `json:"current_request_id,omitempty"`
	CurrentInvitationID        string                        `json:"current_invitation_id,omitempty"`
	PendingInvitationID        string                        `json:"-"`
	PendingInvitationRequestID string                        `json:"-"`
	CurrentCaptureSequence     int                           `json:"current_capture_sequence"`
	SubmissionID               string                        `json:"submission_id,omitempty"`
	State                      VendorWorkState               `json:"state"`
	DeliveryState              VendorWorkDeliveryState       `json:"delivery_state"`
	Recovery                   string                        `json:"recovery,omitempty"`
	ReviewRationale            string                        `json:"review_rationale,omitempty"`
	CancellationReason         string                        `json:"cancellation_reason,omitempty"`
	DueAt                      time.Time                     `json:"due_at"`
	Version                    int64                         `json:"version"`
	CreatedAt                  time.Time                     `json:"created_at"`
	UpdatedAt                  time.Time                     `json:"updated_at"`
	ResponseReceivedAt         *time.Time                    `json:"response_received_at,omitempty"`
	ReviewStartedAt            *time.Time                    `json:"review_started_at,omitempty"`
	AcceptedAt                 *time.Time                    `json:"accepted_at,omitempty"`
	CancelledAt                *time.Time                    `json:"cancelled_at,omitempty"`
}

type VendorWorkCaptureLink struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	LegalEntityID string    `json:"legal_entity_id"`
	WorkRequestID string    `json:"work_request_id"`
	RequestID     string    `json:"request_id"`
	Sequence      int       `json:"sequence"`
	Purpose       string    `json:"purpose"`
	OriginVersion int64     `json:"origin_version"`
	InvitationID  string    `json:"invitation_id,omitempty"`
	SubmissionID  string    `json:"submission_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type VendorWorkInvitationReservationState string

const (
	VendorWorkInvitationPending    VendorWorkInvitationReservationState = "PENDING"
	VendorWorkInvitationFinalized  VendorWorkInvitationReservationState = "FINALIZED"
	VendorWorkInvitationSuperseded VendorWorkInvitationReservationState = "SUPERSEDED"
)

type VendorWorkInvitationReservation struct {
	InvitationID string
	Scope
	WorkRequestID   string
	RequestID       string
	CaptureSequence int
	State           VendorWorkInvitationReservationState
	CreatedAt       time.Time
	ResolvedAt      *time.Time
}

type PrepareVendorWorkInput struct {
	RelationshipID      string                        `json:"relationship_id"`
	RelationshipLinkID  string                        `json:"relationship_link_id"`
	RequestKind         VendorWorkRequestKind         `json:"request_kind,omitempty"`
	Purpose             string                        `json:"purpose"`
	Instructions        string                        `json:"instructions"`
	FormTemplateID      string                        `json:"form_template_id"`
	FormTemplateVersion int64                         `json:"form_template_version"`
	Presentation        formcontract.PresentationMode `json:"presentation,omitempty"`
	VendorAudience      string                        `json:"vendor_audience"`
	DueAt               time.Time                     `json:"due_at"`
}

type SendVendorWorkInput struct {
	ExpectedVersion      int64  `json:"expected_version"`
	VendorAudience       string `json:"vendor_audience"`
	InvitationTTLMinutes int    `json:"invitation_ttl_minutes"`
}

type StartVendorWorkReviewInput struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type RequestVendorWorkChangesInput struct {
	ExpectedVersion      int64     `json:"expected_version"`
	Message              string    `json:"message"`
	FieldIDs             []string  `json:"field_ids"`
	VendorAudience       string    `json:"vendor_audience"`
	DueAt                time.Time `json:"due_at"`
	InvitationTTLMinutes int       `json:"invitation_ttl_minutes"`
}

type AcceptVendorWorkInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Rationale       string `json:"rationale"`
}

type CancelVendorWorkInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason"`
}

type RetryVendorWorkInput struct {
	ExpectedVersion      int64  `json:"expected_version"`
	VendorAudience       string `json:"vendor_audience"`
	InvitationTTLMinutes int    `json:"invitation_ttl_minutes"`
}

type VendorWorkSubmissionInput struct {
	TenantID      string
	WorkRequestID string
	RequestID     string
	SubmissionID  string
	CausationID   string
}

type VendorWorkListInput struct {
	RelationshipID     string         `json:"relationship_id,omitempty"`
	TargetType         LinkTargetType `json:"target_type,omitempty"`
	TargetID           string         `json:"target_id,omitempty"`
	Cursor             string         `json:"cursor,omitempty"`
	Limit              int            `json:"limit,omitempty"`
	VisiblePrincipalID string         `json:"-"`
}

type VendorWorkPage struct {
	Items      []VendorWorkRequest `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type VendorWorkSendOutcome struct {
	Work       VendorWorkRequest                   `json:"work"`
	Invitation *evidence.IssuedInvitation          `json:"invitation,omitempty"`
	Delivery   *evidence.InvitationDeliveryReceipt `json:"delivery,omitempty"`
	State      VendorWorkDeliveryState             `json:"state"`
	Recovery   string                              `json:"recovery,omitempty"`
	CaptureURL string                              `json:"capture_url,omitempty"`
}

type VendorWorkReviewRequest struct {
	RequestID           string                    `json:"request_id"`
	Status              evidence.RequestStatus    `json:"status"`
	Deadline            time.Time                 `json:"deadline"`
	FormTemplateID      string                    `json:"form_template_id"`
	FormTemplateVersion int64                     `json:"form_template_version"`
	Presentation        formcontract.Presentation `json:"presentation"`
}
type VendorWorkReviewResponse struct {
	SubmissionID string    `json:"submission_id"`
	RequestID    string    `json:"request_id"`
	SubmittedAt  time.Time `json:"submitted_at"`
}
type VendorWorkReviewView struct {
	Work      VendorWorkRequest          `json:"work"`
	Request   VendorWorkReviewRequest    `json:"request"`
	Response  VendorWorkReviewResponse   `json:"response"`
	Answers   []AssessmentReviewAnswer   `json:"answers"`
	Documents []AssessmentReviewDocument `json:"documents"`
}

type VendorWorkRepository interface {
	CreateVendorWork(context.Context, VendorWorkRequest) (VendorWorkRequest, error)
	FindActiveVendorWork(context.Context, Scope, string) (VendorWorkRequest, error)
	GetVendorWork(context.Context, Scope, string) (VendorWorkRequest, error)
	AttachVendorWorkCapture(context.Context, Scope, string, int64, VendorWorkCaptureLink, time.Time) (VendorWorkRequest, error)
	ReserveVendorWorkInvitation(context.Context, Scope, string, int64, string, time.Time) (VendorWorkRequest, error)
	MarkVendorWorkSent(context.Context, Scope, string, int64, string, VendorWorkDeliveryState, string, time.Time) (VendorWorkRequest, error)
	MarkVendorWorkPreparationRequired(context.Context, Scope, string, int64, string, time.Time) (VendorWorkRequest, error)
	RecordVendorWorkSubmission(context.Context, VendorWorkSubmissionInput, time.Time) (VendorWorkRequest, error)
	TransitionVendorWork(context.Context, Scope, string, int64, VendorWorkState, string, string, time.Time) (VendorWorkRequest, error)
	RecordVendorWorkChanges(context.Context, Scope, string, int64, VendorWorkCaptureLink, string, string, time.Time, time.Time) (VendorWorkRequest, error)
	ListVendorWork(context.Context, Scope, VendorWorkListInput) (VendorWorkPage, error)
	ResolveVendorWorkCapture(context.Context, string, evidence.RequestOrigin, string) (VendorWorkSubmissionTarget, error)
	HasActiveVendorWork(context.Context, Scope, string) (bool, error)
}

type VendorWorkSubmissionTarget struct {
	Scope
	WorkRequestID string
	WorkVersion   int64
	RequestID     string
}

type vendorWorkEvidence interface {
	CreateRequest(context.Context, evidence.CreateRequestInput) (evidence.Request, error)
	GetRequestByOrigin(context.Context, string, evidence.RequestOrigin) (evidence.Request, error)
	IssueInvitation(context.Context, evidence.IssueInvitationInput) (evidence.IssuedInvitation, error)
	RevokeRequestCapabilities(context.Context, string, string) error
	GetSubmission(context.Context, string, string) (evidence.Submission, error)
	GetArtifact(context.Context, string, string, string) (evidence.Artifact, error)
}

type VendorWorkService struct {
	repo            VendorWorkRepository
	links           RelationshipLinkRepository
	evidence        vendorWorkEvidence
	forms           assessmentFormReader
	delivery        *evidence.InvitationDeliveryService
	relationships   Repository
	guard           AssessmentCommandGuard
	readAuthority   authority.Service
	targets         *RelationshipTargetAccess
	coordinator     *RelationshipLinkCoordinator
	captureBase     *url.URL
	now             func() time.Time
	newID           func() (string, error)
	newInvitationID func() (string, error)
}

func NewVendorWorkService(repo VendorWorkRepository, links RelationshipLinkRepository, evidenceService vendorWorkEvidence, forms assessmentFormReader, delivery *evidence.InvitationDeliveryService, capturePublicBaseURL, environment string) (*VendorWorkService, error) {
	if repo == nil || links == nil || evidenceService == nil || forms == nil {
		return nil, ErrInvalid
	}
	base, err := parseCapturePublicBaseURL(capturePublicBaseURL, environment)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		delivery = evidence.NewInvitationDeliveryService(nil)
	}
	return &VendorWorkService{repo: repo, links: links, evidence: evidenceService, forms: forms, delivery: delivery, captureBase: base, now: time.Now, newID: id.NewUUIDv7, newInvitationID: id.NewUUIDv7}, nil
}

func (s *VendorWorkService) ConfigureRelationshipReader(reader Repository) {
	if s != nil {
		s.relationships = reader
	}
}
func (s *VendorWorkService) ConfigureAuthority(guard AssessmentCommandGuard) {
	if s != nil {
		s.guard = guard
	}
}
func (s *VendorWorkService) ConfigureReadAuthority(service authority.Service) {
	if s != nil {
		s.readAuthority = service
	}
}

func (s *VendorWorkService) ConfigureTargetReader(reader RelationshipTargetReader) {
	if s != nil {
		s.targets = NewRelationshipTargetAccess(reader)
	}
}

func (s *VendorWorkService) ConfigureCoordinator(coordinator *RelationshipLinkCoordinator) {
	if s != nil {
		s.coordinator = coordinator
	}
}

func (s *VendorWorkService) Prepare(ctx context.Context, actor Actor, input PrepareVendorWorkInput) (VendorWorkRequest, error) {
	input.RelationshipID, input.RelationshipLinkID = strings.TrimSpace(input.RelationshipID), strings.TrimSpace(input.RelationshipLinkID)
	input.Purpose, input.Instructions = strings.TrimSpace(input.Purpose), strings.TrimSpace(input.Instructions)
	input.FormTemplateID, input.VendorAudience = strings.TrimSpace(input.FormTemplateID), strings.TrimSpace(input.VendorAudience)
	requestKind, err := normalizeVendorWorkRequestKind(input.RequestKind)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	input.RequestKind = requestKind
	if input.Presentation == "" {
		input.Presentation = formcontract.PresentationAutomatic
	}
	if !validActor(actor) || input.RelationshipID == "" || input.RelationshipLinkID == "" || input.Purpose == "" || len(input.Purpose) > 500 || input.Instructions == "" || len(input.Instructions) > 2000 || input.FormTemplateID == "" || input.FormTemplateVersion < 1 || input.VendorAudience == "" || !input.DueAt.After(s.now().UTC()) {
		return VendorWorkRequest{}, ErrInvalid
	}
	if input.Presentation != formcontract.PresentationAutomatic && input.Presentation != formcontract.PresentationClassic && input.Presentation != formcontract.PresentationWizard {
		return VendorWorkRequest{}, ErrInvalid
	}
	scope := scopeFrom(actor)
	if err := s.authorize(ctx, actor, input.RelationshipID, authority.ResponsibilityOwner, "thirdparty.work.prepare"); err != nil {
		return VendorWorkRequest{}, err
	}
	s.coordinator.Lock()
	coordinatorLocked := true
	defer func() {
		if coordinatorLocked {
			s.coordinator.Unlock()
		}
	}()
	unlockCoordinator := func() {
		if coordinatorLocked {
			s.coordinator.Unlock()
			coordinatorLocked = false
		}
	}
	link, err := s.links.GetRelationshipLink(ctx, scope, input.RelationshipLinkID)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if link.State != RelationshipLinkActive || link.RelationshipID != input.RelationshipID {
		return VendorWorkRequest{}, ErrInvalid
	}
	if s.targets != nil && !s.targets.CanRead(ctx, actor, link.TargetType, link.TargetID) {
		return VendorWorkRequest{}, ErrNotFound
	}
	if err := s.authorizeObject(ctx, actor, string(link.TargetType), link.TargetID, authority.ResponsibilityOwner, "thirdparty.work.prepare"); err != nil {
		return VendorWorkRequest{}, err
	}
	form, err := s.forms.ReusableFormRevision(ctx, scope.TenantID, scope.LegalEntityID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if form.Status != monitoring.LifecycleActive || !form.IsCurrent {
		return VendorWorkRequest{}, monitoring.ErrInactive
	}
	if !vendorWorkFormMatchesRequestKind(form.Code, input.RequestKind) {
		return VendorWorkRequest{}, ErrInvalid
	}
	current, err := s.repo.FindActiveVendorWork(ctx, scope, link.ID)
	if err == nil {
		if current.RelationshipID != input.RelationshipID || current.RequestKind != input.RequestKind || current.Purpose != input.Purpose || current.Instructions != input.Instructions || current.FormTemplateID != form.ID || current.FormTemplateVersion != form.Version || current.Presentation != input.Presentation || !current.DueAt.Equal(input.DueAt.UTC()) {
			return VendorWorkRequest{}, ErrVersionConflict
		}
		if current.CurrentRequestID != "" {
			return current, nil
		}
		unlockCoordinator()
		return s.createCaptureRecoverably(ctx, actor, current, form, input.VendorAudience, input.Instructions, input.DueAt.UTC(), 1, "INITIAL")
	}
	if !errors.Is(err, ErrNotFound) {
		return VendorWorkRequest{}, err
	}
	workID, err := s.newID()
	if err != nil {
		return VendorWorkRequest{}, err
	}
	now := s.now().UTC()
	work, err := s.repo.CreateVendorWork(ctx, VendorWorkRequest{
		ID: workID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: input.RelationshipID, RelationshipLinkID: link.ID,
		TargetType: link.TargetType, TargetID: link.TargetID, RequestKind: input.RequestKind, Purpose: input.Purpose, Instructions: input.Instructions,
		OwnerPrincipalID: actor.PrincipalID, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Presentation: input.Presentation,
		State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent, DueAt: input.DueAt.UTC(), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		stored, readErr := s.repo.FindActiveVendorWork(ctx, scope, link.ID)
		if readErr != nil || stored.RelationshipID != input.RelationshipID || stored.RequestKind != input.RequestKind || stored.Purpose != input.Purpose || stored.Instructions != input.Instructions || stored.FormTemplateID != form.ID || stored.FormTemplateVersion != form.Version || stored.Presentation != input.Presentation || !stored.DueAt.Equal(input.DueAt.UTC()) {
			return VendorWorkRequest{}, err
		}
		work = stored
	}
	unlockCoordinator()
	return s.createCaptureRecoverably(ctx, actor, work, form, input.VendorAudience, input.Instructions, input.DueAt.UTC(), 1, "INITIAL")
}

func (s *VendorWorkService) createCaptureRecoverably(ctx context.Context, actor Actor, work VendorWorkRequest, form monitoring.FormTemplate, audience, instructions string, dueAt time.Time, sequence int, purpose string) (VendorWorkRequest, error) {
	updated, err := s.createCapture(ctx, actor, work, form, audience, instructions, dueAt, sequence, purpose)
	if err == nil {
		return updated, nil
	}
	recovery := "Secure response setup is incomplete. Retry preparation to continue this vendor request."
	recoverable, markErr := s.repo.MarkVendorWorkPreparationRequired(ctx, scopeFrom(actor), work.ID, work.Version, recovery, s.now().UTC())
	if markErr == nil {
		return recoverable, nil
	}
	work.Recovery = recovery
	return work, nil
}

func (s *VendorWorkService) createCapture(ctx context.Context, actor Actor, work VendorWorkRequest, form monitoring.FormTemplate, audience, instructions string, dueAt time.Time, sequence int, purpose string) (VendorWorkRequest, error) {
	request, err := s.ensureCaptureRequest(ctx, actor, work, form, audience, instructions, dueAt, sequence)
	if err != nil {
		return work, err
	}
	linkID, err := s.newID()
	if err != nil {
		return work, err
	}
	updated, err := s.repo.AttachVendorWorkCapture(ctx, scopeFrom(actor), work.ID, work.Version, VendorWorkCaptureLink{
		ID: linkID, TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, WorkRequestID: work.ID, RequestID: request.ID,
		Sequence: sequence, Purpose: purpose, OriginVersion: int64(sequence), CreatedAt: s.now().UTC(),
	}, s.now().UTC())
	if err == nil {
		return updated, nil
	}
	stored, readErr := s.repo.GetVendorWork(ctx, scopeFrom(actor), work.ID)
	if readErr == nil && stored.Version == work.Version+1 && stored.CurrentRequestID == request.ID && stored.CurrentCaptureSequence == sequence {
		return stored, nil
	}
	return work, err
}

func (s *VendorWorkService) ensureCaptureRequest(ctx context.Context, actor Actor, work VendorWorkRequest, form monitoring.FormTemplate, audience, instructions string, dueAt time.Time, sequence int) (evidence.Request, error) {
	origin := evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: int64(sequence)}
	request, err := s.evidence.GetRequestByOrigin(ctx, work.TenantID, origin)
	if errors.Is(err, evidence.ErrNotFound) {
		fields := make([]evidence.Field, len(form.Fields))
		for index, field := range form.Fields {
			fields[index] = evidence.Field{ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required, Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...), Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring}
		}
		presentation := form.Presentation
		if work.Presentation != formcontract.PresentationAutomatic {
			presentation.DefaultMode = work.Presentation
		}
		knownFacts := map[string]string{}
		if s.relationships != nil {
			relationship, relationshipErr := s.relationships.GetRelationship(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.RelationshipID)
			if relationshipErr != nil {
				return evidence.Request{}, relationshipErr
			}
			if name := strings.TrimSpace(relationship.Vendor.LegalName); name != "" {
				knownFacts["Vendor"] = name
			}
			if serviceName := strings.TrimSpace(relationship.Relationship.ServiceName); serviceName != "" {
				knownFacts["Service"] = serviceName
			}
		}
		request, err = s.evidence.CreateRequest(evidence.WithRequestOriginAuthority(ctx, VendorWorkOrigin), evidence.CreateRequestInput{
			TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: work.RelationshipID,
			Title: form.Name, Purpose: work.Purpose, WhyYou: instructions, Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
			Recipient: evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience}, EstimatedMinutes: estimateAssessmentMinutes(len(fields)), Deadline: dueAt,
			KnownFacts: knownFacts, Presentation: presentation, Sections: form.Sections, Fields: fields,
			FormTemplateID: form.ID, FormTemplateVersion: form.Version, Origin: origin, CreatedBy: actor.PrincipalID,
		})
	}
	if err != nil {
		return evidence.Request{}, err
	}
	return request, nil
}

func (s *VendorWorkService) Send(ctx context.Context, actor Actor, workID string, input SendVendorWorkInput) (VendorWorkSendOutcome, error) {
	workID, input.VendorAudience = strings.TrimSpace(workID), strings.TrimSpace(input.VendorAudience)
	if !validActor(actor) || workID == "" || input.ExpectedVersion < 1 || input.VendorAudience == "" || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 {
		return VendorWorkSendOutcome{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), workID)
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if work.Version != input.ExpectedVersion {
		return VendorWorkSendOutcome{}, ErrVersionConflict
	}
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if err := s.authorize(ctx, actor, work.RelationshipID, authority.ResponsibilityOwner, "thirdparty.work.send"); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if work.State != VendorWorkPreparing || work.CurrentRequestID == "" {
		return VendorWorkSendOutcome{}, ErrInvalidAssessmentTransition
	}
	return s.sendCurrent(ctx, actor, work, input.VendorAudience, input.InvitationTTLMinutes)
}

func (s *VendorWorkService) sendCurrent(ctx context.Context, actor Actor, work VendorWorkRequest, audience string, ttlMinutes int) (VendorWorkSendOutcome, error) {
	request, err := s.evidence.GetRequestByOrigin(ctx, work.TenantID, evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: int64(work.CurrentCaptureSequence)})
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if request.ID != work.CurrentRequestID || !evidence.ExternalAudienceMatches(request, audience) {
		return VendorWorkSendOutcome{}, ErrInvalid
	}
	if s.captureBase == nil {
		updated, updateErr := s.repo.MarkVendorWorkSent(ctx, scopeFrom(actor), work.ID, work.Version, "", VendorWorkDeliveryRetryRequired, "Set the secure capture address, then retry sending this vendor request.", s.now().UTC())
		return VendorWorkSendOutcome{Work: updated, State: VendorWorkDeliveryRetryRequired, Recovery: updated.Recovery}, updateErr
	}
	if work.PendingInvitationID != "" {
		if work.PendingInvitationRequestID != request.ID {
			return VendorWorkSendOutcome{}, ErrVersionConflict
		}
		if err := s.evidence.RevokeRequestCapabilities(ctx, work.TenantID, request.ID); err != nil {
			recovery := "Secure access could not be replaced. Retry when revocation is available."
			work.DeliveryState, work.Recovery = VendorWorkDeliveryRetryRequired, recovery
			return VendorWorkSendOutcome{Work: work, State: VendorWorkDeliveryRetryRequired, Recovery: recovery}, nil
		}
	}
	invitationID, err := s.newInvitationID()
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	reserved, err := s.repo.ReserveVendorWorkInvitation(ctx, scopeFrom(actor), work.ID, work.Version, invitationID, s.now().UTC())
	if err != nil {
		stored, readErr := s.repo.GetVendorWork(ctx, scopeFrom(actor), work.ID)
		if readErr != nil || stored.Version != work.Version+1 || stored.PendingInvitationID != invitationID || stored.PendingInvitationRequestID != request.ID || stored.DeliveryState != VendorWorkDeliveryRetryRequired {
			return VendorWorkSendOutcome{}, err
		}
		reserved = stored
	}
	issued, err := s.evidence.IssueInvitation(ctx, evidence.IssueInvitationInput{InvitationID: invitationID, TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, RequestID: request.ID, Audience: audience, Purpose: "Complete the vendor request.", TTLMinutes: ttlMinutes, CreatedBy: actor.PrincipalID})
	if err != nil {
		return VendorWorkSendOutcome{Work: reserved, State: VendorWorkDeliveryRetryRequired, Recovery: reserved.Recovery}, nil
	}
	linkURL := captureInvitationURL(s.captureBase, issued.Token)
	recovery := "Copy the secure link or retry email delivery."
	ready, err := s.repo.MarkVendorWorkSent(ctx, scopeFrom(actor), work.ID, reserved.Version, issued.InvitationID, VendorWorkDeliveryLinkAvailable, recovery, s.now().UTC())
	if err != nil {
		stored, readErr := s.repo.GetVendorWork(ctx, scopeFrom(actor), work.ID)
		if readErr == nil && stored.Version == reserved.Version+1 && stored.CurrentRequestID == request.ID && stored.CurrentInvitationID == issued.InvitationID && stored.PendingInvitationID == "" && stored.DeliveryState == VendorWorkDeliveryLinkAvailable {
			ready, err = stored, nil
		} else {
			revokeErr := s.evidence.RevokeRequestCapabilities(ctx, work.TenantID, request.ID)
			recovery = "Secure-link setup could not be confirmed. Retry to replace the reserved access."
			if revokeErr != nil {
				recovery = "Secure access remains reserved to this request. Retry when revocation is available."
			}
			reserved.DeliveryState, reserved.Recovery = VendorWorkDeliveryRetryRequired, recovery
			return VendorWorkSendOutcome{Work: reserved, State: VendorWorkDeliveryRetryRequired, Recovery: recovery}, nil
		}
	}
	receipt, deliveryErr := s.delivery.Deliver(ctx, evidence.InvitationDeliveryRequest{
		RecipientAddress: audience, InvitationLink: linkURL,
		Message: vendorWorkInvitationMessage(work, issued),
	})
	if deliveryErr != nil || receipt.Status != evidence.InvitationDelivered {
		issued.Token = ""
		return VendorWorkSendOutcome{Work: ready, Invitation: &issued, Delivery: &receipt, State: VendorWorkDeliveryLinkAvailable, CaptureURL: linkURL, Recovery: recovery}, nil
	}
	delivered, deliveryRecordErr := s.repo.MarkVendorWorkSent(ctx, scopeFrom(actor), work.ID, ready.Version, "", VendorWorkDeliveryDelivered, "", s.now().UTC())
	if deliveryRecordErr != nil {
		stored, readErr := s.repo.GetVendorWork(ctx, scopeFrom(actor), work.ID)
		if readErr == nil && stored.Version == ready.Version+1 && stored.CurrentInvitationID == issued.InvitationID && stored.DeliveryState == VendorWorkDeliveryDelivered {
			delivered = stored
		} else {
			issued.Token = ""
			return VendorWorkSendOutcome{Work: ready, Invitation: &issued, Delivery: &receipt, State: VendorWorkDeliveryLinkAvailable, CaptureURL: linkURL, Recovery: recovery}, nil
		}
	}
	issued.Token = ""
	return VendorWorkSendOutcome{Work: delivered, Invitation: &issued, Delivery: &receipt, State: VendorWorkDeliveryDelivered}, nil
}

func normalizeVendorWorkRequestKind(kind VendorWorkRequestKind) (VendorWorkRequestKind, error) {
	kind = VendorWorkRequestKind(strings.ToUpper(strings.TrimSpace(string(kind))))
	if kind == "" {
		kind = VendorWorkGeneral
	}
	switch kind {
	case VendorWorkGeneral, VendorWorkAddressVerification, VendorWorkCertificationRefresh:
		return kind, nil
	default:
		return "", ErrInvalid
	}
}

func vendorWorkFormMatchesRequestKind(code string, kind VendorWorkRequestKind) bool {
	code = strings.TrimSpace(code)
	switch kind {
	case VendorWorkAddressVerification:
		return code == "VENDOR-ADDRESS-VERIFICATION"
	case VendorWorkCertificationRefresh:
		return code == "VENDOR-CERTIFICATION-REFRESH"
	default:
		return code != "VENDOR-ADDRESS-VERIFICATION" && code != "VENDOR-CERTIFICATION-REFRESH"
	}
}

func vendorWorkInvitationMessage(work VendorWorkRequest, issued evidence.IssuedInvitation) evidence.InvitationMessageContext {
	kind, role := evidence.InvitationMessageGeneric, "Vendor contact"
	switch work.RequestKind {
	case VendorWorkAddressVerification:
		kind, role = evidence.InvitationMessageAddressVerification, "Address verification staff contact"
	case VendorWorkCertificationRefresh:
		kind = evidence.InvitationMessageCertificationRefresh
	}
	return evidence.InvitationMessageContext{
		Kind: kind, TaskTitle: work.Purpose, TaskSummary: work.Instructions, RecipientRole: role,
		DueAt: work.DueAt, ExpiresAt: issued.ExpiresAt,
	}
}

func (s *VendorWorkService) StartReview(ctx context.Context, actor Actor, workID string, input StartVendorWorkReviewInput) (VendorWorkRequest, error) {
	return s.transition(ctx, actor, workID, input.ExpectedVersion, VendorWorkUnderReview, "", "")
}

func (s *VendorWorkService) Accept(ctx context.Context, actor Actor, workID string, input AcceptVendorWorkInput) (VendorWorkRequest, error) {
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.Rationale == "" || len(input.Rationale) > 2000 {
		return VendorWorkRequest{}, ErrInvalid
	}
	view, err := s.Response(ctx, actor, strings.TrimSpace(workID))
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if view.Work.Version != input.ExpectedVersion {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	for _, document := range view.Documents {
		if document.ArtifactStatus != evidence.ArtifactAvailable {
			return VendorWorkRequest{}, ErrVendorWorkAcceptanceBlocked
		}
	}
	return s.transition(ctx, actor, workID, input.ExpectedVersion, VendorWorkAccepted, input.Rationale, "")
}

func (s *VendorWorkService) Cancel(ctx context.Context, actor Actor, workID string, input CancelVendorWorkInput) (VendorWorkRequest, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !validActor(actor) || input.ExpectedVersion < 1 || input.Reason == "" || len(input.Reason) > 1000 {
		return VendorWorkRequest{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), strings.TrimSpace(workID))
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if work.Version != input.ExpectedVersion {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return VendorWorkRequest{}, err
	}
	if err := s.authorize(ctx, actor, work.RelationshipID, authority.ResponsibilityOwner, "thirdparty.work.cancel"); err != nil {
		return VendorWorkRequest{}, err
	}
	if work.State == VendorWorkAccepted || work.State == VendorWorkCancelled {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	if work.CurrentRequestID != "" {
		if err := s.evidence.RevokeRequestCapabilities(ctx, work.TenantID, work.CurrentRequestID); err != nil {
			return VendorWorkRequest{}, err
		}
	}
	return s.repo.TransitionVendorWork(ctx, scopeFrom(actor), work.ID, work.Version, VendorWorkCancelled, actor.PrincipalID, input.Reason, s.now().UTC())
}

func (s *VendorWorkService) transition(ctx context.Context, actor Actor, workID string, expected int64, target VendorWorkState, rationale, reason string) (VendorWorkRequest, error) {
	if !validActor(actor) || strings.TrimSpace(workID) == "" || expected < 1 {
		return VendorWorkRequest{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), strings.TrimSpace(workID))
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if work.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return VendorWorkRequest{}, err
	}
	command := "thirdparty.work.review"
	if target == VendorWorkAccepted {
		command = "thirdparty.work.accept"
	}
	if err := s.authorize(ctx, actor, work.RelationshipID, authority.ResponsibilityReviewer, command); err != nil {
		return VendorWorkRequest{}, err
	}
	return s.repo.TransitionVendorWork(ctx, scopeFrom(actor), work.ID, expected, target, actor.PrincipalID, firstNonEmpty(rationale, reason), s.now().UTC())
}

func (s *VendorWorkService) RequestChanges(ctx context.Context, actor Actor, workID string, input RequestVendorWorkChangesInput) (VendorWorkSendOutcome, error) {
	input.Message, input.VendorAudience = strings.TrimSpace(input.Message), strings.TrimSpace(input.VendorAudience)
	if !validActor(actor) || input.ExpectedVersion < 1 || input.Message == "" || len(input.Message) > 2000 || len(input.FieldIDs) == 0 || input.VendorAudience == "" || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 || !input.DueAt.After(s.now().UTC()) {
		return VendorWorkSendOutcome{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), strings.TrimSpace(workID))
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if work.Version != input.ExpectedVersion || work.State != VendorWorkUnderReview {
		return VendorWorkSendOutcome{}, ErrInvalidAssessmentTransition
	}
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if err := s.authorize(ctx, actor, work.RelationshipID, authority.ResponsibilityReviewer, "thirdparty.work.request_changes"); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	form, err := s.forms.ReusableFormRevision(ctx, work.TenantID, work.LegalEntityID, work.FormTemplateID, work.FormTemplateVersion)
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	requested := map[string]bool{}
	for _, fieldID := range input.FieldIDs {
		requested[strings.TrimSpace(fieldID)] = true
	}
	byID := make(map[string]monitoring.TemplateField, len(form.Fields))
	for _, field := range form.Fields {
		byID[field.ID] = field
	}
	for fieldID := range requested {
		if _, exists := byID[fieldID]; !exists || fieldID == "" {
			return VendorWorkSendOutcome{}, ErrInvalid
		}
	}
	if len(requested) == 0 {
		return VendorWorkSendOutcome{}, ErrInvalid
	}
	selected := make(map[string]bool, len(requested))
	var include func(string) error
	include = func(fieldID string) error {
		if selected[fieldID] {
			return nil
		}
		field, exists := byID[fieldID]
		if !exists {
			return ErrInvalid
		}
		selected[fieldID] = true
		if field.Condition != nil {
			return include(field.Condition.FieldID)
		}
		return nil
	}
	for fieldID := range requested {
		if err := include(fieldID); err != nil {
			return VendorWorkSendOutcome{}, err
		}
	}
	fields := make([]monitoring.TemplateField, 0, len(selected))
	for _, field := range form.Fields {
		if selected[field.ID] {
			fields = append(fields, field)
		}
	}
	form.Fields = append([]monitoring.TemplateField(nil), fields...)
	sequence := work.CurrentCaptureSequence + 1
	request, err := s.ensureCaptureRequest(ctx, actor, work, form, input.VendorAudience, input.Message, input.DueAt.UTC(), sequence)
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	linkID, err := s.newID()
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	updated, err := s.repo.RecordVendorWorkChanges(ctx, scopeFrom(actor), work.ID, work.Version, VendorWorkCaptureLink{
		ID: linkID, TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, WorkRequestID: work.ID, RequestID: request.ID,
		Sequence: sequence, Purpose: "CLARIFICATION", OriginVersion: int64(sequence), CreatedAt: s.now().UTC(),
	}, actor.PrincipalID, input.Message, input.DueAt.UTC(), s.now().UTC())
	if err != nil {
		stored, readErr := s.repo.GetVendorWork(ctx, scopeFrom(actor), work.ID)
		if readErr != nil || stored.Version != work.Version+1 || stored.State != VendorWorkChangesRequested || stored.CurrentRequestID != request.ID || stored.CurrentCaptureSequence != sequence || stored.ReviewerPrincipalID != actor.PrincipalID || stored.ReviewRationale != input.Message || !stored.DueAt.Equal(input.DueAt.UTC()) {
			return VendorWorkSendOutcome{}, err
		}
		updated = stored
	}
	outcome, sendErr := s.sendCurrent(ctx, actor, updated, input.VendorAudience, input.InvitationTTLMinutes)
	if sendErr == nil {
		return outcome, nil
	}
	recovery := "The clarification was recorded, but secure delivery could not be prepared. Retry sending from this request."
	updated.DeliveryState, updated.Recovery = VendorWorkDeliveryRetryRequired, recovery
	return VendorWorkSendOutcome{Work: updated, State: VendorWorkDeliveryRetryRequired, Recovery: recovery}, nil
}

func (s *VendorWorkService) RecordSubmission(ctx context.Context, input VendorWorkSubmissionInput) (VendorWorkRequest, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.WorkRequestID) == "" || strings.TrimSpace(input.RequestID) == "" || strings.TrimSpace(input.SubmissionID) == "" || strings.TrimSpace(input.CausationID) == "" {
		return VendorWorkRequest{}, ErrInvalid
	}
	return s.repo.RecordVendorWorkSubmission(ctx, input, s.now().UTC())
}

func (s *VendorWorkService) Get(ctx context.Context, actor Actor, id string) (VendorWorkRequest, error) {
	if !validActor(actor) || strings.TrimSpace(id) == "" {
		return VendorWorkRequest{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), strings.TrimSpace(id))
	if err != nil {
		return VendorWorkRequest{}, err
	}
	if err := s.authorizeRead(ctx, actor, work); err != nil {
		return VendorWorkRequest{}, err
	}
	return work, nil
}

func (s *VendorWorkService) List(ctx context.Context, actor Actor, input VendorWorkListInput) (VendorWorkPage, error) {
	if !validActor(actor) || input.Limit < 0 || input.Limit > 100 || (input.TargetType != "" && input.TargetType != LinkTargetProgram && input.TargetType != LinkTargetMatter) {
		return VendorWorkPage{}, ErrInvalid
	}
	input.RelationshipID, input.TargetID, input.Cursor = strings.TrimSpace(input.RelationshipID), strings.TrimSpace(input.TargetID), strings.TrimSpace(input.Cursor)
	if input.RelationshipID == "" && (input.TargetType == "" || input.TargetID == "") {
		return VendorWorkPage{}, ErrInvalid
	}
	if (input.TargetType == "") != (input.TargetID == "") {
		return VendorWorkPage{}, ErrInvalid
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	input.VisiblePrincipalID = strings.TrimSpace(actor.PrincipalID)
	visible := make([]VendorWorkRequest, 0, input.Limit)
	scan := input
	scanned := 0
	for {
		page, err := s.repo.ListVendorWork(ctx, scopeFrom(actor), scan)
		if err != nil {
			return VendorWorkPage{}, err
		}
		for index, work := range page.Items {
			scanned++
			if s.authorizeRead(ctx, actor, work) != nil {
				continue
			}
			visible = append(visible, work)
			if len(visible) == input.Limit {
				next := ""
				if index < len(page.Items)-1 || page.NextCursor != "" {
					next = encodeCursor(work.UpdatedAt, work.ID)
				}
				return VendorWorkPage{Items: visible, NextCursor: next}, nil
			}
		}
		if page.NextCursor == "" {
			return VendorWorkPage{Items: visible}, nil
		}
		if scanned >= thirdPartyVisibilityScanLimit {
			return VendorWorkPage{Items: visible, NextCursor: page.NextCursor}, nil
		}
		scan.Cursor = page.NextCursor
	}
}

func (s *VendorWorkService) Response(ctx context.Context, actor Actor, id string) (VendorWorkReviewView, error) {
	work, err := s.Get(ctx, actor, id)
	if err != nil {
		return VendorWorkReviewView{}, err
	}
	if work.CurrentRequestID == "" || work.SubmissionID == "" {
		return VendorWorkReviewView{}, ErrNotFound
	}
	request, err := s.evidence.GetRequestByOrigin(ctx, work.TenantID, evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: int64(work.CurrentCaptureSequence)})
	if err != nil || request.ID != work.CurrentRequestID {
		return VendorWorkReviewView{}, ErrNotFound
	}
	submission, err := s.evidence.GetSubmission(ctx, work.TenantID, work.SubmissionID)
	if err != nil || submission.RequestID != request.ID {
		return VendorWorkReviewView{}, ErrNotFound
	}
	contract, err := formcontract.Normalize(reviewContract(request))
	if err != nil {
		return VendorWorkReviewView{}, err
	}
	visibleFields, err := formcontract.VisibleFields(contract, submission.Answers)
	if err != nil {
		return VendorWorkReviewView{}, err
	}
	visible := map[string]formcontract.Field{}
	for _, field := range visibleFields {
		visible[field.ID] = field
	}
	view := VendorWorkReviewView{Work: work, Request: VendorWorkReviewRequest{RequestID: request.ID, Status: request.Status, Deadline: request.Deadline, FormTemplateID: request.FormTemplateID, FormTemplateVersion: request.FormTemplateVersion, Presentation: request.Presentation}, Response: VendorWorkReviewResponse{SubmissionID: submission.ID, RequestID: submission.RequestID, SubmittedAt: submission.SubmittedAt}, Answers: make([]AssessmentReviewAnswer, 0, len(contract.Fields)), Documents: []AssessmentReviewDocument{}}
	for _, field := range contract.Fields {
		answer := AssessmentReviewAnswer{FieldID: field.ID, Label: field.Label, Type: field.Type, Required: field.Required, Visibility: AssessmentAnswerConditionallyOmitted}
		if _, ok := visible[field.ID]; ok {
			answer.Visibility = AssessmentAnswerVisible
			if value, exists := submission.Answers[field.ID]; exists {
				copy := value
				answer.Value = &copy
			}
			if provenance, exists := submission.AnswerProvenance[field.ID]; exists {
				copy := provenance
				answer.Provenance = &copy
			}
		}
		view.Answers = append(view.Answers, answer)
		if answer.Value == nil || !reviewArtifactField(field.Type) {
			continue
		}
		for _, artifactID := range reviewArtifactIDs(*answer.Value) {
			artifact, readErr := s.evidence.GetArtifact(ctx, work.TenantID, request.ID, artifactID)
			if readErr != nil || artifact.SubmissionID != submission.ID {
				return VendorWorkReviewView{}, ErrNotFound
			}
			document := AssessmentReviewDocument{FieldID: field.ID, ArtifactID: artifact.ID, FileName: artifact.FileName, MediaType: artifact.MediaType, SizeBytes: artifact.SizeBytes, ArtifactStatus: artifact.Status, Status: "SUBMITTED", EvidenceClass: AssessmentEvidenceVendorSupplied}
			if answer.Value.Document != nil {
				document.DocumentType = answer.Value.Document.DocumentType
				document.Reference = answer.Value.Document.Reference
				document.IssuedBy = answer.Value.Document.IssuedBy
				document.IssuedOn = answer.Value.Document.IssuedOn
				document.ExpiresOn = answer.Value.Document.ExpiresOn
			}
			view.Documents = append(view.Documents, document)
		}
	}
	return view, nil
}

func (s *VendorWorkService) authorizeRead(ctx context.Context, actor Actor, work VendorWorkRequest) error {
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return err
	}
	if s.relationships == nil {
		return ErrNotFound
	}
	relationship, err := s.relationships.GetRelationship(ctx, scopeFrom(actor), work.RelationshipID)
	if err != nil {
		return ErrNotFound
	}
	if actor.PrincipalID == work.OwnerPrincipalID || actor.PrincipalID == work.ReviewerPrincipalID || actor.PrincipalID == relationship.Relationship.BusinessOwnerPrincipalID {
		return nil
	}
	if s.readAuthority == nil {
		return ErrNotFound
	}
	for _, route := range []struct {
		responsibility authority.Responsibility
		decisionType   string
	}{
		{responsibility: authority.ResponsibilityOwner, decisionType: "thirdparty.work.send"},
		{responsibility: authority.ResponsibilityReviewer, decisionType: "thirdparty.work.review"},
	} {
		resolution, resolveErr := s.readAuthority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "VENDOR_RELATIONSHIP", ObjectID: work.RelationshipID, Responsibility: route.responsibility, DecisionType: route.decisionType, Materiality: 3})
		if resolveErr == nil && resolution.AllowsPrincipal(actor.PrincipalID) {
			return nil
		}
	}
	return ErrNotFound
}

func (s *VendorWorkService) Retry(ctx context.Context, actor Actor, workID string, input RetryVendorWorkInput) (VendorWorkSendOutcome, error) {
	workID, input.VendorAudience = strings.TrimSpace(workID), strings.TrimSpace(input.VendorAudience)
	if !validActor(actor) || workID == "" || input.ExpectedVersion < 1 || input.VendorAudience == "" || input.InvitationTTLMinutes < 5 || input.InvitationTTLMinutes > 30*24*60 {
		return VendorWorkSendOutcome{}, ErrInvalid
	}
	work, err := s.repo.GetVendorWork(ctx, scopeFrom(actor), workID)
	if err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if work.Version != input.ExpectedVersion {
		return VendorWorkSendOutcome{}, ErrVersionConflict
	}
	if err := s.authorizeWorkTarget(ctx, actor, work); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if err := s.authorize(ctx, actor, work.RelationshipID, authority.ResponsibilityOwner, "thirdparty.work.retry"); err != nil {
		return VendorWorkSendOutcome{}, err
	}
	if work.CurrentRequestID == "" && work.State == VendorWorkPreparing && work.DeliveryState == VendorWorkDeliveryRetryRequired {
		form, formErr := s.forms.ReusableFormRevision(ctx, work.TenantID, work.LegalEntityID, work.FormTemplateID, work.FormTemplateVersion)
		if formErr != nil {
			return VendorWorkSendOutcome{}, formErr
		}
		prepared, prepareErr := s.createCaptureRecoverably(ctx, actor, work, form, input.VendorAudience, work.Instructions, work.DueAt, 1, "INITIAL")
		if prepareErr != nil {
			return VendorWorkSendOutcome{}, prepareErr
		}
		if prepared.CurrentRequestID == "" {
			return VendorWorkSendOutcome{Work: prepared, State: prepared.DeliveryState, Recovery: prepared.Recovery}, nil
		}
		return s.sendCurrent(ctx, actor, prepared, input.VendorAudience, input.InvitationTTLMinutes)
	}
	if work.CurrentRequestID == "" || (work.DeliveryState != VendorWorkDeliveryLinkAvailable && work.DeliveryState != VendorWorkDeliveryRetryRequired) || (work.State != VendorWorkPreparing && work.State != VendorWorkAwaitingVendor && work.State != VendorWorkChangesRequested) {
		return VendorWorkSendOutcome{}, ErrInvalidAssessmentTransition
	}
	if work.CurrentRequestID != "" {
		if err := s.evidence.RevokeRequestCapabilities(ctx, work.TenantID, work.CurrentRequestID); err != nil {
			return VendorWorkSendOutcome{}, err
		}
	}
	return s.sendCurrent(ctx, actor, work, input.VendorAudience, input.InvitationTTLMinutes)
}

func (s *VendorWorkService) authorizeWorkTarget(ctx context.Context, actor Actor, work VendorWorkRequest) error {
	if s.targets != nil && !s.targets.CanRead(ctx, actor, work.TargetType, work.TargetID) {
		return ErrNotFound
	}
	return nil
}

func (s *VendorWorkService) authorize(ctx context.Context, actor Actor, relationshipID string, responsibility authority.Responsibility, command string) error {
	return s.authorizeObject(ctx, actor, "VENDOR_RELATIONSHIP", relationshipID, responsibility, command)
}

func (s *VendorWorkService) authorizeObject(ctx context.Context, actor Actor, objectType, objectID string, responsibility authority.Responsibility, command string) error {
	if s.guard == nil {
		return nil
	}
	if guard, ok := s.guard.(*commandauth.Guard); ok && guard.Mode() == commandauth.ModeOff {
		return nil
	}
	verified, err := identity.Require(ctx)
	if err != nil {
		return err
	}
	if err := verified.Valid(s.now().UTC()); err != nil || verified.LegalEntityID == "*" {
		if err != nil {
			return err
		}
		return identity.ErrInvalidIdentity
	}
	if verified.TenantID != actor.TenantID || verified.LegalEntityID != actor.LegalEntityID || verified.PrincipalID != actor.PrincipalID {
		return ErrVendorWorkIdentityMismatch
	}
	decision, err := s.guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: objectType, ObjectID: objectID, Responsibility: responsibility, DecisionType: command, Materiality: 3})
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return commandauth.ErrNotAuthorized
	}
	if err := decision.Actor.Valid(s.now().UTC()); err != nil || decision.Actor.TenantID != actor.TenantID || decision.Actor.LegalEntityID != actor.LegalEntityID || decision.Actor.PrincipalID != actor.PrincipalID {
		return ErrVendorWorkIdentityMismatch
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

var _ vendorWorkEvidence = (*evidence.Service)(nil)
