package thirdparty

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const AssessmentApplyResponseCommand = "thirdparty.assessment.response.apply"

type FieldApplicationDecisionValue string

const (
	FieldApplicationAccept FieldApplicationDecisionValue = "ACCEPT"
	FieldApplicationReject FieldApplicationDecisionValue = "REJECT"
)

type ApplyAssessmentResponseInput struct {
	ExpectedAssessmentVersion  int64                      `json:"expected_assessment_version"`
	ExpectedSubmissionRevision int64                      `json:"expected_submission_revision"`
	Decisions                  []FieldApplicationDecision `json:"decisions"`
}

type FieldApplicationDecision struct {
	FieldID   string                        `json:"field_id"`
	Decision  FieldApplicationDecisionValue `json:"decision"`
	Rationale string                        `json:"rationale"`
}

type ResponseApplicationReceipt struct {
	ID                      string                     `json:"id"`
	AssessmentID            string                     `json:"assessment_id"`
	DistributionID          string                     `json:"distribution_id,omitempty"`
	ResponseRevisionID      string                     `json:"response_revision_id"`
	VendorID                string                     `json:"vendor_id"`
	ActorPrincipalID        string                     `json:"actor_principal_id"`
	AcceptedFieldIDs        []string                   `json:"accepted_field_ids"`
	RejectedFieldIDs        []string                   `json:"rejected_field_ids"`
	Decisions               []FieldApplicationDecision `json:"decisions"`
	PriorVendorVersion      int64                      `json:"prior_vendor_version"`
	ResultVendorVersion     int64                      `json:"result_vendor_version"`
	ResultAssessmentVersion int64                      `json:"result_assessment_version"`
	AppliedAt               time.Time                  `json:"applied_at"`
}

type DocumentReplacementApplication struct {
	FieldID               string
	PriorDocumentID       string
	PriorVersion          int64
	ReplacementID         string
	ReplacementArtifactID string
	DocumentType          string
}

type AssessmentApplicationRecord struct {
	Scope
	AssessmentID               string
	ExpectedAssessmentVersion  int64
	ResponseRevisionID         string
	ExpectedSubmissionRevision int64
	Vendor                     Vendor
	PriorVendorVersion         int64
	IdentityChanged            bool
	AcceptedFieldIDs           []string
	RejectedFieldIDs           []string
	Decisions                  []FieldApplicationDecision
	DocumentReplacements       []DocumentReplacementApplication
	ActorPrincipalID           string
	ReceiptID                  string
	AppliedAt                  time.Time
}

type AssessmentApplicationResult struct {
	Receipt ResponseApplicationReceipt `json:"receipt"`
	Review  AssessmentReviewView       `json:"review"`
}

type AssessmentApplicationRepository interface {
	GetResponseApplicationReceipt(context.Context, Scope, string, string) (ResponseApplicationReceipt, error)
	ApplyAssessmentResponse(context.Context, AssessmentApplicationRecord) (ResponseApplicationReceipt, error)
	GetRelationship(context.Context, Scope, string) (Aggregate, error)
	ListAssessmentDocuments(context.Context, Scope, string, int) ([]AssessmentDocument, error)
}

type AssessmentApplicationService struct {
	assessments *AssessmentService
	reviews     *AssessmentReviewService
	repo        AssessmentApplicationRepository
}

func NewAssessmentApplicationService(assessments *AssessmentService, reviews *AssessmentReviewService, repo AssessmentApplicationRepository) *AssessmentApplicationService {
	return &AssessmentApplicationService{assessments: assessments, reviews: reviews, repo: repo}
}

