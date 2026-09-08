package aigovernance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestCreateGovernedPolicyValidatesBaselineExceptionScope(t *testing.T) {
	repo, _, baseline := baselineExceptionFixture(t)
	service := NewService(repo, nil, nil, nil)
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	expires := now.Add(2 * time.Hour)
	scope, _ := json.Marshal(GatewayBaselineExceptionScope{
		TargetBaselineID: baseline.ID, TargetBaselineVersion: baseline.Version,
		WorkloadRecordIDs: []string{"workload-record"}, Environments: []string{"PRODUCTION"},
		WaivedRuleIDs: []string{"high-risk"}, Justification: "Approved bounded validation exception",
	})
	created, err := service.CreateGovernedPolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: aigateway.GatewayBaselineExceptionCodeRoot + ":ticket-123",
		Name: "Ticket 123", ActionClass: aigateway.GatewayBaselineExceptionActionClass,
		Eligibility: scope, Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
		RolloutMode: aigateway.RolloutShadow, MakerID: "maker-a", EffectiveUntil: &expires,
	})
	if err != nil {
		t.Fatalf("CreateGovernedPolicy() error = %v", err)
	}
	if created.Status != "DRAFT" || created.RolloutMode != aigateway.RolloutShadow || created.Code != aigateway.GatewayBaselineExceptionCodeRoot+":ticket-123" {
		t.Fatalf("created exception = %#v", created)
	}
}

func TestCreateGovernedPolicyRejectsUnknownWaivedRule(t *testing.T) {
	repo, _, baseline := baselineExceptionFixture(t)
	service := NewService(repo, nil, nil, nil)
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	expires := now.Add(time.Hour)
	scope, _ := json.Marshal(GatewayBaselineExceptionScope{
		TargetBaselineID: baseline.ID, TargetBaselineVersion: baseline.Version,
		WorkloadRecordIDs: []string{"workload-record"}, Environments: []string{"PRODUCTION"},
		WaivedRuleIDs: []string{"missing-rule"}, Justification: "Must not survive rule drift",
	})
	_, err := service.CreateGovernedPolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: aigateway.GatewayBaselineExceptionCodeRoot + ":bad-rule",
		Name: "Bad rule", ActionClass: aigateway.GatewayBaselineExceptionActionClass,
		Eligibility: scope, Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
		RolloutMode: aigateway.RolloutShadow, MakerID: "maker-a", EffectiveUntil: &expires,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateGovernedPolicy() error = %v, want invalid", err)
	}
}

func TestCreateGovernedPolicyRejectsPartialExceptionShape(t *testing.T) {
	repo, _, _ := baselineExceptionFixture(t)
	service := NewService(repo, nil, nil, nil)
	_, err := service.CreateGovernedPolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: aigateway.GatewayBaselineExceptionCodeRoot + ":spoof",
		Name: "Spoof", ActionClass: "MODEL_REQUEST", MakerID: "maker-a",
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreateGovernedPolicy() error = %v, want invalid", err)
	}
}

func TestTransitionGovernedPolicyRevalidatesExceptionExpiry(t *testing.T) {
	repo, _, baseline := baselineExceptionFixture(t)
	service := NewService(repo, nil, nil, nil)
	now := time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	expires := now.Add(time.Hour)
	scope, _ := json.Marshal(GatewayBaselineExceptionScope{
		TargetBaselineID: baseline.ID, TargetBaselineVersion: baseline.Version,
		WorkloadRecordIDs: []string{"workload-record"}, Environments: []string{"PRODUCTION"},
		WaivedRuleIDs: []string{"high-risk"}, Justification: "Short lived validation",
	})
	created, err := service.CreateGovernedPolicy(context.Background(), CreatePolicyInput{
		TenantID: "tenant-a", Code: aigateway.GatewayBaselineExceptionCodeRoot + ":expires",
		Name: "Expires", ActionClass: aigateway.GatewayBaselineExceptionActionClass,
		Eligibility: scope, Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
		RolloutMode: aigateway.RolloutShadow, MakerID: "maker-a", EffectiveUntil: &expires,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return expires.Add(time.Second) }
	_, err = service.TransitionGovernedPolicy(context.Background(), "submit", TransitionInput{
		TenantID: "tenant-a", ID: created.ID, ActorID: "maker-a", ExpectedVersion: created.RecordVersion,
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("TransitionGovernedPolicy() error = %v, want invalid transition", err)
	}
}
