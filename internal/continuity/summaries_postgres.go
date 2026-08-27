//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (r *PostgresRepository) ListProgramSummaries(ctx context.Context, tenant string, query SummaryQuery) (ProgramSummaryPage, error) {
	cursor, err := decodeProgramSummaryCursor(query.Cursor)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	limit := boundedLimit(query.Limit)
	hasCursor := cursor.ID != ""
	enforceEntity, actorTenant, actorEntity := postgresActorScope(ctx)
	actor, enforceVisibility := identity.FromContext(ctx)
	principalID := ""
	if enforceVisibility {
		principalID = actor.PrincipalID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			p.id::text,t.id::text,COALESCE(p.legal_entity_id::text,''),p.code,p.name,p.program_type,p.status,
			p.owning_function,COALESCE(p.owner_principal_id::text,''),COALESCE(p.authority_principal_id::text,''),
			p.jurisdiction,p.scope,p.effective_from,p.effective_until,p.created_at,p.updated_at,p.version,
			COALESCE(effective_state.overall_state,'UNKNOWN'),COALESCE(ps.dimensions,'{}'::jsonb),COALESCE(ps.reasons,'[]'::jsonb),
			CASE WHEN $9 THEN COALESCE(visible.open_matter_count,0) ELSE COALESCE(ps.open_matter_count,0) END,
			CASE WHEN $9 THEN visible.latest_visible_at ELSE NULL END,
			ps.generated_at,COALESCE(ps.program_version,0),COALESCE(ps.projection_version,0),
			(SELECT count(*) FROM program_requirements pr WHERE pr.tenant_id=p.tenant_id AND pr.program_id=p.id),
			(SELECT count(*) FROM control_implementations ci WHERE ci.tenant_id=p.tenant_id AND ci.program_id=p.id),
			(SELECT count(*) FROM evidence_contracts ec WHERE ec.tenant_id=p.tenant_id AND ec.program_id=p.id)
		FROM programs p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT overall_state,dimensions,reasons,open_matter_count,generated_at,program_version,projection_version
			FROM program_state_snapshots
			WHERE tenant_id=p.tenant_id AND program_id=p.id
			ORDER BY generated_at DESC,projection_version DESC
			LIMIT 1
		) ps ON TRUE
		LEFT JOIN LATERAL (
			SELECT count(DISTINCT m.id)::int AS open_matter_count, max(m.updated_at) AS latest_visible_at
			FROM matter_links ml
			JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
			WHERE $9
			  AND ml.tenant_id=p.tenant_id
			  AND ml.program_id=p.id
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND CASE
					WHEN NOT (m.scope ? 'access') THEN true
					WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false
					WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
					WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
						CASE
							WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false
							ELSE
								NOT EXISTS (
									SELECT 1
									FROM jsonb_array_elements(m.scope->'allowed_principal_ids') entry(value)
									WHERE jsonb_typeof(entry.value)<>'string'
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') nonblank(value)
									WHERE btrim(nonblank.value)<>''
								)
								AND EXISTS (
									SELECT 1
									FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(value)
									WHERE btrim(allowed.value)=$10
								)
						END
					ELSE false
				END
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
				CASE WHEN $9 THEN CASE WHEN COALESCE(visible.open_matter_count,0)>0 THEN 'AT_RISK' ELSE 'CURRENT' END ELSE ps.dimensions->>'exception' END,
				ps.dimensions->>'assurance',ps.dimensions->>'deadline',ps.dimensions->>'source_quality'
			]::text[] AS values) state_values
		) effective_state ON TRUE
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (NOT $9 OR ((t.id::text=$11 OR t.slug=$11) AND p.legal_entity_id IS NOT NULL AND ($12='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$12 OR le.code=$12) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		  AND ($2='' OR p.status=$2)
		  AND ($3='' OR p.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))
		  AND ($13='' OR lower(btrim(p.jurisdiction))=lower(btrim($13)))
		  AND ($14='' OR effective_state.overall_state=$14)
		  AND (NOT $15 OR COALESCE(p.owner_principal_id::text,'')=$16)
		  AND (
			NOT $4 OR
			CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END > $5 OR
			(CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END = $5 AND
				(p.updated_at < $6 OR (p.updated_at = $6 AND p.id < NULLIF($7,'')::uuid)))
		  )
		ORDER BY CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END,
			p.updated_at DESC,p.id DESC
		LIMIT $8`, tenant, query.Status, query.Search, hasCursor, cursor.Rank, cursor.UpdatedAt, cursor.ID, limit+1, enforceEntity, principalID, actorTenant, actorEntity, query.Jurisdiction, query.OverallState, query.AssignedToMe, query.principalID)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	defer rows.Close()
	values := make([]ProgramSummary, 0, limit+1)
	for rows.Next() {
		var value ProgramSummary
		var dimensionsRaw, reasonsRaw json.RawMessage
		var dimensions ComplianceDimensions
		var latestVisibleAt, generatedAt *time.Time
		if err := rows.Scan(
			&value.Program.ID, &value.Program.TenantID, &value.Program.LegalEntityID, &value.Program.Code, &value.Program.Name,
			&value.Program.Type, &value.Program.Status, &value.Program.OwningFunction, &value.Program.OwnerPrincipalID,
			&value.Program.AuthorityPrincipalID, &value.Program.Jurisdiction, &value.Program.Scope, &value.Program.EffectiveFrom,
			&value.Program.EffectiveUntil, &value.Program.CreatedAt, &value.Program.UpdatedAt, &value.Program.Version,
			&value.OverallState, &dimensionsRaw, &reasonsRaw, &value.OpenMatterCount, &latestVisibleAt, &generatedAt,
			&value.AssessedProgramVersion, &value.ProjectionVersion,
			&value.RequirementCount, &value.SafeguardCount, &value.EvidenceCheckCount,
		); err != nil {
			return ProgramSummaryPage{}, err
		}
		if err := json.Unmarshal(reasonsRaw, &value.Reasons); err != nil {
			return ProgramSummaryPage{}, err
		}
		if enforceVisibility && generatedAt != nil {
			if err := json.Unmarshal(dimensionsRaw, &dimensions); err != nil {
				return ProgramSummaryPage{}, err
			}
			state := programStateForVisibleMatters(ProgramStateSnapshot{
				TenantID:          value.Program.TenantID,
				ProgramID:         value.Program.ID,
				Overall:           value.OverallState,
				Dimensions:        dimensions,
				Reasons:           value.Reasons,
				OpenMatterCount:   value.OpenMatterCount,
				ProgramVersion:    value.AssessedProgramVersion,
				ProjectionVersion: value.ProjectionVersion,
			}, value.OpenMatterCount)
			value.OverallState = state.Overall
			value.Reasons = state.Reasons
			value.OpenMatterCount = state.OpenMatterCount
			value.ProjectionVersion = state.ProjectionVersion
			latest := time.Time{}
			if latestVisibleAt != nil {
				latest = latestVisibleAt.UTC()
			}
			visibleGeneratedAt := actorVisibleProgramStateTime(value.Program.UpdatedAt, latest)
			value.StateGeneratedAt = &visibleGeneratedAt
		} else {
			value.StateGeneratedAt = generatedAt
		}
		value.ReasonsTotal = len(value.Reasons)
		if len(value.Reasons) > 6 {
			value.Reasons = value.Reasons[:6]
		}
		value.ReasonsOmitted = max(0, value.ReasonsTotal-len(value.Reasons))
		value.ProgramVersion = value.Program.Version
		value.ProjectionStale = value.AssessedProgramVersion < value.Program.Version
		switch value.Program.Status {
		case ProgramDraft:
			value.StateLabel = "Setup in progress"
		case ProgramPaused:
			value.StateLabel = "Paused"
		case ProgramRetired:
			value.StateLabel = "Ended"
		default:
			value.StateLabel = programStateLabel(value.OverallState)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return ProgramSummaryPage{}, err
	}
	page := ProgramSummaryPage{GeneratedAt: time.Now().UTC()}
	if len(values) > limit {
		last := values[limit-1]
		page.NextCursor, err = encodeSummaryCursor(programSummaryCursor{Rank: programStatusRank(last.Program.Status), UpdatedAt: last.Program.UpdatedAt, ID: last.Program.ID})
		if err != nil {
			return ProgramSummaryPage{}, err
		}
		values = values[:limit]
	}
	page.Items = values
	return page, nil
}

func (r *PostgresRepository) ListMatterSummaries(ctx context.Context, tenant string, query SummaryQuery) (MatterSummaryPage, error) {
	cursor, err := decodeMatterSummaryCursor(query.Cursor)
	if err != nil {
		return MatterSummaryPage{}, err
	}
	limit := boundedLimit(query.Limit)
	hasCursor := cursor.ID != ""
	actor, enforceVisibility := identity.FromContext(ctx)
	principalID := ""
	actorTenant := ""
	if enforceVisibility {
		principalID = actor.PrincipalID
		actorTenant = actor.TenantID
	}
	enforceEntity, actorEntityTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `
		SELECT
			m.id::text,t.id::text,COALESCE(m.legal_entity_id::text,''),m.reference,m.matter_type,m.status,m.priority,m.title,m.summary,m.scope,
			m.source_type,COALESCE(m.source_id::text,''),m.trigger_type,COALESCE(m.trigger_id::text,''),m.trigger_key,m.known_facts,m.missing_facts,m.contradictions,
			COALESCE(m.owner_principal_id::text,''),m.required_authority,m.due_at,m.closed_at,m.closure_reason,m.reopen_count,
			m.created_at,m.updated_at,m.version,
			(SELECT count(DISTINCT ml.program_id) FROM matter_links ml WHERE ml.tenant_id=m.tenant_id AND ml.matter_id=m.id AND ml.program_id IS NOT NULL AND ml.retired_at IS NULL),
			(SELECT count(*) FROM matter_actions ma WHERE ma.tenant_id=m.tenant_id AND ma.matter_id=m.id AND ma.status NOT IN ('IMPLEMENTED','CANCELLED')),
			(SELECT count(*) FROM verification_contracts vc WHERE vc.tenant_id=m.tenant_id AND vc.matter_id=m.id AND vc.status='ACTIVE'),
			latest.result,latest.observed_at
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		LEFT JOIN LATERAL (
			SELECT vr.result,vr.observed_at
			FROM verification_results vr
			WHERE vr.tenant_id=m.tenant_id AND vr.matter_id=m.id
			ORDER BY vr.observed_at DESC,vr.id DESC
			LIMIT 1
		) latest ON TRUE
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (NOT $13 OR ((t.id::text=$14 OR t.slug=$14) AND m.legal_entity_id IS NOT NULL AND ($15='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$15 OR le.code=$15) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		  AND ($2='' OR ($2='OPEN' AND m.status NOT IN ('CLOSED','CANCELLED')) OR m.status=$2)
		  AND ($3='' OR m.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))
		  AND ($12='' OR EXISTS (
			SELECT 1 FROM matter_links program_link
			WHERE program_link.tenant_id=m.tenant_id AND program_link.matter_id=m.id AND program_link.program_id=NULLIF($12,'')::uuid AND program_link.retired_at IS NULL
		  ))
		  AND ($16='' OR m.matter_type=$16)
		  AND ($17=0 OR m.priority=$17)
		  AND (NOT $20 OR COALESCE(m.owner_principal_id::text,'')=$21)
		  AND (
			$18='' OR
			($18='NO_DUE_DATE' AND m.due_at IS NULL) OR
			($18='OVERDUE' AND m.due_at<$19 AND m.status NOT IN ('CLOSED','CANCELLED')) OR
			($18='DUE_7_DAYS' AND m.due_at>=$19 AND m.due_at<=$19+interval '7 days' AND m.status NOT IN ('CLOSED','CANCELLED')) OR
			($18='DUE_30_DAYS' AND m.due_at>=$19 AND m.due_at<=$19+interval '30 days' AND m.status NOT IN ('CLOSED','CANCELLED'))
		  )
		  AND (NOT $4 OR t.id::text=$6 OR t.slug=$6)
		  AND (
			NOT $4 OR
			CASE
				WHEN NOT (m.scope ? 'access') THEN true
				WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false
				WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
				WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
					CASE
						WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false
						ELSE
							NOT EXISTS (
								SELECT 1
								FROM jsonb_array_elements(m.scope->'allowed_principal_ids') entry(value)
								WHERE jsonb_typeof(entry.value)<>'string'
							)
							AND EXISTS (
								SELECT 1
								FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') nonblank(value)
								WHERE btrim(nonblank.value)<>''
							)
							AND EXISTS (
								SELECT 1
								FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(value)
								WHERE btrim(allowed.value)=$5
							)
					END
				ELSE false
			END
		  )
		  AND (
			NOT $7 OR m.priority < $8 OR
			(m.priority = $8 AND (m.updated_at < $9 OR (m.updated_at = $9 AND m.id < NULLIF($10,'')::uuid)))
		  )
		ORDER BY m.priority DESC,m.updated_at DESC,m.id DESC
		LIMIT $11`, tenant, query.Status, query.Search, enforceVisibility, principalID, actorTenant, hasCursor, cursor.Priority, cursor.UpdatedAt, cursor.ID, limit+1, query.ProgramID, enforceEntity, actorEntityTenant, actorEntity, query.MatterType, query.Priority, query.DueCondition, query.asOf, query.AssignedToMe, query.principalID)
	if err != nil {
		return MatterSummaryPage{}, err
	}
	defer rows.Close()
	values := make([]MatterSummary, 0, limit+1)
	for rows.Next() {
		var value MatterSummary
		var sourceID, triggerID string
		var latestOutcome *VerificationResultStatus
		var latestAt *time.Time
		if err := rows.Scan(
			&value.Matter.ID, &value.Matter.TenantID, &value.Matter.LegalEntityID, &value.Matter.Reference, &value.Matter.Type, &value.Matter.Status,
			&value.Matter.Priority, &value.Matter.Title, &value.Matter.Summary, &value.Matter.Scope, &value.Matter.SourceType,
			&sourceID, &value.Matter.TriggerType, &triggerID, &value.Matter.TriggerKey, &value.Matter.KnownFacts,
			&value.Matter.MissingFacts, &value.Matter.Contradictions, &value.Matter.OwnerPrincipalID, &value.Matter.RequiredAuthority,
			&value.Matter.DueAt, &value.Matter.ClosedAt, &value.Matter.ClosureReason, &value.Matter.ReopenCount,
			&value.Matter.CreatedAt, &value.Matter.UpdatedAt, &value.Matter.Version,
			&value.ProgramCount, &value.OpenActionCount, &value.OutcomeCheckCount, &latestOutcome, &latestAt,
		); err != nil {
			return MatterSummaryPage{}, err
		}
		value.Matter.SourceID = sourceID
		value.Matter.TriggerID = triggerID
		if latestOutcome != nil {
			value.LatestOutcome = *latestOutcome
		}
		value.LatestOutcomeAt = latestAt
		value.TypeLabel = matterTypeLabel(value.Matter.Type)
		value.StatusLabel = matterStatusLabel(value.Matter.Status)
		value.NextAction = matterNextAction(value.Matter.Status)
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return MatterSummaryPage{}, err
	}
	page := MatterSummaryPage{GeneratedAt: time.Now().UTC()}
	if len(values) > limit {
		last := values[limit-1].Matter
		page.NextCursor, err = encodeSummaryCursor(matterSummaryCursor{Priority: last.Priority, UpdatedAt: last.UpdatedAt, ID: last.ID})
		if err != nil {
			return MatterSummaryPage{}, err
		}
		values = values[:limit]
	}
	page.Items = values
	return page, nil
}

var _ SummaryRepository = (*PostgresRepository)(nil)
var _ SummaryRepository = (*MemoryRepository)(nil)

func normalizedSummaryStatus(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
