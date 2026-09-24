//go:build postgres

package reporting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ProgramReportPageSQL returns one bounded Program page. The visibility-aware
// open-Matter count is calculated in SQL, and the page CTE applies tenant,
// legal-entity, scope, filter, and keyset predicates before its limit.
func ProgramReportPageSQL(filterFragment string, filterArgumentCount int) string {
	return reportProgramMatterPageSQL(filterFragment, filterArgumentCount, true)
}

// MatterReportPageSQL returns one bounded Matter exception page. The access
// policy is deliberately evaluated in the page CTE's WHERE clause, before its
// keyset and limit, and uses the canonical fail-closed predicate.
func MatterReportPageSQL(filterFragment string, filterArgumentCount int) string {
	return reportProgramMatterPageSQL(filterFragment, filterArgumentCount, false)
}

func reportProgramMatterPageSQL(filterFragment string, filterArgumentCount int, program bool) string {
	if strings.TrimSpace(filterFragment) == "" || filterArgumentCount < 0 || filterArgumentCount > maxReportFilterNodes {
		return ""
	}
	hasCursor := 6 + filterArgumentCount
	cursorRank := hasCursor + 1
	cursorDate := hasCursor + 2
	cursorID := hasCursor + 3
	limit := hasCursor + 4
	if program {
		template := `
WITH program_rows AS (
  SELECT p.id::text,
         p.tenant_id::text,
         p.legal_entity_id::text,
         p.code,
         p.name,
         p.program_type,
         p.status,
         p.owning_function,
         p.owner_principal_id,
         COALESCE(p.authority_principal_id::text,'') AS authority_principal_id,
         p.jurisdiction,
         p.effective_from,
         p.effective_until,
         p.created_at,
         p.updated_at,
         p.version,
         COALESCE(ps.program_version,0) AS assessed_program_version,
         COALESCE(ps.projection_version,0) AS projection_version,
         COALESCE(visible.open_matter_count,0) AS visible_open_matter_count,
         (COALESCE(visible.open_matter_count,0)>0) AS has_open_matters,
         ps.generated_at AS state_generated_at,
         COALESCE(effective_state.overall_state,'UNKNOWN') AS overall_state,
         COALESCE(adjusted_reasons.reasons,'[]'::jsonb) AS reasons
  FROM programs p
  LEFT JOIN LATERAL (
    SELECT overall_state,dimensions,reasons,generated_at,program_version,projection_version
    FROM program_state_snapshots
    WHERE tenant_id=p.tenant_id AND program_id=p.id
    ORDER BY generated_at DESC,projection_version DESC
    LIMIT 1
  ) ps ON TRUE
  LEFT JOIN LATERAL (
    SELECT count(DISTINCT m.id)::integer AS open_matter_count
    FROM matter_links ml
    JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
    WHERE ml.tenant_id=p.tenant_id
      AND ml.program_id=p.id
      AND ml.retired_at IS NULL
      AND m.legal_entity_id=p.legal_entity_id
      AND m.status NOT IN ('CLOSED','CANCELLED')
      AND ` + strings.ReplaceAll(MatterReportVisibilitySQL, "a.", "m.") + `
  ) visible ON TRUE
  LEFT JOIN LATERAL (
    SELECT CASE
      WHEN ps.generated_at IS NULL THEN 'UNKNOWN'
      WHEN 'OVERDUE'=ANY(state_values.values) THEN 'OVERDUE'
      WHEN 'GAP_IDENTIFIED'=ANY(state_values.values) THEN 'GAP_IDENTIFIED'
      WHEN 'EVIDENCE_INSUFFICIENT'=ANY(state_values.values) THEN 'EVIDENCE_INSUFFICIENT'
      WHEN 'IMPLEMENTATION_PENDING'=ANY(state_values.values) THEN 'IMPLEMENTATION_PENDING'
      WHEN 'AT_RISK'=ANY(state_values.values) THEN 'AT_RISK'
      WHEN 'UNDER_REVIEW'=ANY(state_values.values) THEN 'UNDER_REVIEW'
      WHEN ps.dimensions->>'applicability'='NOT_APPLICABLE' THEN 'NOT_APPLICABLE'
      WHEN 'UNKNOWN'=ANY(state_values.values) OR array_position(state_values.values,NULL) IS NOT NULL THEN 'UNKNOWN'
      ELSE 'CURRENT'
    END AS overall_state
    FROM (SELECT ARRAY[
      ps.dimensions->>'interpretation',ps.dimensions->>'applicability',ps.dimensions->>'control_design',ps.dimensions->>'implementation',
      ps.dimensions->>'evidence_sufficiency',ps.dimensions->>'operating_effectiveness',
      ps.dimensions->>'exception',
      CASE WHEN COALESCE(visible.open_matter_count,0)>0 THEN 'AT_RISK' ELSE 'CURRENT' END,
      ps.dimensions->>'assurance',ps.dimensions->>'deadline',ps.dimensions->>'source_quality'
    ]::text[] AS values) state_values
  ) effective_state ON TRUE
  LEFT JOIN LATERAL (
    SELECT CASE WHEN ps.generated_at IS NULL THEN '[]'::jsonb ELSE
      COALESCE((SELECT jsonb_agg(reason)
        FROM jsonb_array_elements(ps.reasons) reason
        WHERE upper(btrim(COALESCE(reason->>'code','')))<>'OPEN_MATTERS'),'[]'::jsonb)
      || CASE WHEN COALESCE(visible.open_matter_count,0)>0 THEN jsonb_build_array(jsonb_build_object(
        'code','OPEN_MATTERS','summary',format('%s open issue(s) or change(s) affect this program.',visible.open_matter_count)
      )) ELSE '[]'::jsonb END
    END AS reasons
  ) adjusted_reasons ON TRUE
  WHERE p.tenant_id = $1::uuid
    AND p.legal_entity_id = $2::uuid
    AND ($3='' OR p.id=NULLIF($3,'')::uuid)
    AND p.updated_at<=$5::timestamptz
), page AS MATERIALIZED (
  SELECT a.id::text,
         a.tenant_id::text,
         a.legal_entity_id::text,
         a.code,
         a.name,
         a.program_type,
         a.status,
         a.owning_function,
         COALESCE(a.owner_principal_id::text,'') AS owner_principal_id,
         a.authority_principal_id,
         a.jurisdiction,
         a.effective_from,
         a.effective_until,
         a.created_at,
         a.updated_at,
         a.version,
         a.assessed_program_version,
         a.projection_version,
         (a.assessed_program_version<a.version) AS projection_stale,
         a.overall_state,
         a.has_open_matters,
         a.state_generated_at,
         COALESCE((a.reasons->>'count')::integer, jsonb_array_length(a.reasons)) AS reasons_total,
         COALESCE((SELECT jsonb_agg(limited.value)
           FROM (SELECT value FROM jsonb_array_elements(a.reasons) WITH ORDINALITY
                 WHERE value<>'null'::jsonb ORDER BY ordinality LIMIT 6) limited),'[]'::jsonb) AS reasons,
         GREATEST(0,COALESCE((a.reasons->>'count')::integer,jsonb_array_length(a.reasons))-6) AS reasons_omitted
  FROM program_rows a
  WHERE (__REPORT_FILTER__)
    AND (
      $__HAS_CURSOR__=false OR
      ` + ProgramReportStatusRankSQL + ` > $__CURSOR_RANK__::integer OR
      (` + ProgramReportStatusRankSQL + `=$__CURSOR_RANK__::integer AND
        (a.updated_at<$__CURSOR_DATE__::timestamptz OR
         (a.updated_at=$__CURSOR_DATE__::timestamptz AND a.id<NULLIF($__CURSOR_ID__,'')::text)))
    )
  ORDER BY ` + ProgramReportStatusRankSQL + `,a.updated_at DESC,a.id DESC
  LIMIT $__LIMIT__
)
SELECT * FROM page
ORDER BY CASE page.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END,page.updated_at DESC,page.id DESC`
		return strings.NewReplacer(
			"__REPORT_FILTER__", filterFragment,
			"$__HAS_CURSOR__", fmt.Sprintf("$%d", hasCursor),
			"$__CURSOR_RANK__", fmt.Sprintf("$%d", cursorRank),
			"$__CURSOR_DATE__", fmt.Sprintf("$%d", cursorDate),
			"$__CURSOR_ID__", fmt.Sprintf("$%d", cursorID),
			"$__LIMIT__", fmt.Sprintf("$%d", limit),
		).Replace(template)
	}

	template := `
WITH matter_rows AS (
  SELECT a.id::text,
         a.tenant_id::text,
         a.legal_entity_id::text,
         a.reference,
         a.matter_type,
         a.status,
         a.priority,
         a.title,
         a.summary,
         a.owner_principal_id,
         a.required_authority,
         a.due_at,
         a.closed_at,
         a.closure_reason,
         a.reopen_count,
         a.created_at,
         a.updated_at,
         a.version,
         a.scope,
         COALESCE(latest.result,'') AS latest_verification_result,
         latest.observed_at AS latest_verification_at,
         action_counts.open_action_count,
         outcome_counts.outcome_check_count,
         reason_values.reasons AS reasons
  FROM matters a
  LEFT JOIN LATERAL (
    SELECT vr.result,vr.observed_at
    FROM verification_results vr
    WHERE vr.tenant_id=a.tenant_id AND vr.matter_id=a.id
    ORDER BY vr.observed_at DESC,vr.id DESC
    LIMIT 1
  ) latest ON TRUE
  LEFT JOIN LATERAL (
    SELECT count(*)::integer AS open_action_count
    FROM matter_actions ma
    WHERE ma.tenant_id=a.tenant_id AND ma.matter_id=a.id AND ma.status NOT IN ('IMPLEMENTED','CANCELLED')
  ) action_counts ON TRUE
  LEFT JOIN LATERAL (
    SELECT count(*)::integer AS outcome_check_count
    FROM verification_contracts vc
    WHERE vc.tenant_id=a.tenant_id AND vc.matter_id=a.id AND vc.status='ACTIVE'
  ) outcome_counts ON TRUE
  LEFT JOIN LATERAL (
    SELECT COALESCE(jsonb_agg(reason) FILTER (WHERE reason IS NOT NULL),'[]'::jsonb) AS reasons
    FROM (
      SELECT jsonb_build_object('code','MISSING_FACT','summary',entry.value #>> '{}') AS reason
      FROM jsonb_array_elements(a.missing_facts) AS entry
      WHERE jsonb_typeof(entry.value)='string'
      UNION ALL
      SELECT jsonb_build_object('code','CONTRADICTION','summary',entry.value #>> '{}') AS reason
      FROM jsonb_array_elements(a.contradictions) AS entry
      WHERE jsonb_typeof(entry.value)='string'
      UNION ALL
      SELECT CASE WHEN a.status NOT IN ('CLOSED','CANCELLED') THEN jsonb_build_object('code','OPEN_ISSUE','summary','Issue or change is still open.') ELSE NULL::jsonb END
      UNION ALL
      SELECT CASE WHEN a.due_at IS NOT NULL AND a.due_at<$5::timestamptz AND a.status NOT IN ('CLOSED','CANCELLED') THEN jsonb_build_object('code','OVERDUE','summary','Obligation is past its recorded due date.') ELSE NULL::jsonb END
      UNION ALL
      SELECT CASE WHEN action_counts.open_action_count>0 THEN jsonb_build_object('code','OPEN_ACTION','summary','One or more actions remain open.') ELSE NULL::jsonb END
      UNION ALL
      SELECT CASE WHEN latest.result IN ('FAIL','INCONCLUSIVE') THEN jsonb_build_object('code','OUTCOME_CHECK','summary','Latest outcome check requires attention.') ELSE NULL::jsonb END
      UNION ALL
      SELECT CASE WHEN a.owner_principal_id IS NULL THEN jsonb_build_object('code','UNASSIGNED','summary','No accountable owner is recorded.') ELSE NULL::jsonb END
    ) reason_values
  ) reason_values ON TRUE
  WHERE a.tenant_id=$1::uuid
    AND a.legal_entity_id=$2::uuid
    AND a.updated_at<=$5::timestamptz
), page AS MATERIALIZED (
  SELECT a.id::text,
         a.tenant_id::text,
         a.legal_entity_id::text,
         a.reference,
         a.matter_type,
         a.status,
         a.priority,
         a.title,
         a.summary,
         COALESCE(a.owner_principal_id::text,'') AS owner_principal_id,
         a.required_authority,
         a.due_at,
         a.closed_at,
         a.closure_reason,
         a.reopen_count,
         a.created_at,
         a.updated_at,
         a.version,
         a.latest_verification_result,
         a.latest_verification_at,
         a.open_action_count,
         a.outcome_check_count,
         jsonb_array_length(a.reasons) AS reasons_total,
         COALESCE((SELECT jsonb_agg(limited.value)
           FROM (SELECT value FROM jsonb_array_elements(a.reasons) WITH ORDINALITY
                 WHERE value<>'null'::jsonb ORDER BY ordinality LIMIT 6) limited),'[]'::jsonb) AS reasons,
         GREATEST(0,jsonb_array_length(a.reasons)-6) AS reasons_omitted,
         (CASE WHEN a.matter_type='EXCEPTION' AND a.status NOT IN ('CLOSED','CANCELLED') THEN 'Open exception'
            WHEN a.due_at IS NOT NULL AND a.due_at<$5::timestamptz AND a.status NOT IN ('CLOSED','CANCELLED') THEN 'Overdue obligation'
            ELSE '' END) AS exceptions
  FROM matter_rows a
  WHERE ($3='' OR a.id=NULLIF($3,'')::text)
    AND ` + MatterReportVisibilitySQL + `
    AND ` + MatterReportExceptionPredicateSQL + `
    AND (__REPORT_FILTER__)
    AND (
      $__HAS_CURSOR__=false OR
      a.priority<$__CURSOR_RANK__::integer OR
      (a.priority=$__CURSOR_RANK__::integer AND
        (a.updated_at<$__CURSOR_DATE__::timestamptz OR
         (a.updated_at=$__CURSOR_DATE__::timestamptz AND a.id<NULLIF($__CURSOR_ID__,'')::text)))
    )
  ORDER BY a.priority DESC,a.updated_at DESC,a.id DESC
  LIMIT $__LIMIT__
)
SELECT * FROM page
ORDER BY page.priority DESC,page.updated_at DESC,page.id DESC`
	return strings.NewReplacer(
		"__REPORT_FILTER__", filterFragment,
		"$__HAS_CURSOR__", fmt.Sprintf("$%d", hasCursor),
		"$__CURSOR_RANK__", fmt.Sprintf("$%d", cursorRank),
		"$__CURSOR_DATE__", fmt.Sprintf("$%d", cursorDate),
		"$__CURSOR_ID__", fmt.Sprintf("$%d", cursorID),
		"$__LIMIT__", fmt.Sprintf("$%d", limit),
	).Replace(template)
}

