//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"testing"
	"time"
)

func TestPostgresActivationSimulationPersistsMissingGateCounts(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	repository := NewPostgresRepository(pool)
	policy, err := repository.ProposeActivationPolicy(ctx, ActivationPolicy{
		ID:                          "33333333-3333-7333-8333-333333333351",
		TenantID:                    "third-party-bank",
		LegalEntityID:               thirdPartyEntityA,
		AllowedConclusions:          []AssessmentConclusion{AssessmentSatisfactory},
		MaximumAssessmentAgeDays:    365,
		RequiredDecisionTypes:       []string{},
		AddressVerificationRequired: true,
		BlockingMatterTypes:         []string{},
		EffectiveFrom:               now,
		Status:                      ActivationPolicyDraft,
		ProposedBy:                  thirdPartyPrincipal,
		ProposalRationale:           "Exercise the durable activation simulation record.",
		CreatedAt:                   now,
		UpdatedAt:                   now,
		Version:                     1,
	})
	if err != nil {
		t.Fatal(err)
	}

	stored, err := repository.StoreActivationSimulation(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, ActivationSimulation{
		ID:                   "33333333-3333-7333-8333-333333333352",
		PolicyID:             policy.ID,
		PolicyVersion:        policy.Version,
		CandidateCount:       3,
		EligibleCount:        1,
		MissingGateCounts:    map[string]int{"ADDRESS_OUTCOME": 2, "CURRENT_ASSESSMENT": 1},
		PopulationIsComplete: true,
		EvaluatedBy:          thirdPartyPrincipal,
		EvaluatedAt:          now,
		ExpiresAt:            now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.GetActivationSimulation(ctx, Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, stored.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.PopulationIsComplete || loaded.MissingGateCounts["ADDRESS_OUTCOME"] != 2 || loaded.MissingGateCounts["CURRENT_ASSESSMENT"] != 1 {
		t.Fatalf("simulation round trip = %#v", loaded)
	}
}
