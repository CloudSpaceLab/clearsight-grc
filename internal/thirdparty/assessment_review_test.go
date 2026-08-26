package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type assessmentReviewEvidenceStub struct {
	requests    map[string]evidence.Request
	submissions map[string]evidence.Submission
	artifacts   map[string]evidence.Artifact
}

func (s *assessmentReviewEvidenceStub) GetRequest(_ context.Context, tenantID, requestID string) (evidence.Request, error) {
	value, ok := s.requests[requestID]
	if !ok || value.TenantID != tenantID {
		return evidence.Request{}, evidence.ErrNotFound
	}
	return value, nil
}

func (s *assessmentReviewEvidenceStub) GetSubmission(_ context.Context, tenantID, submissionID string) (evidence.Submission, error) {
	value, ok := s.submissions[submissionID]
	if !ok || value.TenantID != tenantID {
		return evidence.Submission{}, evidence.ErrNotFound
	}
	return value, nil
}

func (s *assessmentReviewEvidenceStub) GetArtifact(_ context.Context, tenantID, requestID, artifactID string) (evidence.Artifact, error) {
	value, ok := s.artifacts[artifactID]
	if !ok || value.TenantID != tenantID || value.RequestID != requestID {
		return evidence.Artifact{}, evidence.ErrNotFound
	}
	return value, nil
}

type assessmentReviewMatterStub struct {
	values []AssessmentReviewMatter
}

func (s assessmentReviewMatterStub) ListAssessmentReviewMatters(_ context.Context, _ Actor, _ Scope, _ string, _ int) ([]AssessmentReviewMatter, error) {
	return append([]AssessmentReviewMatter(nil), s.values...), nil
}

func TestAssessmentReviewReadReturnsProvenanceCoverageDocumentsAndSharedScore(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	view, err := service.GetReview(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Requests) != 1 || view.Requests[0].RequestID != "request-1" || view.Requests[0].Sequence != 1 {
		t.Fatalf("unexpected ordered request links %#v", view.Requests)
	}
	if view.Response == nil || view.Response.SubmissionID != "submission-1" || view.Response.AnswerCount != 5 || view.Response.ArtifactCount != 2 {
		t.Fatalf("unexpected response summary %#v", view.Response)
	}
	if view.Coverage.RequiredFields != 3 || view.Coverage.AnsweredRequired != 3 || view.Coverage.Ratio != 1 {
		t.Fatalf("unexpected answer coverage %#v", view.Coverage)
	}
	answer := reviewAnswerByID(t, view.Answers, "access_control")
	if answer.Provenance == nil || answer.Provenance.Origin != evidence.AnswerRespondentCorrected || answer.Provenance.SourceValue == nil {
		t.Fatalf("source-prefilled correction provenance was not retained %#v", answer.Provenance)
	}
	if len(view.Documents) != 1 {
		t.Fatalf("expected one typed vendor document, got %#v", view.Documents)
	}
	document := view.Documents[0]
	if document.DocumentType != "SOC_2_TYPE_II" || document.Reference != "SOC2-2026" || document.ArtifactStatus != evidence.ArtifactAvailable {
		t.Fatalf("unexpected vendor document metadata %#v", document)
	}
	if document.EvidenceClass != AssessmentEvidenceVendorSupplied {
		t.Fatalf("scan availability must not promote vendor evidence, got %q", document.EvidenceClass)
	}
	if view.ProvisionalScore == nil || view.ProvisionalScore.Score == nil || *view.ProvisionalScore.Score != 50 || view.ProvisionalScore.Coverage != 1 {
		t.Fatalf("unexpected shared form score %#v", view.ProvisionalScore)
	}
	if view.Assessment.Conclusion != AssessmentUnsatisfactory {
		t.Fatalf("assessment conclusion must remain distinct from score: %#v", view.Assessment)
	}
	if len(view.Matters) != 1 || view.Matters[0].MatterID != "matter-1" {
		t.Fatalf("unexpected current matter projections %#v", view.Matters)
	}
	_ = reader
}

