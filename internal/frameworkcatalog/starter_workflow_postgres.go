//go:build postgres

package frameworkcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"github.com/jackc/pgx/v5"
	"strings"
)

func (r *PostgresRepository) StarterRequirements(ctx context.Context, tenant, entity, code string) ([]continuity.StarterRequirement, error) {
	rows, err := r.pool.Query(ctx, `SELECT l.requirement_code,l.title,l.statement,l.reference_label,l.version,l.form_template_id::text,l.form_template_version,f.name,f.status,f.approved_by IS NOT NULL,l.group_id,
 COALESCE((SELECT s->>'title' FROM jsonb_array_elements(f.sections) s WHERE s->>'id'=l.group_id),''),
 COALESCE((SELECT jsonb_agg(q) FROM jsonb_array_elements(f.fields) q WHERE q->>'section_id'=l.group_id),'[]'::jsonb)
 FROM framework_starter_requirement_links l JOIN tenants t ON t.id=l.tenant_id
 JOIN monitoring_form_templates f ON f.tenant_id=l.tenant_id AND f.legal_entity_id=l.legal_entity_id AND f.id=l.form_template_id AND f.version=l.form_template_version
 LEFT JOIN framework_starter_requirements d ON d.template_code=l.template_code AND d.requirement_code=l.requirement_code
 WHERE (t.id::text=$1 OR t.slug=$1) AND l.legal_entity_id=$2::uuid AND l.template_code=$3 ORDER BY COALESCE(d.ordinal,1000),l.requirement_code LIMIT 100`, tenant, entity, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]continuity.StarterRequirement, 0)
	for rows.Next() {
		var v continuity.StarterRequirement
		var fields []byte
		if err = rows.Scan(&v.Code, &v.Title, &v.Statement, &v.Reference, &v.Version, &v.FormID, &v.FormVersion, &v.FormName, &v.FormStatus, &v.FormApproved, &v.GroupID, &v.GroupTitle, &fields); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(fields, &v.Fields); err != nil {
			return nil, err
		}
		v.OriginalStatement = v.Statement
		v.OriginalTitle = v.Title
		items = append(items, v)
	}
	return items, rows.Err()
}
func (r *PostgresRepository) ValidateStarterRespondents(ctx context.Context, tenant, entity string, ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return continuity.ErrInvalidState
		}
		seen[id] = true
		var exists bool
		err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_scopes s JOIN tenants t ON t.id=s.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND s.legal_entity_id=$2::uuid AND s.id::text=$3 AND s.kind IN ('DEPARTMENT','ORGANIZATION_UNIT','FUNCTION','BUSINESS_UNIT') AND s.status='ACTIVE' AND s.valid_from<=clock_timestamp() AND (s.valid_until IS NULL OR s.valid_until>clock_timestamp()))`, tenant, entity, id).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return continuity.ErrInvalidState
		}
	}
	return nil
}
func (r *PostgresRepository) ValidateStarterSnapshot(ctx context.Context, tenant, entity string, c continuity.StarterConfiguration) error {
	if err := r.ValidateStarterRespondents(ctx, tenant, entity, c.RespondentScopeIDs); err != nil {
		return err
	}
	for _, v := range c.Requirements {
		var ok bool
		err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid AND f.id::text=$3 AND f.version=$4 AND f.approved_by IS NOT NULL AND f.status IN ('ACTIVE','RETIRED') AND EXISTS(SELECT 1 FROM jsonb_array_elements(f.sections) s WHERE s->>'id'=$5))`, tenant, entity, v.FormID, v.FormVersion, v.GroupID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return continuity.ErrInvalidState
		}
	}
	return nil
}

