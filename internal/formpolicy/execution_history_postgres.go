//go:build postgres

package formpolicy

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (repo *PostgresRepository) GetExecutionResult(ctx context.Context, tenant, entity, policy, execution string) (ExecutionResultSource, error) {
	var executionID pgtype.UUID
	if err := executionID.Scan(execution); err != nil || !executionID.Valid {
		return ExecutionResultSource{}, ErrNotFound
	}
	var value ExecutionResultSource
	err := repo.pool.QueryRow(ctx, `SELECT e.id::text,e.state,e.result_basis,e.assessment_version,e.created_at,e.response_revision_id::text,COALESCE(e.matter_id::text,'')
 FROM (SELECT id,tenant_id,legal_entity_id,policy_id,state,result_basis,assessment_version,created_at,response_revision_id,matter_id FROM form_response_policy_executions WHERE id=$4
 UNION ALL SELECT id,tenant_id,legal_entity_id,policy_id,'FAILED' AS state,result_basis,assessment_version,created_at,response_revision_id,NULL::uuid AS matter_id FROM form_response_policy_execution_failures WHERE id=$4) e
 JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id::text=$3`, tenant, entity, policy, executionID).Scan(&value.ID, &value.State, &value.ResultBasis, &value.AssessmentVersion, &value.CreatedAt, &value.ResponseRevisionID, &value.MatterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionResultSource{}, ErrNotFound
	}
	return value, err
}

func (repo *PostgresRepository) ListExecutionHistory(ctx context.Context, tenant, entity, policy string, limit int) ([]ExecutionHistoryItem, error) {
	rows, err := repo.pool.Query(ctx, `SELECT e.id::text,e.state,e.result_basis,e.assessment_version,e.created_at FROM (SELECT id,tenant_id,legal_entity_id,policy_id,state,result_basis,assessment_version,created_at FROM form_response_policy_executions UNION ALL SELECT id,tenant_id,legal_entity_id,policy_id,'FAILED' AS state,result_basis,assessment_version,created_at FROM form_response_policy_execution_failures) e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id::text=$3 ORDER BY e.created_at DESC,e.id DESC LIMIT $4`, tenant, entity, policy, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []ExecutionHistoryItem{}
	for rows.Next() {
		var value ExecutionHistoryItem
		if err := rows.Scan(&value.ID, &value.State, &value.ResultBasis, &value.AssessmentVersion, &value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
