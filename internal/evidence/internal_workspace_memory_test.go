package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestInternalMemoryWorkspaceReadRechecksNamedRecipient(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	request := Request{ID: "request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestReady, Deadline: now.Add(time.Hour), Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "person", State: RecipientStateAssigned}, Fields: []Field{{ID: "q1", Label: "Access reviewed", Type: "yes_no"}}}
	candidate := RecipientCandidate{PrincipalID: "person", TenantID: "bank", LegalEntityIDs: []string{"entity"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{candidate})
	distributions := NewMemoryDistributionStore(repo, nil, nil)
	distributions.distributions["distribution"] = FormDistribution{ID: "distribution", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionOpen, Deadline: now.Add(time.Hour)}
	distributions.workspaces["distribution"] = ResponseWorkspace{ID: "workspace", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", Status: ResponseWorkspaceOpen, Version: 1}
	distributions.recipients["distribution"] = []memoryDistributionRecipient{{safe: DistributionRecipient{ID: "recipient", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", RequestID: "request", Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "person", State: DistributionRecipientPending}}}
	distributions.requestDistribution["request"] = "distribution"
	store := NewMemoryDistributionAccessStore(distributions)
	ctx := identity.WithActor(context.Background(), actor)
	service := &DistributionAccessService{store: store, now: func() time.Time { return now }}
	_, view, err := service.GetInternalWorkspace(ctx, "request")
	if err != nil || view.Workspace.ID != "workspace" {
		t.Fatalf("authorized request: %+v %v", view, err)
	}
	if _, err := service.SaveInternalWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 1}); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("save without authority route: %v", err)
	}
	service = service.WithInternalWorkspaceAuthorizer(func(_ context.Context, got identity.Actor, gotRequest Request, command string) (string, error) {
		if got.PrincipalID != "person" || gotRequest.ID != "request" || command != "forms.response.save" {
			return "", errors.New("wrong authority target")
		}
		return "v1:performer", nil
	})
	saved, err := service.SaveInternalWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 1, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}})
	if err != nil || saved.Workspace.Version != 2 || saved.FieldProvenance["q1"].Assurance != AssuranceInternalAuthenticated {
		t.Fatalf("save: %+v %v", saved, err)
	}
	if got := distributions.events[len(distributions.events)-1].ActorID; got != "person" {
		t.Fatalf("event actor = %q", got)
	}
	if got := distributions.events[len(distributions.events)-1].Payload["authority_route"]; got != "v1:performer" {
		t.Fatalf("event authority = %v", got)
	}
	if len(distributions.events) != len(distributions.outbox) {
		t.Fatal("save audit and outbox diverged")
	}
	service.WithInternalWorkspaceAuthorizer(func(context.Context, identity.Actor, Request, string) (string, error) {
		return "", errors.New("authority unavailable")
	})
	if _, err := service.SaveInternalWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 2, Edits: []FieldEdit{{FieldID: "q1", BaseSequence: 2, Value: formcontract.TextAnswer("No")}}}); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("authority failure: %v", err)
	}
	if len(distributions.events) != 1 || len(distributions.outbox) != 1 {
		t.Fatal("authority failure changed material state")
	}
	edit := store.workspaceStates["distribution"].edits[0]
	if edit.Origin != "INTERNAL" || edit.PrincipalID != "person" || edit.SessionID != "" || edit.RouteID != "" || edit.Assurance != AssuranceInternalAuthenticated {
		t.Fatalf("internal save used incorrect provenance: %+v", edit)
	}
	_, err = store.saveInternalMemoryWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 1, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("No")}}}, now)
	if !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("stale field should conflict: %v", err)
	}
	other := actor
	other.PrincipalID = "other"
	if _, _, err := store.getInternalMemoryWorkspace(identity.WithActor(context.Background(), other), "request", now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("wrong principal: %v", err)
	}
	distributions.mu.Lock()
	distributions.recipients["distribution"][0].safe.State = DistributionRecipientRevoked
	distributions.mu.Unlock()
	_, err = store.saveInternalMemoryWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 2, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("No"), BaseSequence: 2}}}, now)
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked recipient saved: %v", err)
	}
	if len(distributions.events) != 1 || len(distributions.outbox) != 1 {
		t.Fatal("denied save changed audit or outbox")
	}
	if _, _, err := store.getInternalMemoryWorkspace(ctx, "request", now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked recipient: %v", err)
	}
	distributions.mu.Lock()
	distributions.recipients["distribution"][0].safe.State = DistributionRecipientPending
	distributions.mu.Unlock()
	repo.mu.Lock()
	candidate.Active = false
	repo.candidates["person"] = candidate
	repo.mu.Unlock()
	if _, _, err := store.getInternalMemoryWorkspace(ctx, "request", now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("inactive principal: %v", err)
	}
}

