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
	link       RelationshipLink
	actor      Actor
	audience   string
	now        time.Time
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
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
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
	repository := NewMemoryVendorWorkRepository()
	service, err := NewVendorWorkService(repository, links, evidenceService, forms, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
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
	return vendorWorkFixture{service: service, repository: repository, links: links, forms: forms, evidence: evidenceService, link: link, actor: actor, audience: "security@vendor.example", now: now}
}

func TestVendorWorkRejectsDuplicateExternalAddressVerificationPath(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	addVendorWorkForm(t, fixture, "address-form", "VENDOR-ADDRESS-VERIFICATION", []monitoring.TemplateField{{
		ID: "address", SectionID: "evidence", Label: "Registered address", Type: formcontract.TypeLongText, Required: true,
	}})

	_, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID,
		RequestKind: VendorWorkAddressVerification, Purpose: "Verify the registered address.",
		Instructions:   "Record the observed address and supporting evidence.",
		FormTemplateID: "address-form", FormTemplateVersion: 1, VendorAudience: fixture.audience,
		DueAt: fixture.now.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("address verification prepare error = %v, want ErrInvalid", err)
	}
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
	token := vendorWorkCaptureToken(t, sent.CaptureURL)
	session, err := fixture.evidence.RedeemInvitation(context.Background(), token, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := fixture.evidence.SubmitSession(context.Background(), session.SessionToken, map[string]formcontract.AnswerValue{"service_current": formcontract.TextAnswer("Yes")}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{
		TenantID: "bank", WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: receipt.SubmissionID, CausationID: "event-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if received.State != VendorWorkResponseReceived || received.SubmissionID != receipt.SubmissionID {
		t.Fatalf("received work = %#v", received)
	}
	view, err := fixture.service.Response(context.Background(), fixture.actor, received.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Response.SubmissionID != receipt.SubmissionID || len(view.Answers) != 1 || view.Answers[0].Value == nil {
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

	clarificationToken := vendorWorkCaptureToken(t, changes.CaptureURL)
	clarificationSession, err := fixture.evidence.RedeemInvitation(context.Background(), clarificationToken, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	clarificationReceipt, err := fixture.evidence.SubmitSession(context.Background(), clarificationSession.SessionToken, map[string]formcontract.AnswerValue{"service_current": formcontract.TextAnswer("No")}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err = fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{
		TenantID: "bank", WorkRequestID: prepared.ID, RequestID: changes.Work.CurrentRequestID, SubmissionID: clarificationReceipt.SubmissionID, CausationID: "event-2",
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

func TestVendorWorkRejectsRequestKindAndGovernedFormMismatch(t *testing.T) {
	fixture := newVendorWorkFixture(t)
	_, err := fixture.service.Prepare(context.Background(), fixture.actor, PrepareVendorWorkInput{
		RelationshipID: "relationship-1", RelationshipLinkID: fixture.link.ID, RequestKind: VendorWorkAddressVerification,
		Purpose: "Confirm the registered address.", Instructions: "Check the address and provide evidence.",
		FormTemplateID: "form-1", FormTemplateVersion: 3, VendorAudience: fixture.audience, DueAt: fixture.now.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("mismatched address request error = %v", err)
	}
	if len(fixture.repository.work) != 0 {
		t.Fatal("mismatched governed form reached persistence")
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
	initialSession, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, sent.CaptureURL), fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	isoArtifact, err := fixture.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{TenantID: "bank", RequestID: prepared.CurrentRequestID, FieldID: "iso_certificate", FileName: "iso-original.pdf", MediaType: "application/pdf", SessionToken: initialSession.SessionToken}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	pciArtifact, err := fixture.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{TenantID: "bank", RequestID: prepared.CurrentRequestID, FieldID: "pci_attestation", FileName: "pci-current.pdf", MediaType: "application/pdf", SessionToken: initialSession.SessionToken}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	initialReceipt, err := fixture.evidence.SubmitSession(context.Background(), initialSession.SessionToken, map[string]formcontract.AnswerValue{
		"certifications_applicable": formcontract.TextAnswer("Yes"),
		"iso_applicable":            formcontract.TextAnswer("Yes"),
		"pci_applicable":            formcontract.TextAnswer("Yes"),
		"iso_certificate":           {Document: &formcontract.DocumentAnswer{ArtifactID: isoArtifact.ID, DocumentType: "ISO_27001"}},
		"pci_attestation":           {Document: &formcontract.DocumentAnswer{ArtifactID: pciArtifact.ID, DocumentType: "PCI_DSS"}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err := fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: "bank", WorkRequestID: prepared.ID, RequestID: prepared.CurrentRequestID, SubmissionID: initialReceipt.SubmissionID, CausationID: "certification-submission"})
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
	replacementSession, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, changes.CaptureURL), fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	replacementArtifact, err := fixture.evidence.StoreArtifact(context.Background(), evidence.ArtifactInput{TenantID: "bank", RequestID: request.ID, FieldID: "iso_certificate", FileName: "iso-replacement.pdf", MediaType: "application/pdf", SessionToken: replacementSession.SessionToken}, bytes.NewBufferString(testVendorWorkPDF))
	if err != nil {
		t.Fatal(err)
	}
	replacementReceipt, err := fixture.evidence.SubmitSession(context.Background(), replacementSession.SessionToken, map[string]formcontract.AnswerValue{
		"certifications_applicable": formcontract.TextAnswer("Yes"),
		"iso_applicable":            formcontract.TextAnswer("Yes"),
		"iso_certificate":           {Document: &formcontract.DocumentAnswer{ArtifactID: replacementArtifact.ID, DocumentType: "ISO_27001"}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	received, err = fixture.service.RecordSubmission(context.Background(), VendorWorkSubmissionInput{TenantID: "bank", WorkRequestID: prepared.ID, RequestID: request.ID, SubmissionID: replacementReceipt.SubmissionID, CausationID: "certification-replacement"})
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
	session, err := fixture.evidence.RedeemInvitation(context.Background(), token, fixture.audience)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := fixture.service.Cancel(context.Background(), fixture.actor, sent.Work.ID, CancelVendorWorkInput{ExpectedVersion: sent.Work.Version, Reason: "The Program no longer needs this vendor response."})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != VendorWorkCancelled || cancelled.CancellationReason == "" {
		t.Fatalf("cancelled work = %#v", cancelled)
	}
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), token, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("cancelled invitation remained usable: %v", err)
	}
	if _, _, err := fixture.evidence.SessionRequest(context.Background(), session.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
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
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), firstToken, fixture.audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
		t.Fatalf("previous capability error = %v", err)
	}
	if _, err := fixture.evidence.RedeemInvitation(context.Background(), vendorWorkCaptureToken(t, retried.CaptureURL), fixture.audience); err != nil {
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
	fixture.service.evidence = &vendorWorkEvidenceFailure{vendorWorkEvidence: fixture.evidence, createFailures: 1}
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

type vendorWorkEvidenceFailure struct {
	vendorWorkEvidence
	createFailures int
}

func (f *vendorWorkEvidenceFailure) CreateRequest(ctx context.Context, input evidence.CreateRequestInput) (evidence.Request, error) {
	if f.createFailures > 0 {
		f.createFailures--
		return evidence.Request{}, errors.New("capture store unavailable")
	}
	return f.vendorWorkEvidence.CreateRequest(ctx, input)
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
