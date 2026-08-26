package httpapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type countingPrincipalResolver struct {
	scalarCalls int
	batchCalls  int
	batchIDs    []string
	useBatch    bool
}

func (s *countingPrincipalResolver) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	return access.Resolution{}, access.ErrIdentityNotProvisioned
}

func (s *countingPrincipalResolver) ResolvePrincipal(_ context.Context, tenant, principalID, entity string) (access.Resolution, error) {
	s.scalarCalls++
	return access.Resolution{TenantID: tenant, LegalEntityID: entity, PrincipalID: principalID, DisplayName: "Name " + principalID, Kind: "PERSON"}, nil
}

type countingBatchPrincipalResolver struct{ *countingPrincipalResolver }

func (s *countingBatchPrincipalResolver) ResolvePrincipals(_ context.Context, tenant, entity string, principalIDs []string) ([]access.PrincipalResolveOutcome, error) {
	s.batchCalls++
	s.batchIDs = append([]string(nil), principalIDs...)
	outcomes := make([]access.PrincipalResolveOutcome, len(principalIDs))
	for index, principalID := range principalIDs {
		outcomes[index].Resolution = access.Resolution{TenantID: tenant, LegalEntityID: entity, PrincipalID: principalID, DisplayName: "Name " + principalID, Kind: "PERSON"}
	}
	return outcomes, nil
}