func (r *PostgresRepository) captureProgramSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	var highWater *time.Time
	var population int
	if err := r.pool.QueryRow(ctx, `SELECT max(p.updated_at),count(*)::integer FROM programs p
		WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid
		  AND ($3='' OR p.id=NULLIF($3,'')::uuid)`, scope.TenantID, scope.LegalEntityID, definition.ScopeRef).
		Scan(&highWater, &population); err != nil {
		return SourceBoundary{}, fmt.Errorf("read Program report source boundary: %w", err)
	}
	captured := time.Now().UTC()
	if highWater == nil {
		highWater = &captured
	}
	return SourceBoundary{CapturedAt: captured, ProjectionVersion: "program-state-report.v1",
		SourceHighWater: map[string]time.Time{"programs": highWater.UTC()}, Population: population, PopulationComplete: false}, nil
}

func (r *PostgresRepository) captureMatterSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	var highWater *time.Time
	var population int
	if err := r.pool.QueryRow(ctx, `SELECT max(m.updated_at),count(*)::integer FROM matters m
		WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid
		  AND ($3='' OR m.id=NULLIF($3,'')::uuid)`, scope.TenantID, scope.LegalEntityID, definition.ScopeRef).
		Scan(&highWater, &population); err != nil {
		return SourceBoundary{}, fmt.Errorf("read Matter report source boundary: %w", err)
	}
	captured := time.Now().UTC()
	if highWater == nil {
		highWater = &captured
	}
	return SourceBoundary{CapturedAt: captured, ProjectionVersion: "matters-current-report.v1",
		SourceHighWater: map[string]time.Time{"matters": highWater.UTC()}, Population: population, PopulationComplete: false}, nil
}