func TestAssessmentReviewReadShowsConditionalOmissionWithoutReducingCoverage(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	request := reader.requests["request-1"]
	request.Fields = append(request.Fields, evidence.Field{
		ID: "incident_detail", SectionID: "security", Label: "Incident details", Type: string(formcontract.TypeLongText), Required: true,
		Condition: &formcontract.VisibilityCondition{FieldID: "has_incident", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}},
	})
	reader.requests["request-1"] = request

	view, err := service.GetReview(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	omitted := reviewAnswerByID(t, view.Answers, "incident_detail")
	if omitted.Visibility != AssessmentAnswerConditionallyOmitted || omitted.Value != nil || omitted.Provenance != nil {
		t.Fatalf("unexpected conditional omission %#v", omitted)
	}
	if view.Coverage.RequiredFields != 3 || view.Coverage.AnsweredRequired != 3 || view.Coverage.Ratio != 1 {
		t.Fatalf("hidden required field must not reduce coverage %#v", view.Coverage)
	}
}

func TestAssessmentReviewReadFailsClosedOnScopeRequestAndSubmissionMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(Actor, Assessment, *assessmentReviewEvidenceStub) Actor
	}{
		{name: "verified scope", mutate: func(actor Actor, _ Assessment, _ *assessmentReviewEvidenceStub) Actor {
			actor.LegalEntityID = "entity-b"
			return actor
		}},
		{name: "request origin", mutate: func(actor Actor, _ Assessment, reader *assessmentReviewEvidenceStub) Actor {
			value := reader.requests["request-1"]
			value.Origin.ID = "assessment-other"
			reader.requests["request-1"] = value
			return actor
		}},
		{name: "submission request", mutate: func(actor Actor, _ Assessment, reader *assessmentReviewEvidenceStub) Actor {
			value := reader.submissions["submission-1"]
			value.RequestID = "request-other"
			reader.submissions["submission-1"] = value
			return actor
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, actor, assessment, reader := assessmentReviewFixture(t)
			actor = test.mutate(actor, assessment, reader)
			if _, err := service.GetReview(context.Background(), actor, assessment.ID); !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected scoped not found, got %v", err)
			}
		})
	}
}

func TestAssessmentReviewReadRejectsAnswersForConditionallyOmittedFields(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	request := reader.requests["request-1"]
	request.Fields = append(request.Fields, evidence.Field{
		ID: "incident_detail", SectionID: "security", Label: "Incident details", Type: string(formcontract.TypeLongText), Required: true,
		Condition: &formcontract.VisibilityCondition{FieldID: "has_incident", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}},
	})
	reader.requests["request-1"] = request
	submission := reader.submissions["submission-1"]
	submission.Answers["incident_detail"] = formcontract.TextAnswer("Should not be accepted")
	reader.submissions["submission-1"] = submission

	if _, err := service.GetReview(context.Background(), actor, assessment.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected hidden answer to fail closed, got %v", err)
	}
}

