package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var matterBoardBriefBoundaryQueries = []struct {
	key   string
	query string
}{
	{"matters", `SELECT max(updated_at) FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`},
	{"matter_actions", `SELECT max(updated_at) FROM matter_actions WHERE tenant_id=$1::uuid AND matter_id=$3::uuid`},
	{"matter_decisions", `SELECT max(updated_at) FROM matter_decisions WHERE tenant_id=$1::uuid AND matter_id=$3::uuid`},
	{"verification_results", `SELECT max(created_at) FROM verification_results WHERE tenant_id=$1::uuid AND matter_id=$3::uuid`},
	{"operational_losses", `SELECT max(updated_at) FROM operational_losses WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND matter_id=$3::uuid`},
	{"operational_loss_recoveries", `SELECT max(r.created_at) FROM operational_loss_recoveries r JOIN operational_losses l ON l.tenant_id=r.tenant_id AND l.legal_entity_id=r.legal_entity_id AND l.id=r.loss_id WHERE l.tenant_id=$1::uuid AND l.legal_entity_id=$2::uuid AND l.matter_id=$3::uuid`},
	{"form_distributions", `SELECT max(updated_at) FROM capture_form_distributions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND subject_type='MATTER' AND subject_id=$3::uuid`},
	{"form_responses", `SELECT max(r.created_at) FROM capture_response_revisions r JOIN capture_form_distributions d ON d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id AND d.id=r.distribution_id WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.subject_type='MATTER' AND d.subject_id=$3::uuid`},
	{"vendor_work", `SELECT max(updated_at) FROM third_party_work_requests WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND target_type='MATTER' AND target_id=$3::uuid`},
	{"matter_links", `SELECT max(GREATEST(created_at,COALESCE(retired_at,created_at))) FROM matter_links WHERE tenant_id=$1::uuid AND matter_id=$3::uuid`},
	{"programs", `SELECT max(p.updated_at) FROM matter_links ml JOIN programs p ON p.tenant_id=ml.tenant_id AND p.id=ml.program_id WHERE ml.tenant_id=$1::uuid AND ml.matter_id=$3::uuid AND p.legal_entity_id=$2::uuid`},
	{"monitoring_results", `SELECT max(r.created_at) FROM matters m JOIN monitoring_results r ON m.source_type='MONITORING_RESULT' AND r.tenant_id=m.tenant_id AND r.id::text=m.source_id WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.id=$3::uuid`},
}

var noBoardBriefRows = time.Unix(0, 0).UTC()

func (r *PostgresRepository) captureMatterBoardBriefSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	if definition.ScopeKind != ScopeMatter || definition.ScopeRef == "" {
		return SourceBoundary{}, ErrInvalid
	}
	high, err := r.readMatterBoardBriefHighWater(ctx, scope, definition.ScopeRef)
	if err != nil {
		return SourceBoundary{}, err
	}
	if high["matters"].Equal(noBoardBriefRows) {
		return SourceBoundary{}, ErrNotFound
	}
	return SourceBoundary{
		CapturedAt:         time.Now().UTC(),
		ProjectionVersion:  "matter-board-brief.v1",
		SourceHighWater:    high,
		Population:         1,
		PopulationComplete: true,
	}, nil
}

func (r *PostgresRepository) readMatterBoardBriefHighWater(ctx context.Context, scope ReportScope, matterID string) (map[string]time.Time, error) {
	high := make(map[string]time.Time, len(matterBoardBriefBoundaryQueries))
	for _, source := range matterBoardBriefBoundaryQueries {
		var value *time.Time
		if err := r.pool.QueryRow(ctx, source.query, scope.TenantID, scope.LegalEntityID, matterID).Scan(&value); err != nil {
			return nil, fmt.Errorf("read board brief %s boundary: %w", source.key, err)
		}
		if value == nil {
			high[source.key] = noBoardBriefRows
		} else {
			high[source.key] = value.UTC()
		}
	}
	return high, nil
}

func (r *PostgresRepository) validateMatterBoardBriefBoundary(ctx context.Context, scope ReportScope, run ReportRun) error {
	current, err := r.readMatterBoardBriefHighWater(ctx, scope, run.ScopeRef)
	if err != nil {
		return err
	}
	for _, source := range matterBoardBriefBoundaryQueries {
		expected, ok := run.SourceBoundary.SourceHighWater[source.key]
		if !ok || expected.IsZero() || !current[source.key].Equal(expected) {
			return &sourceBoundaryError{Expected: run.SourceBoundary.Population, Actual: 0}
		}
	}
	return nil
}

