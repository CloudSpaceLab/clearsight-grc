package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestRuntimeAuthenticationUsesOnlyActiveGovernedCredentialAndServerMetadata(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	policy := activatePolicy(t, ctx, service, aigateway.RolloutShadow, "RUNTIME")
	secret := "runtime-secret-1"
	digest := sha256.Sum256([]byte(secret))
	workload, err := service.CreateWorkload(ctx, CreateWorkloadInput{TenantID: "bank", WorkloadID: "agent-1", Code: "AGENT-1", Name: "Agent 1", Purpose: "risk", Environment: "prod", OwnerPrincipalID: "owner", AllowedModels: []string{"general"}, RequestsPerMinute: 10, TokensPerMinute: 1000, CostMicroUSDPerMinute: 10000, MaxConcurrent: 2, VerifiedMetadata: map[string]string{"classification": "CONFIDENTIAL"}, PolicyID: policy.ID, PolicyVersion: policy.Version, KeySHA256: hex.EncodeToString(digest[:]), MakerID: "maker"})
	if err != nil {
		t.Fatal(err)
	}
	workload, _ = service.TransitionWorkload(ctx, "submit", TransitionInput{TenantID: "bank", ID: workload.ID, ActorID: "maker", ExpectedVersion: workload.RecordVersion})
	workload, _ = service.TransitionWorkload(ctx, "approve", TransitionInput{TenantID: "bank", ID: workload.ID, ActorID: "checker", ExpectedVersion: workload.RecordVersion})
	workload, err = service.TransitionWorkload(ctx, "activate", TransitionInput{TenantID: "bank", ID: workload.ID, ActorID: "checker", ExpectedVersion: workload.RecordVersion})
	if err != nil {
		t.Fatal(err)
	}

	provider := NewRuntimeProvider(repo, nil)
	if _, err := provider.Authenticate(ctx, "Bearer wrong-secret"); !errors.Is(err, aigateway.ErrUnauthorized) {
		t.Fatalf("wrong secret must fail: %v", err)
	}
	authenticated, err := provider.Authenticate(ctx, "Bearer "+secret)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.TenantID != "bank" || authenticated.Policy.ID != policy.ID {
		t.Fatalf("wrong governed identity: %#v", authenticated)
	}

	facts, err := provider.ResolveFacts(ctx, *authenticated, aigateway.Request{Metadata: map[string]string{"classification": "PUBLIC"}}, []aigateway.BindingRequirement{{Mode: aigateway.ResolutionMetadata, FactKey: "classification", MetadataKey: "classification", Required: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].State != aigateway.FactKnown || facts[0].Value != "CONFIDENTIAL" || facts[0].Source != "WORKLOAD_METADATA" {
		t.Fatalf("caller metadata overrode server truth: %#v", facts)
	}
}