func TestInternalMemorySubmissionRecordsVerifiedPrincipalAndRevision(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	request := Request{ID: "request", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestReady, Deadline: now.Add(time.Hour), Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "person", State: RecipientStateAssigned}, Fields: []Field{{ID: "q1", Label: "Access reviewed", Type: "yes_no", Required: true}}}
	candidate := RecipientCandidate{PrincipalID: "person", TenantID: "bank", LegalEntityIDs: []string{"entity"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{candidate})
	distributions := NewMemoryDistributionStore(repo, nil, nil)
	distributions.distributions["distribution"] = FormDistribution{ID: "distribution", TenantID: "bank", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 1, Status: DistributionOpen, Deadline: now.Add(time.Hour)}
	distributions.workspaces["distribution"] = ResponseWorkspace{ID: "workspace", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", Status: ResponseWorkspaceOpen, Version: 1}
	distributions.recipients["distribution"] = []memoryDistributionRecipient{{safe: DistributionRecipient{ID: "recipient", TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution", RequestID: "request", Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "person", State: DistributionRecipientPending}}}
	distributions.requestDistribution["request"] = "distribution"
	store := NewMemoryDistributionAccessStore(distributions)
	ctx := identity.WithActor(context.Background(), actor)
	if _, err := store.submitInternalMemoryWorkspace(ctx, "request", SubmitWorkspaceInput{ExpectedVersion: 1}, now); err == nil {
		t.Fatal("missing required answer was accepted")
	}
	view, err := store.saveInternalMemoryWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: 1, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitInternalMemoryWorkspace(ctx, "request", SubmitWorkspaceInput{ExpectedVersion: 1}, now); !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("stale submit: %v", err)
	}
	type submissionOutcome struct {
		result WorkspaceSubmissionResult
		err    error
	}
	outcomes := make(chan submissionOutcome, 2)
	for range 2 {
		go func() {
			result, err := store.submitInternalMemoryWorkspace(ctx, "request", SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}, now)
			outcomes <- submissionOutcome{result: result, err: err}
		}()
	}
	var result WorkspaceSubmissionResult
	var successful, conflicted int
	for range 2 {
		value := <-outcomes
		if value.err == nil {
			successful++
			result = value.result
		} else if errors.Is(value.err, ErrWorkspaceConflict) {
			conflicted++
		} else {
			t.Fatalf("concurrent submit: %v", value.err)
		}
	}
	if successful != 1 || conflicted != 1 {
		t.Fatalf("concurrent submit successes=%d conflicts=%d", successful, conflicted)
	}
	submission := repo.submissions[result.Submission.SubmissionID]
	if submission.SubmittedBy != "person" || submission.Channel != "INTERNAL" || submission.SessionID != "" {
		t.Fatalf("submission provenance: %+v", submission)
	}
	if result.Revision.SubmissionID != submission.ID || result.Revision.AchievedAssurance != AssuranceInternalAuthenticated || result.Revision.SubmittedRecipientPrincipalID != "person" {
		t.Fatalf("revision provenance: %+v", result.Revision)
	}
	if len(distributions.events) != len(distributions.outbox) || distributions.events[len(distributions.events)-1].ActorID != "person" {
		t.Fatal("submission audit or outbox provenance missing")
	}
	nextView, err := store.saveInternalMemoryWorkspace(ctx, "request", SaveWorkspaceInput{ExpectedVersion: result.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", BaseSequence: view.FieldSequences["q1"], Value: formcontract.TextAnswer("No")}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.submitInternalMemoryWorkspace(ctx, "request", SubmitWorkspaceInput{ExpectedVersion: nextView.Workspace.Version}, now)
	if err != nil {
		t.Fatal(err)
	}
	history := distributions.responseRevisions["distribution"]
	if len(history) != 2 || history[0].Current || !history[1].Current || history[1].SupersedesRevisionID != result.Revision.ID || second.Revision.Revision != 2 {
		t.Fatalf("incorrect revision history: %+v", history)
	}
	before := len(distributions.events)
	distributions.mu.Lock()
	distributions.recipients["distribution"][0].safe.State = DistributionRecipientRevoked
	distributions.mu.Unlock()
	if _, err := store.submitInternalMemoryWorkspace(ctx, "request", SubmitWorkspaceInput{ExpectedVersion: second.Workspace.Version}, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked recipient submitted: %v", err)
	}
	if len(distributions.events) != before || len(distributions.outbox) != before || len(repo.submissions) != 2 {
		t.Fatal("denied submission changed material state")
	}
}
