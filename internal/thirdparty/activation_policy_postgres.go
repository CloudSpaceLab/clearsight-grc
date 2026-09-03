//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) StoreActivationSimulation(ctx context.Context, scope Scope, value ActivationSimulation) (ActivationSimulation, error) {
	missing, err := json.Marshal(value.MissingGateCounts)
	if err != nil {
		return ActivationSimulation{}, err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO third_party_activation_policy_simulations(id,tenant_id,legal_entity_id,policy_id,policy_version,candidate_count,eligible_count,missing_gate_counts,population_is_complete,evaluated_by,evaluated_at,expires_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,$8::jsonb,$9,$10::uuid,$11,$12)`,
		value.ID, scope.TenantID, scope.LegalEntityID, value.PolicyID, value.PolicyVersion, value.CandidateCount, value.EligibleCount, missing, value.PopulationIsComplete, value.EvaluatedBy, value.EvaluatedAt, value.ExpiresAt)
	if err != nil {
		return ActivationSimulation{}, fmt.Errorf("store activation policy simulation: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) GetActivationSimulation(ctx context.Context, scope Scope, simulationID string) (ActivationSimulation, error) {
	var value ActivationSimulation
	var missing []byte
	err := r.pool.QueryRow(ctx, `SELECT s.id::text,s.policy_id::text,s.policy_version,s.candidate_count,s.eligible_count,s.missing_gate_counts,s.population_is_complete,s.evaluated_by::text,s.evaluated_at,s.expires_at
		FROM third_party_activation_policy_simulations s JOIN tenants t ON t.id=s.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND s.legal_entity_id::text=$2 AND s.id::text=$3`, scope.TenantID, scope.LegalEntityID, simulationID).
		Scan(&value.ID, &value.PolicyID, &value.PolicyVersion, &value.CandidateCount, &value.EligibleCount, &missing, &value.PopulationIsComplete, &value.EvaluatedBy, &value.EvaluatedAt, &value.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ActivationSimulation{}, ErrActivationSimulationRequired
	}
	if err != nil {
		return ActivationSimulation{}, err
	}
	if err = json.Unmarshal(missing, &value.MissingGateCounts); err != nil {
		return ActivationSimulation{}, err
	}
	return value, nil
}

func (r *PostgresRepository) ProposeActivationPolicy(ctx context.Context, policy ActivationPolicy) (ActivationPolicy, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ActivationPolicy{}, err
	}
	defer tx.Rollback(ctx)
	tenantID, err := resolveTenant(ctx, tx, policy.TenantID)
	if err != nil {
		return ActivationPolicy{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tenantID+":"+policy.LegalEntityID+":third-party-activation-policy"); err != nil {
		return ActivationPolicy{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(policy_number),0)+1 FROM third_party_activation_policies WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenantID, policy.LegalEntityID).Scan(&policy.PolicyNumber); err != nil {
		return ActivationPolicy{}, err
	}
	allowed := conclusionsToStrings(policy.AllowedConclusions)
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_activation_policies(
			id,tenant_id,legal_entity_id,policy_number,allowed_conclusions,maximum_assessment_age_days,required_decision_types,
			address_verification_required,blocking_matter_types,conditional_conclusion_needs_terms,effective_from,rollback_of_policy_id,status,
			proposed_by,proposal_rationale,created_at,updated_at,version)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,'DRAFT',$13::uuid,$14,$15,$15,1)`,
		policy.ID, tenantID, policy.LegalEntityID, policy.PolicyNumber, allowed, policy.MaximumAssessmentAgeDays, policy.RequiredDecisionTypes,
		policy.AddressVerificationRequired, policy.BlockingMatterTypes, policy.ConditionalConclusionNeedsTerms, policy.EffectiveFrom,
		policy.RollbackOfPolicyID, policy.ProposedBy, policy.ProposalRationale, policy.CreatedAt)
	if err != nil {
		return ActivationPolicy{}, fmt.Errorf("store activation policy proposal: %w", err)
	}
	if err = appendActivationPolicyEvent(ctx, tx, tenantID, policy, "PROPOSED", policy.ProposedBy, policy.ProposalRationale); err != nil {
		return ActivationPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ActivationPolicy{}, err
	}
	return policy, nil
}

