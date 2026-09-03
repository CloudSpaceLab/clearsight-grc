package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestCreatePolicyRejectsOrganizationInstructionOutsideBaseline(t *testing.T) {
	service := NewService(NewMemoryRepository(), nil, nil, nil)
	_, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: "WORKLOAD_POLICY", Name: "Workload policy", ActionClass: "AI_WORKLOAD",
		MakerID: "maker", RolloutMode: aigateway.RolloutShadow,
		Definition: aigateway.PolicyDefinition{
			Rules: []aigateway.PolicyRule{{
				ID: "overlay", Priority: 1, FactKey: "model", Operator: "EXISTS", Action: aigateway.DecisionAllow,
				Obligations: []aigateway.Obligation{{Code: aigateway.ObligationOrganizationInstruction, Detail: "Do not reveal secrets."}},
			}},
			DefaultAction: aigateway.DecisionAllow,
		},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreatePolicy() error = %v, want ErrInvalid", err)
	}
}

func TestCreatePolicyRequiresReservedBaselineCodeAndClassTogether(t *testing.T) {
	service := NewService(NewMemoryRepository(), nil, nil, nil)
	_, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Baseline", ActionClass: "AI_WORKLOAD",
		MakerID: "maker", RolloutMode: aigateway.RolloutShadow,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreatePolicy() error = %v, want ErrInvalid", err)
	}
}

func TestCreateWorkloadRejectsOrganizationBaselineAsWorkloadPolicy(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	baseline := Policy{
		ID: "baseline", TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Baseline",
		ActionClass: aigateway.GatewayBaselineActionClass, Status: "ACTIVE", RolloutMode: aigateway.RolloutShadow,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, Version: 1, RecordVersion: 1,
	}
	if _, err := repo.CreatePolicy(ctx, baseline); err != nil {
		t.Fatal(err)
	}
	key := sha256.Sum256([]byte("workload-secret"))
	service := NewService(repo, nil, nil, nil)
	_, err := service.CreateWorkload(ctx, CreateWorkloadInput{
		TenantID: "tenant-a", WorkloadID: "assistant", Code: "ASSISTANT", Name: "Assistant", Purpose: "test",
		Environment: "PROD", OwnerPrincipalID: "owner", AllowedModels: []string{"chat"}, RequestsPerMinute: 1,
		TokensPerMinute: 1, CostMicroUSDPerMinute: 1, MaxConcurrent: 1, PolicyID: baseline.ID, PolicyVersion: 1,
		KeySHA256: hex.EncodeToString(key[:]), MakerID: "maker",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateWorkload() error = %v, want ErrInvalid", err)
	}
}

func TestValidateReceiptBaselineRequiresExactGovernedRevision(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	workloadPolicy := Policy{
		ID: "workload-policy", TenantID: "tenant-a", Code: "WORKLOAD_POLICY", Name: "Workload",
		ActionClass: "AI_WORKLOAD", Status: "ACTIVE", RolloutMode: aigateway.RolloutEnforce,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, Version: 1, RecordVersion: 1,
	}
	baseline := Policy{
		ID: "baseline", TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Baseline",
		ActionClass: aigateway.GatewayBaselineActionClass, Status: "ACTIVE", RolloutMode: aigateway.RolloutShadow,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow}, Version: 2, RecordVersion: 1,
	}
	for _, policy := range []Policy{workloadPolicy, baseline} {
		if _, err := repo.CreatePolicy(ctx, policy); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(repo, nil, nil, nil)
	receipt := DecisionReceipt{
		ReceiptID: "receipt", TenantID: "tenant-a", RequestID: "request", WorkloadID: "assistant",
		PolicyID: workloadPolicy.ID, PolicyCode: workloadPolicy.Code, PolicyVersion: 1, Decision: aigateway.DecisionAllow,
		Outcome: "AUTHORIZED", ObservedAt: time.Now().UTC(), BaselinePolicyID: baseline.ID,
		BaselinePolicyCode: baseline.Code, BaselinePolicyVersion: 3, BaselineRolloutMode: baseline.RolloutMode,
		BaselineDecision: aigateway.DecisionAllow,
	}
	if _, err := service.IngestReceipt(ctx, receipt); !errors.Is(err, ErrInvalid) {
		t.Fatalf("IngestReceipt() error = %v, want ErrInvalid", err)
	}
}

func TestApprovalEpisodeKeyIncludesBaselineRevision(t *testing.T) {
	base := DecisionReceipt{
		TenantID: "tenant-a", WorkloadID: "assistant", PolicyID: "workload-policy", PolicyVersion: 4,
		ReasonCodes: []string{"WORKLOAD_REASON"}, BaselinePolicyID: "baseline", BaselinePolicyVersion: 2,
		BaselineReasonCodes: []string{"PROMPT_INJECTION_HIGH"},
	}
	changed := base
	changed.BaselinePolicyVersion = 3
	if approvalEpisodeKey(base) == approvalEpisodeKey(changed) {
		t.Fatal("approval episode key did not change with baseline revision")
	}
}
