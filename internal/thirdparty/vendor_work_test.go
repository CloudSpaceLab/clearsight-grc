package thirdparty

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type vendorWorkFixture struct {
	service    *VendorWorkService
	repository *MemoryVendorWorkRepository
	links      *MemoryRelationshipLinkRepository
	forms      *monitoring.MemoryRepository
	evidence   *evidence.Service
	access     *evidence.DistributionAccessService
	dispatcher *evidence.WorkflowDistributionDispatcher
	otp        *vendorWorkOTPDelivery
	link       RelationshipLink
	actor      Actor
	audience   string
	now        time.Time
}

type vendorWorkOTPDelivery struct {
	values []evidence.DistributionOTPDelivery
}

func (delivery *vendorWorkOTPDelivery) DeliverDistributionOTP(_ context.Context, value evidence.DistributionOTPDelivery) error {
	delivery.values = append(delivery.values, value)
	return nil
}

type vendorWorkDistributionFormReader struct{ forms *monitoring.MemoryRepository }

func (reader vendorWorkDistributionFormReader) GetDistributionFormRevision(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (evidence.DistributionFormRevision, error) {
	form, err := reader.forms.ReusableFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return evidence.DistributionFormRevision{}, err
	}
	return evidence.DistributionFormRevision{
		ID: form.ID, TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, Version: form.Version,
		Sensitivity: form.Sensitivity, Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile,
		Sections: append([]formcontract.Section(nil), form.Sections...), Fields: append([]formcontract.Field(nil), form.Fields...),
		Active: form.Status == monitoring.LifecycleActive && form.IsCurrent,
	}, nil
}

type vendorWorkEvidenceRepository struct {
	*evidence.MemoryRepository
}

func (r *vendorWorkEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	if tenant != "bank" || subjectType != "VENDOR_RELATIONSHIP" || subjectID == "" {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: "entity-a", SubjectType: subjectType, SubjectID: subjectID}, nil
}