func (r *PostgresRepository) listProgramReportRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	filter, args, err := ReportFilterSQLForDataset(DatasetPrograms, run.Filter, 6)
	if err != nil {
		return ReportPage{}, fmt.Errorf("build Program report filter: %w", err)
	}
	query := ProgramReportPageSQL(filter, len(args))
	if query == "" {
		return ReportPage{}, fmt.Errorf("Program report page SQL was empty for filter %q with %d arguments", filter, len(args))
	}
	position, err := decodeProgramReportCursor(cursor)
	if err != nil {
		return ReportPage{}, fmt.Errorf("decode Program report cursor %q: %w", cursor, err)
	}
	highWater := run.SourceBoundary.SourceHighWater["programs"]
	if highWater.IsZero() {
		highWater = run.AsOf
	}
	hasCursor := position.ID != ""
	rank, updatedAt, id := 0, time.Time{}, ""
	if hasCursor {
		rank, updatedAt, id = position.Rank, position.UpdatedAt, position.ID
	}
	queryArgs := append([]any{scope.TenantID, scope.LegalEntityID, run.ScopeRef, run.RequestedByRef, highWater}, args...)
	queryArgs = append(queryArgs, hasCursor, rank, updatedAt, id, limit+1)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return ReportPage{}, fmt.Errorf("read bounded Program report rows: %w", err)
	}
	defer rows.Close()
	page := ReportPage{Rows: make([]ReportRow, 0, limit+1), Columns: []string{
		"tenant_id", "legal_entity_id", "code", "name", "program_type", "status", "owning_function", "owner_principal_id",
		"authority_principal_id", "jurisdiction", "effective_from", "effective_until", "created_at", "updated_at", "version",
		"assessed_program_version", "projection_version", "projection_stale", "overall_state", "has_open_matters",
		"state_generated_at", "reasons_total", "reasons", "reasons_omitted",
	}}
	for rows.Next() {
		row, err := scanProgramReportRow(rows)
		if err != nil {
			return ReportPage{}, fmt.Errorf("scan bounded Program report row: %w", err)
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return ReportPage{}, fmt.Errorf("read bounded Program report rows: %w", err)
	}
	if len(page.Rows) > limit {
		page.Rows = page.Rows[:limit]
		page.NextCursor, err = encodeProgramReportCursor(page.Rows[len(page.Rows)-1])
		if err != nil {
			return ReportPage{}, fmt.Errorf("encode Program report cursor: %w", err)
		}
	}
	return page, nil
}

