package aigovernance

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGatewayEmergencyControlFreezeAndUnfreezeAreVersioned(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	initial, err := service.GatewayEmergencyControl(context.Background(), "bank", "production")
	if err != nil {
		t.Fatal(err)
	}
	if initial.Frozen || initial.RecordVersion != 0 || initial.Environment != "PRODUCTION" {
		t.Fatalf("initial control = %#v", initial)
	}

	frozen, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "PRODUCTION", Frozen: true,
		Reason: "Potential provider credential compromise under investigation", ActorID: "admin-a", ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !frozen.Frozen || frozen.RecordVersion != 1 || frozen.ActorID != "admin-a" || frozen.ID == "" {
		t.Fatalf("frozen control = %#v", frozen)
	}

	now = now.Add(time.Minute)
	restored, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "production", Frozen: false,
		Reason: "Credential rotated and provider boundary revalidated", ActorID: "admin-b", ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Frozen || restored.RecordVersion != 2 || restored.ID != frozen.ID || restored.ActorID != "admin-b" {
		t.Fatalf("restored control = %#v", restored)
	}
}

func TestGatewayEmergencyControlRejectsStaleAndNoOpChanges(t *testing.T) {
	service := NewService(NewMemoryRepository(), nil, nil, nil)
	if _, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "PRODUCTION", Frozen: false, Reason: "Nothing to restore", ActorID: "admin", ExpectedVersion: 0,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("unfreeze before freeze error = %v", err)
	}
	frozen, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "PRODUCTION", Frozen: true, Reason: "Incident response freeze", ActorID: "admin", ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "PRODUCTION", Frozen: false, Reason: "Stale operator view", ActorID: "admin", ExpectedVersion: 0,
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	if _, err := service.SetGatewayEmergencyControl(context.Background(), SetGatewayEmergencyControlInput{
		TenantID: "bank", Environment: "PRODUCTION", Frozen: true, Reason: "Duplicate freeze", ActorID: "admin", ExpectedVersion: frozen.RecordVersion,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("duplicate freeze error = %v", err)
	}
}
