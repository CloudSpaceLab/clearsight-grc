package governance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEscalationSequenceRevisionAndRollbackPreserveGovernedLineage(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 5, 17, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	initial := json.RawMessage(`{
		"rules":[{"id":"owner","legal_entity_id":"` + testEntityA + `","object_type":"MATTER","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","decision_type":"matter.test","priority":100,"selector":{"kind":"ROLE","ref":"RISK_OWNER"}}],
		"escalations":[{"id":"overdue","trigger":"OVERDUE","steps":[{"after":"15m","responsibility":"ACCOUNTABLE_OWNER"}]}]
	}`)
	policy, err := service.CreatePolicy(ctx, CreatePolicyInput{
		TenantID: "bank", LegalEntityID: testEntityA, Code: "MATTER", Name: "Matter routing", MakerID: "maker", Definition: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.SubmitPolicy(ctx, TransitionInput{TenantID: "bank", LegalEntityID: testEntityA, ID: policy.ID, ActorID: "maker", ExpectedVersion: policy.Version})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	policy, err = service.ApprovePolicy(ctx, TransitionInput{
		TenantID: "bank", LegalEntityID: testEntityA, ID: policy.ID, ActorID: "checker", ExpectedVersion: policy.Version, Rationale: "Initial route reviewed",
	})
	if err != nil {
		t.Fatal(err)
	}
	initialChecksum := policy.Checksum

	levelUp := 1
	now = now.Add(time.Minute)
	revision, err := service.ProposeEscalationSequenceRevision(ctx, EscalationSequenceRevisionInput{
		TenantID: "bank", LegalEntityID: testEntityA, PolicyID: policy.ID, SequenceID: "overdue",
		ActorID: "sequence-maker", ExpectedPolicyVersion: policy.Version,
		TerminalHandling: "KEEP_OPEN", RecoveryAction: "REVIEW_ROUTE",
		Steps: []EscalationSequenceStepInput{
			{After: "15m", Responsibility: "ACCOUNTABLE_OWNER"},
			{After: "1h", Responsibility: "ESCALATION_OWNER", DepartmentLevelsUp: &levelUp, TargetRoles: []string{"risk manager"}},
			{After: "4h", Responsibility: "AUTHORIZER", TargetPositionIDs: []string{"019fffff-ffff-7fff-8fff-fffffffffff1"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.Version != 2 || revision.BaseVersion != 1 {
		t.Fatalf("unexpected sequence revision: %#v", revision)
	}

	current, err := repo.GetPolicy(ctx, "bank", policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentVersion != 1 || current.Checksum != initialChecksum {
		t.Fatalf("proposal changed active policy: %#v", current)
	}
	sequences, err := ParseEscalationSequences(revision.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if len(sequences) != 1 || len(sequences[0].Steps) != 3 {
		t.Fatalf("expected three configured escalation levels, got %#v", sequences)
	}
	if got := sequences[0].Steps[1].TargetRoles; len(got) != 1 || got[0] != "RISK_MANAGER" {
		t.Fatalf("target role was not normalized: %#v", got)
	}
	if got := sequences[0].Steps[2].TargetPositionIDs; len(got) != 1 {
		t.Fatalf("target position was not retained: %#v", got)
	}
	if _, err := service.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: "bank", LegalEntityID: testEntityA, PolicyID: policy.ID, RevisionVersion: revision.Version,
		ActorID: "sequence-maker", ExpectedPolicyVersion: current.Version, Rationale: "Self approval",
	}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("expected independent approval requirement, got %v", err)
	}

	now = now.Add(time.Minute)
	active, err := service.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: "bank", LegalEntityID: testEntityA, PolicyID: policy.ID, RevisionVersion: revision.Version,
		ActorID: "sequence-checker", ExpectedPolicyVersion: current.Version, Rationale: "Sequence checked",
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.CurrentVersion != 2 {
		t.Fatalf("sequence revision not activated: %#v", active)
	}

	now = now.Add(time.Minute)
	rollback, err := service.ProposeEscalationRollback(ctx, EscalationRollbackInput{
		TenantID: "bank", LegalEntityID: testEntityA, PolicyID: policy.ID, SourceVersion: 1,
		ActorID: "rollback-maker", ExpectedPolicyVersion: active.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rollback.Version != 3 || rollback.BaseVersion != 2 || rollback.Checksum != initialChecksum {
		t.Fatalf("rollback must create a new revision from approved v1: %#v", rollback)
	}
	stillActive, _ := repo.GetPolicy(ctx, "bank", policy.ID)
	if stillActive.CurrentVersion != 2 {
		t.Fatalf("rollback proposal changed active policy: %#v", stillActive)
	}

	now = now.Add(time.Minute)
	restored, err := service.ApprovePolicyRevision(ctx, ApprovePolicyRevisionInput{
		TenantID: "bank", LegalEntityID: testEntityA, PolicyID: policy.ID, RevisionVersion: rollback.Version,
		ActorID: "rollback-checker", ExpectedPolicyVersion: stillActive.Version, Rationale: "Restore approved v1 sequence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.CurrentVersion != 3 || restored.Checksum != initialChecksum {
		t.Fatalf("rollback did not activate as a new lineage version: %#v", restored)
	}
	old, err := repo.GetPolicyVersion(ctx, "bank", testEntityA, policy.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if old.EffectiveUntil == nil {
		t.Fatal("superseded version did not receive an effective-until boundary")
	}
}

func TestEscalationSequenceRevisionRejectsNonMonotonicLevels(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := canonicalEscalationSequence(EscalationSequenceRevisionInput{
		SequenceID: "overdue",
		Steps: []EscalationSequenceStepInput{
			{After: "2h", Responsibility: "ACCOUNTABLE_OWNER"},
			{After: "1h", Responsibility: "ESCALATION_OWNER"},
		},
	})
	if err == nil {
		t.Fatal("expected non-monotonic escalation thresholds to be rejected")
	}
}
