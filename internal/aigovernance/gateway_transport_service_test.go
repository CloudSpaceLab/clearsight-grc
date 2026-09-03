package aigovernance

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func testGatewayTransportDefinition() aigateway.TransportDefinition {
	requireUsage := true
	return aigateway.TransportDefinition{
		Providers: []aigateway.TransportProviderConfig{{
			ID: "openai-primary", Name: "OpenAI primary", Kind: aigateway.ProviderKindOpenAI,
			BaseURL: "https://api.openai.com", SecretRef: "env:OPENAI_API_KEY", State: aigateway.ProviderStateEnabled,
			RequireUsage: &requireUsage, Regions: []string{"US"},
		}},
		Models: []aigateway.ModelConfig{{Alias: "safe-chat", Routes: []aigateway.RouteConfig{{
			ID: "openai-gpt", ProviderID: "openai-primary", Model: "gpt-5", Weight: 100,
		}}}},
	}
}

func TestGatewayTransportLifecycleRequiresIndependentChecker(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()
	created, err := service.CreateGatewayTransport(ctx, CreateGatewayTransportInput{
		TenantID: "tenant-a", Environment: "production", Definition: testGatewayTransportDefinition(),
		MakerID: "maker", ChangeReason: "Initial governed provider route",
	})
	if err != nil {
		t.Fatalf("CreateGatewayTransport() error = %v", err)
	}
	if created.Status != GatewayTransportDraft || created.Version != 1 || len(created.Checksum) != 64 {
		t.Fatalf("created = %#v", created)
	}
	submitted, err := service.TransitionGatewayTransport(ctx, "submit", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "maker", ExpectedVersion: created.RecordVersion})
	if err != nil {
		t.Fatalf("submit error = %v", err)
	}
	if _, err := service.TransitionGatewayTransport(ctx, "approve", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "maker", ExpectedVersion: submitted.RecordVersion}); err != ErrMakerChecker {
		t.Fatalf("maker approval error = %v, want %v", err, ErrMakerChecker)
	}
	approved, err := service.TransitionGatewayTransport(ctx, "approve", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "checker", ExpectedVersion: submitted.RecordVersion})
	if err != nil {
		t.Fatalf("approve error = %v", err)
	}
	active, err := service.TransitionGatewayTransport(ctx, "activate", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "checker", ExpectedVersion: approved.RecordVersion})
	if err != nil {
		t.Fatalf("activate error = %v", err)
	}
	if active.Status != GatewayTransportActive || active.Checksum != created.Checksum {
		t.Fatalf("active = %#v", active)
	}
}

func TestGatewayTransportActivationAtomicallySupersedesPriorActive(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	ctx := context.Background()
	activate := func(reason string) GatewayTransportRevision {
		created, err := service.CreateGatewayTransport(ctx, CreateGatewayTransportInput{
			TenantID: "tenant-a", Environment: "PRODUCTION", Definition: testGatewayTransportDefinition(),
			MakerID: "maker", ChangeReason: reason,
		})
		if err != nil {
			t.Fatal(err)
		}
		submitted, err := service.TransitionGatewayTransport(ctx, "submit", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "maker", ExpectedVersion: created.RecordVersion})
		if err != nil {
			t.Fatal(err)
		}
		approved, err := service.TransitionGatewayTransport(ctx, "approve", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "checker", ExpectedVersion: submitted.RecordVersion})
		if err != nil {
			t.Fatal(err)
		}
		active, err := service.TransitionGatewayTransport(ctx, "activate", GatewayTransportTransitionInput{TenantID: "tenant-a", ID: created.ID, ActorID: "checker", ExpectedVersion: approved.RecordVersion})
		if err != nil {
			t.Fatal(err)
		}
		return active
	}
	first := activate("First")
	second := activate("Second")
	if second.Version != first.Version+1 {
		t.Fatalf("versions first=%d second=%d", first.Version, second.Version)
	}
	firstStored, err := service.GetGatewayTransport(ctx, "tenant-a", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstStored.Status != GatewayTransportSuspended {
		t.Fatalf("first status = %s, want SUSPENDED", firstStored.Status)
	}
	current, err := service.ActiveGatewayTransport(ctx, "tenant-a", "PRODUCTION")
	if err != nil {
		t.Fatal(err)
	}
	if current.ID != second.ID {
		t.Fatalf("active id = %s, want %s", current.ID, second.ID)
	}
}

func TestGatewayTransportStoresOpaqueSecretReferenceOnly(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	value, err := service.CreateGatewayTransport(context.Background(), CreateGatewayTransportInput{
		TenantID: "tenant-a", Environment: "PRODUCTION", Definition: testGatewayTransportDefinition(),
		MakerID: "maker", ChangeReason: "Secret reference test",
	})
	if err != nil {
		t.Fatal(err)
	}
	provider := value.Definition.Providers[0]
	if provider.SecretRef != "env:OPENAI_API_KEY" {
		t.Fatalf("secret ref = %q", provider.SecretRef)
	}
}