func newVendorWorkFixture(t *testing.T) vendorWorkFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	links := NewMemoryRelationshipLinkRepository()
	links.AllowRelationship(actor.TenantID, actor.LegalEntityID, "relationship-1")
	links.AllowTarget(actor.TenantID, actor.LegalEntityID, LinkTargetProgram, "program-1")
	linkService := NewRelationshipLinkService(links)
	linkService.now = func() time.Time { return now }
	linkService.newID = func() (string, error) { return "link-1", nil }
	link, err := linkService.Link(context.Background(), actor, "relationship-1", LinkRelationshipInput{
		TargetType: LinkTargetProgram, TargetID: "program-1", PurposeCode: "EVIDENCE_PROVIDER", PurposeLabel: "Evidence provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	forms := monitoring.NewMemoryRepository()
	_, err = forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{
		ID: "form-1", TenantID: "bank", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "VENDOR-CONTROL", Name: "Quarterly service confirmation", Purpose: "Confirm current service and control information.",
		Sensitivity:  "CONFIDENTIAL",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "service", Title: "Service"}},
		Fields:       []monitoring.TemplateField{{ID: "service_current", SectionID: "service", Label: "Is the service information current?", Type: formcontract.TypeYesNo, Required: true}},
		Lifecycle:    monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	evidenceRepository := &vendorWorkEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil)}
	evidenceService := evidence.NewServiceWithClock(evidenceRepository, evidence.NewMemoryObjectStore(), func() time.Time { return now })
	var recipientKey [32]byte
	var accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index], accessKey[index] = 0x31, 0x42
	}
	keyring, err := evidence.NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	distributionStore := evidence.NewMemoryDistributionStore(evidenceRepository.MemoryRepository, vendorWorkDistributionFormReader{forms: forms}, keyring)
	distributions := evidence.NewDistributionService(distributionStore)
	accessStore := evidence.NewMemoryDistributionAccessStore(distributionStore)
	otp := &vendorWorkOTPDelivery{}
	access, err := evidence.NewDistributionAccessService(accessStore, keyring, otp, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := evidence.NewWorkflowDistributionDispatcher(distributions, access)
	repository := NewMemoryVendorWorkRepository()
	service, err := NewVendorWorkService(repository, links, evidenceService, forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
	service.ConfigureDistributionDispatcher(dispatcher)
	relationships := NewMemoryRepository()
	_, err = relationships.CreateRelationship(context.Background(), CreateRecord{
		Vendor:       Vendor{ID: "vendor-1", TenantID: "bank", LegalName: "Northstar Hosting Limited", Status: VendorActive, Version: 1, CreatedAt: now, UpdatedAt: now},
		Relationship: Relationship{ID: "relationship-1", TenantID: "bank", LegalEntityID: "entity-a", VendorID: "vendor-1", ServiceName: "Managed transaction screening", Status: RelationshipActive, Version: 1, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureRelationshipReader(relationships)
	ids := []string{"work-1", "request-1", "request-link-1", "request-2", "request-link-2"}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	return vendorWorkFixture{service: service, repository: repository, links: links, forms: forms, evidence: evidenceService, access: access, dispatcher: dispatcher, otp: otp, link: link, actor: actor, audience: "security@vendor.example", now: now}
}

func TestCertificationRefreshRequiresActiveRelationship(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	addVendorWorkForm(t, fixture, "certification-form", "VENDOR-CERTIFICATION-REFRESH", []monitoring.TemplateField{{
		ID: "certificate", SectionID: "evidence", Label: "Certification evidence", Type: formcontract.TypeVendorDocument, Required: true,
	}})
	fixture.service.relationships.(*MemoryRepository).mu.Lock()
	relationship := fixture.service.relationships.(*MemoryRepository).relationships["relationship-1"]
	relationship.Status = RelationshipProposed
	fixture.service.relationships.(*MemoryRepository).relationships[relationship.ID] = relationship
	fixture.service.relationships.(*MemoryRepository).mu.Unlock()

	_, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID,
		RequestKind: VendorWorkCertificationRefresh, Purpose: "Collect current certification evidence.",
		Instructions:   "Provide the current ISO 27001 and PCI DSS evidence that applies.",
		FormTemplateID: "certification-form", FormTemplateVersion: 1, VendorAudience: fixture.audience,
		DueAt: fixture.now.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrRelationshipNotActive) {
		t.Fatalf("certification prepare error = %v, want ErrRelationshipNotActive", err)
	}
}

func addVendorWorkForm(t *testing.T, fixture vendorWorkFixture, id, code string, fields []monitoring.TemplateField, customSections ...[]formcontract.Section) {
	t.Helper()
	sections := []formcontract.Section{{ID: "evidence", Title: "Evidence"}}
	if len(customSections) > 0 {
		sections = customSections[0]
	}
	_, err := fixture.forms.CreateFormRevision(context.Background(), monitoring.FormTemplate{
		ID: id, TenantID: "bank", LegalEntityID: "entity-a", ProgramID: "program-1", Code: code, Name: code, Purpose: "Collect the evidence required for this vendor request.",
		Sensitivity:  "CONFIDENTIAL",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     sections, Fields: fields,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestVendorWorkLifecyclePreservesCaptureAndReviewBoundaries(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID,
		Purpose: "Confirm the service information needed for the Program review.", Instructions: "Review the known service details and correct anything that changed.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, Presentation: formcontract.PresentationClassic, VendorAudience: fixture.audience, DueAt: fixture.now.Add(7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.State != VendorWorkPreparing || prepared.CurrentRequestID == "" || prepared.TargetType != LinkTargetProgram || prepared.TargetID != "program-1" || prepared.OwnerPrincipalID != fixture.actor.PrincipalID || prepared.Presentation != formcontract.PresentationClassic {
		t.Fatalf("prepared work = %#v", prepared)
	}
	capture, err := fixture.evidence.GetRequest(context.Background(), "bank", prepared.CurrentRequestID)
	if err != nil {
		t.Fatal(err)
	}
	if capture.KnownFacts["Vendor"] != "Northstar Hosting Limited" || capture.KnownFacts["Service"] != "Managed transaction screening" || capture.KnownFacts["target_id"] != "" || capture.KnownFacts["target_type"] != "" {
		t.Fatalf("capture known facts = %#v", capture.KnownFacts)
	}

	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if sent.Work.State != VendorWorkAwaitingVendor || sent.Work.CurrentInvitationID == "" || sent.CaptureURL == "" || sent.State != VendorWorkDeliveryLinkAvailable {
		t.Fatalf("sent outcome = %#v", sent)
	}
	session := redeemVendorWorkAccess(t, fixture, sent.CaptureURL)
	result := submitVendorWorkAnswers(t, fixture, session.SessionToken, map[string]formcontract.AnswerValue{"service_current": formcontract.TextAnswer("Yes")})
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{
		TenantID: "bank", WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: result.Submission.SubmissionID, CausationID: "event-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if received.State != VendorWorkResponseReceived || received.SubmissionID != result.Submission.SubmissionID {
		t.Fatalf("received work = %#v", received)
	}
	view, err := fixture.service.Response(context.Background(), fixture.actor, received.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Response.SubmissionID != result.Submission.SubmissionID || len(view.Answers) != 1 || view.Answers[0].Value == nil {
		t.Fatalf("response view = %#v", view)
	}
	if answer, ok := view.Answers[0].Value.ScalarText(); !ok || answer != "Yes" {
		t.Fatalf("response answer = %#v", view.Answers[0])
	}
	if _, err := fixture.service.Response(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "unassigned-user"}, received.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unassigned response read = %v", err)
	}

	reviewing, err := fixture.service.StartReview(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, received.ID, StartVendorWorkReviewInput{ExpectedVersion: received.Version})
	if err != nil {
		t.Fatal(err)
	}
	if reviewing.State != VendorWorkUnderReview || reviewing.ReviewerPrincipalID != "reviewer-1" {
		t.Fatalf("reviewing work = %#v", reviewing)
	}

	changes, err := fixture.service.RequestChanges(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, RequestVendorWorkChangesInput{
		ExpectedVersion: reviewing.Version, Message: "Confirm the updated service owner.", FieldIDs: []string{"service_current"},
		VendorAudience: fixture.audience, DueAt: fixture.now.Add(5 * 24 * time.Hour), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if changes.Work.State != VendorWorkChangesRequested || changes.Work.CurrentRequestID == prepared.CurrentRequestID || changes.Work.CurrentCaptureSequence != 2 || changes.CaptureURL == "" {
		t.Fatalf("changes outcome = %#v", changes)
	}

	clarificationSession := redeemVendorWorkAccess(t, fixture, changes.CaptureURL)
	clarificationResult := submitVendorWorkAnswers(t, fixture, clarificationSession.SessionToken, map[string]formcontract.AnswerValue{"service_current": formcontract.TextAnswer("No")})
	received, err = fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{
		TenantID: "bank", WorkRequestID: prepared.ID, RequestID: changes.Work.CurrentRequestID, SubmissionID: clarificationResult.Submission.SubmissionID, CausationID: "event-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	reviewing, err = fixture.service.StartReview(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, received.ID, StartVendorWorkReviewInput{ExpectedVersion: received.Version})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := fixture.service.Accept(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, reviewing.ID, AcceptVendorWorkInput{ExpectedVersion: reviewing.Version, Rationale: "The response addresses the requested service confirmation."})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.State != VendorWorkAccepted || accepted.AcceptedAt == nil {
		t.Fatalf("accepted work = %#v", accepted)
	}

	page, err := fixture.service.List(context.Background(), fixture.actor, VendorWorkListInput{RelationshipID: "relationship-1", TargetType: LinkTargetProgram, TargetID: "program-1", Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != accepted.ID {
		t.Fatalf("work page=%#v err=%v", page, err)
	}
}

func TestVendorWorkCanonicalOTPLinkReopensUntilExpiryAndSubmitsAfterAutosave(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID,
		Purpose: "Confirm the service information needed for the Program review.", Instructions: "Review the known service details and correct anything that changed.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{
		ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	selector := vendorWorkCaptureToken(t, sent.CaptureURL)
	first := redeemVendorWorkAccess(t, fixture, sent.CaptureURL)
	if _, err := fixture.access.StartDistributionAccess(context.Background(), selector); err != nil {
		t.Fatalf("unexpired vendor-work selector could not be reopened: %v", err)
	}
	second := redeemVendorWorkAccess(t, fixture, sent.CaptureURL)
	if first.SessionID == second.SessionID || first.SessionToken == second.SessionToken {
		t.Fatal("reopening the vendor-work link reused the prior bounded session")
	}
	result := submitVendorWorkAnswers(t, fixture, second.SessionToken, map[string]formcontract.AnswerValue{
		"service_current": formcontract.TextAnswer("Yes"),
	})
	if result.Submission.SubmissionID == "" {
		t.Fatalf("canonical OTP submission did not produce a receipt: %#v", result)
	}
}

func TestVendorWorkRejectsUnknownRequestKindBeforePersistence(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	_, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, RequestKind: VendorWorkRequestKind("UNBOUNDED_ACTION"),
		Purpose: "Confirm the registered address.", Instructions: "Check the address and provide evidence.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v", err)
	}
	if len(fixture.repository.work) != 0 {
		t.Fatal("invalid request kind reached persistence")
	}
}

func TestVendorWorkRejectsSpecialGovernedFormForGeneralRequest(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	addVendorWorkForm(t, fixture, "address-form", "VENDOR-ADDRESS-VERIFICATION", []monitoring.TemplateField{{ID: "address_result", SectionID: "evidence", Label: "Address result", Type: formcontract.TypeYesNo, Required: true}})
	_, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, RequestKind: VendorWorkGeneral,
		Purpose: "Confirm the service information.", Instructions: "Review the service information.",
		FormTemplateID: "address-form", FormTemplateVersion: 1, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("general request with address form error = %v", err)
	}
	if len(fixture.repository.work) != 0 {
		t.Fatal("special governed form reached general request persistence")
	}
}

func TestCertificationChangeRequestIncludesConditionalApplicabilityField(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	addVendorWorkForm(t, fixture, "certification-form", "VENDOR-CERTIFICATION-REFRESH", []monitoring.TemplateField{
		{ID: "certifications_applicable", SectionID: "scope", Label: "Are certifications required?", Type: formcontract.TypeYesNo, Required: true},
		{ID: "iso_applicable", SectionID: "certifications", Label: "Does ISO 27001 apply?", Type: formcontract.TypeYesNo, Required: true},
		{ID: "iso_certificate", SectionID: "certifications", Label: "Current ISO 27001 certificate", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Condition: &formcontract.VisibilityCondition{FieldID: "iso_applicable", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
		{ID: "pci_applicable", SectionID: "certifications", Label: "Does PCI DSS apply?", Type: formcontract.TypeYesNo, Required: true},
		{ID: "pci_attestation", SectionID: "certifications", Label: "Current PCI DSS attestation", Type: formcontract.TypeVendorDocument, Required: true, AcceptedFormats: []string{"application/pdf"}, Condition: &formcontract.VisibilityCondition{FieldID: "pci_applicable", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
	}, []formcontract.Section{
		{ID: "scope", Title: "Scope"},
		{ID: "certifications", Title: "Certifications", Condition: &formcontract.VisibilityCondition{FieldID: "certifications_applicable", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
	})
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, RequestKind: VendorWorkCertificationRefresh,
		Purpose: "Provide a current ISO 27001 certificate.", Instructions: "Confirm applicability and upload the current certificate.",
		FormTemplateID: "certification-form", FormTemplateVersion: 1, VendorAudience: fixture.audience, DueAt: fixture.now.Add(7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	initialSession := redeemVendorWorkAccess(t, fixture, sent.CaptureURL)
	isoArtifact, err := fixture.evidence.StoreArtifactForDistributionSession(context.Background(), fixture.access, initialSession.SessionToken, evidence.ArtifactInput{TenantID: "bank", RequestID: prepared.CurrentRequestID, FieldID: "iso_certificate", FileName: "iso-original.pdf", MediaType: "application/pdf"}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	pciArtifact, err := fixture.evidence.StoreArtifactForDistributionSession(context.Background(), fixture.access, initialSession.SessionToken, evidence.ArtifactInput{TenantID: "bank", RequestID: prepared.CurrentRequestID, FieldID: "pci_attestation", FileName: "pci-current.pdf", MediaType: "application/pdf"}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	initialResult := submitVendorWorkAnswers(t, fixture, initialSession.SessionToken, map[string]formcontract.AnswerValue{
		"certifications_applicable": formcontract.TextAnswer("Yes"),
		"iso_applicable":            formcontract.TextAnswer("Yes"),
		"pci_applicable":            formcontract.TextAnswer("Yes"),
		"iso_certificate":           {Document: &formcontract.DocumentAnswer{ArtifactID: isoArtifact.ID, DocumentType: "ISO_27001"}},
		"pci_attestation":           {Document: &formcontract.DocumentAnswer{ArtifactID: pciArtifact.ID, DocumentType: "PCI_DSS"}},
	})
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: "bank", WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: initialResult.Submission.SubmissionID, CausationID: "certification-submission"})
	if err != nil {
		t.Fatal(err)
	}
	reviewer := Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}
	reviewing, err := fixture.service.StartReview(context.Background(), reviewer, received.ID, StartVendorWorkReviewInput{ExpectedVersion: received.Version})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := fixture.service.RequestChanges(context.Background(), reviewer, reviewing.ID, RequestVendorWorkChangesInput{
		ExpectedVersion: reviewing.Version, Message: "Replace the ISO 27001 certificate.", FieldIDs: []string{"iso_certificate"},
		VendorAudience: fixture.audience, DueAt: fixture.now.Add(5 * 24 * time.Hour), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sent.Work.CurrentRequestID == changes.Work.CurrentRequestID {
		t.Fatal("change request did not create a bounded replacement collection")
	}
	request, err := fixture.evidence.GetRequest(context.Background(), "bank", changes.Work.CurrentRequestID)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Fields) != 3 || request.Fields[0].ID != "certifications_applicable" || request.Fields[1].ID != "iso_applicable" || request.Fields[2].ID != "iso_certificate" {
		t.Fatalf("change request fields = %#v", request.Fields)
	}
	replacementSession := redeemVendorWorkAccess(t, fixture, changes.CaptureURL)
	replacementArtifact, err := fixture.evidence.StoreArtifactForDistributionSession(context.Background(), fixture.access, replacementSession.SessionToken, evidence.ArtifactInput{TenantID: "bank", RequestID: request.ID, FieldID: "iso_certificate", FileName: "iso-replacement.pdf", MediaType: "application/pdf"}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	replacementResult := submitVendorWorkAnswers(t, fixture, replacementSession.SessionToken, map[string]formcontract.AnswerValue{
		"certifications_applicable": formcontract.TextAnswer("Yes"),
		"iso_applicable":            formcontract.TextAnswer("Yes"),
		"iso_certificate":           {Document: &formcontract.DocumentAnswer{ArtifactID: replacementArtifact.ID, DocumentType: "ISO_27001"}},
	})
	received, err = fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: "bank", WorkRequestID: prepared.ID, RequestID: request.ID, SubmissionID: replacementResult.Submission.SubmissionID, CausationID: "certification-replacement"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := fixture.service.Response(context.Background(), fixture.actor, received.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Documents) != 2 {
		t.Fatalf("merged certification documents = %#v", view.Documents)
	}
	documents := map[string]AssessmentReviewDocument{}
	for _, document := range view.Documents {
		documents[document.FieldID] = document
	}
	if documents["iso_certificate"].RequestID != request.ID || documents["iso_certificate"].FileName != "iso-replacement.pdf" || documents["pci_attestation"].RequestID != prepared.CurrentRequestID || documents["pci_attestation"].FileName != "pci-current.pdf" {
		t.Fatalf("merged certification basis = %#v", documents)
	}
}

const testVendorWorkPDF = "%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"

func TestCancelVendorWorkRevokesCurrentInvitationAndRedeemedSession(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	token := vendorWorkCaptureToken(t, sent.CaptureURL)
	session := redeemVendorWorkAccess(t, fixture, sent.CaptureURL)
	cancelled, err := fixture.service.Cancel(context.Background(), fixture.actor, sent.Work.ID, CancelVendorWorkInput{ExpectedVersion: sent.Work.Version, Reason: "The Program no longer needs this vendor response."})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != VendorWorkCancelled || cancelled.CancellationReason == "" {
		t.Fatalf("cancelled work = %#v", cancelled)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), token); !errors.Is(err, evidence.ErrDistributionAccessUnavailable) {
		t.Fatalf("cancelled invitation remained usable: %v", err)
	}
	if _, _, err := fixture.access.SessionRequest(context.Background(), session.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("cancelled session remained usable: %v", err)
	}
}

func TestRetryVendorWorkReissuesAndRevokesPreviousCapability(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := fixture.service.Send(context.Background(), fixture.actor, prepared.ID, SendVendorWorkInput{ExpectedVersion: prepared.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	firstToken := vendorWorkCaptureToken(t, first.CaptureURL)
	retried, err := fixture.service.Retry(context.Background(), fixture.actor, prepared.ID, RetryVendorWorkInput{ExpectedVersion: first.Work.Version, VendorAudience: fixture.audience, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if retried.Work.CurrentInvitationID == first.Work.CurrentInvitationID || retried.CaptureURL == "" || retried.Work.Version <= first.Work.Version {
		t.Fatalf("retry outcome = %#v", retried)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), firstToken); !errors.Is(err, evidence.ErrDistributionAccessUnavailable) {
		t.Fatalf("previous capability error = %v", err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), vendorWorkCaptureToken(t, retried.CaptureURL)); err != nil {
		t.Fatalf("replacement capability: %v", err)
	}
}

func TestEndRelationshipLinkConflictsWhileVendorWorkIsActive(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	prepared, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.", FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	links := NewRelationshipLinkService(fixture.links)
	links.ConfigureActiveWorkGuard(fixture.repository)
	if _, err := links.End(context.Background(), fixture.actor, fixture.link.ID, EndRelationshipLinkInput{ExpectedVersion: fixture.link.Version, Reason: "This association is no longer needed."}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("end link with active work %s: %v", prepared.ID, err)
	}
}

func TestPrepareVendorWorkReturnsRecoverableStateAndResumesCapture(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	fixture.service.dispatch = &vendorWorkDispatcherFailure{vendorWorkDispatcher: fixture.dispatcher, dispatchFailures: 1}
	input := PrepareVendorWorkInput{RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, Purpose: "Confirm service information.", Instructions: "Review the request.", FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour)}
	first, err := fixture.service.Prepare(context.Background(), fixture.actor, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.CurrentRequestID != "" || first.DeliveryState != VendorWorkDeliveryRetryRequired || first.Recovery == "" {
		t.Fatalf("recoverable prepare = %#v", first)
	}
	resumed, err := fixture.service.Prepare(context.Background(), fixture.actor, input)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ID != first.ID || resumed.CurrentRequestID == "" || resumed.Recovery != "" || resumed.State != VendorWorkPreparing {
		t.Fatalf("resumed prepare = %#v", resumed)
	}
}

type vendorWorkDispatcherFailure struct {
	vendorWorkDispatcher
	dispatchFailures int
}

func (f *vendorWorkDispatcherFailure) Dispatch(ctx context.Context, input evidence.WorkflowDistributionDispatchInput) (evidence.WorkflowDistributionDispatch, error) {
	if f.dispatchFailures > 0 {
		f.dispatchFailures--
		return evidence.WorkflowDistributionDispatch{}, errors.New("capture store unavailable")
	}
	return f.vendorWorkDispatcher.Dispatch(ctx, input)
}

func vendorWorkCaptureToken(t *testing.T, raw string) string {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	fragment, err := url.ParseQuery(value.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	token := fragment.Get("form_access")
	if token == "" {
		t.Fatalf("capture URL has no token: %q", raw)
	}
	return token
}

func redeemVendorWorkAccess(t *testing.T, fixture vendorWorkFixture, rawURL string) evidence.RedeemedDistributionSession {
	t.Helper()
	selector := vendorWorkCaptureToken(t, rawURL)
	start, err := fixture.access.StartDistributionAccess(context.Background(), selector)
	if err != nil || start.Policy != evidence.AccessDirectEmailOTP || len(start.Recipients) != 1 {
		t.Fatalf("start vendor-work access = (%#v, %v)", start, err)
	}
	receipt, err := fixture.access.SendOTP(context.Background(), selector, start.Recipients[0].SelectorID)
	if err != nil || len(fixture.otp.values) == 0 {
		t.Fatalf("send vendor-work OTP = (%#v, %v)", receipt, err)
	}
	delivered := fixture.otp.values[len(fixture.otp.values)-1]
	redeemed, err := fixture.access.VerifyOTP(context.Background(), selector, receipt.ChallengeID, delivered.Code)
	if err != nil {
		t.Fatalf("verify vendor-work OTP = %v", err)
	}
	return redeemed
}

func submitVendorWorkAnswers(t *testing.T, fixture vendorWorkFixture, sessionToken string, answers map[string]formcontract.AnswerValue) evidence.WorkspaceSubmissionResult {
	t.Helper()
	view, err := fixture.access.GetResponseWorkspace(context.Background(), sessionToken)
	if err != nil {
		t.Fatal(err)
	}
	edits := make([]evidence.FieldEdit, 0, len(answers))
	for fieldID, value := range answers {
		edits = append(edits, evidence.FieldEdit{FieldID: fieldID, Value: value, BaseSequence: view.FieldSequences[fieldID]})
	}
	saved, err := fixture.access.SaveResponseWorkspace(context.Background(), sessionToken, evidence.SaveWorkspaceInput{
		ExpectedVersion: view.Workspace.Version, PresentationMode: view.PresentationMode, Edits: edits,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.access.SubmitResponseWorkspace(context.Background(), sessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