func (r *PostgresRepository) listMatterReportRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	filter, args, err := ReportFilterSQLForDataset(DatasetMatterExceptions, run.Filter, 6)
	if err != nil {
		return ReportPage{}, fmt.Errorf("build Matter report filter: %w", err)
	}
	query := MatterReportPageSQL(filter, len(args))
	if query == "" {
		return ReportPage{}, fmt.Errorf("Matter report page SQL was empty for filter %q with %d arguments", filter, len(args))
	}
	position, err := decodeMatterReportCursor(cursor)
	if err != nil {
		return ReportPage{}, fmt.Errorf("decode Matter report cursor %q: %w", cursor, err)
	}
	highWater := run.SourceBoundary.SourceHighWater["matters"]
	if highWater.IsZero() {
		highWater = run.AsOf
	}
	hasCursor := position.ID != ""
	priority, updatedAt, id := 0, time.Time{}, ""
	if hasCursor {
		priority, updatedAt, id = position.Priority, position.UpdatedAt, position.ID
	}
	queryArgs := append([]any{scope.TenantID, scope.LegalEntityID, run.ScopeRef, run.RequestedByRef, highWater}, args...)
	queryArgs = append(queryArgs, hasCursor, priority, updatedAt, id, limit+1)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return ReportPage{}, fmt.Errorf("read bounded Matter report rows: %w", err)
	}
	defer rows.Close()
	page := ReportPage{Rows: make([]ReportRow, 0, limit+1), Columns: []string{
		"tenant_id", "legal_entity_id", "reference", "matter_type", "status", "priority", "title", "summary",
		"owner_principal_id", "required_authority", "due_at", "closed_at", "closure_reason", "reopen_count",
		"created_at", "updated_at", "version", "latest_verification_result", "latest_verification_at", "open_action_count",
		"outcome_check_count", "reasons_total", "reasons", "reasons_omitted", "exceptions",
	}}
	for rows.Next() {
		row, err := scanMatterReportRow(rows)
		if err != nil {
			return ReportPage{}, fmt.Errorf("scan bounded Matter report row: %w", err)
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return ReportPage{}, fmt.Errorf("read bounded Matter report rows: %w", err)
	}
	if len(page.Rows) > limit {
		page.Rows = page.Rows[:limit]
		page.NextCursor, err = encodeMatterReportCursor(page.Rows[len(page.Rows)-1])
		if err != nil {
			return ReportPage{}, fmt.Errorf("encode Matter report cursor: %w", err)
		}
	}
	return page, nil
}

