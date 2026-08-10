package governance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func validDefinition() json.RawMessage {
	return json.RawMessage(`{"rules":[{"id":"r1","responsibility":"AUTHORIZER","selector":{"kind":"ROLE","ref":"CRO"}}]}`)
}

func TestPolicyMakerCheckerLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryRepository())
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	p, err := svc.CreatePolicy(ctx, CreatePolicyInput{TenantID: "t1", Code: "risk", Name: "Risk", MakerID: "maker", Definition: validDefinition()})
	if err != nil {
		t.Fatal(err)
	}
	p, err = svc.SubmitPolicy(ctx, TransitionInput{TenantID: "t1", ID: p.ID, ActorID: "maker", ExpectedVersion: p.Version})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ApprovePolicy(ctx, TransitionInput{TenantID: "t1", ID: p.ID, ActorID: "maker", ExpectedVersion: p.Version}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("expected maker checker, got %v", err)
	}
	p, err = svc.ApprovePolicy(ctx, TransitionInput{TenantID: "t1", ID: p.ID, ActorID: "checker", ExpectedVersion: p.Version})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != PolicyActive || p.CheckerID != "checker" {
		t.Fatalf("unexpected policy %#v", p)
	}
}

func TestPolicyCreationRejectsInvalidEscalationSequence(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	definition := json.RawMessage(`{
		"rules":[{"id":"r1","responsibility":"ESCALATION_OWNER","selector":{"kind":"ROLE","ref":"RISK_MANAGER"}}],
		"escalations":[{
			"id":"overdue-review",
			"trigger":"OVERDUE",
			"steps":[
				{"after":"4h","responsibility":"ACCOUNTABLE_OWNER","department_levels_up":0},
				{"after":"4h","responsibility":"ESCALATION_OWNER","department_levels_up":1}
			]
		}]
	}`)
	_, err := svc.CreatePolicy(context.Background(), CreatePolicyInput{TenantID: "t1", Code: "risk", Name: "Risk", MakerID: "maker", Definition: definition})
	if err == nil || !strings.Contains(err.Error(), "must increase") {
		t.Fatalf("expected escalation validation error, got %v", err)
	}
}

func TestDelegationRejectsReverseCycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo)
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	first, _ := svc.CreateDelegation(ctx, CreateDelegationInput{TenantID: "t1", FromPrincipalID: "a", ToPrincipalID: "b", Responsibility: "REVIEWER", StartsAt: now.Add(time.Hour), EndsAt: now.Add(24 * time.Hour), MakerID: "maker-1"})
	first, _ = svc.SubmitDelegation(ctx, TransitionInput{TenantID: "t1", ID: first.ID, ActorID: "maker-1", ExpectedVersion: first.Version})
	if _, err := svc.ApproveDelegation(ctx, TransitionInput{TenantID: "t1", ID: first.ID, ActorID: "checker", ExpectedVersion: first.Version}); err != nil {
		t.Fatal(err)
	}
	second, _ := svc.CreateDelegation(ctx, CreateDelegationInput{TenantID: "t1", FromPrincipalID: "b", ToPrincipalID: "a", Responsibility: "REVIEWER", StartsAt: now.Add(2 * time.Hour), EndsAt: now.Add(12 * time.Hour), MakerID: "maker-2"})
	second, _ = svc.SubmitDelegation(ctx, TransitionInput{TenantID: "t1", ID: second.ID, ActorID: "maker-2", ExpectedVersion: second.Version})
	if _, err := svc.ApproveDelegation(ctx, TransitionInput{TenantID: "t1", ID: second.ID, ActorID: "checker-2", ExpectedVersion: second.Version}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
