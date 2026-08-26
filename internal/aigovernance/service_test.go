package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func activatePolicy(t *testing.T, ctx context.Context, service *Service, rollout aigateway.RolloutMode, code string) Policy {
	t.Helper()
	policy, err := service.CreatePolicy(ctx, CreatePolicyInput{TenantID: "bank", Code: code, Name: code, ActionClass: "MODEL_REQUEST", Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, RolloutMode: rollout, MakerID: "maker"})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.TransitionPolicy(ctx, "submit", TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "maker", ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionPolicy(ctx, "approve", TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "maker", ExpectedVersion: policy.RecordVersion}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker must not approve own policy, got %v", err)
	}
	policy, err = service.TransitionPolicy(ctx, "approve", TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "checker", ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.TransitionPolicy(ctx, "activate", TransitionInput{TenantID: "bank", ID: policy.ID, ActorID: "checker", ExpectedVersion: policy.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestEnforcementRequiresPriorShadowRevisionAndVersionsAdvance(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	enforce, err := service.CreatePolicy(ctx, CreatePolicyInput{TenantID: "bank", Code: "AI-CONTROL", Name: "AI control", ActionClass: "MODEL_REQUEST", Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, RolloutMode: aigateway.RolloutEnforce, MakerID: "maker"})
	if err != nil {
		t.Fatal(err)
	}
	enforce, _ = service.TransitionPolicy(ctx, "submit", TransitionInput{TenantID: "bank", ID: enforce.ID, ActorID: "maker", ExpectedVersion: enforce.RecordVersion})
	enforce, _ = service.TransitionPolicy(ctx, "approve", TransitionInput{TenantID: "bank", ID: enforce.ID, ActorID: "checker", ExpectedVersion: enforce.RecordVersion})
	if _, err := service.TransitionPolicy(ctx, "activate", TransitionInput{TenantID: "bank", ID: enforce.ID, ActorID: "checker", ExpectedVersion: enforce.RecordVersion}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("enforcement without shadow history must fail, got %v", err)
	}

	shadow := activatePolicy(t, ctx, service, aigateway.RolloutShadow, "AI-CONTROL-2")
	if shadow.Version != 1 {
		t.Fatalf("first policy revision = %d", shadow.Version)
	}
	enforce2, err := service.CreatePolicy(ctx, CreatePolicyInput{TenantID: "bank", Code: "AI-CONTROL-2", Name: "AI control enforced", ActionClass: "MODEL_REQUEST", Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, RolloutMode: aigateway.RolloutEnforce, MakerID: "maker-2"})
	if err != nil {
		t.Fatal(err)
	}
	if enforce2.Version != 2 {
		t.Fatalf("second policy revision = %d, want 2", enforce2.Version)
	}
	enforce2, _ = service.TransitionPolicy(ctx, "submit", TransitionInput{TenantID: "bank", ID: enforce2.ID, ActorID: "maker-2", ExpectedVersion: enforce2.RecordVersion})
	enforce2, _ = service.TransitionPolicy(ctx, "approve", TransitionInput{TenantID: "bank", ID: enforce2.ID, ActorID: "checker-2", ExpectedVersion: enforce2.RecordVersion})
	if _, err := service.TransitionPolicy(ctx, "activate", TransitionInput{TenantID: "bank", ID: enforce2.ID, ActorID: "checker-2", ExpectedVersion: enforce2.RecordVersion}); err != nil {
		t.Fatalf("enforcement after shadow history should activate: %v", err)
	}
}

func TestWorkloadActivationRequiresActiveExactPolicyAndGlobalCredentialUniqueness(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	policy := activatePolicy(t, ctx, service, aigateway.RolloutShadow, "MODEL-GUARD")
	key := sha256.Sum256([]byte("shared-secret"))
	keyHex := hex.EncodeToString(key[:])
	create := func(tenant, workloadID, code string) Workload {
		w, err := service.CreateWorkload(ctx, CreateWorkloadInput{TenantID: tenant, WorkloadID: workloadID, Code: code, Name: code, Purpose: "governed", Environment: "prod", OwnerPrincipalID: "owner", AllowedModels: []string{"general"}, RequestsPerMinute: 10, TokensPerMinute: 1000, CostMicroUSDPerMinute: 10000, MaxConcurrent: 2, PolicyID: policy.ID, PolicyVersion: policy.Version, KeySHA256: keyHex, MakerID: "maker"})
		if err != nil {
			t.Fatal(err)
		}
		w, err = service.TransitionWorkload(ctx, "submit", TransitionInput{TenantID: tenant, ID: w.ID, ActorID: "maker", ExpectedVersion: w.RecordVersion})
		if err != nil {
			t.Fatal(err)
		}
		w, err = service.TransitionWorkload(ctx, "approve", TransitionInput{TenantID: tenant, ID: w.ID, ActorID: "checker", ExpectedVersion: w.RecordVersion})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	first := create("bank", "gateway-a", "GATEWAY-A")
	first, err := service.TransitionWorkload(ctx, "activate", TransitionInput{TenantID: "bank", ID: first.ID, ActorID: "checker", ExpectedVersion: first.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 {
		t.Fatalf("first workload revision = %d", first.Version)
	}

	// Same bearer secret cannot identify two active tenants because authentication
	// intentionally has no caller-supplied tenant hint.
	repo.policies[memKey("other-bank", policy.ID)] = Policy{ID: policy.ID, TenantID: "other-bank", Code: policy.Code, Name: policy.Name, ActionClass: policy.ActionClass, Definition: policy.Definition, Status: "ACTIVE", RolloutMode: policy.RolloutMode, MakerID: "maker", CheckerID: "checker", Checksum: policy.Checksum, Version: policy.Version, RecordVersion: 1}
	second := create("other-bank", "gateway-b", "GATEWAY-B")
	if _, err := service.TransitionWorkload(ctx, "activate", TransitionInput{TenantID: "other-bank", ID: second.ID, ActorID: "checker", ExpectedVersion: second.RecordVersion}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate active credential must conflict globally, got %v", err)
	}
}

func TestReceiptIngestionIsIdempotentAndApprovalSignalsConvergeOnOneEpisode(t *testing.T) {
	ctx := context.Background()
	autoRepo := autonomy.NewMemoryRepository()
	auto := autonomy.NewService(autoRepo)
	repo := NewMemoryRepository()
	service := NewService(repo, auto, nil, nil)
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	policy := activatePolicy(t, ctx, service, aigateway.RolloutShadow, "AI-CONTROL")

	base := DecisionReceipt{ReceiptID: "receipt-1", TenantID: "bank", RequestID: "request-1", WorkloadID: "workload", PolicyID: policy.ID, PolicyCode: policy.Code, PolicyVersion: policy.Version, Decision: aigateway.DecisionRequireApproval, ReasonCodes: []string{"HIGH_IMPACT_TOOL"}, Outcome: "BLOCKED", ObservedAt: now}
	inserted, err := service.IngestReceipt(ctx, base)
	if err != nil || !inserted {
		t.Fatalf("first receipt: inserted=%v err=%v", inserted, err)
	}
	inserted, err = service.IngestReceipt(ctx, base)
	if err != nil || inserted {
		t.Fatalf("duplicate receipt: inserted=%v err=%v", inserted, err)
	}

	second := base
	second.ReceiptID = "receipt-2"
	second.RequestID = "request-2"
	inserted, err = service.IngestReceipt(ctx, second)
	if err != nil || !inserted {
		t.Fatalf("second request receipt: inserted=%v err=%v", inserted, err)
	}
	readiness, err := auto.Readiness(ctx, "bank")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.ActiveDrifts) != 1 {
		t.Fatalf("repeated material requests must converge on one episode, got %d drifts", len(readiness.ActiveDrifts))
	}

	conflict := base
	conflict.Outcome = "AUTHORIZED"
	if _, err := service.IngestReceipt(ctx, conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("receipt ID reuse with different content must conflict, got %v", err)
	}
}

func TestExecutionGrantRequiresExactApprovedMatterDecisionAndIsSingleUse(t *testing.T) {
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), "bank", "entity-1")
	matterRepo := continuity.NewMemoryRepository()
	matterRepo.RegisterLegalEntity("bank", "entity-1", "BANK")
	matters := continuity.NewService(matterRepo)
	service := NewService(NewMemoryRepository(), nil, nil, matters)
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	matter, err := matters.CreateMatter(ctx, continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-1", Type: continuity.MatterAuthorityRequest, Priority: 5, Title: "Approve high-impact AI action", Summary: "Approve one exact AI tool action.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	action := json.RawMessage(`{"tool":"freeze_account","arguments":{"account":"A-100","reason":"fraud"}}`)
	actionHash := sha256.Sum256(compactJSON(action))
	selected := hex.EncodeToString(actionHash[:])
	for _, step := range []struct {
		status continuity.DecisionStatus
		actor  string
	}{{continuity.DecisionProposed, "requester"}, {continuity.DecisionInReview, "reviewer"}, {continuity.DecisionApproved, "authorizer"}} {
		input := continuity.AddDecisionInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "AI_EXECUTION_GRANT", Status: step.status, Rationale: "Exact governed action.", AuthorityPrincipalID: step.actor}
		if step.status == continuity.DecisionApproved {
			input.SelectedOption = selected
		}
		matter, err = matters.RecordDecisionLifecycle(ctx, input)
		if err != nil {
			t.Fatalf("%s: %v", step.status, err)
		}
	}
	approved := continuity.CurrentDecisionForType(matter.Decisions, "AI_EXECUTION_GRANT")
	if approved == nil || approved.Status != continuity.DecisionApproved {
		t.Fatalf("approved decision missing: %#v", approved)
	}

	grant, err := service.CreateExecutionGrant(ctx, CreateGrantInput{TenantID: "bank", WorkloadID: "payments-agent", MatterID: matter.Matter.ID, DecisionID: approved.ID, Action: action, ActorID: "authorizer", TTLMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if grant.Token == "" || grant.State != "ACTIVE" {
		t.Fatalf("grant is not executable: %#v", grant)
	}
	if _, err := service.ConsumeExecutionGrant(ctx, "bank", "payments-agent", action, grant.Token); err != nil {
		t.Fatalf("first exact use failed: %v", err)
	}
	if _, err := service.ConsumeExecutionGrant(ctx, "bank", "payments-agent", action, grant.Token); !errors.Is(err, ErrGrantInvalid) {
		t.Fatalf("grant replay must fail, got %v", err)
	}
	changed := json.RawMessage(`{"tool":"freeze_account","arguments":{"account":"A-101","reason":"fraud"}}`)
	if _, err := service.ConsumeExecutionGrant(ctx, "bank", "payments-agent", changed, grant.Token); !errors.Is(err, ErrGrantInvalid) {
		t.Fatalf("changed arguments must fail, got %v", err)
	}
}

func TestRetentionMaintainerExpiresBoundedGovernanceState(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	policy := activatePolicy(t, ctx, service, aigateway.RolloutShadow, "RETENTION")

	for i := 0; i < 3; i++ {
		r := DecisionReceipt{
			ReceiptID: fmt.Sprintf("expired-%d", i), TenantID: "bank", RequestID: fmt.Sprintf("request-%d", i),
			WorkloadID: "retention-agent", PolicyID: policy.ID, PolicyCode: policy.Code, PolicyVersion: policy.Version,
			Decision: aigateway.DecisionAllow, Outcome: "AUTHORIZED", ObservedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
		}
		if inserted, err := service.IngestReceipt(ctx, r); err != nil || !inserted {
			t.Fatalf("seed receipt %d: inserted=%v err=%v", i, inserted, err)
		}
	}
	maintainer := &RetentionMaintainer{Repo: repo}
	if changed, err := maintainer.Maintain(ctx, now, 2); err != nil || changed != 2 {
		t.Fatalf("first bounded retention pass = %d, %v; want 2, nil", changed, err)
	}
	if changed, err := maintainer.Maintain(ctx, now, 2); err != nil || changed != 1 {
		t.Fatalf("second bounded retention pass = %d, %v; want 1, nil", changed, err)
	}
	repo.mu.Lock()
	remaining := len(repo.receipts)
	repo.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("expired receipts remain: %d", remaining)
	}
}

func TestPolicyRejectsContradictoryStreamingResponseControls(t *testing.T) {
	def := aigateway.PolicyDefinition{
		DefaultAction:   aigateway.DecisionAllow,
		ResponseControl: aigateway.ResponseControl{AllowStreaming: true, RedactPatterns: []string{`secret`}},
	}
	if err := validatePolicyDefinition(def); err == nil {
		t.Fatal("streaming with whole-response redaction must be rejected")
	}
}