func (r *PostgresRepository) listMatterBoardBriefRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	if run.Dataset != DatasetMatterBoardBrief || run.ScopeKind != ScopeMatter || run.ScopeRef == "" || cursor != "" || limit < 1 {
		return ReportPage{}, ErrInvalid
	}
	if err := r.validateMatterBoardBriefBoundary(ctx, scope, run); err != nil {
		return ReportPage{}, err
	}
	h := run.SourceBoundary.SourceHighWater

	var id, reference, title, status, summary, organizationScope, affectedArea, ownerName string
	var version int64
	var priority int
	var dueAt *time.Time
	var programsRaw, actionsRaw, decisionsRaw, outcomesRaw, lossesRaw, formsRaw, vendorRaw, sourceRaw []byte
	err := r.pool.QueryRow(ctx, `
WITH target AS (
	SELECT a.*
	FROM matters a
	WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid AND a.id=$3::uuid
	  AND a.updated_at=$5::timestamptz
	  AND `+MatterReportVisibilitySQL+`
)
SELECT t.id::text,t.reference,t.title,t.status,t.version,t.priority,t.summary,
	COALESCE(array_to_string(os.department_path,' / '),os.name,''),COALESCE(t.scope->>'affected_area',''),
	COALESCE(owner.display_name,''),t.due_at,
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object('code',bounded.code,'name',bounded.name) ORDER BY bounded.code)
		FROM (
			SELECT p.code,p.name
			FROM matter_links ml
			JOIN programs p ON p.tenant_id=ml.tenant_id AND p.id=ml.program_id
			WHERE ml.tenant_id=t.tenant_id AND ml.matter_id=t.id
			  AND ml.created_at<=$14::timestamptz
			  AND (ml.retired_at IS NULL OR ml.retired_at>$14::timestamptz)
			  AND p.legal_entity_id=t.legal_entity_id AND p.updated_at<=$15::timestamptz
			ORDER BY p.code,p.id
			LIMIT 20
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object('title',bounded.title,'status',bounded.status,'owner',bounded.owner,'due_at',bounded.due_at) ORDER BY bounded.created_at,bounded.id)
		FROM (
			SELECT ma.id,ma.title,ma.status,COALESCE(op.display_name,'') AS owner,ma.due_at,ma.created_at
			FROM matter_actions ma
			LEFT JOIN principals op ON op.tenant_id=ma.tenant_id AND op.id=ma.owner_principal_id
			WHERE ma.tenant_id=t.tenant_id AND ma.matter_id=t.id AND ma.updated_at<=$6::timestamptz
			ORDER BY ma.created_at,ma.id
			LIMIT 25
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object('type',bounded.decision_type,'status',bounded.status,'selected_option',bounded.selected_option,'decided_at',bounded.decided_at) ORDER BY bounded.created_at,bounded.id)
		FROM (
			SELECT md.id,md.decision_type,md.status,md.selected_option,md.decided_at,md.created_at
			FROM matter_decisions md
			WHERE md.tenant_id=t.tenant_id AND md.matter_id=t.id AND md.updated_at<=$7::timestamptz
			ORDER BY md.created_at,md.id
			LIMIT 25
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object('result',bounded.result,'observed_at',bounded.observed_at,'rationale',bounded.rationale) ORDER BY bounded.observed_at DESC,bounded.id DESC)
		FROM (
			SELECT vr.id,vr.result,vr.observed_at,vr.rationale
			FROM verification_results vr
			WHERE vr.tenant_id=t.tenant_id AND vr.matter_id=t.id AND vr.created_at<=$8::timestamptz
			ORDER BY vr.observed_at DESC,vr.id DESC
			LIMIT 20
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'code',bounded.code,'title',bounded.title,'gross',bounded.gross_amount_minor,
			'recovered',bounded.recovered_amount_minor,
			'net',bounded.gross_amount_minor-bounded.recovered_amount_minor,
			'currency',bounded.currency
		) ORDER BY bounded.occurred_at DESC,bounded.id)
		FROM (
			SELECT l.id,l.code,l.title,l.gross_amount_minor,l.currency,l.occurred_at,
				COALESCE((
					SELECT sum(CASE WHEN rec.kind='RECOVERY' THEN rec.amount_minor ELSE -rec.amount_minor END)
					FROM operational_loss_recoveries rec
					WHERE rec.tenant_id=l.tenant_id AND rec.legal_entity_id=l.legal_entity_id
					  AND rec.loss_id=l.id AND rec.created_at<=$10::timestamptz
				),0) AS recovered_amount_minor
			FROM operational_losses l
			WHERE l.tenant_id=t.tenant_id AND l.legal_entity_id=t.legal_entity_id
			  AND l.matter_id=t.id AND l.updated_at<=$9::timestamptz AND l.status='ACTIVE'
			ORDER BY l.occurred_at DESC,l.id
			LIMIT 20
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'title',bounded.title,'status',bounded.status,'deadline',bounded.deadline,
			'response_state',bounded.response_state,'concern',bounded.concern
		) ORDER BY bounded.created_at DESC,bounded.id DESC)
		FROM (
			SELECT d.id,d.title,d.status,d.deadline,d.created_at,
				COALESCE(rr.state,'') AS response_state,COALESCE(rr.concern_band,'') AS concern
			FROM capture_form_distributions d
			LEFT JOIN LATERAL (
				SELECT r.state,r.concern_band
				FROM capture_response_revisions r
				WHERE r.tenant_id=d.tenant_id AND r.legal_entity_id=d.legal_entity_id
				  AND r.distribution_id=d.id AND r.created_at<=$12::timestamptz
				ORDER BY r.revision DESC,r.id DESC
				LIMIT 1
			) rr ON TRUE
			WHERE d.tenant_id=t.tenant_id AND d.legal_entity_id=t.legal_entity_id
			  AND d.subject_type='MATTER' AND d.subject_id=t.id AND d.updated_at<=$11::timestamptz
			ORDER BY d.created_at DESC,d.id DESC
			LIMIT 20
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object('purpose',bounded.purpose,'state',bounded.state,'due_at',bounded.due_at) ORDER BY bounded.updated_at DESC,bounded.id DESC)
		FROM (
			SELECT w.id,w.purpose,w.state,w.due_at,w.updated_at
			FROM third_party_work_requests w
			WHERE w.tenant_id=t.tenant_id AND w.legal_entity_id=t.legal_entity_id
			  AND w.target_type='MATTER' AND w.target_id=t.id AND w.updated_at<=$13::timestamptz
			ORDER BY w.updated_at DESC,w.id DESC
			LIMIT 20
		) bounded
	),'[]'::jsonb),
	COALESCE((
		SELECT jsonb_build_array(jsonb_build_object(
			'type','Monitoring result',
			'observed_at',r.evaluated_at,
			'completeness',CASE WHEN r.source_receipt->>'completeness' IN ('COMPLETE','PARTIAL','UNKNOWN') THEN r.source_receipt->>'completeness' ELSE '' END,
			'records',CASE WHEN COALESCE(r.source_receipt->>'count','') ~ '^[0-9]+$' THEN (r.source_receipt->>'count')::bigint ELSE NULL END
		))
		FROM monitoring_results r
		WHERE t.source_type='MONITORING_RESULT' AND r.tenant_id=t.tenant_id
		  AND r.id::text=t.source_id AND r.created_at<=$16::timestamptz
		LIMIT 1
	),'[]'::jsonb)
FROM target t
LEFT JOIN organization_scopes os ON os.tenant_id=t.tenant_id AND os.legal_entity_id=t.legal_entity_id AND os.id=t.organization_scope_id
LEFT JOIN principals owner ON owner.tenant_id=t.tenant_id AND owner.id=t.owner_principal_id
`, scope.TenantID, scope.LegalEntityID, run.ScopeRef, run.RequestedByRef,
		h["matters"], h["matter_actions"], h["matter_decisions"], h["verification_results"],
		h["operational_losses"], h["operational_loss_recoveries"], h["form_distributions"], h["form_responses"],
		h["vendor_work"], h["matter_links"], h["programs"], h["monitoring_results"]).
		Scan(&id, &reference, &title, &status, &version, &priority, &summary, &organizationScope, &affectedArea, &ownerName, &dueAt,
			&programsRaw, &actionsRaw, &decisionsRaw, &outcomesRaw, &lossesRaw, &formsRaw, &vendorRaw, &sourceRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportPage{}, &sourceBoundaryError{Expected: run.SourceBoundary.Population, Actual: 0}
	}
	if err != nil {
		return ReportPage{}, fmt.Errorf("read Matter board brief projection: %w", err)
	}
	values := map[string]any{
		"reference": reference, "title": title, "status": status, "version": version, "priority": priority, "summary": summary,
		"organization_scope": organizationScope, "affected_area": affectedArea, "owner_name": ownerName,
	}
	if dueAt != nil {
		values["due_at"] = dueAt.UTC()
	}
	for key, raw := range map[string][]byte{
		"programs": programsRaw, "actions": actionsRaw, "decisions": decisionsRaw, "outcomes": outcomesRaw,
		"losses": lossesRaw, "forms": formsRaw, "vendor_work": vendorRaw, "source_context": sourceRaw,
	} {
		var decoded []any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return ReportPage{}, fmt.Errorf("decode board brief %s: %w", key, err)
		}
		values[key] = decoded
	}
	return ReportPage{Rows: []ReportRow{{ID: id, Values: values}}}, nil
}