func scanProgramReportRow(row reportingRowScanner) (ReportRow, error) {
	var id, tenantID, entityID, code, name, programType, status, function, ownerID, authorityID, jurisdiction string
	var effectiveFrom time.Time
	var effectiveUntil, stateGeneratedAt pgtype.Timestamptz
	var createdAt, updatedAt time.Time
	var version, assessedVersion, projectionVersion int64
	var stale bool
	var overallState string
	var hasOpen bool
	var reasonsTotal, reasonsOmitted int
	var reasonsRaw []byte
	if err := row.Scan(&id, &tenantID, &entityID, &code, &name, &programType, &status, &function, &ownerID,
		&authorityID, &jurisdiction, &effectiveFrom, &effectiveUntil, &createdAt, &updatedAt, &version,
		&assessedVersion, &projectionVersion, &stale, &overallState, &hasOpen, &stateGeneratedAt, &reasonsTotal,
		&reasonsRaw, &reasonsOmitted); err != nil {
		return ReportRow{}, err
	}
	var reasons []any
	if err := json.Unmarshal(reasonsRaw, &reasons); err != nil {
		return ReportRow{}, fmt.Errorf("decode Program report reasons: %w", err)
	}
	var until any
	if effectiveUntil.Valid {
		until = effectiveUntil.Time.UTC()
	}
	var generated any
	if stateGeneratedAt.Valid {
		generated = stateGeneratedAt.Time.UTC()
	}
	return ReportRow{ID: id, Values: map[string]any{
		"tenant_id": tenantID, "legal_entity_id": entityID, "code": code, "name": name, "program_type": programType,
		"status": status, "owning_function": function, "owner_principal_id": ownerID, "authority_principal_id": authorityID,
		"jurisdiction": jurisdiction, "effective_from": effectiveFrom.UTC(), "effective_until": until,
		"created_at": createdAt.UTC(), "updated_at": updatedAt.UTC(), "version": version,
		"assessed_program_version": assessedVersion, "projection_version": projectionVersion, "projection_stale": stale,
		"overall_state": overallState, "has_open_matters": hasOpen, "state_generated_at": generated,
		"reasons_total": reasonsTotal, "reasons": reasons, "reasons_omitted": reasonsOmitted,
	}}, nil
}