func TestAssessmentReviewReadOmitsProtectedDeliveryAndStorageFields(t *testing.T) {
	service, actor, assessment, _ := assessmentReviewFixture(t)
	view, err := service.GetReview(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(raw)
	for _, required := range []string{`"provisional_score":`, `"matters":[`, `"artifact_status":"AVAILABLE"`, `"expires_on":"2027-05-31"`, `"status":"OPEN"`} {
		if !strings.Contains(serialized, required) {
			t.Fatalf("review view omitted canonical reviewer field %s: %s", required, serialized)
		}
	}
	for _, stale := range []string{`"score_details"`, `"findings"`, `"valid_until"`} {
		if strings.Contains(serialized, stale) {
			t.Fatalf("review view retained stale reviewer field %s: %s", stale, serialized)
		}
	}
	for _, protected := range []string{"vendor@example.test", "session-secret", "invitation-secret", "tenant/request/storage-key", "reviewer private note", "submitted_by", "storage_key", "invitation_id", "session_id"} {
		if strings.Contains(serialized, protected) {
			t.Fatalf("review view exposed protected value %q: %s", protected, serialized)
		}
	}
}

func TestAssessmentReviewReadReturnsEmptyMattersWhenReaderIsUnavailable(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	service = NewAssessmentReviewService(service.assessments, service.links, reader, nil)
	configureAssessmentReviewTestAuthority(service, assessment.ID, actor.PrincipalID)
	view, err := service.GetReview(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Matters == nil || len(view.Matters) != 0 {
		t.Fatalf("expected an explicit empty matter projection, got %#v", view.Matters)
	}
}

func TestAssessmentReviewReadFailsClosedOnMatterProjectionScopeMismatch(t *testing.T) {
	service, actor, assessment, _ := assessmentReviewFixture(t)
	service.matters = assessmentReviewMatterStub{values: []AssessmentReviewMatter{{TenantID: "bank-b", LegalEntityID: actor.LegalEntityID, AssessmentID: assessment.ID, MatterID: "matter-2", Type: "VENDOR_DEFICIENCY", Status: "OPEN", Title: "Unscoped matter"}}}
	if _, err := service.GetReview(context.Background(), actor, assessment.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected mismatched matter scope to fail closed, got %v", err)
	}
}

func TestAssessmentReviewReadUsesCurrentReviewerRouteAndRejectsRevokedReviewer(t *testing.T) {
	service, _, assessment, _ := assessmentReviewFixture(t)
	service.ConfigureAuthority(authority.NewResolver("delegated-review-test", []authority.Rule{{
		ID: "assessment-reviewer", TenantID: "bank-a", LegalEntityID: "entity-a", ObjectType: assessmentObjectType, ObjectID: assessment.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: AssessmentReviewCommand, MinMateriality: 3,
		CandidatePrincipals: []authority.Principal{{ID: "reviewer-a", Kind: "PERSON"}, {ID: "delegated-reviewer", Kind: "PERSON"}}, ResolutionStrategy: "CANDIDATE_SET", Priority: 1,
	}}))
	delegated := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "delegated-reviewer"}
	if _, err := service.GetReview(context.Background(), delegated, assessment.ID); err != nil {
		t.Fatalf("current delegated reviewer should be allowed: %v", err)
	}

	service.ConfigureAuthority(authority.NewResolver("reviewer-revoked-test", []authority.Rule{{
		ID: "assessment-reviewer", TenantID: "bank-a", LegalEntityID: "entity-a", ObjectType: assessmentObjectType, ObjectID: assessment.ID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: AssessmentReviewCommand, MinMateriality: 3,
		Principal: authority.Principal{ID: "delegated-reviewer", Kind: "PERSON"}, Priority: 1,
	}}))
	revoked := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}
	if _, err := service.GetReview(context.Background(), revoked, assessment.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reviewer removed from the current route must be denied, got %v", err)
	}
	if _, err := service.GetReview(context.Background(), delegated, assessment.ID); err != nil {
		t.Fatalf("current replacement reviewer should remain allowed: %v", err)
	}
}

func TestAssessmentReviewReadAllowsStarterAndCurrentRelationshipOwner(t *testing.T) {
	service, _, assessment, _ := assessmentReviewFixture(t)
	service.authority = nil
	for _, principalID := range []string{"starter-a", "owner-a"} {
		t.Run(principalID, func(t *testing.T) {
			actor := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: principalID}
			if _, err := service.GetReview(context.Background(), actor, assessment.ID); err != nil {
				t.Fatalf("assessment starter or current relationship owner should be allowed: %v", err)
			}
		})
	}
}

func TestAssessmentReviewReadRequiresCurrentSubmissionForReviewStates(t *testing.T) {
	service, actor, assessment, _ := assessmentReviewFixture(t)
	repository := service.links.(*MemoryAssessmentRepository)
	current := repository.assessments[assessment.ID]
	current.SubmissionID = ""
	repository.assessments[assessment.ID] = current
	if _, err := service.GetReview(context.Background(), actor, assessment.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected incomplete review state to fail closed, got %v", err)
	}
}