func (s *AssessmentApplicationService) ApplyResponse(ctx context.Context, _ Actor, assessmentID, responseRevisionID string, input ApplyAssessmentResponseInput) (AssessmentApplicationResult, error) {
	assessmentID, responseRevisionID = strings.TrimSpace(assessmentID), strings.TrimSpace(responseRevisionID)
	decisions, err := normalizeApplicationDecisions(input.Decisions)
	if s == nil || s.assessments == nil || s.reviews == nil || s.repo == nil || err != nil || !validAssessmentIdentifiers(assessmentID, responseRevisionID) || input.ExpectedAssessmentVersion < 1 || input.ExpectedSubmissionRevision != 1 {
		return AssessmentApplicationResult{}, ErrInvalid
	}
	actor, err := s.assessments.authorize(ctx, assessmentID, assessmentObjectType, AssessmentApplyResponseCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	scope := scopeFrom(actor)
	if receipt, receiptErr := s.repo.GetResponseApplicationReceipt(ctx, scope, assessmentID, responseRevisionID); receiptErr == nil {
		if !receiptMatchesDecisions(receipt, decisions) {
			return AssessmentApplicationResult{}, ErrVersionConflict
		}
		view, viewErr := s.reviews.GetReview(ctx, actor, assessmentID)
		return AssessmentApplicationResult{Receipt: receipt, Review: view}, viewErr
	} else if !errors.Is(receiptErr, ErrNotFound) {
		return AssessmentApplicationResult{}, receiptErr
	}
	view, err := s.reviews.GetReview(ctx, actor, assessmentID)
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	if view.Assessment.Version != input.ExpectedAssessmentVersion || view.Assessment.Status != AssessmentUnderReview || view.Assessment.SubmissionID != responseRevisionID || view.Response == nil || view.Response.SubmissionID != responseRevisionID {
		return AssessmentApplicationResult{}, ErrVersionConflict
	}
	baselineAnswers := map[string]AssessmentReviewAnswer{}
	for _, answer := range view.Answers {
		if answer.Visibility == AssessmentAnswerVisible && answer.Baseline != nil {
			baselineAnswers[answer.FieldID] = answer
		}
	}
	if len(baselineAnswers) == 0 || len(decisions) != len(baselineAnswers) {
		return AssessmentApplicationResult{}, ErrInvalid
	}
	aggregate, err := s.repo.GetRelationship(ctx, scope, view.Assessment.RelationshipID)
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	updated := aggregate.Vendor
	accepted, rejected := []string{}, []string{}
	replacements := []DocumentReplacementApplication{}
	storedDocuments, err := s.repo.ListAssessmentDocuments(ctx, scope, assessmentID, assessmentReviewMaxArtifacts+1)
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	documentByArtifact := map[string]AssessmentDocument{}
	for _, document := range storedDocuments {
		documentByArtifact[document.ArtifactID] = document
	}
	identityChanged := false
	priorVersion := int64(0)
	for _, decision := range decisions {
		answer, exists := baselineAnswers[decision.FieldID]
		if !exists || answer.Baseline.SubjectID != view.Assessment.RelationshipID {
			return AssessmentApplicationResult{}, ErrInvalid
		}
		if decision.Decision == FieldApplicationReject {
			rejected = append(rejected, decision.FieldID)
			continue
		}
		accepted = append(accepted, decision.FieldID)
		switch answer.Baseline.TargetKey {
		case "VENDOR.IDENTITY.LEGAL_NAME", "VENDOR.IDENTITY.TRADING_NAME", "VENDOR.IDENTITY.REGISTRATION_REFERENCE", "VENDOR.IDENTITY.JURISDICTION", "VENDOR.IDENTITY.REGISTERED_ADDRESS", "VENDOR.IDENTITY.WEBSITE_DOMAIN":
			if answer.Baseline.RecordID != aggregate.Vendor.ID || answer.Baseline.RecordVersion < 1 || answer.Value == nil || answer.Value.Text == nil {
				return AssessmentApplicationResult{}, ErrInvalid
			}
			if priorVersion == 0 {
				priorVersion = answer.Baseline.RecordVersion
			} else if priorVersion != answer.Baseline.RecordVersion {
				return AssessmentApplicationResult{}, ErrVersionConflict
			}
			before := updated
			if err := applyIdentityAnswer(&updated, answer.Baseline.TargetKey, strings.TrimSpace(*answer.Value.Text)); err != nil {
				return AssessmentApplicationResult{}, err
			}
			identityChanged = identityChanged || updated != before
		default:
			if !strings.HasPrefix(answer.Baseline.TargetKey, "VENDOR.DOCUMENT.") || answer.Value == nil || answer.Value.Document == nil {
				return AssessmentApplicationResult{}, ErrInvalid
			}
			replacement, ok := documentByArtifact[answer.Value.Document.ArtifactID]
			if !ok || replacement.Status != AssessmentDocumentValidated || replacement.DocumentType != strings.TrimPrefix(answer.Baseline.TargetKey, "VENDOR.DOCUMENT.") {
				return AssessmentApplicationResult{}, ErrInvalid
			}
			replacements = append(replacements, DocumentReplacementApplication{FieldID: decision.FieldID, PriorDocumentID: answer.Baseline.RecordID, PriorVersion: answer.Baseline.RecordVersion, ReplacementID: replacement.ID, ReplacementArtifactID: replacement.ArtifactID, DocumentType: replacement.DocumentType})
		}
	}
	if priorVersion == 0 {
		priorVersion = aggregate.Vendor.Version
	}
	receiptID, err := s.assessments.newID()
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	receipt, err := s.repo.ApplyAssessmentResponse(ctx, AssessmentApplicationRecord{Scope: scope, AssessmentID: assessmentID, ExpectedAssessmentVersion: input.ExpectedAssessmentVersion, ResponseRevisionID: responseRevisionID, ExpectedSubmissionRevision: input.ExpectedSubmissionRevision, Vendor: updated, PriorVendorVersion: priorVersion, IdentityChanged: identityChanged, AcceptedFieldIDs: accepted, RejectedFieldIDs: rejected, Decisions: decisions, DocumentReplacements: replacements, ActorPrincipalID: actor.PrincipalID, ReceiptID: receiptID, AppliedAt: s.assessments.now().UTC()})
	if err != nil {
		return AssessmentApplicationResult{}, err
	}
	refreshed, err := s.reviews.GetReview(ctx, actor, assessmentID)
	if err != nil {
		return AssessmentApplicationResult{Receipt: receipt}, nil
	}
	return AssessmentApplicationResult{Receipt: receipt, Review: refreshed}, nil
}

func normalizeApplicationDecisions(values []FieldApplicationDecision) ([]FieldApplicationDecision, error) {
	if len(values) == 0 || len(values) > formcontract.MaxFields {
		return nil, ErrInvalid
	}
	seen := map[string]bool{}
	result := make([]FieldApplicationDecision, len(values))
	for index, value := range values {
		value.FieldID, value.Rationale = strings.TrimSpace(value.FieldID), strings.TrimSpace(value.Rationale)
		value.Decision = FieldApplicationDecisionValue(strings.ToUpper(strings.TrimSpace(string(value.Decision))))
		if value.FieldID == "" || seen[value.FieldID] || (value.Decision != FieldApplicationAccept && value.Decision != FieldApplicationReject) || value.Rationale == "" || len(value.Rationale) > 2000 {
			return nil, ErrInvalid
		}
		seen[value.FieldID] = true
		result[index] = value
	}
	sort.Slice(result, func(i, j int) bool { return result[i].FieldID < result[j].FieldID })
	return result, nil
}

func applyIdentityAnswer(vendor *Vendor, targetKey, value string) error {
	switch targetKey {
	case "VENDOR.IDENTITY.LEGAL_NAME":
		if value == "" {
			return ErrInvalid
		}
		vendor.LegalName = value
	case "VENDOR.IDENTITY.TRADING_NAME":
		vendor.TradingName = value
	case "VENDOR.IDENTITY.REGISTRATION_REFERENCE":
		vendor.RegistrationRef = value
	case "VENDOR.IDENTITY.JURISDICTION":
		vendor.Jurisdiction = value
	case "VENDOR.IDENTITY.REGISTERED_ADDRESS":
		vendor.RegisteredAddress = value
	case "VENDOR.IDENTITY.WEBSITE_DOMAIN":
		domain, err := normalizeOptionalWebsiteDomain(value)
		if err != nil {
			return ErrInvalid
		}
		vendor.WebsiteDomain = domain
	default:
		return ErrInvalid
	}
	return nil
}

func receiptMatchesDecisions(receipt ResponseApplicationReceipt, decisions []FieldApplicationDecision) bool {
	if len(receipt.Decisions) == len(decisions) {
		matches := true
		for index := range decisions {
			if receipt.Decisions[index] != decisions[index] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	accepted, rejected := []string{}, []string{}
	for _, decision := range decisions {
		if decision.Decision == FieldApplicationAccept {
			accepted = append(accepted, decision.FieldID)
		} else {
			rejected = append(rejected, decision.FieldID)
		}
	}
	sort.Strings(accepted)
	sort.Strings(rejected)
	wantAccepted, wantRejected := append([]string(nil), receipt.AcceptedFieldIDs...), append([]string(nil), receipt.RejectedFieldIDs...)
	sort.Strings(wantAccepted)
	sort.Strings(wantRejected)
	return strings.Join(accepted, "\x00") == strings.Join(wantAccepted, "\x00") && strings.Join(rejected, "\x00") == strings.Join(wantRejected, "\x00")
}
