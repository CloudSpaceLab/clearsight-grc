//go:build postgres

package formpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
)

func (repo *PostgresRepository) ListAutomationChoices(ctx context.Context, tenant, entity, form string, version int64, limit int) ([]AutomationChoice, error) {
	rows, err := repo.pool.Query(ctx, `SELECT DISTINCT ON(ap.code) ap.id::text,ap.name,ap.status,ap.version,ap.eligibility,ap.blast_radius_limit,ap.verification_contract,ap.rollout_mode,ap.effective_from,ap.effective_until,
 (SELECT count(*) FROM form_response_policy_definitions used JOIN legal_entities le ON le.id=used.legal_entity_id AND le.tenant_id=used.tenant_id WHERE used.tenant_id=ap.tenant_id AND (le.id::text=$6 OR le.code=$6) AND used.automation_policy_id=ap.id AND used.automation_policy_version=ap.version)
 FROM automation_policies ap JOIN tenants t ON t.id=ap.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND ap.action_class=$2 AND ap.eligibility->>'form_template_id'=$3 AND ap.eligibility->>'form_template_version'=$4
 AND EXISTS(SELECT 1 FROM monitoring_form_templates f JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id WHERE f.tenant_id=ap.tenant_id AND f.id::text=$3 AND f.version::text=$4 AND (le.id::text=$6 OR le.code=$6))
 AND NOT EXISTS(SELECT 1 FROM automation_policies newer WHERE newer.tenant_id=ap.tenant_id AND newer.code=ap.code AND newer.version>ap.version)
 ORDER BY ap.code,ap.version DESC LIMIT $5`, tenant, ActionClassCreateMatter, form, fmt.Sprint(version), limit, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []AutomationChoice{}
	for rows.Next() {
		var value AutomationChoice
		var eligibility, blast, outcome []byte
		if err := rows.Scan(&value.ID, &value.Name, &value.Status, &value.Version, &eligibility, &blast, &outcome, &value.Rollout, &value.EffectiveFrom, &value.EffectiveUntil, &value.UseCount); err != nil {
			return nil, err
		}
		if json.Unmarshal(eligibility, &value.Eligibility) != nil || json.Unmarshal(blast, &value.BlastRadius) != nil || json.Unmarshal(outcome, &value.Outcome) != nil {
			return nil, ErrInvalid
		}
		value.Purpose = value.Outcome.ExpectedOutcome
		result = append(result, value)
	}
	return result, rows.Err()
}

func insertManagedAutomationTx(ctx context.Context, tx pgx.Tx, value Policy, eligibility, blast, outcome []byte) error {
	_, err := tx.Exec(ctx, `INSERT INTO automation_policies(id,tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,status,effective_from,effective_until,version,rollout_mode,maker_id,checksum,record_version,created_at,updated_at)
 VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9,$10,$11,$12,$13,$14::uuid,$15,$16,$17,$18)`,
		value.AutomationPolicyID, value.TenantID, "forms:"+value.LegalEntityID+":"+value.Code, value.Name, ActionClassCreateMatter,
		eligibility, blast, outcome, value.Status, value.EffectiveFrom, value.EffectiveUntil, value.AutomationPolicyVersion, value.Rollout, value.MakerID, value.Checksum, value.RecordVersion, value.CreatedAt, value.UpdatedAt)
	return normalizePostgresError(err)
}

func updateManagedAutomationTx(ctx context.Context, tx pgx.Tx, value Policy, expected int64) error {
	tag, err := tx.Exec(ctx, `UPDATE automation_policies SET status=$4,checker_id=NULLIF($5,'')::uuid,submitted_at=$6,approved_at=$7,activated_at=$8,suspended_at=$9,retired_at=$10,record_version=$11,updated_at=$12
 WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=$3 AND record_version=$13 AND checksum=$14 AND action_class=$15`,
		value.AutomationPolicyID, value.TenantID, value.AutomationPolicyVersion, value.Status, value.CheckerID, value.SubmittedAt, value.ApprovedAt, value.ActivatedAt, value.SuspendedAt, value.RetiredAt, value.RecordVersion, value.UpdatedAt, expected, value.Checksum, ActionClassCreateMatter)
	if err != nil {
		return normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}