func scanMatterReportRow(row reportingRowScanner) (ReportRow, error) {
	var id, tenantID, entityID, reference, matterType, status, title, summary, ownerID, requiredAuthority, closureReason, latestResult, exceptions string
	var priority, reopenCount, openActions, outcomeChecks, reasonsTotal, reasonsOmitted int
	var dueAt, closedAt, latestAt pgtype.Timestamptz
	var createdAt, updatedAt time.Time
	var version int64
	var reasonsRaw []byte
	if err := row.Scan(&id, &tenantID, &entityID, &reference, &matterType, &status, &priority, &title, &summary,
		&ownerID, &requiredAuthority, &dueAt, &closedAt, &closureReason, &reopenCount, &createdAt, &updatedAt, &version,
		&latestResult, &latestAt, &openActions, &outcomeChecks, &reasonsTotal, &reasonsRaw, &reasonsOmitted, &exceptions); err != nil {
		return ReportRow{}, err
	}
	var reasons []any
	if err := json.Unmarshal(reasonsRaw, &reasons); err != nil {
		return ReportRow{}, fmt.Errorf("decode Matter report reasons: %w", err)
	}
	var due, closed, latest any
	if dueAt.Valid {
		due = dueAt.Time.UTC()
	}
	if closedAt.Valid {
		closed = closedAt.Time.UTC()
	}
	if latestAt.Valid {
		latest = latestAt.Time.UTC()
	}
	return ReportRow{ID: id, Values: map[string]any{
		"tenant_id": tenantID, "legal_entity_id": entityID, "reference": reference, "matter_type": matterType,
		"status": status, "priority": priority, "title": title, "summary": summary, "owner_principal_id": ownerID,
		"required_authority": requiredAuthority, "due_at": due, "closed_at": closed, "closure_reason": closureReason,
		"reopen_count": reopenCount, "created_at": createdAt.UTC(), "updated_at": updatedAt.UTC(), "version": version,
		"latest_verification_result": latestResult, "latest_verification_at": latest, "open_action_count": openActions,
		"outcome_check_count": outcomeChecks, "reasons_total": reasonsTotal, "reasons": reasons,
		"reasons_omitted": reasonsOmitted, "exceptions": exceptions,
	}}, nil
}