func TestPrincipalLabelsUseOneStableUniqueBatch(t *testing.T) {
	resolver := &countingBatchPrincipalResolver{countingPrincipalResolver: &countingPrincipalResolver{}}
	api := &API{deps: Dependencies{Access: resolver}}
	ctx, complete := api.withPrincipalLabels(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity"}, []string{"owner", "reviewer", "owner", " "})
	if !complete || resolver.batchCalls != 1 || resolver.scalarCalls != 0 || fmt.Sprint(resolver.batchIDs) != "[owner reviewer]" {
		t.Fatalf("unexpected resolution: complete=%t batch=%d scalar=%d ids=%v", complete, resolver.batchCalls, resolver.scalarCalls, resolver.batchIDs)
	}
	if principal, cached := cachedPrincipalLabel(ctx, "reviewer"); !cached || principal == nil || principal.DisplayName != "Name reviewer" {
		t.Fatalf("reviewer label was not cached: cached=%t principal=%#v", cached, principal)
	}
}

func TestPrincipalLabelsFallbackMemoizesEachUniquePrincipal(t *testing.T) {
	resolver := &countingPrincipalResolver{}
	api := &API{deps: Dependencies{Access: resolver}}
	ctx, complete := api.withPrincipalLabels(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity"}, []string{"owner", "owner", "reviewer"})
	if !complete || resolver.scalarCalls != 2 {
		t.Fatalf("fallback calls=%d complete=%t", resolver.scalarCalls, complete)
	}
	if _, cached := cachedPrincipalLabel(ctx, "owner"); !cached {
		t.Fatal("owner label was not cached")
	}
}

func TestPrincipalLabelsBoundLargeResponsibilitySets(t *testing.T) {
	resolver := &countingBatchPrincipalResolver{countingPrincipalResolver: &countingPrincipalResolver{}}
	api := &API{deps: Dependencies{Access: resolver}}
	ids := make([]string, access.MaxPrincipalBatchSize+1)
	for index := range ids {
		ids[index] = fmt.Sprintf("principal-%03d", index)
	}
	ctx, complete := api.withPrincipalLabels(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity"}, ids)
	if complete || resolver.batchCalls != 1 || len(resolver.batchIDs) != access.MaxPrincipalBatchSize {
		t.Fatalf("large batch complete=%t calls=%d size=%d", complete, resolver.batchCalls, len(resolver.batchIDs))
	}
	if principal, cached := cachedPrincipalLabel(ctx, ids[len(ids)-1]); !cached || principal != nil {
		t.Fatalf("overflow principal must have an explicit unavailable cache entry: cached=%t principal=%#v", cached, principal)
	}
}

type labelBatchAuthority struct{ principalID string }

func (s labelBatchAuthority) resolution() authority.Resolution {
	principal := authority.Principal{ID: s.principalID, DisplayName: "Current actor", Kind: "PERSON"}
	return authority.Resolution{Principal: principal, CandidatePrincipals: []authority.Principal{principal}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: s.principalID, OriginPrincipalID: s.principalID}}, RuleID: "route", PolicyVersion: "v1"}
}

func (s labelBatchAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return s.resolution(), nil
}

func (s labelBatchAuthority) ResolveMany(_ context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	outcomes := make([]authority.ResolveOutcome, len(inputs))
	for index := range outcomes {
		outcomes[index].Resolution = s.resolution()
	}
	return outcomes, nil
}

func (s labelBatchAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s labelBatchAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s labelBatchAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestMatterAndProgramOperationsBatchStoredPrincipalLabels(t *testing.T) {
	resolver := &countingBatchPrincipalResolver{countingPrincipalResolver: &countingPrincipalResolver{}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}
	api := &API{deps: Dependencies{Access: resolver, Authority: labelBatchAuthority{principalID: "owner"}}}
	now := time.Now().UTC()
	matter := continuity.MatterAggregate{
		Matter:  continuity.Matter{ID: "matter", TenantID: "bank", LegalEntityID: "entity", Status: continuity.MatterInitialReview, OwnerPrincipalID: "owner", Priority: 2, Version: 1},
		Actions: []continuity.Action{{ID: "action", OwnerPrincipalID: "owner", Status: continuity.ActionPlanned}},
	}
	matterResponse := api.buildMatterOperations(identity.WithActor(t.Context(), actor), actor, matter, now)
	if !matterResponse.ResponsibilityLabelsComplete || resolver.batchCalls != 1 || resolver.scalarCalls != 0 || len(resolver.batchIDs) != 1 {
		t.Fatalf("matter labels complete=%t batch=%d scalar=%d ids=%v", matterResponse.ResponsibilityLabelsComplete, resolver.batchCalls, resolver.scalarCalls, resolver.batchIDs)
	}

	resolver.batchCalls, resolver.scalarCalls, resolver.batchIDs = 0, 0, nil
	program := continuity.ProgramAggregate{
		Program:                continuity.Program{ID: "program", TenantID: "bank", LegalEntityID: "entity", Status: continuity.ProgramActive, OwnerPrincipalID: "owner", AuthorityPrincipalID: "reviewer", Version: 1},
		ControlImplementations: []continuity.ControlImplementation{{ID: "safeguard", OwnerPrincipalID: "owner", Status: continuity.ImplementationImplemented}},
	}
	programResponse := api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, now)
	if !programResponse.ResponsibilityLabelsComplete || resolver.batchCalls != 1 || resolver.scalarCalls != 0 || fmt.Sprint(resolver.batchIDs) != "[owner reviewer]" {
		t.Fatalf("program labels complete=%t batch=%d scalar=%d ids=%v", programResponse.ResponsibilityLabelsComplete, resolver.batchCalls, resolver.scalarCalls, resolver.batchIDs)
	}
}

func TestStoredResponsibilityRemainsVisibleWhenNameLookupFails(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "viewer"}
	api := &API{deps: Dependencies{Authority: labelBatchAuthority{principalID: "viewer"}}}
	response := api.buildMatterOperations(identity.WithActor(t.Context(), actor), actor, continuity.MatterAggregate{
		Matter: continuity.Matter{ID: "matter", TenantID: "bank", LegalEntityID: "entity", Status: continuity.MatterInitialReview, OwnerPrincipalID: "recorded-owner", Priority: 2, Version: 1},
	}, time.Now().UTC())
	if response.ResponsibilityLabelsComplete || len(response.ResponsibleParties) != 1 || response.ResponsibleParties[0].DisplayName != unavailableResponsiblePartyName {
		t.Fatalf("degraded responsibility response: %#v", response)
	}
}
