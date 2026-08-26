package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type countingMatterBatchAuthority struct {
	batchCalls  int
	scalarCalls int
	inputs      []authority.ResolveInput
	batchErr    error
	truncate    int
}

func (s *countingMatterBatchAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	s.scalarCalls++
	return authority.Resolution{}, errors.New("scalar authority resolution must not be used for Matter operations")
}

func (s *countingMatterBatchAuthority) ResolveMany(_ context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	s.batchCalls++
	s.inputs = append([]authority.ResolveInput(nil), inputs...)
	if s.batchErr != nil {
		return nil, s.batchErr
	}
	length := len(inputs) - s.truncate
	if length < 0 {
		length = 0
	}
	outcomes := make([]authority.ResolveOutcome, length)
	for index := range outcomes {
		outcomes[index].Resolution = authority.Resolution{
			Principal:           authority.Principal{ID: "actor-1", DisplayName: "Current responsible person"},
			CandidatePrincipals: []authority.Principal{{ID: "actor-1", DisplayName: "Current responsible person"}},
			EffectiveOrigins:    []authority.EffectiveOrigin{{PrincipalID: "actor-1", OriginPrincipalID: "actor-1"}},
		}
	}
	return outcomes, nil
}

func (s *countingMatterBatchAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s *countingMatterBatchAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s *countingMatterBatchAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

type scalarOnlyMatterAuthority struct{ scalarCalls int }

func (s *scalarOnlyMatterAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	s.scalarCalls++
	return authority.Resolution{Principal: authority.Principal{ID: "actor-1", DisplayName: "Current responsible person"}}, nil
}

func (s *scalarOnlyMatterAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (s *scalarOnlyMatterAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (s *scalarOnlyMatterAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func TestMatterOperationsResolveAuthorityInOneStableBatchForAnyActionAndOutcomeCount(t *testing.T) {
	for _, count := range []int{1, 40} {
		t.Run(fmt.Sprintf("records_%d", count), func(t *testing.T) {
			now := time.Now().UTC()
			aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
				ID: "matter-batch", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap,
				Status: continuity.MatterActionsInProgress, Priority: 4, OwnerPrincipalID: "actor-1",
				CreatedAt: now, UpdatedAt: now, Version: 7,
			}}
			for index := 0; index < count; index++ {
				aggregate.Actions = append(aggregate.Actions, continuity.Action{
					ID: fmt.Sprintf("action-%02d", index), MatterID: aggregate.Matter.ID, Title: "Remediate control gap",
					OwnerPrincipalID: "actor-1", RequiredResponsibility: string(authority.ResponsibilityPerformer),
					Status: continuity.ActionPlanned, CreatedAt: now, UpdatedAt: now, Version: 1,
				})
				aggregate.VerificationContracts = append(aggregate.VerificationContracts, continuity.VerificationContract{
					ID: fmt.Sprintf("outcome-%02d", index), MatterID: aggregate.Matter.ID,
					ExpectedOutcome: "The control gap remains resolved.", AuthorityPrincipalID: "actor-1",
					Status: continuity.VerificationActive, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour), Version: 1,
				})
			}

			resolver := &countingMatterBatchAuthority{}
			api := &API{deps: Dependencies{Authority: resolver}}
			payload := api.buildMatterOperations(t.Context(), identity.Actor{
				TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "actor-1",
			}, aggregate, now)

			if resolver.batchCalls != 1 || resolver.scalarCalls != 0 {
				t.Fatalf("authority calls = batch %d scalar %d, want one batch and zero scalar for %d actions/outcomes", resolver.batchCalls, resolver.scalarCalls, count)
			}
			cursor := 0
			actionTransitions, outcomeRecords := 0, 0
			for _, operation := range payload.Operations {
				if cursor >= len(resolver.inputs) {
					t.Fatalf("operation %s/%s has no aligned primary batch input", operation.Command, operation.SubresourceID)
				}
				primary := resolver.inputs[cursor]
				cursor++
				if primary.TenantID != "bank" || primary.LegalEntityID != "entity-a" || primary.ObjectType != "MATTER" || primary.ObjectID != "matter-batch" ||
					primary.DecisionType != operation.Command || string(primary.Responsibility) != operation.Responsibility || !primary.At.Equal(now) {
					t.Fatalf("primary input for %s/%s = %#v", operation.Command, operation.SubresourceID, primary)
				}
				if operation.Command == "matter.action.add" || operation.Command == "matter.action.assign" {
					if cursor >= len(resolver.inputs) {
						t.Fatalf("operation %s/%s has no aligned candidate batch input", operation.Command, operation.SubresourceID)
					}
					candidate := resolver.inputs[cursor]
					cursor++
					if candidate.DecisionType != operation.Command || candidate.Responsibility != authority.ResponsibilityPerformer || candidate.ObjectID != "matter-batch" {
						t.Fatalf("candidate input for %s/%s = %#v", operation.Command, operation.SubresourceID, candidate)
					}
				}
				if operation.Command == "matter.action.transition" {
					actionTransitions++
				}
				if operation.Command == "matter.outcome.record" {
					outcomeRecords++
				}
			}
			if cursor != len(resolver.inputs) {
				t.Fatalf("%d batch inputs were not aligned to operations", len(resolver.inputs)-cursor)
			}
			if actionTransitions != count || outcomeRecords != count {
				t.Fatalf("materialized actions/outcomes = %d/%d, want %d/%d", actionTransitions, outcomeRecords, count, count)
			}
		})
	}
}

func TestMatterOperationsFailEveryOperationClosedWhenBatchAuthorityCannotComplete(t *testing.T) {
	now := time.Now().UTC()
	aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-batch", TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap,
		Status: continuity.MatterAssessment, Priority: 4, OwnerPrincipalID: "actor-1",
		CreatedAt: now, UpdatedAt: now, Version: 7,
	}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "actor-1"}

	for _, test := range []struct {
		name      string
		authority authority.Service
	}{
		{
			name: "batch interface unavailable",
			authority: func() authority.Service {
				return &scalarOnlyMatterAuthority{}
			}(),
		},
		{
			name: "batch failure",
			authority: func() authority.Service {
				return &countingMatterBatchAuthority{batchErr: errors.New("authority database unavailable")}
			}(),
		},
		{
			name: "incomplete batch",
			authority: func() authority.Service {
				return &countingMatterBatchAuthority{truncate: 1}
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &API{deps: Dependencies{Authority: test.authority}}
			payload := api.buildMatterOperations(t.Context(), actor, aggregate, now)
			if payload.AuthorityAvailable {
				t.Fatal("authority was reported available after batch resolution could not complete")
			}
			if len(payload.Operations) == 0 {
				t.Fatal("fail-closed response omitted the operations that need authority recovery")
			}
			for _, operation := range payload.Operations {
				if operation.CanAct || !strings.Contains(operation.Reason, "Responsibility could not be checked") {
					t.Fatalf("operation did not fail closed: %#v", operation)
				}
			}
			switch resolver := test.authority.(type) {
			case *scalarOnlyMatterAuthority:
				if resolver.scalarCalls != 0 {
					t.Fatalf("scalar authority fallback was used %d times", resolver.scalarCalls)
				}
			case *countingMatterBatchAuthority:
				if resolver.batchCalls != 1 || resolver.scalarCalls != 0 {
					t.Fatalf("authority calls = batch %d scalar %d", resolver.batchCalls, resolver.scalarCalls)
				}
			}
		})
	}
}
