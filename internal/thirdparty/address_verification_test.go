package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestAddressVerificationProvisionerCreatesOneCanonicalInternalJourney(t *testing.T) {
	now := time.Date(2026, 9, 2, 11, 0, 0, 0, time.UTC)
	assessments := addressVerificationAssessmentFixture(t, now)
	matters := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	requests := newAddressVerificationEvidenceRepository([]evidence.Request{{
		ID: "registration-request", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1}, Status: evidence.RequestSubmitted,
	}})
	evidenceService := evidence.NewServiceWithClock(requests, evidence.NewMemoryObjectStore(), func() time.Time { return now })
	forms := addressVerificationForms(t)
	inbox := workflowruntime.NewMemoryRepository()
	provisioner := NewAddressVerificationProvisioner(inbox, requests, assessments, matters, evidenceService, forms)
	provisioner.now = func() time.Time { return now }
	event := addressVerificationRegistrationEvent(now)

	if err := provisioner.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := provisioner.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	matter, err := matters.MatterByTriggerKey(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", "thirdparty-address-verification:assessment-1")
	if err != nil {
		t.Fatal(err)
	}
	if matter.Matter.Type != continuity.MatterVendorReview || len(matter.Actions) != 1 || len(matter.VerificationContracts) != 1 {
		t.Fatalf("address verification matter = %#v", matter)
	}
	action := matter.Actions[0]
	if action.Status != continuity.ActionPlanned || action.OwnerPrincipalID != "" || matter.VerificationContracts[0].ActionID != action.ID {
		t.Fatalf("address verification action = %#v contract = %#v", action, matter.VerificationContracts[0])
	}
	if _, err := requests.GetRequestByOrigin(context.Background(), "bank", evidence.RequestOrigin{Type: AddressVerificationRequestOrigin, ID: action.ID, Version: action.Version}); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("unassigned action request error = %v, want not found", err)
	}
	allMatters, err := matters.ListMatters(continuity.WithTrustedSystemScope(context.Background()), "bank", "", 20)
	if err != nil || len(allMatters) != 1 {
		t.Fatalf("matter count = %d, err = %v", len(allMatters), err)
	}
	allRequests, err := requests.ListRequests(context.Background(), "bank", 20)
	if err != nil || len(allRequests) != 1 {
		t.Fatalf("request count = %d, err = %v", len(allRequests), err)
	}

	configureAddressVerificationCandidates(t, requests, matter.Matter.ID)
	assigned, assignmentEvent := assignAddressVerificationAction(t, matters, matter, "staff-1", "Assign the current verifier.", now.Add(time.Minute))
	consumer := NewAddressVerificationAssignmentConsumer(inbox, assessments, matters, evidenceService, forms)
	if err := consumer.Publish(context.Background(), assignmentEvent); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), assignmentEvent); err != nil {
		t.Fatal(err)
	}
	updated, err := matters.GetMatter(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	currentAction, _ := addressVerificationAction(updated.Actions)
	if currentAction.Status != continuity.ActionInProgress || currentAction.OwnerPrincipalID != "staff-1" {
		t.Fatalf("assigned address verification action = %#v", currentAction)
	}
	request, err := requests.GetRequestByOrigin(context.Background(), "bank", evidence.RequestOrigin{Type: AddressVerificationRequestOrigin, ID: assigned.ID, Version: assigned.Version})
	if err != nil {
		t.Fatal(err)
	}
	if request.SubjectType != "MATTER" || request.SubjectID != matter.Matter.ID || request.AudienceType != "INTERNAL" || request.Recipient.Type != evidence.RecipientInternalPrincipal || request.Recipient.PrincipalID != "staff-1" || request.FormTemplateID != "address-form" {
		t.Fatalf("address verification request = %#v", request)
	}
	resolver := NewAddressVerificationAssignmentTargetResolver(requests)
	target, err := resolver.ResolveAssignmentNotificationTarget(context.Background(), workflow.AssignmentNotificationTarget{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: assigned.ID, ActionVersion: assigned.Version, PrincipalID: "staff-1",
	})
	if err != nil || target != "/#work/evidence/"+request.ID {
		t.Fatalf("assignment target = %q, err = %v", target, err)
	}
}

