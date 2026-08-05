//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

func (r *PostgresRepository) ListProgramSummaries(ctx context.Context, tenant string, query SummaryQuery) (ProgramSummaryPage, error) {
	cursor, err := decodeProgramSummaryCursor(query.Cursor)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	limit := boundedLimit(query.Limit)
	hasCursor := cursor.ID != ""
	rows, err := r.pool.Query(ctx, `
		SELECT
			p.id::text,t.slug,COALESCE(p.legal_entity_id::text,''),p.code,p.name,p.program_type,p.status,
			p.owning_function,COALESCE(p.owner_principal_id::text,''),COALESCE(p.authority_principal_id::text,''),
			p.jurisdiction,p.scope,p.effective_from,p.effective_until,p.created_at,p.updated_at,p.version,
			COALESCE(ps.overall_state,'UNKNOWN'),COALESCE(ps.reasons,'[]'::jsonb),COALESCE(ps.open_matter_count,0),ps.generated_at,
			(SELECT count(*) FROM program_requirements pr WHERE pr.tenant_id=p.tenant_id AND pr.program_id=p.id),
			(SELECT count(*) FROM control_implementations ci WHERE ci.tenant_id=p.tenant_id AND ci.program_id=p.id),
			(SELECT count(*) FROM evidence_contracts ec WHERE ec.tenant_id=p.tenant_id AND ec.program_id=p.id)
		FROM programs p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT overall_state,reasons,open_matter_count,generated_at
			FROM program_state_snapshots
			WHERE tenant_id=p.tenant_id AND program_id=p.id
			ORDER BY generated_at DESC,id DESC
			LIMIT 1
		) ps ON TRUE
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND ($2='' OR p.status=$2)
		  AND ($3='' OR p.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))
		  AND (
			NOT $4 OR
			CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END > $5 OR
			(CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END = $5 AND
				(p.updated_at < $6 OR (p.updated_at = $6 AND p.id < NULLIF($7,'')::uuid)))
		  )
		ORDER BY CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END,
			p.updated_at DESC,p.id DESC
		LIMIT $8`, tenant, query.Status, query.Search, hasCursor, cursor.Rank, cursor.UpdatedAt, cursor.ID, limit+1)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	defer rows.Close()
	values := make([]ProgramSummary, 0, limit+1)
	for rows.Next() {
		var value ProgramSummary
		var reasons json.RawMessage
		var generatedAt *time.Time
		if err := rows.Scan(
			&value.Program.ID, &value.Program.TenantID, &value.Program.LegalEntityID, &value.Program.Code, &value.Program.Name,
			&value.Program.Type, &value.Program.Status, &value.Program.OwningFunction, &value.Program.OwnerPrincipalID,
			&value.Program.AuthorityPrincipalID, &value.Program.Jurisdiction, &value.Program.Scope, &value.Program.EffectiveFrom,
			&value.Program.EffectiveUntil, &value.Program.CreatedAt, &value.Program.UpdatedAt, &value.Program.Version,
			&value.OverallState, &reasons, &value.OpenMatterCount, &generatedAt,
			&value.RequirementCount, &value.SafeguardCount, &value.EvidenceCheckCount,
		); err != nil {
			return ProgramSummaryPage{}, err
		}
		if err := json.Unmarshal(reasons, &value.Reasons); err != nil {
			return ProgramSummaryPage{}, err
		}
		if len(value.Reasons) > 6 {
			value.Reasons = value.Reasons[:6]
		}
		value.StateGeneratedAt = generatedAt
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
	rows, err := r.pool.Query(ctx, `
		SELECT
			m.id::text,t.slug,m.reference,m.matter_type,m.status,m.priority,m.title,m.summary,m.scope,
			m.source_type,COALESCE(m.source_id::text,''),m.trigger_type,COALESCE(m.trigger_id::text,''),m.trigger_key,m.known_facts,m.missing_facts,m.contradictions,
			COALESCE(m.owner_principal_id::text,''),m.required_authority,m.due_at,m.closed_at,m.closure_reason,m.reopen_count,
			m.created_at,m.updated_at,m.version,
			(SELECT count(DISTINCT ml.program_id) FROM matter_links ml WHERE ml.tenant_id=m.tenant_id AND ml.matter_id=m.id AND ml.program_id IS NOT NULL),
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
		  AND ($2='' OR ($2='OPEN' AND m.status NOT IN ('CLOSED','CANCELLED')) OR m.status=$2)
		  AND ($3='' OR m.search_document @@ websearch_to_tsquery('simple'::regconfig,$3))
		  AND (
			NOT $4 OR m.priority < $5 OR
			(m.priority = $5 AND (m.updated_at < $6 OR (m.updated_at = $6 AND m.id < NULLIF($7,'')::uuid)))
		  )
		ORDER BY m.priority DESC,m.updated_at DESC,m.id DESC
		LIMIT $8`, tenant, query.Status, query.Search, hasCursor, cursor.Priority, cursor.UpdatedAt, cursor.ID, limit+1)
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
			&value.Matter.ID, &value.Matter.TenantID, &value.Matter.Reference, &value.Matter.Type, &value.Matter.Status,
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
