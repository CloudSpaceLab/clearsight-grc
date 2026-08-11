package governance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEscalationGuardRevisionPreservesActivePolicyUntilIndependentApproval(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo)
	now := time.Date(2026, 8, 11, 5, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	definition := json.RawMessage(`{
		"rules":[{"id":"auditor-route","responsibility":"ESCALATION_OWNER","selector":{"kind":"ROLE","ref":"AUDITOR"}}],
		"metadata":{"owner":"risk-governance"},
		"escalations":[{"id":"compliance-overdue","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER"}]}]
	}`)
	policy, err := svc.CreatePolicy(ctx, CreatePolicyInput{TenantID: "bank", Code: "BANK", Name: "Bank", MakerID: "maker-0", Definition: definition})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.SubmitPolicy(ctx, TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "maker-0", ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.ApprovePolicy(ctx, TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "checker-0", ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	activeDefinition := append([]byte(nil), policy.Definition...)
	activeCurrentVersion := policy.CurrentVersion

	revision, err := svc.ProposeEscalationGuardRevision(ctx, EscalationGuardRevisionInput{
		TenantID: "bank", PolicyID: policy.ID, SequenceID: "compliance-overdue", StepIndex: 0,
		SourceRoles: []string{"Compliance Officer"}, TargetRoles: []string{"Supervisor"},
		TargetGroupIDs: []string{"019fede5-67de-733a-95ae-97f4db546c1e"},
		ActorID: "guard-maker", ExpectedPolicyVersion: policy.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.Version <= activeCurrentVersion || revision.MakerID != "guard-maker" {
		t.Fatalf("unexpected revision: %#v", revision)
	}

	current, err := svc.ListPolicies(ctx, "bank")
	if err != nil || len(current) != 1 {
		t.Fatalf("load active policy after proposal: %#v err=%v", current, err)
	}
	policy = current[0]
	if policy.CurrentVersion != activeCurrentVersion {
		t.Fatalf("proposal changed active version: got %d want %d", policy.CurrentVersion, activeCurrentVersion)
	}
	if string(policy.Definition) != string(activeDefinition) {
		t.Fatalf("proposal changed active definition: %s", policy.Definition)
	}

	sequences, err := ParseEscalationSequences(revision.Definition)
	if err != nil {
		t.Fatal(err)
	}
	step := sequences[0].Steps[0]
	if len(step.SourceRoles) != 1 || step.SourceRoles[0] != "COMPLIANCE_OFFICER" || len(step.TargetRoles) != 1 || step.TargetRoles[0] != "SUPERVISOR" || len(step.TargetGroupIDs) != 1 {
		t.Fatalf("guard revision did not preserve normalized constraints: %#v", step)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(revision.Definition, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["metadata"]; !ok {
		t.Fatal("unrelated policy metadata was dropped by guard revision")
	}
	if _, ok := doc["rules"]; !ok {
		t.Fatal("authority rules were dropped by guard revision")
	}

	if _, err := svc.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: "bank", PolicyID: policy.ID, RevisionVersion: revision.Version,
		ActorID: "guard-maker", ExpectedPolicyVersion: policy.Version, Rationale: "self approve",
	}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("expected maker-checker rejection, got %v", err)
	}

	approved, err := svc.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: "bank", PolicyID: policy.ID, RevisionVersion: revision.Version,
		ActorID: "guard-checker", ExpectedPolicyVersion: policy.Version, Rationale: "Reviewed escalation population",
	})
	if err != nil {
		t.Fatal(err)
	}
	if approved.CurrentVersion != revision.Version || approved.CheckerID != "guard-checker" {
		t.Fatalf("revision not activated by checker: %#v", approved)
	}
	approvedSequences, err := ParseEscalationSequences(approved.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if got := approvedSequences[0].Steps[0].TargetRoles; len(got) != 1 || got[0] != "SUPERVISOR" {
		t.Fatalf("approved definition missing guard: %#v", approvedSequences[0].Steps[0])
	}
}

func TestEscalationGuardRevisionCannotOverwriteAnotherMakersPendingRevision(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo)
	now := time.Date(2026, 8, 11, 5, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	definition := json.RawMessage(`{
		"rules":[{"id":"r1","responsibility":"ESCALATION_OWNER","selector":{"kind":"ROLE","ref":"AUDITOR"}}],
		"escalations":[{"id":"overdue","trigger":"OVERDUE","steps":[{"after":"0s","responsibility":"ESCALATION_OWNER"}]}]
	}`)
	policy, err := svc.CreatePolicy(ctx, CreatePolicyInput{TenantID: "bank", Code: "BANK", Name: "Bank", MakerID: "maker-0", Definition: definition})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.SubmitPolicy(ctx, TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "maker-0", ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = svc.ApprovePolicy(ctx, TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "checker-0", ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProposeEscalationGuardRevision(ctx, EscalationGuardRevisionInput{
		TenantID: "bank", PolicyID: policy.ID, SequenceID: "overdue", StepIndex: 0,
		SourceRoles: []string{"COMPLIANCE_OFFICER"}, ActorID: "maker-a", ExpectedPolicyVersion: policy.Version,
	}); err != nil {
		t.Fatal(err)
	}
	current, _ := repo.GetPolicy(ctx, "bank", policy.ID)
	_, err = svc.ProposeEscalationGuardRevision(ctx, EscalationGuardRevisionInput{
		TenantID: "bank", PolicyID: policy.ID, SequenceID: "overdue", StepIndex: 0,
		TargetRoles: []string{"SUPERVISOR"}, ActorID: "maker-b", ExpectedPolicyVersion: current.Version,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected pending revision ownership conflict, got %v", err)
	}
}
