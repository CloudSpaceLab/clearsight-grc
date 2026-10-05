//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func loadOutsideAppetiteMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("risks_outside_appetite")
	rows, err := tx.Query(ctx, `
		WITH selected AS (
			SELECT risk.id,risk.name,risk.version,
			       CASE
			         WHEN assessment.id IS NULL OR assessment.risk_version<>risk.version THEN 'UNKNOWN'
			         WHEN assessment.appetite_statement_id IS NULL THEN 'UNKNOWN'
			         WHEN appetite.id IS NULL OR assessment.appetite_statement_id<>appetite.id THEN 'UNKNOWN'
			         ELSE COALESCE(NULLIF(assessment.appetite_position,''),'UNKNOWN')
			       END state
			FROM risks risk
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM risk_assessments candidate
				WHERE candidate.tenant_id=risk.tenant_id
				  AND candidate.legal_entity_id=risk.legal_entity_id
				  AND candidate.risk_id=risk.id
				ORDER BY candidate.risk_version DESC,candidate.id DESC
				LIMIT 1
			) assessment ON true
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM (
					SELECT statement.*
					FROM risk_appetite_statements statement
					WHERE statement.tenant_id=risk.tenant_id
					  AND statement.legal_entity_id=risk.legal_entity_id
					  AND statement.risk_id=risk.id
					  AND statement.effective_from<=$3
					ORDER BY statement.version DESC,statement.id DESC
					LIMIT 1
				) candidate
				WHERE candidate.status='ACTIVE'
				  AND (candidate.effective_until IS NULL OR $3<candidate.effective_until)
			) appetite ON true
			WHERE risk.tenant_id=$1::uuid
			  AND risk.legal_entity_id=$2::uuid
			  AND risk.status='ACTIVE'
		)
		SELECT id::text,name,state FROM selected ORDER BY id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load outside-appetite metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title, state string
		if err := rows.Scan(&id, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "BREACHED" {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "RISK", TargetID: id, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadIndicatorBreachMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("indicator_breaches")
	rows, err := tx.Query(ctx, `
		WITH current_links AS (
			SELECT DISTINCT ON (link.risk_id,link.monitoring_check_id)
			       link.id,link.risk_id,link.program_id,link.monitoring_check_id,link.monitoring_check_version,link.kind
			FROM risk_indicator_links link
			JOIN risks risk
			  ON risk.tenant_id=link.tenant_id
			 AND risk.legal_entity_id=link.legal_entity_id
			 AND risk.id=link.risk_id
			 AND risk.status='ACTIVE'
			WHERE link.tenant_id=$1::uuid
			  AND link.legal_entity_id=$2::uuid
			ORDER BY link.risk_id,link.monitoring_check_id,link.risk_version DESC,link.id DESC
		)
		SELECT link.id::text,risk.id::text,risk.name,
		       CASE
		         WHEN check_config.revision_id IS NULL OR check_config.status<>'ACTIVE' OR NOT check_config.is_current THEN 'UNKNOWN'
		         WHEN program.id IS NULL OR program.status<>'ACTIVE' THEN 'UNKNOWN'
		         WHEN result.id IS NULL THEN 'UNKNOWN'
		         WHEN result.evaluated_at < $3-(check_config.freshness_minutes*interval '1 minute') THEN 'UNKNOWN'
		         WHEN NULLIF(result.evaluation->>'coverage','')::numeric < check_config.minimum_coverage THEN 'UNKNOWN'
		         WHEN result.evaluation->>'band'='CRITICAL' THEN 'CRITICAL'
		         WHEN result.evaluation->>'band'='HIGH' THEN 'HIGH'
		         WHEN result.evaluation->>'band'='MODERATE' THEN 'WATCH'
		         WHEN result.evaluation->>'band'='LOW' THEN 'NORMAL'
		         ELSE 'UNKNOWN'
		       END state
		FROM current_links link
		JOIN risks risk ON risk.id=link.risk_id AND risk.tenant_id=$1::uuid AND risk.legal_entity_id=$2::uuid
		LEFT JOIN monitoring_checks check_config
		  ON check_config.tenant_id=$1::uuid
		 AND check_config.id=link.monitoring_check_id
		 AND check_config.version=link.monitoring_check_version
		 AND check_config.program_id=link.program_id
		LEFT JOIN programs program
		  ON program.tenant_id=$1::uuid
		 AND program.id=link.program_id
		 AND program.legal_entity_id=$2::uuid
		LEFT JOIN LATERAL (
			SELECT candidate.*
			FROM monitoring_results candidate
			WHERE candidate.tenant_id=$1::uuid
			  AND candidate.program_id=link.program_id
			  AND candidate.monitoring_check_id=link.monitoring_check_id
			  AND candidate.monitoring_check_version=link.monitoring_check_version
			  AND candidate.evaluated_at<=$3
			ORDER BY candidate.evaluated_at DESC,candidate.id DESC
			LIMIT 1
		) result ON true
		ORDER BY link.id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load indicator-breach metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var memberID, riskID, title, state string
		if err := rows.Scan(&memberID, &riskID, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "HIGH" || state == "CRITICAL" {
			result.Members = append(result.Members, domainMetricMember{MemberID: memberID, TargetType: "RISK", TargetID: riskID, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadAssuranceFailureMetric(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("assurance_failures")
	rows, err := tx.Query(ctx, `
		WITH selected_risks AS (
			SELECT risk.id,risk.tenant_id,risk.legal_entity_id,risk.name
			FROM risks risk
			WHERE risk.tenant_id=$1::uuid
			  AND risk.legal_entity_id=$2::uuid
			  AND risk.status='ACTIVE'
		), eligible_links AS (
			SELECT risk_link.risk_id,
			       implementation.tenant_id,
			       implementation.program_id,
			       implementation.id implementation_id,
			       implementation.status implementation_status
			FROM risk_control_links risk_link
			JOIN control_catalog_implementation_links catalog_link
			  ON catalog_link.id=risk_link.catalog_link_id
			 AND catalog_link.tenant_id=risk_link.tenant_id
			 AND catalog_link.legal_entity_id=risk_link.legal_entity_id
			JOIN control_definitions definition
			  ON definition.id=catalog_link.definition_id
			 AND definition.tenant_id=catalog_link.tenant_id
			 AND definition.status='ACTIVE'
			JOIN control_implementations implementation
			  ON implementation.id=catalog_link.implementation_id
			 AND implementation.tenant_id=catalog_link.tenant_id
			 AND implementation.program_id=catalog_link.program_id
			 AND implementation.status NOT IN ('INACTIVE','RETIRED')
			 AND implementation.effective_from<=$3
			 AND (implementation.effective_until IS NULL OR $3<implementation.effective_until)
			WHERE risk_link.tenant_id=$1::uuid
			  AND risk_link.legal_entity_id=$2::uuid
			  AND EXISTS (SELECT 1 FROM selected_risks risk WHERE risk.id=risk_link.risk_id)
		), contract_facts AS (
			SELECT risk.id risk_id,
			       CASE
			         WHEN link.implementation_id IS NULL OR link.implementation_status<>'IMPLEMENTED' THEN 'UNKNOWN'
			         WHEN contract.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.id IS NULL THEN 'UNKNOWN'
			         WHEN assessment.valid_until IS NOT NULL AND NOT ($3<assessment.valid_until) THEN 'UNKNOWN'
			         WHEN assessment.conclusion IN ('UNSUPPORTED','CONTRADICTED') THEN 'FAILED'
			         WHEN assessment.conclusion='PARTIALLY_SUPPORTED' OR assessment.coverage<contract.minimum_coverage THEN 'PARTIAL'
			         WHEN assessment.conclusion='SUPPORTED' THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN eligible_links link ON link.risk_id=risk.id
			LEFT JOIN evidence_contracts contract
			  ON contract.tenant_id=link.tenant_id
			 AND contract.program_id=link.program_id
			 AND contract.control_implementation_id=link.implementation_id
			 AND contract.status='ACTIVE'
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM evidence_assessments candidate
				WHERE candidate.tenant_id=contract.tenant_id
				  AND candidate.program_id=contract.program_id
				  AND candidate.contract_id=contract.id
				  AND candidate.assessed_at<=$3
				ORDER BY candidate.assessed_at DESC,candidate.id DESC
				LIMIT 1
			) assessment ON true
		), risk_state AS (
			SELECT risk.id,risk.name,
			       CASE
			         WHEN bool_or(fact.state='FAILED') THEN 'FAILED'
			         WHEN bool_or(fact.state='UNKNOWN') AND bool_or(fact.state IN ('SUPPORTED','PARTIAL')) THEN 'PARTIAL'
			         WHEN bool_or(fact.state='UNKNOWN') THEN 'UNKNOWN'
			         WHEN bool_or(fact.state='PARTIAL') THEN 'PARTIAL'
			         WHEN bool_or(fact.state='SUPPORTED') THEN 'SUPPORTED'
			         ELSE 'UNKNOWN'
			       END state
			FROM selected_risks risk
			LEFT JOIN contract_facts fact ON fact.risk_id=risk.id
			GROUP BY risk.id,risk.name
		)
		SELECT id::text,name,state FROM risk_state ORDER BY id`, scope.TenantID, scope.LegalEntityID, at.UTC())
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load assurance-failure metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title, state string
		if err := rows.Scan(&id, &title, &state); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if state == "UNKNOWN" {
			result.Unknown++
		}
		if state == "FAILED" {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "RISK", TargetID: id, Title: title, State: state})
		}
	}
	return result, rows.Err()
}

func loadLossWithoutIssueMetric(ctx context.Context, tx pgx.Tx, scope domainScope, _ time.Time) (domainMetricResult, error) {
	definition, _ := DomainDefinition("losses_without_issue")
	rows, err := tx.Query(ctx, `
		SELECT loss.id::text,loss.title,loss.matter_id IS NULL
		FROM operational_losses loss
		WHERE loss.tenant_id=$1::uuid
		  AND loss.legal_entity_id=$2::uuid
		  AND loss.status='ACTIVE'
		ORDER BY loss.id`, scope.TenantID, scope.LegalEntityID)
	if err != nil {
		return domainMetricResult{}, fmt.Errorf("load loss-without-issue metric: %w", err)
	}
	defer rows.Close()
	result := domainMetricResult{Definition: definition, Members: []domainMetricMember{}}
	for rows.Next() {
		var id, title string
		var withoutIssue bool
		if err := rows.Scan(&id, &title, &withoutIssue); err != nil {
			return domainMetricResult{}, err
		}
		result.Population++
		if withoutIssue {
			result.Members = append(result.Members, domainMetricMember{MemberID: id, TargetType: "LOSS", TargetID: id, Title: title, State: "WITHOUT_ISSUE"})
		}
	}
	return result, rows.Err()
}
