//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
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

func TestPostgresActivationServiceSimulatesIncompleteProposedRelationships(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	repository := NewPostgresRepository(pool)
	for index, status := range []AssessmentStatus{AssessmentCollecting, AssessmentReadyToSend, AssessmentSetupPending} {
		relationship := seedAssessmentRelationship(t, pool, "Hosted activation candidate "+string(rune('A'+index)))
		candidateTime := now.Add(time.Duration(index) * time.Minute)
		assessmentID := []string{
			"33333333-3333-7333-8333-333333333361",
			"33333333-3333-7333-8333-333333333362",
			"33333333-3333-7333-8333-333333333363",
		}[index]
		if _, err := repository.CreateAssessment(ctx, postgresAssessmentRecord(assessmentID, relationship, candidateTime)); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			matterID := "33333333-3333-7333-8333-333333333366"
			decisionID := "33333333-3333-7333-8333-333333333367"
			if _, err := pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,known_facts,missing_facts,contradictions,created_at,updated_at)
				VALUES($1::uuid,$2::uuid,$3::uuid,'MAT-ACTIVATION-REVIEW','VENDOR_REVIEW','DECISION_REQUIRED',3,'Review vendor onboarding','Confirm the onboarding assessment before activation.','{}'::jsonb,'THIRD_PARTY_ASSESSMENT',$4,'VENDOR_REGISTRATION_SUBMITTED','{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$5,$5)`, matterID, thirdPartyTenantID, thirdPartyEntityA, assessmentID, candidateTime); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO matter_decisions(id,tenant_id,matter_id,decision_type,status,options,rationale,conditions,authority_principal_id,decided_at,created_at,updated_at,version)
				VALUES($1::uuid,$2::uuid,$3::uuid,'VENDOR_APPROVAL','APPROVED','[]'::jsonb,'Assessment approved for activation simulation.','[]'::jsonb,$4::uuid,$5,$5,$5,1)`, decisionID, thirdPartyTenantID, matterID, thirdPartyPrincipal, candidateTime); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `UPDATE third_party_assessments SET review_matter_id=$3::uuid WHERE tenant_id=$1::uuid AND id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, assessmentID, matterID); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := pool.Exec(ctx, `UPDATE third_party_assessments SET status=$3,version=2,updated_at=$4 WHERE tenant_id=$1::uuid AND id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, assessmentID, status, candidateTime); err != nil {
			t.Fatal(err)
		}
	}

	maker := identity.Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal, Kind: "PERSON"}
	service := NewActivationService(repository, activationGuard{actor: maker})
	service.now = func() time.Time { return now }
	ids := []string{"33333333-3333-7333-8333-333333333364", "33333333-3333-7333-8333-333333333365"}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	policy, err := service.ProposePolicy(activationContext(maker), ProposeActivationPolicyInput{
		LegalEntityID: thirdPartyEntityA, AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory, AssessmentSatisfactoryWithConditions},
		MaximumAssessmentAgeDays: 365, RequiredDecisionTypes: []string{"VENDOR_APPROVAL"}, AddressVerificationRequired: true,
		BlockingMatterTypes: []string{"VENDOR_DEFICIENCY"}, ConditionalConclusionNeedsTerms: true, EffectiveFrom: now.Add(-time.Minute),
		Rationale: "Exercise every incomplete hosted activation candidate before independent approval.",
	})
	if err != nil {
		t.Fatal(err)
	}
	policy, err = service.SubmitPolicy(activationContext(maker), policy.ID, policy.Version, "Submit the complete reference policy for independent review.")
	if err != nil {
		t.Fatal(err)
	}
	simulation, err := service.SimulatePolicy(activationContext(maker), policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if simulation.CandidateCount != 3 || simulation.EligibleCount != 0 || !simulation.PopulationIsComplete || simulation.PolicyVersion != policy.Version {
		t.Fatalf("simulation = %#v", simulation)
	}
	if simulation.MissingGateCounts["CURRENT_ASSESSMENT"] != 3 || simulation.MissingGateCounts["ADDRESS_OUTCOME"] != 3 {
		t.Fatalf("missing gates = %#v", simulation.MissingGateCounts)
	}
}