type programReportCursor struct {
	Rank      int       `json:"r"`
	UpdatedAt time.Time `json:"u"`
	ID        string    `json:"i"`
}

type matterReportCursor struct {
	Priority  int       `json:"p"`
	UpdatedAt time.Time `json:"u"`
	ID        string    `json:"i"`
}

func encodeProgramReportCursor(row ReportRow) (string, error) {
	status, _ := row.Values["status"].(string)
	updated, ok := row.Values["updated_at"].(time.Time)
	if !ok || !isUUID(row.ID) || !validReportProgramStatus(status) {
		return "", ErrInvalid
	}
	return encodeOpaqueReportCursor(fmt.Sprintf(`{"r":%d,"u":%q,"i":%q}`, programStatusRank(status), updated.UTC().Format(time.RFC3339Nano), row.ID)), nil
}

func decodeProgramReportCursor(value string) (programReportCursor, error) {
	if strings.TrimSpace(value) == "" {
		return programReportCursor{}, nil
	}
	raw, err := decodeOpaqueReportCursor(value)
	if err != nil {
		return programReportCursor{}, err
	}
	var cursor programReportCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.Rank < 0 || cursor.Rank > 3 || cursor.UpdatedAt.IsZero() || !isUUID(cursor.ID) {
		return programReportCursor{}, ErrInvalid
	}
	return cursor, nil
}

func encodeMatterReportCursor(row ReportRow) (string, error) {
	priority, ok := row.Values["priority"].(int)
	updated, updatedOK := row.Values["updated_at"].(time.Time)
	if !ok || !updatedOK || priority < 1 || priority > 5 || !isUUID(row.ID) {
		return "", ErrInvalid
	}
	return encodeOpaqueReportCursor(fmt.Sprintf(`{"p":%d,"u":%q,"i":%q}`, priority, updated.UTC().Format(time.RFC3339Nano), row.ID)), nil
}

func decodeMatterReportCursor(value string) (matterReportCursor, error) {
	if strings.TrimSpace(value) == "" {
		return matterReportCursor{}, nil
	}
	raw, err := decodeOpaqueReportCursor(value)
	if err != nil {
		return matterReportCursor{}, err
	}
	var cursor matterReportCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.Priority < 1 || cursor.Priority > 5 || cursor.UpdatedAt.IsZero() || !isUUID(cursor.ID) {
		return matterReportCursor{}, ErrInvalid
	}
	return cursor, nil
}

func validReportProgramStatus(value string) bool {
	return value == "ACTIVE" || value == "PAUSED" || value == "DRAFT" || value == "RETIRED"
}

func encodeOpaqueReportCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeOpaqueReportCursor(value string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) == 0 || len(raw) > 512 {
		return nil, ErrInvalid
	}
	return raw, nil
}

func programStatusRank(status string) int {
	switch status {
	case "ACTIVE":
		return 0
	case "PAUSED":
		return 1
	case "DRAFT":
		return 2
	default:
		return 3
	}
}