func assessmentReviewFixture(t *testing.T) (*AssessmentReviewService, Actor, Assessment, *assessmentReviewEvidenceStub) {
	t.Helper()
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	actor := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}
	assessment := Assessment{
		ID: "assessment-1", TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RelationshipID: "relationship-1",
		ReviewKind: AssessmentReviewOnboarding, StableEpisodeKey: "episode-1", Status: AssessmentCompleted,
		FormTemplateID: "form-1", FormTemplateVersion: 3, CurrentRequestID: "request-1", SubmissionID: "submission-1",
		ReviewMatterID: "matter-review", ReviewDueAt: now.Add(48 * time.Hour), StartedByPrincipalID: "starter-a", StartedAt: now.Add(-24 * time.Hour),
		Conclusion: AssessmentUnsatisfactory, ConclusionRationale: "Material gaps require remediation.", ReviewerPrincipalID: actor.PrincipalID,
		Version: 6, CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now,
	}
	assessmentRepo := NewMemoryAssessmentRepository()
	assessmentRepo.vendors["vendor-1"] = Vendor{ID: "vendor-1", TenantID: actor.TenantID, LegalName: "Vendor Limited", Status: VendorActive, Version: 1}
	assessmentRepo.relationships[assessment.RelationshipID] = Relationship{
		ID: assessment.RelationshipID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, VendorID: "vendor-1",
		ServiceName: "Payment processing", BusinessOwnerPrincipalID: "owner-a", Status: RelationshipUnderReview, Version: 1,
	}
	assessmentRepo.assessments[assessment.ID] = assessment
	assessmentRepo.requestLinks[assessment.ID] = []AssessmentRequestLink{{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, AssessmentID: assessment.ID, RequestID: "request-1",
		Purpose: AssessmentRequestInitial, Sequence: 1, OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: 1,
		InvitationID: "invitation-secret", CreatedAt: now.Add(-23 * time.Hour),
	}}
	no := "No"
	document := formcontract.DocumentAnswer{ArtifactID: "artifact-1", DocumentType: "SOC_2_TYPE_II", Reference: "SOC2-2026", IssuedBy: "Independent auditor", IssuedOn: "2026-06-01", ExpiresOn: "2027-05-31"}
	source := "Yes"
	reader := &assessmentReviewEvidenceStub{
		requests: map[string]evidence.Request{"request-1": {
			ID: "request-1", TenantID: actor.TenantID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: assessment.RelationshipID,
			Title: "Vendor due diligence", Purpose: "Collect due diligence evidence.", Recipient: evidence.Recipient{AudienceHint: "vendor@example.test"},
			Deadline: now.Add(24 * time.Hour), Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
			Sections: []formcontract.Section{{ID: "security", Title: "Security"}}, FormTemplateID: assessment.FormTemplateID, FormTemplateVersion: assessment.FormTemplateVersion,
			Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}, Status: evidence.RequestSubmitted, Version: 2,
			Fields: []evidence.Field{
				{ID: "access_control", SectionID: "security", Label: "Access controls enforced", Type: string(formcontract.TypeYesNo), Required: true, Options: []string{"Yes", "No"}, Scoring: &formcontract.Scoring{Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}},
				{ID: "has_incident", SectionID: "security", Label: "Material incident reported", Type: string(formcontract.TypeYesNo), Required: true, Options: []string{"Yes", "No"}, Scoring: &formcontract.Scoring{Weight: 1, AnswerScores: map[string]int{"Yes": 100, "No": 0}, CriticalAnswers: []string{"Yes"}}},
				{ID: "assurance_report", SectionID: "security", Label: "Assurance report", Type: string(formcontract.TypeVendorDocument), Required: true, AcceptedFormats: []string{"application/pdf"}},
				{ID: "security_policy", SectionID: "security", Label: "Security policy", Type: string(formcontract.TypeFile), AcceptedFormats: []string{"application/pdf"}},
				{ID: "security_policy_copy", SectionID: "security", Label: "Security policy reference", Type: string(formcontract.TypeFile), AcceptedFormats: []string{"application/pdf"}},
			},
		}},
		submissions: map[string]evidence.Submission{"submission-1": {
			ID: "submission-1", TenantID: actor.TenantID, RequestID: "request-1", SessionID: "session-secret", SubmittedBy: "vendor@example.test", Channel: "MAGIC_LINK",
			Answers: map[string]formcontract.AnswerValue{
				"access_control":       formcontract.TextAnswer(no),
				"has_incident":         formcontract.TextAnswer("No"),
				"assurance_report":     {Document: &document},
				"security_policy":      {ArtifactIDs: []string{"artifact-2"}},
				"security_policy_copy": {ArtifactIDs: []string{"artifact-2"}},
			},
			AnswerProvenance: map[string]evidence.AnswerProvenance{
				"access_control": {Origin: evidence.AnswerRespondentCorrected, SourceValue: nil},
				"has_incident":   {Origin: evidence.AnswerRespondentEntered},
			},
			SubmittedAt: now.Add(-time.Hour),
		}},
		artifacts: map[string]evidence.Artifact{"artifact-1": {
			ID: "artifact-1", TenantID: actor.TenantID, RequestID: "request-1", SubmissionID: "submission-1", FileName: "soc2-report.pdf",
			MediaType: "application/pdf", SizeBytes: 2048, SHA256: "digest", StorageKey: "tenant/request/storage-key", Status: evidence.ArtifactAvailable,
		}, "artifact-2": {
			ID: "artifact-2", TenantID: actor.TenantID, RequestID: "request-1", SubmissionID: "submission-1", FileName: "security-policy.pdf",
			MediaType: "application/pdf", SizeBytes: 1024, SHA256: "digest-2", StorageKey: "tenant/request/storage-key-2", Status: evidence.ArtifactStoredUnscanned,
		}},
	}
	sourceValue := sourceaccess.StringValue(source)
	reader.submissions["submission-1"].AnswerProvenance["access_control"] = evidence.AnswerProvenance{Origin: evidence.AnswerRespondentCorrected, SourceValue: &sourceValue}
	assessmentService := NewAssessmentService(assessmentRepo, nil)
	matters := assessmentReviewMatterStub{values: []AssessmentReviewMatter{{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, AssessmentID: assessment.ID, MatterID: "matter-1", Type: "VENDOR_DEFICIENCY", Status: "OPEN", Title: "Access-control evidence gap"}}}
	service := NewAssessmentReviewService(assessmentService, assessmentRepo, reader, matters)
	configureAssessmentReviewTestAuthority(service, assessment.ID, actor.PrincipalID)
	return service, actor, assessment, reader
}

func configureAssessmentReviewTestAuthority(service *AssessmentReviewService, assessmentID, principalID string) {
	service.ConfigureAuthority(authority.NewResolver("review-test", []authority.Rule{{
		ID: "assessment-reviewer", TenantID: "bank-a", LegalEntityID: "entity-a", ObjectType: assessmentObjectType, ObjectID: assessmentID,
		Responsibility: authority.ResponsibilityReviewer, DecisionType: AssessmentReviewCommand, MinMateriality: 3,
		Principal: authority.Principal{ID: principalID, Kind: "PERSON"}, Priority: 1,
	}}))
}

func reviewAnswerByID(t *testing.T, answers []AssessmentReviewAnswer, fieldID string) AssessmentReviewAnswer {
	t.Helper()
	for _, answer := range answers {
		if answer.FieldID == fieldID {
			return answer
		}
	}
	t.Fatalf("answer %q not found in %#v", fieldID, answers)
	return AssessmentReviewAnswer{}
}
