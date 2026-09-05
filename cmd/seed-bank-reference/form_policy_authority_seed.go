//go:build postgres

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	acceptanceAuthorityCode   = "CLEARSIGHT-REFERENCE-FORM-AUTOMATION"
	acceptanceAuthorityRuleID = "reference-form-policy-service"
)

type acceptanceAuthorityDefinition struct {
	Rules []acceptanceAuthorityRule `json:"rules"`
}

type acceptanceAuthorityRule struct {
	ID             string                      `json:"id"`
	LegalEntityID  string                      `json:"legal_entity_id"`
	ObjectType     string                      `json:"object_type"`
	ObjectID       string                      `json:"object_id"`
	Responsibility string                      `json:"responsibility"`
	DecisionType   string                      `json:"decision_type"`
	MinMateriality int                         `json:"min_materiality"`
	Priority       int                         `json:"priority"`
	Selector       acceptanceAuthoritySelector `json:"selector"`
}

type acceptanceAuthoritySelector struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

func ensureAcceptanceExecutionAuthority(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig) error {
	if pool == nil {
		return fmt.Errorf("response-policy acceptance authority requires PostgreSQL")
	}
	var tenantID, legalEntityID string
	if err := pool.QueryRow(ctx, `
		SELECT tenant.id::text,entity.id::text
		FROM tenants tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id AND entity.valid_until IS NULL
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		LIMIT 1`, seed.TenantID, seed.LegalEntityID).Scan(&tenantID, &legalEntityID); err != nil {
		return fmt.Errorf("resolve response-policy acceptance authority scope: %w", err)
	}

	serviceID, err := deterministicSeedUUID(ctx, pool, "clearsight:"+tenantID+":form-response-policy-service")
	if err != nil {
		return err
	}
	policyID, err := deterministicSeedUUID(ctx, pool, "clearsight:"+tenantID+":form-response-policy-authority")
	if err != nil {
		return err
	}
	versionID, err := deterministicSeedUUID(ctx, pool, "clearsight:"+tenantID+":form-response-policy-authority:v1")
	if err != nil {
		return err
	}

	definition := acceptanceAuthorityDefinition{Rules: []acceptanceAuthorityRule{{
		ID:             acceptanceAuthorityRuleID,
		LegalEntityID:  legalEntityID,
		ObjectType:     "FORM_RESPONSE_POLICY",
		ObjectID:       "*",
		Responsibility: "PERFORMER",
		DecisionType:   "forms.response-policy.execute",
		MinMateriality: 0,
		Priority:       300,
		Selector:       acceptanceAuthoritySelector{Kind: "PRINCIPAL", Ref: serviceID},
	}}}
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return fmt.Errorf("encode response-policy acceptance authority: %w", err)
	}
	checksumBytes := sha256.Sum256(definitionJSON)
	checksum := hex.EncodeToString(checksumBytes[:])

	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,external_ref,display_name,status,valid_from,valid_until)
		VALUES($1::uuid,$2::uuid,'SERVICE','reference-form-response-policy-automation','Form response policy automation','ACTIVE','2020-01-01T00:00:00Z',NULL)
		ON CONFLICT(id) DO UPDATE SET
			tenant_id=EXCLUDED.tenant_id,
			kind=EXCLUDED.kind,
			external_ref=EXCLUDED.external_ref,
			display_name=EXCLUDED.display_name,
			status='ACTIVE',
			valid_until=NULL`, serviceID, tenantID); err != nil {
		return fmt.Errorf("seed response-policy service principal: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO routing_policies(
			id,tenant_id,legal_entity_id,code,name,status,current_version,
			maker_id,checker_id,submitted_at,approved_at,version
		)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,'Reference form response automation','ACTIVE',1,$5::uuid,$6::uuid,clock_timestamp(),clock_timestamp(),1)
		ON CONFLICT(id) DO NOTHING`, policyID, tenantID, legalEntityID, acceptanceAuthorityCode, seed.ActorID, seed.ReviewerPrincipalID); err != nil {
		return fmt.Errorf("seed response-policy authority policy: %w", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO routing_policy_versions(
			id,policy_id,legal_entity_id,version,definition,checksum,effective_from,
			created_by,approved_by,approved_at
		)
		VALUES($1::uuid,$2::uuid,$3::uuid,1,$4::jsonb,$5,'2020-01-01T00:00:00Z',$6::uuid,$7::uuid,clock_timestamp())
		ON CONFLICT(policy_id,version) DO NOTHING`, versionID, policyID, legalEntityID, string(definitionJSON), checksum, seed.ActorID, seed.ReviewerPrincipalID); err != nil {
		return fmt.Errorf("seed response-policy authority version: %w", err)
	}

	var fixtureValid bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM routing_policies policy
			JOIN routing_policy_versions version ON version.policy_id=policy.id AND version.version=1
			WHERE policy.id=$1::uuid
			  AND policy.tenant_id=$2::uuid
			  AND policy.legal_entity_id=$3::uuid
			  AND policy.code=$4
			  AND policy.status='ACTIVE'
			  AND policy.current_version=1
			  AND version.id=$5::uuid
			  AND version.legal_entity_id=$3::uuid
			  AND version.definition=$6::jsonb
			  AND version.checksum=$7
			  AND version.approved_by=$8::uuid
			  AND version.approved_at IS NOT NULL
		)`, policyID, tenantID, legalEntityID, acceptanceAuthorityCode, versionID, string(definitionJSON), checksum, seed.ReviewerPrincipalID).Scan(&fixtureValid); err != nil {
		return fmt.Errorf("verify response-policy authority fixture: %w", err)
	}
	if !fixtureValid {
		return fmt.Errorf("response-policy authority fixture differs from the governed reference definition")
	}
	if _, err := pool.Exec(ctx, `SELECT refresh_effective_authority_routes($1::uuid)`, tenantID); err != nil {
		return fmt.Errorf("refresh response-policy authority projection: %w", err)
	}

	var routeValid bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM effective_authority_routes
			WHERE tenant_id=$1::uuid
			  AND source_policy_id=$2::uuid
			  AND source_rule_id=$3
			  AND legal_entity_ref=$4
			  AND object_type='FORM_RESPONSE_POLICY'
			  AND object_id='*'
			  AND responsibility='PERFORMER'
			  AND decision_type='forms.response-policy.execute'
			  AND selector_kind='PRINCIPAL'
			  AND selector_ref=$5
		)`, tenantID, policyID, acceptanceAuthorityRuleID, legalEntityID, serviceID).Scan(&routeValid); err != nil {
		return fmt.Errorf("verify response-policy service route: %w", err)
	}
	if !routeValid {
		return fmt.Errorf("response-policy automation service route was not projected")
	}
	return nil
}

func deterministicSeedUUID(ctx context.Context, pool *pgxpool.Pool, value string) (string, error) {
	var id string
	if err := pool.QueryRow(ctx, `SELECT md5($1)::uuid::text`, value).Scan(&id); err != nil {
		return "", fmt.Errorf("derive reference fixture id: %w", err)
	}
	return id, nil
}