// UpdateStarterRequirement changes the bank library only. Existing adoption snapshots
// remain unchanged. Actor/scope are verified, route authority is checked by httpapi.
func (r *PostgresRepository) UpdateStarterRequirement(ctx context.Context, code string, v continuity.StarterRequirement, rationale string) (int64, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	// Verify exact entity, form revision and nonempty question group before taking a link.
	var valid bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM monitoring_form_templates f WHERE f.tenant_id=$1::uuid AND f.legal_entity_id=$2::uuid AND f.id::text=$3 AND f.version=$4 AND EXISTS(SELECT 1 FROM jsonb_array_elements(f.sections) s WHERE s->>'id'=$5) AND EXISTS(SELECT 1 FROM jsonb_array_elements(f.fields) q WHERE q->>'section_id'=$5))`, actor.TenantID, actor.LegalEntityID, v.FormID, v.FormVersion, v.GroupID).Scan(&valid)
	if err != nil {
		return 0, err
	}
	if !valid {
		return 0, continuity.ErrInvalidState
	}
	var version int64
	if v.Version == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO framework_starter_requirement_links(tenant_id,legal_entity_id,template_code,requirement_code,title,statement,reference_label,form_template_id,form_template_version,group_id) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,'Bank-defined starter requirement; framework mapping requires review',$7::uuid,$8,$9) ON CONFLICT DO NOTHING RETURNING version`, actor.TenantID, actor.LegalEntityID, code, v.Code, v.Title, v.Statement, v.FormID, v.FormVersion, v.GroupID).Scan(&version)
	} else {
		err = tx.QueryRow(ctx, `UPDATE framework_starter_requirement_links SET title=$5,statement=$6,form_template_id=$7::uuid,form_template_version=$8,group_id=$9,version=version+1 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND template_code=$3 AND requirement_code=$4 AND version=$10 RETURNING version`, actor.TenantID, actor.LegalEntityID, code, v.Code, v.Title, v.Statement, v.FormID, v.FormVersion, v.GroupID, v.Version).Scan(&version)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, continuity.ErrVersionConflict
	}
	if err != nil {
		return 0, err
	}
	v.Version = version
	b, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	var eventID string
	err = tx.QueryRow(ctx, `INSERT INTO framework_starter_requirement_events(tenant_id,legal_entity_id,template_code,requirement_code,version,actor_id,payload,rationale) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::uuid,$7::jsonb,$8) RETURNING id::text`, actor.TenantID, actor.LegalEntityID, code, v.Code, version, actor.PrincipalID, b, rationale).Scan(&eventID)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FRAMEWORK_STARTER_REQUIREMENT',$2::uuid,'FRAMEWORK_STARTER_REQUIREMENT_UPDATED',$3::jsonb,clock_timestamp(),clock_timestamp(),clock_timestamp())`, actor.TenantID, eventID, b)
	if err != nil {
		return 0, err
	}
	return version, tx.Commit(ctx)
}

// StarterRespondentUnits exposes only routing names in the Program legal entity.
// httpapi verifies current Program ownership/review before calling it.
func (r *PostgresRepository) StarterRespondentUnits(ctx context.Context, tenant, entity, query string) ([]runtimecontext.ScopeNode, error) {
	rows, err := r.pool.Query(ctx, `SELECT s.id::text,s.name,s.kind FROM organization_scopes s JOIN tenants t ON t.id=s.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND s.legal_entity_id=$2::uuid AND s.kind IN ('ORGANIZATION_UNIT','DEPARTMENT','FUNCTION','BUSINESS_UNIT') AND s.status='ACTIVE' AND s.valid_from<=clock_timestamp() AND (s.valid_until IS NULL OR s.valid_until>clock_timestamp()) AND ($3='' OR s.name ILIKE '%'||$3||'%' OR s.code ILIKE '%'||$3||'%') ORDER BY s.name,s.id LIMIT 50`, tenant, entity, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]runtimecontext.ScopeNode, 0)
	for rows.Next() {
		var v runtimecontext.ScopeNode
		if err = rows.Scan(&v.ID, &v.Name, &v.Kind); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