func (r *PostgresRepository) TransitionActivationPolicy(ctx context.Context, scope Scope, policyID string, expectedVersion int64, to ActivationPolicyStatus, actorID, rationale string, at time.Time) (ActivationPolicy, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ActivationPolicy{}, err
	}
	defer tx.Rollback(ctx)
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return ActivationPolicy{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tenantID+":"+scope.LegalEntityID+":third-party-activation-policy"); err != nil {
		return ActivationPolicy{}, err
	}
	policy, err := scanActivationPolicy(tx.QueryRow(ctx, activationPolicySelect+` WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid FOR UPDATE`, tenantID, scope.LegalEntityID, policyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	if err != nil {
		return ActivationPolicy{}, err
	}
	if policy.Version != expectedVersion {
		return ActivationPolicy{}, ErrVersionConflict
	}
	if err := validatePolicyTransition(policy.Status, to); err != nil {
		return ActivationPolicy{}, err
	}

	if to == ActivationPolicyPendingApproval {
		policy.Status, policy.UpdatedAt, policy.Version = to, at.UTC(), policy.Version+1
		if _, err = tx.Exec(ctx, `UPDATE third_party_activation_policies SET status=$4,updated_at=$5,version=$6 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$7`, tenantID, scope.LegalEntityID, policy.ID, policy.Status, policy.UpdatedAt, policy.Version, expectedVersion); err != nil {
			return ActivationPolicy{}, err
		}
		if err = appendActivationPolicyEvent(ctx, tx, tenantID, policy, "SUBMITTED", actorID, rationale); err != nil {
			return ActivationPolicy{}, err
		}
	} else {
		rows, queryErr := tx.Query(ctx, activationPolicySelect+` WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid AND p.status='ACTIVE' AND p.id<>$3::uuid AND (p.effective_until IS NULL OR p.effective_until>$4) FOR UPDATE`, tenantID, scope.LegalEntityID, policy.ID, policy.EffectiveFrom)
		if queryErr != nil {
			return ActivationPolicy{}, queryErr
		}
		var priors []ActivationPolicy
		for rows.Next() {
			prior, scanErr := scanActivationPolicy(rows)
			if scanErr != nil {
				rows.Close()
				return ActivationPolicy{}, scanErr
			}
			priors = append(priors, prior)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return ActivationPolicy{}, err
		}
		rows.Close()
		for _, prior := range priors {
			priorEnd := policy.EffectiveFrom
			// Keep the incumbent active until the exact replacement boundary. A
			// future-dated approval must not create an activation-policy outage.
			prior.EffectiveUntil, prior.UpdatedAt, prior.Version = &priorEnd, at.UTC(), prior.Version+1
			if _, err = tx.Exec(ctx, `UPDATE third_party_activation_policies SET effective_until=$4,updated_at=$5,version=$6 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$7`, tenantID, scope.LegalEntityID, prior.ID, priorEnd, at.UTC(), prior.Version, prior.Version-1); err != nil {
				return ActivationPolicy{}, err
			}
			if err = appendActivationPolicyEvent(ctx, tx, tenantID, prior, "REPLACED", actorID, "Effective interval ends when independently approved policy "+policy.ID+" begins."); err != nil {
				return ActivationPolicy{}, err
			}
		}
		policy.Status, policy.ApprovedBy, policy.ApprovalRationale, policy.UpdatedAt, policy.Version = ActivationPolicyActive, actorID, rationale, at.UTC(), policy.Version+1
		if _, err = tx.Exec(ctx, `UPDATE third_party_activation_policies SET status='ACTIVE',approved_by=$4::uuid,approval_rationale=$5,updated_at=$6,version=$7 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$8`, tenantID, scope.LegalEntityID, policy.ID, actorID, rationale, at.UTC(), policy.Version, expectedVersion); err != nil {
			return ActivationPolicy{}, err
		}
		if err = appendActivationPolicyEvent(ctx, tx, tenantID, policy, "APPROVED", actorID, rationale); err != nil {
			return ActivationPolicy{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return ActivationPolicy{}, err
	}
	return policy, nil
}

func (r *PostgresRepository) GetActivationPolicy(ctx context.Context, scope Scope, policyID string) (ActivationPolicy, error) {
	policy, err := scanActivationPolicy(r.pool.QueryRow(ctx, activationPolicySelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.id::text=$3`, scope.TenantID, scope.LegalEntityID, policyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	return policy, err
}

func (r *PostgresRepository) CurrentActivationPolicy(ctx context.Context, scope Scope, at time.Time) (ActivationPolicy, error) {
	rows, err := r.pool.Query(ctx, activationPolicySelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.status='ACTIVE' AND p.effective_from<=$3 AND (p.effective_until IS NULL OR p.effective_until>$3) ORDER BY p.effective_from DESC,p.id DESC LIMIT 2`, scope.TenantID, scope.LegalEntityID, at)
	if err != nil {
		return ActivationPolicy{}, err
	}
	defer rows.Close()
	var values []ActivationPolicy
	for rows.Next() {
		value, scanErr := scanActivationPolicy(rows)
		if scanErr != nil {
			return ActivationPolicy{}, scanErr
		}
		values = append(values, value)
	}
	if err = rows.Err(); err != nil {
		return ActivationPolicy{}, err
	}
	if len(values) != 1 {
		return ActivationPolicy{}, ErrActivationPolicyUnavailable
	}
	return values[0], nil
}

func (r *PostgresRepository) ListActivationCandidates(ctx context.Context, scope Scope, cursor string, limit int) (ActivationCandidatePage, error) {
	args := []any{scope.TenantID, scope.LegalEntityID}
	whereCursor := ""
	if cursor != "" {
		cursorTime, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return ActivationCandidatePage{}, ErrInvalid
		}
		args = append(args, cursorTime, cursorID)
		whereCursor = " AND (r.updated_at,r.id) < ($3,$4::uuid)"
	}
	args = append(args, limit+1)
	query := relationshipSelect + ` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id::text=$2 AND r.status IN ('PROPOSED','UNDER_REVIEW')` + whereCursor + ` ORDER BY r.updated_at DESC,r.id DESC LIMIT $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return ActivationCandidatePage{}, err
	}
	defer rows.Close()
	items := make([]Relationship, 0, limit+1)
	for rows.Next() {
		value, scanErr := scanAggregate(rows)
		if scanErr != nil {
			return ActivationCandidatePage{}, scanErr
		}
		items = append(items, value.Relationship)
	}
	if err = rows.Err(); err != nil {
		return ActivationCandidatePage{}, err
	}
	page := ActivationCandidatePage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (r *PostgresRepository) ReadActivationFacts(ctx context.Context, scope Scope, relationshipID string, policy ActivationPolicy) (Relationship, ActivationFacts, error) {
	return readActivationFacts(ctx, r.pool, scope.TenantID, scope.LegalEntityID, relationshipID, policy)
}

func (r *PostgresRepository) CommitRelationshipActivation(ctx context.Context, commit ActivationCommit) (Relationship, ActivationReceipt, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	defer tx.Rollback(ctx)
	tenantID, err := resolveTenant(ctx, tx, commit.TenantID)
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	policy, err := scanActivationPolicy(tx.QueryRow(ctx, activationPolicySelect+` WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid FOR UPDATE`, tenantID, commit.LegalEntityID, commit.Policy.ID))
	if err != nil || policy.Version != commit.Policy.Version || policy.Status != ActivationPolicyActive || policy.EffectiveFrom.After(commit.EffectiveAt) || (policy.EffectiveUntil != nil && !policy.EffectiveUntil.After(commit.EffectiveAt)) {
		return Relationship{}, ActivationReceipt{}, ErrActivationPolicyUnavailable
	}
	var currentVersion int64
	if err = tx.QueryRow(ctx, `SELECT version FROM third_party_relationships WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid FOR UPDATE`, tenantID, commit.LegalEntityID, commit.RelationshipID).Scan(&currentVersion); errors.Is(err, pgx.ErrNoRows) {
		return Relationship{}, ActivationReceipt{}, ErrNotFound
	}
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	if currentVersion != commit.ExpectedVersion {
		return Relationship{}, ActivationReceipt{}, ErrVersionConflict
	}
	relationship, facts, err := readActivationFacts(ctx, tx, tenantID, commit.LegalEntityID, commit.RelationshipID, policy)
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	facts.DecisionAuthoritiesCurrent = commit.Facts.DecisionAuthoritiesCurrent
	if !gatesSatisfied(activationGates(relationship, facts, policy, commit.EffectiveAt)) || facts.AssessmentID != commit.Facts.AssessmentID || facts.AssessmentVersion != commit.Facts.AssessmentVersion || facts.VerificationResultID != commit.Facts.VerificationResultID || strings.Join(facts.DecisionIDs, ",") != strings.Join(commit.Facts.DecisionIDs, ",") {
		return Relationship{}, ActivationReceipt{}, ErrActivationIneligible
	}
	if _, err = tx.Exec(ctx, `UPDATE third_party_relationships SET status='ACTIVE',effective_from=$4,updated_at=$4,version=version+1 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$5`, tenantID, commit.LegalEntityID, commit.RelationshipID, commit.EffectiveAt, commit.ExpectedVersion); err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	stored, err := scanAggregate(tx.QueryRow(ctx, relationshipSelect+` WHERE t.id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`, tenantID, commit.LegalEntityID, commit.RelationshipID))
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	receipt := ActivationReceipt{ID: commit.ReceiptID, TenantID: commit.TenantID, LegalEntityID: commit.LegalEntityID, RelationshipID: stored.Relationship.ID, RelationshipVersion: stored.Relationship.Version, PolicyID: policy.ID, PolicyVersion: policy.Version, AssessmentID: facts.AssessmentID, AssessmentVersion: facts.AssessmentVersion, DecisionIDs: facts.DecisionIDs, AddressMatterID: facts.AddressMatterID, VerificationResultID: facts.VerificationResultID, ActivatedBy: commit.ActorID, ActivatedAt: commit.EffectiveAt, Rationale: commit.Rationale}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_activation_receipts(id,tenant_id,legal_entity_id,relationship_id,relationship_version,policy_id,policy_version,assessment_id,assessment_version,decision_ids,address_matter_id,verification_result_id,activated_by,activated_at,rationale) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9,$10::uuid[],NULLIF($11,'')::uuid,NULLIF($12,'')::uuid,$13::uuid,$14,$15)`, receipt.ID, tenantID, receipt.LegalEntityID, receipt.RelationshipID, receipt.RelationshipVersion, receipt.PolicyID, receipt.PolicyVersion, receipt.AssessmentID, receipt.AssessmentVersion, receipt.DecisionIDs, receipt.AddressMatterID, receipt.VerificationResultID, receipt.ActivatedBy, receipt.ActivatedAt, receipt.Rationale)
	if err != nil {
		return Relationship{}, ActivationReceipt{}, fmt.Errorf("store activation receipt: %w", err)
	}
	eventID, err := appendRelationshipEvent(ctx, tx, tenantID, stored.Relationship, commit.ActorID, "VendorRelationshipActivated")
	if err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	if err = r.commitThirdPartyEvents(ctx, tx, relationshipCommitProof(eventID, stored.Relationship, "VendorRelationshipActivated")); err != nil {
		return Relationship{}, ActivationReceipt{}, err
	}
	return stored.Relationship, receipt, nil
}

type activationQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readActivationFacts(ctx context.Context, q activationQuerier, tenant, entity, relationshipID string, policy ActivationPolicy) (Relationship, ActivationFacts, error) {
	aggregate, err := scanAggregate(q.QueryRow(ctx, relationshipSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id::text=$2 AND r.id::text=$3`, tenant, entity, relationshipID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Relationship{}, ActivationFacts{}, ErrNotFound
	}
	if err != nil {
		return Relationship{}, ActivationFacts{}, err
	}
	facts := ActivationFacts{}
	var completedAt *time.Time
	var reviewMatterID string
	err = q.QueryRow(ctx, `
		SELECT a.id::text,a.version,a.status,COALESCE(a.conclusion,''),a.completed_at,COALESCE(a.review_matter_id::text,'')
		FROM third_party_assessments a JOIN tenants t ON t.id=a.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id::text=$2 AND a.relationship_id::text=$3 AND a.review_kind='ONBOARDING'
		ORDER BY a.started_at DESC,a.id DESC LIMIT 1`, tenant, entity, relationshipID).
		Scan(&facts.AssessmentID, &facts.AssessmentVersion, &facts.AssessmentStatus, &facts.AssessmentConclusion, &completedAt, &reviewMatterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return aggregate.Relationship, facts, nil
	}
	if err != nil {
		return Relationship{}, ActivationFacts{}, err
	}
	if completedAt != nil {
		facts.AssessmentCompletedAt = completedAt.UTC()
	}
	if reviewMatterID != "" {
		rows, queryErr := q.Query(ctx, `SELECT id::text,decision_type,conditions,COALESCE(authority_principal_id::text,'') FROM matter_decisions d JOIN tenants t ON t.id=d.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND d.matter_id::text=$2 AND d.status IN ('APPROVED','CONDITIONALLY_APPROVED') AND (d.expires_at IS NULL OR d.expires_at>clock_timestamp()) ORDER BY d.created_at DESC,d.id DESC`, tenant, reviewMatterID)
		if queryErr != nil {
			return Relationship{}, ActivationFacts{}, queryErr
		}
		seen := map[string]bool{}
		for rows.Next() {
			var decisionID, decisionType, decisionAuthorityID string
			var conditions []byte
			if scanErr := rows.Scan(&decisionID, &decisionType, &conditions, &decisionAuthorityID); scanErr != nil {
				rows.Close()
				return Relationship{}, ActivationFacts{}, scanErr
			}
			decisionType = strings.ToUpper(decisionType)
			if !seen[decisionType] {
				seen[decisionType] = true
				facts.SatisfiedDecisionTypes = append(facts.SatisfiedDecisionTypes, decisionType)
				facts.DecisionIDs = append(facts.DecisionIDs, decisionID)
				facts.DecisionDependencies = append(facts.DecisionDependencies, ActivationDecisionDependency{ID: decisionID, MatterID: reviewMatterID, DecisionType: decisionType, AuthorityPrincipalID: decisionAuthorityID})
			}
			if string(conditions) != "[]" && string(conditions) != "{}" && string(conditions) != "null" {
				facts.ConditionsRecorded = true
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return Relationship{}, ActivationFacts{}, err
		}
		rows.Close()
	}
	var addressStatus string
	err = q.QueryRow(ctx, `
		SELECT m.id::text,m.status,COALESCE(vr.id::text,''),COALESCE(vr.result='PASS',false)
		FROM matters m JOIN tenants t ON t.id=m.tenant_id
		LEFT JOIN LATERAL (
			SELECT vr.id,vr.result FROM verification_contracts vc
			JOIN verification_results vr ON vr.tenant_id=vc.tenant_id AND vr.matter_id=vc.matter_id AND vr.contract_id=vc.id
			WHERE vc.tenant_id=m.tenant_id AND vc.matter_id=m.id AND vc.status='ACTIVE'
			ORDER BY vr.observed_at DESC,vr.id DESC LIMIT 1
		) vr ON true
		WHERE (t.id::text=$1 OR t.slug=$1) AND m.legal_entity_id::text=$2 AND m.source_type='THIRD_PARTY_ASSESSMENT' AND m.source_id=$3 AND m.trigger_type='VENDOR_REGISTRATION_SUBMITTED'
		ORDER BY m.created_at DESC,m.id DESC LIMIT 1`, tenant, entity, facts.AssessmentID).
		Scan(&facts.AddressMatterID, &addressStatus, &facts.VerificationResultID, &facts.VerificationPassed)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Relationship{}, ActivationFacts{}, err
	}
	facts.AddressMatterClosed = addressStatus == "CLOSED"
	if len(policy.BlockingMatterTypes) > 0 {
		if err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.legal_entity_id::text=$2 AND m.status NOT IN ('CLOSED','CANCELLED') AND m.matter_type=ANY($3::text[]) AND (m.scope->>'relationship_id'=$4 OR (m.source_type='THIRD_PARTY_ASSESSMENT' AND m.source_id=$5)))`, tenant, entity, policy.BlockingMatterTypes, relationshipID, facts.AssessmentID).Scan(&facts.HasBlockingMatter); err != nil {
			return Relationship{}, ActivationFacts{}, err
		}
	}
	if err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.legal_entity_id::text=$2 AND m.status NOT IN ('CLOSED','CANCELLED') AND jsonb_array_length(m.contradictions)>0 AND (m.scope->>'relationship_id'=$3 OR (m.source_type='THIRD_PARTY_ASSESSMENT' AND m.source_id=$4)))`, tenant, entity, relationshipID, facts.AssessmentID).Scan(&facts.HasUnresolvedContradiction); err != nil {
		return Relationship{}, ActivationFacts{}, err
	}
	return aggregate.Relationship, facts, nil
}

const activationPolicySelect = `SELECT p.id::text,t.slug,p.legal_entity_id::text,p.policy_number,p.allowed_conclusions,p.maximum_assessment_age_days,p.required_decision_types,p.address_verification_required,p.blocking_matter_types,p.conditional_conclusion_needs_terms,p.effective_from,p.effective_until,COALESCE(p.rollback_of_policy_id::text,''),p.status,p.proposed_by::text,COALESCE(p.approved_by::text,''),p.proposal_rationale,p.approval_rationale,p.created_at,p.updated_at,p.version FROM third_party_activation_policies p JOIN tenants t ON t.id=p.tenant_id`

func scanActivationPolicy(row rowScanner) (ActivationPolicy, error) {
	var value ActivationPolicy
	var conclusions []string
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyNumber, &conclusions, &value.MaximumAssessmentAgeDays, &value.RequiredDecisionTypes, &value.AddressVerificationRequired, &value.BlockingMatterTypes, &value.ConditionalConclusionNeedsTerms, &value.EffectiveFrom, &value.EffectiveUntil, &value.RollbackOfPolicyID, &value.Status, &value.ProposedBy, &value.ApprovedBy, &value.ProposalRationale, &value.ApprovalRationale, &value.CreatedAt, &value.UpdatedAt, &value.Version)
	for _, conclusion := range conclusions {
		value.AllowedConclusions = append(value.AllowedConclusions, AssessmentConclusion(conclusion))
	}
	return value, err
}

func appendActivationPolicyEvent(ctx context.Context, tx pgx.Tx, tenantID string, policy ActivationPolicy, eventType, actorID, rationale string) error {
	_, err := tx.Exec(ctx, `INSERT INTO third_party_activation_policy_events(tenant_id,legal_entity_id,policy_id,policy_version,event_type,actor_principal_id,rationale,payload,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::uuid,$7,jsonb_build_object('policy_number',$8::int,'status',$9::text,'effective_from',$10::timestamptz,'effective_until',$11::timestamptz),$12)`, tenantID, policy.LegalEntityID, policy.ID, policy.Version, eventType, actorID, rationale, policy.PolicyNumber, policy.Status, policy.EffectiveFrom, policy.EffectiveUntil, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append activation policy event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'THIRD_PARTY_ACTIVATION_POLICY',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'status',$5::text),$6,$6)`, tenantID, policy.ID, "ThirdPartyActivationPolicy"+eventType, policy.Version, policy.Status, policy.UpdatedAt)
	return err
}

func conclusionsToStrings(values []AssessmentConclusion) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = string(v)
	}
	return result
}

var _ ActivationRepository = (*PostgresRepository)(nil)
