package commandauth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type authorityStub struct {
	principal string
	err       error
}

func (s authorityStub) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	if s.err != nil {
		return authority.Resolution{}, s.err
	}
	return authority.Resolution{Principal: authority.Principal{ID: s.principal, DisplayName: "Selected person"}, RuleID: "rule-1", PolicyVersion: "v1"}, nil
}
func (s authorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (s authorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (s authorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func actorContext() context.Context {
	now := time.Now().UTC()
	return identity.WithActor(context.Background(), identity.Actor{TenantID: "bank-demo", PrincipalID: "person-1", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)})
}

func TestGuardEnforcesSelectedPrincipal(t *testing.T) {
	guard, err := New(authorityStub{principal: "person-1"}, ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	decision, err := guard.Authorize(actorContext(), Request{TenantID: "bank-demo", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4})
	if err != nil || !decision.Allowed || decision.Resolution == nil {
		t.Fatalf("expected command to be authorized: %#v err=%v", decision, err)
	}
}

func TestGuardRejectsDifferentSelectedPrincipal(t *testing.T) {
	guard, _ := New(authorityStub{principal: "person-2"}, ModeEnforce, slog.Default())
	decision, err := guard.Authorize(actorContext(), Request{TenantID: "bank-demo", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: authority.ResponsibilityAuthorizer})
	if !errors.Is(err, ErrNotAuthorized) || decision.Allowed {
		t.Fatalf("expected rejection: %#v err=%v", decision, err)
	}
}

func TestAuditModeRecordsButAllows(t *testing.T) {
	guard, _ := New(authorityStub{err: authority.ErrNoRoute}, ModeAudit, slog.Default())
	decision, err := guard.Authorize(actorContext(), Request{TenantID: "bank-demo", ObjectType: "PROGRAM", ObjectID: "program-1", Responsibility: authority.ResponsibilityReviewer})
	if err != nil || !decision.Allowed || decision.Enforced {
		t.Fatalf("expected audit decision to allow: %#v err=%v", decision, err)
	}
}

func TestGuardRejectsDifferentLegalEntity(t *testing.T) {
	guard, _ := New(authorityStub{principal: "person-1"}, ModeEnforce, slog.Default())
	decision, err := guard.Authorize(actorContext(), Request{TenantID: "bank-demo", LegalEntityID: "bank-gh", ObjectType: "MATTER", ObjectID: "matter-1", Responsibility: authority.ResponsibilityAuthorizer})
	if !errors.Is(err, ErrLegalEntityMismatch) || decision.Allowed {
		t.Fatalf("expected legal-entity rejection: %#v err=%v", decision, err)
	}
}
