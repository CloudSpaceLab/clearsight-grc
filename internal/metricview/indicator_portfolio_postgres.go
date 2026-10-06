//go:build postgres

package metricview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func (r *DomainRepository) ListIndicators(ctx context.Context, tenantID, legalEntityID string, input IndicatorPortfolioFilter) (IndicatorPortfolioPage, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return IndicatorPortfolioPage{}, ErrIndicatorPortfolioInvalid
	}
	tenantID, legalEntityID = strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" || legalEntityID == "*" {
		return IndicatorPortfolioPage{}, ErrIndicatorPortfolioInvalid
	}
	filter, cursor, err := normalizeIndicatorPortfolioFilter(input)
	if err != nil {
		return IndicatorPortfolioPage{}, err
	}
	now := time.Now().UTC()
	rows, err := r.pool.Query(ctx, `
		WITH scope AS (
			SELECT tenant.id tenant_id,entity.id legal_entity_id
			FROM tenants tenant
			JOIN legal_entities entity ON entity.tenant_id=tenant.id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
		), current_links AS (
			SELECT DISTINCT ON (link.risk_id,link.monitoring_check_id,link.monitoring_check_version,link.kind)
			       link.id,link.risk_id,link.risk_version,link.program_id,
			       link.monitoring_check_id,link.monitoring_check_version,link.kind
			FROM risk_indicator_links link
			JOIN scope
			  ON scope.tenant_id=link.tenant_id
			 AND scope.legal_entity_id=link.legal_entity_id
			JOIN risks risk
			  ON risk.tenant_id=link.tenant_id
			 AND risk.legal_entity_id=link.legal_entity_id
			 AND risk.id=link.risk_id
			 AND risk.status='ACTIVE'
			ORDER BY link.risk_id,link.monitoring_check_id,link.monitoring_check_version,link.kind,
			         link.risk_version DESC,link.id DESC
		), grouped AS (
			SELECT link.monitoring_check_id,link.monitoring_check_version,link.program_id,link.kind,
			       jsonb_agg(DISTINCT jsonb_build_object('id',risk.id::text,'name',risk.name)) AS risks,
			       string_agg(DISTINCT lower(risk.name),' ') AS risk_search
			FROM current_links link
			JOIN scope ON true
			JOIN risks risk
			  ON risk.tenant_id=scope.tenant_id
			 AND risk.legal_entity_id=scope.legal_entity_id
			 AND risk.id=link.risk_id
			GROUP BY link.monitoring_check_id,link.monitoring_check_version,link.program_id,link.kind
		), materialized AS (
			SELECT grouped.kind,grouped.risks,
			       check_config.id::text check_id,check_config.version check_version,check_config.code check_code,
			       check_config.name check_name,check_config.claim,check_config.status check_status,
			       check_config.is_current,check_config.input_kind,check_config.minimum_coverage,check_config.freshness_minutes,
			       check_config.measurement check_measurement,check_config.source_rules,
			       program.id::text program_id,program.name program_name,program.status program_status,
			       COALESCE(owner.display_name,'') owner_display_name,
			       COALESCE(reviewer.display_name,'') reviewer_display_name,
			       COALESCE(result.id::text,'') result_id,result.evaluated_at,
			       NULLIF(result.evaluation->>'score','')::double precision score,
			       COALESCE(result.evaluation->>'band','') band,
			       NULLIF(result.evaluation->>'coverage','')::double precision coverage,
			       result.evaluation->'measurement' result_measurement,
			       CASE
			         WHEN check_config.id IS NULL OR check_config.status<>'ACTIVE' OR NOT check_config.is_current THEN 'UNKNOWN'
			         WHEN program.id IS NULL OR program.status<>'ACTIVE' THEN 'UNKNOWN'
			         WHEN result.id IS NULL THEN 'UNKNOWN'
			         WHEN result.evaluated_at < $3-(check_config.freshness_minutes*interval '1 minute') THEN 'UNKNOWN'
			         WHEN NULLIF(result.evaluation->>'coverage','')::numeric < check_config.minimum_coverage THEN 'UNKNOWN'
			         WHEN result.evaluation ? 'measurement' THEN
			           CASE result.evaluation->'measurement'->>'condition'
			             WHEN 'BREACHED' THEN 'BREACH'
			             WHEN 'WITHIN' THEN 'NORMAL'
			             ELSE 'UNKNOWN'
			           END
			         WHEN result.evaluation->>'band' IN ('HIGH','CRITICAL') THEN 'BREACH'
			         WHEN result.evaluation->>'band'='MODERATE' THEN 'WATCH'
			         WHEN result.evaluation->>'band'='LOW' THEN 'NORMAL'
			         ELSE 'UNKNOWN'
			       END state,
			       CASE
			         WHEN check_config.id IS NULL OR check_config.status<>'ACTIVE' OR NOT check_config.is_current THEN 'Monitoring check is no longer current.'
			         WHEN program.id IS NULL OR program.status<>'ACTIVE' THEN 'Source Program is not active.'
			         WHEN result.id IS NULL THEN 'No current monitoring result.'
			         WHEN result.evaluated_at < $3-(check_config.freshness_minutes*interval '1 minute') THEN 'Latest monitoring result is stale.'
			         WHEN NULLIF(result.evaluation->>'coverage','')::numeric < check_config.minimum_coverage THEN 'Monitoring coverage is below the approved minimum.'
			         WHEN result.evaluation ? 'measurement' AND result.evaluation->'measurement'->>'condition'='BREACHED' THEN 'Latest native measurement is outside its approved limit.'
			         WHEN result.evaluation ? 'measurement' AND result.evaluation->'measurement'->>'condition'='WITHIN' THEN 'Latest native measurement is within its approved limit.'
			         WHEN result.evaluation ? 'measurement' THEN 'Native measurement or approved limit is unavailable.'
			         WHEN result.evaluation->>'band'='LOW' THEN 'Latest complete result is in the low band.'
			         WHEN result.evaluation->>'band'='MODERATE' THEN 'Latest complete result is in the moderate band.'
			         WHEN result.evaluation->>'band' IN ('HIGH','CRITICAL') THEN 'Latest complete result is in a high or critical band.'
			         ELSE 'Latest monitoring result is not assessed.'
			       END reason,
			       grouped.risk_search
			FROM grouped
			JOIN scope ON true
			LEFT JOIN monitoring_checks check_config
			  ON check_config.tenant_id=scope.tenant_id
			 AND check_config.id=grouped.monitoring_check_id
			 AND check_config.version=grouped.monitoring_check_version
			 AND check_config.program_id=grouped.program_id
			LEFT JOIN programs program
			  ON program.tenant_id=scope.tenant_id
			 AND program.legal_entity_id=scope.legal_entity_id
			 AND program.id=grouped.program_id
			LEFT JOIN principals owner
			  ON owner.tenant_id=scope.tenant_id AND owner.id=check_config.owner_principal_id
			LEFT JOIN principals reviewer
			  ON reviewer.tenant_id=scope.tenant_id AND reviewer.id=check_config.reviewer_principal_id
			LEFT JOIN LATERAL (
				SELECT candidate.*
				FROM monitoring_results candidate
				WHERE candidate.tenant_id=scope.tenant_id
				  AND candidate.program_id=grouped.program_id
				  AND candidate.monitoring_check_id=grouped.monitoring_check_id
				  AND candidate.monitoring_check_version=grouped.monitoring_check_version
				  AND candidate.evaluated_at<=$3
				ORDER BY candidate.evaluated_at DESC,candidate.id DESC
				LIMIT 1
			) result ON true
		)
		SELECT kind,risks,program_id,program_name,check_id,check_version,check_code,check_name,claim,
		       check_status,input_kind,owner_display_name,reviewer_display_name,minimum_coverage,freshness_minutes,
		       check_measurement,source_rules,result_id,evaluated_at,score,band,coverage,result_measurement,state,reason
		FROM materialized
		WHERE ($4='' OR kind=$4)
		  AND ($5='' OR state=$5)
		  AND ($6='' OR lower(check_name) LIKE '%'||lower($6)||'%'
		            OR lower(check_code) LIKE '%'||lower($6)||'%'
		            OR lower(program_name) LIKE '%'||lower($6)||'%'
		            OR risk_search LIKE '%'||lower($6)||'%')
		  AND ($7='' OR (lower(check_name),check_id,check_version,kind) > ($7,$8,$9,$10))
		ORDER BY lower(check_name),check_id,check_version,kind
		LIMIT $11
	`,
		tenantID, legalEntityID, now, string(filter.Kind), string(filter.State), filter.Search,
		cursor.Name, cursor.CheckID, cursor.Version, string(cursor.Kind), filter.Limit+1,
	)
	if err != nil {
		return IndicatorPortfolioPage{}, fmt.Errorf("list indicator portfolio: %w", err)
	}
	defer rows.Close()

	page := IndicatorPortfolioPage{GeneratedAt: now, Items: make([]IndicatorPortfolioItem, 0, filter.Limit)}
	for rows.Next() {
		var (
			item                        IndicatorPortfolioItem
			risksJSON, checkJSON        []byte
			sourceRulesJSON, resultJSON []byte
			resultID, band              string
			evaluatedAt                 *time.Time
			score, coverage             *float64
		)
		if err := rows.Scan(
			&item.Kind, &risksJSON, &item.ProgramID, &item.ProgramName, &item.CheckID, &item.CheckVersion,
			&item.CheckCode, &item.CheckName, &item.Claim, &item.CheckStatus, &item.InputKind,
			&item.OwnerDisplayName, &item.ReviewerDisplayName, &item.MinimumCoverage, &item.FreshnessMinutes,
			&checkJSON, &sourceRulesJSON, &resultID, &evaluatedAt, &score, &band, &coverage, &resultJSON,
			&item.State, &item.Reason,
		); err != nil {
			return IndicatorPortfolioPage{}, err
		}
		if err := json.Unmarshal(risksJSON, &item.Risks); err != nil {
			return IndicatorPortfolioPage{}, fmt.Errorf("decode indicator risks: %w", err)
		}
		item.ResultID, item.EvaluatedAt, item.Score, item.Coverage = resultID, evaluatedAt, score, coverage
		item.Band = monitoring.RiskBand(band)
		if len(resultJSON) > 0 && string(resultJSON) != "null" {
			var measurement monitoring.NativeMeasurement
			if err := json.Unmarshal(resultJSON, &measurement); err != nil {
				return IndicatorPortfolioPage{}, fmt.Errorf("decode indicator measurement: %w", err)
			}
			item.NativeMeasurement = &measurement
		} else if len(checkJSON) > 0 && string(checkJSON) != "null" {
			var spec monitoring.MeasurementSpec
			var rules []monitoring.SourceRule
			if err := json.Unmarshal(checkJSON, &spec); err != nil {
				return IndicatorPortfolioPage{}, fmt.Errorf("decode indicator definition: %w", err)
			}
			if len(sourceRulesJSON) > 0 && string(sourceRulesJSON) != "null" {
				if err := json.Unmarshal(sourceRulesJSON, &rules); err != nil {
					return IndicatorPortfolioPage{}, fmt.Errorf("decode indicator limits: %w", err)
				}
			}
			item.NativeMeasurement = monitoring.MeasurementDefinition(&spec, rules)
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return IndicatorPortfolioPage{}, err
	}
	if len(page.Items) > filter.Limit {
		last := page.Items[filter.Limit-1]
		next, err := encodeIndicatorPortfolioCursor(last)
		if err != nil {
			return IndicatorPortfolioPage{}, err
		}
		page.NextCursor = next
		page.Items = page.Items[:filter.Limit]
	}
	return page, nil
}

var _ IndicatorPortfolioReader = (*DomainRepository)(nil)
