package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExecutionResultReadsExactScopedReceiptBeyondHistoryPage(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	policy := createPolicyFixture(t, service, actor, RolloutShadow)
	repo := service.repo.(*MemoryRepository)
	for i := 0; i < 60; i++ {
		repo.executions[fmt.Sprint(i)] = ExecutionReceipt{ID: fmt.Sprint(i), TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PolicyID: policy.ID, State: ExecutionApplied, ResponseRevisionID: "private-response", MatterID: "private-issue", CreatedAt: time.Unix(int64(i), 0)}
	}
	result, err := service.ExecutionResult(context.Background(), actor, policy.ID, "0")
	if err != nil || result.ID != "0" || result.ResponseRevisionID != "private-response" {
		t.Fatalf("exact result=%+v err=%v", result, err)
	}
	payload, _ := json.Marshal(result)
	if strings.Contains(string(payload), "private-") {
		t.Fatalf("unchecked IDs leaked: %s", payload)
	}
	for _, scope := range []Actor{{TenantID: "other", LegalEntityID: "entity", PrincipalID: "maker"}, {TenantID: "bank", LegalEntityID: "other", PrincipalID: "maker"}} {
		if _, err := service.ExecutionResult(context.Background(), scope, policy.ID, "0"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("scope err=%v", err)
		}
	}
	for _, target := range []struct{ policy, id string }{{"other", "0"}, {policy.ID, "missing"}} {
		if _, err := service.ExecutionResult(context.Background(), actor, target.policy, target.id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("target err=%v", err)
		}
	}
	repo.executionFailures["failed"] = ExecutionReceipt{ID: "failed", TenantID: "bank", LegalEntityID: "entity", PolicyID: policy.ID, State: ExecutionFailed, ResponseRevisionID: "failed-response"}
	if result, err := service.ExecutionResult(context.Background(), actor, policy.ID, "failed"); err != nil || result.ResponseRevisionID != "failed-response" || result.MatterID != "" {
		t.Fatalf("failure result=%+v err=%v", result, err)
	}
}