func TestAddressVerificationSubmissionImplementsActionButDoesNotVerifyOrCloseMatter(t *testing.T) {
	now := time.Date(2026, 9, 2, 11, 0, 0, 0, time.UTC)
	assessments := addressVerificationAssessmentFixture(t, now)
	matters := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	requests := newAddressVerificationEvidenceRepository([]evidence.Request{{
		ID: "registration-request", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1}, Status: evidence.RequestSubmitted,
	}})
	evidenceService := evidence.NewServiceWithClock(requests, evidence.NewMemoryObjectStore(), func() time.Time { return now })
	inbox := workflowruntime.NewMemoryRepository()
	provisioner := NewAddressVerificationProvisioner(inbox, requests, assessments, matters, evidenceService, addressVerificationForms(t))
	provisioner.now = func() time.Time { return now }
	if err := provisioner.Publish(context.Background(), addressVerificationRegistrationEvent(now)); err != nil {
		t.Fatal(err)
	}
	matter, err := matters.MatterByTriggerKey(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", "thirdparty-address-verification:assessment-1")
	if err != nil {
		t.Fatal(err)
	}
	configureAddressVerificationCandidates(t, requests, matter.Matter.ID)
	assigned, assignmentEvent := assignAddressVerificationAction(t, matters, matter, "staff-1", "Assign the current verifier.", now.Add(time.Minute))
	assignmentConsumer := NewAddressVerificationAssignmentConsumer(inbox, assessments, matters, evidenceService, addressVerificationForms(t))
	if err := assignmentConsumer.Publish(context.Background(), assignmentEvent); err != nil {
		t.Fatal(err)
	}
	request, err := requests.GetRequestByOrigin(context.Background(), "bank", evidence.RequestOrigin{Type: AddressVerificationRequestOrigin, ID: assigned.ID, Version: assigned.Version})
	if err != nil {
		stored, _ := requests.ListRequests(context.Background(), "bank", 20)
		t.Fatalf("get address request for action %#v: %v; stored=%#v", assigned, err, stored)
	}
	receipt, err := evidenceService.Submit(context.Background(), evidence.Submission{
		TenantID: "bank", LegalEntityID: "entity-a", RequestID: request.ID, SubmittedBy: "staff-1", Channel: "INTERNAL",
		Answers: map[string]formcontract.AnswerValue{"address_result": formcontract.TextAnswer("Yes")}, ExpectedVersion: request.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	consumer := NewAddressVerificationSubmissionConsumer(inbox, requests, matters)
	payload, _ := json.Marshal(map[string]string{"submission_id": receipt.SubmissionID, "channel": "INTERNAL"})
	event := workflowruntime.OutboxEvent{ID: "address-event", TenantID: "bank", AggregateType: "EVIDENCE_REQUEST", AggregateID: request.ID, EventType: "EvidenceResponseSubmitted", Payload: payload, OccurredAt: now.Add(time.Minute)}

	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	updated, err := matters.GetMatter(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Actions[0].Status != continuity.ActionImplemented || len(updated.VerificationResults) != 0 || updated.Matter.Status == continuity.MatterClosed {
		t.Fatalf("submission crossed implementation boundary: %#v", updated)
	}
}

func TestAddressVerificationReassignmentKeepsOneRequestForTheCurrentPerformer(t *testing.T) {
	now := time.Date(2026, 9, 2, 11, 0, 0, 0, time.UTC)
	assessments := addressVerificationAssessmentFixture(t, now)
	matters := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	requests := newAddressVerificationEvidenceRepository([]evidence.Request{{
		ID: "registration-request", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1",
		Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: "assessment-1", Version: 1}, Status: evidence.RequestSubmitted,
	}})
	evidenceService := evidence.NewServiceWithClock(requests, evidence.NewMemoryObjectStore(), func() time.Time { return now })
	forms := addressVerificationForms(t)
	inbox := workflowruntime.NewMemoryRepository()
	provisioner := NewAddressVerificationProvisioner(inbox, requests, assessments, matters, evidenceService, forms)
	if err := provisioner.Publish(context.Background(), addressVerificationRegistrationEvent(now)); err != nil {
		t.Fatal(err)
	}
	matter, err := matters.MatterByTriggerKey(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", "thirdparty-address-verification:assessment-1")
	if err != nil {
		t.Fatal(err)
	}
	configureAddressVerificationCandidates(t, requests, matter.Matter.ID)
	firstAction, firstEvent := assignAddressVerificationAction(t, matters, matter, "staff-1", "Assign the current verifier.", now.Add(time.Minute))
	consumer := NewAddressVerificationAssignmentConsumer(inbox, assessments, matters, evidenceService, forms)
	if err := consumer.Publish(context.Background(), firstEvent); err != nil {
		t.Fatal(err)
	}
	firstRequest, err := requests.GetRequestByOrigin(context.Background(), "bank", evidence.RequestOrigin{Type: AddressVerificationRequestOrigin, ID: firstAction.ID, Version: firstAction.Version})
	if err != nil {
		t.Fatal(err)
	}
	current, err := matters.GetMatter(continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-a"), "bank", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondAction, secondEvent := assignAddressVerificationAction(t, matters, current, "staff-2", "Cover the verifier's approved absence.", now.Add(2*time.Minute))
	if err := consumer.Publish(context.Background(), secondEvent); err != nil {
		t.Fatal(err)
	}
	currentRequest, err := requests.GetRequest(context.Background(), "bank", firstRequest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentRequest.ID != firstRequest.ID || currentRequest.Recipient.PrincipalID != "staff-2" || currentRequest.Version <= firstRequest.Version {
		t.Fatalf("reassigned address request = %#v; first = %#v", currentRequest, firstRequest)
	}
	if secondAction.OwnerPrincipalID != "staff-2" {
		t.Fatalf("reassigned action = %#v", secondAction)
	}
	resolver := NewAddressVerificationAssignmentTargetResolver(requests)
	target, err := resolver.ResolveAssignmentNotificationTarget(context.Background(), workflow.AssignmentNotificationTarget{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: secondAction.ID, ActionVersion: secondAction.Version, PrincipalID: "staff-2",
	})
	if err != nil || target != "/#work/evidence/"+firstRequest.ID {
		t.Fatalf("reassigned target = %q, err = %v", target, err)
	}
	allRequests, err := requests.ListRequests(context.Background(), "bank", 20)
	if err != nil || len(allRequests) != 2 {
		t.Fatalf("request count after reassignment = %d, err = %v", len(allRequests), err)
	}
}

func assignAddressVerificationAction(t *testing.T, matters *continuity.Service, aggregate continuity.MatterAggregate, owner, rationale string, occurredAt time.Time) (continuity.Action, workflowruntime.OutboxEvent) {
	t.Helper()
	matter := aggregate.Matter
	action, found := addressVerificationAction(aggregate.Actions)
	if !found {
		t.Fatal("address verification action is unavailable")
	}
	previousOwner := action.OwnerPrincipalID
	updated, err := matters.AssignAction(continuity.WithTrustedSystemEntityScope(context.Background(), matter.TenantID, matter.LegalEntityID), continuity.AssignActionInput{
		TenantID: matter.TenantID, MatterID: matter.ID, ActionID: action.ID, ExpectedVersion: matter.Version,
		OwnerPrincipalID: owner, ActorID: "owner-1", Rationale: rationale,
	})
	if err != nil {
		t.Fatal(err)
	}
	action, found = addressVerificationAction(updated.Actions)
	if !found {
		t.Fatal("assigned address verification action is unavailable")
	}
	payload, err := json.Marshal(map[string]any{
		"action": action, "previous_owner_principal_id": previousOwner, "owner_principal_id": owner, "rationale": rationale,
	})
	if err != nil {
		t.Fatal(err)
	}
	return action, workflowruntime.OutboxEvent{ID: "assignment-" + owner, TenantID: matter.TenantID, AggregateType: "MATTER", AggregateID: matter.ID, EventType: continuity.EventActionAssigned, Payload: payload, OccurredAt: occurredAt}
}

func addressVerificationAssessmentFixture(t *testing.T, now time.Time) *MemoryAssessmentRepository {
	t.Helper()
	repository := NewMemoryAssessmentRepository()
	_, err := repository.CreateRelationship(context.Background(), CreateRecord{
		Vendor:       Vendor{ID: "vendor-1", TenantID: "bank", LegalName: "Northstar Hosting Limited", RegisteredAddress: "1 Example Street", Status: VendorActive, Version: 1, CreatedAt: now, UpdatedAt: now},
		Relationship: Relationship{ID: "relationship-1", TenantID: "bank", LegalEntityID: "entity-a", VendorID: "vendor-1", ServiceName: "Transaction screening", BusinessOwnerPrincipalID: "owner-1", Criticality: CriticalityImportant, PrivacyRole: PrivacyProcessor, Status: RelationshipProposed, Version: 1, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	repository.assessments["assessment-1"] = Assessment{ID: "assessment-1", TenantID: "bank", LegalEntityID: "entity-a", RelationshipID: "relationship-1", ReviewKind: AssessmentReviewOnboarding, Status: AssessmentSubmitted, ReviewDueAt: now.Add(14 * 24 * time.Hour), Version: 5, CreatedAt: now.Add(-time.Hour), UpdatedAt: now}
	repository.requestLinks["assessment-1"] = []AssessmentRequestLink{{
		TenantID: "bank", LegalEntityID: "entity-a", AssessmentID: "assessment-1", RequestID: "registration-request",
		Purpose: AssessmentRequestInitial, Sequence: 1, OriginType: AssessmentRequestOrigin, OriginID: "assessment-1", OriginSequence: 1,
	}}
	return repository
}

func addressVerificationForms(t *testing.T) *monitoring.MemoryRepository {
	t.Helper()
	repository := monitoring.NewMemoryRepository()
	_, err := repository.CreateFormRevision(context.Background(), monitoring.FormTemplate{
		ID: "address-form", TenantID: "bank", LegalEntityID: "entity-a", Code: "VENDOR-ADDRESS-VERIFICATION", Name: "Verify vendor address", Purpose: "Confirm the vendor registered address using current evidence.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}, Sections: []formcontract.Section{{ID: "address", Title: "Address verification"}},
		Fields:    []monitoring.TemplateField{{ID: "address_result", SectionID: "address", Label: "Does the observed address match the registered address?", Type: formcontract.TypeYesNo, Required: true}},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func addressVerificationRegistrationEvent(now time.Time) workflowruntime.OutboxEvent {
	payload, _ := json.Marshal(map[string]string{"submission_id": "registration-submission", "channel": "EXTERNAL"})
	return workflowruntime.OutboxEvent{ID: "registration-event", TenantID: "bank", AggregateType: "EVIDENCE_REQUEST", AggregateID: "registration-request", EventType: "EvidenceResponseSubmitted", Payload: payload, OccurredAt: now}
}

type addressVerificationEvidenceRepository struct {
	*evidence.MemoryRepository
}

func newAddressVerificationEvidenceRepository(requests []evidence.Request) *addressVerificationEvidenceRepository {
	return &addressVerificationEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, requests)}
}

func configureAddressVerificationCandidates(t *testing.T, repository *addressVerificationEvidenceRepository, matterID string) {
	t.Helper()
	requests, err := repository.ListRequests(context.Background(), "bank", 20)
	if err != nil {
		t.Fatal(err)
	}
	subjects := map[string]bool{"MATTER:" + matterID: true}
	repository.MemoryRepository = evidence.NewMemoryRepositoryWithRecipientCandidates(nil, requests, []evidence.RecipientCandidate{
		{PrincipalID: "owner-1", DisplayName: "Program Owner", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true, ReadableSubjects: subjects},
		{PrincipalID: "staff-1", DisplayName: "Verifier One", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true, ReadableSubjects: subjects},
		{PrincipalID: "staff-2", DisplayName: "Verifier Two", TenantID: "bank", LegalEntityIDs: []string{"entity-a"}, Kind: "PERSON", Active: true, ReadableSubjects: subjects},
	})
}

func (r *addressVerificationEvidenceRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	return evidence.SubjectScope{TenantID: tenant, LegalEntityID: "entity-a", SubjectType: subjectType, SubjectID: subjectID}, nil
}

func (r *addressVerificationEvidenceRepository) CanReadSubject(_ context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	return tenant == "bank" && (principalID == "owner-1" || principalID == "staff-1" || principalID == "staff-2") && subjectType == "MATTER" && subjectID != "", nil
}
