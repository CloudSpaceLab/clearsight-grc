package commandauth

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type candidateAuthority struct{ resolution authority.Resolution }

func (c candidateAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return c.resolution, nil
}
func (candidateAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (candidateAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (candidateAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestGuardAllowsAnyEffectiveCandidate(t *testing.T) {
	service := candidateAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "owner"},
			{ID: "delegate"},
		},
		Strategy: "CANDIDATE_SET",
	}}
	guard, err := New(service, ModeEnforce, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ctx := identity.WithActor(context.Background(), identity.Actor{
		TenantID: "bank", PrincipalID: "delegate", LegalEntityID: "entity", Kind: "PERSON",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	decision, err := guard.Authorize(ctx, Request{
		TenantID: "bank", LegalEntityID: "entity", ObjectType: "MATTER", ObjectID: "matter-1",
		Responsibility: authority.ResponsibilityOwner, DecisionType: "matter.action.add", Materiality: 3,
	})
	if err != nil || !decision.Allowed {
		t.Fatalf("effective delegate was rejected: %#v err=%v", decision, err)
	}
}
