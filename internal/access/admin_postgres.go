//go:build postgres

package access

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdministrator struct{ pool *pgxpool.Pool }

func NewPostgresAdministrator(pool *pgxpool.Pool) *PostgresAdministrator {
	return &PostgresAdministrator{pool: pool}
}

func (a *PostgresAdministrator) Overview(ctx context.Context, tenant, legalEntity string, limit int) (AdminOverview, error) {
	tenant = strings.TrimSpace(tenant)
	legalEntity = strings.TrimSpace(legalEntity)
	if tenant == "" || legalEntity == "" {
		return AdminOverview{}, ErrAdminInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var tenantID, entityID string
	if err := a.pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1`, tenant).Scan(&tenantID); errors.Is(err, pgx.ErrNoRows) {
		return AdminOverview{}, ErrAdminNotFound
	} else if err != nil {
		return AdminOverview{}, err
	}
	if err := a.pool.QueryRow(ctx, `SELECT id::text FROM legal_entities WHERE tenant_id=$1::uuid AND (id::text=$2 OR code=$2) AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until) LIMIT 1`, tenantID, legalEntity).Scan(&entityID); errors.Is(err, pgx.ErrNoRows) {
		return AdminOverview{}, ErrAdminNotFound
	} else if err != nil {
		return AdminOverview{}, err
	}

	result := AdminOverview{}
	rows, err := a.pool.Query(ctx, `
		SELECT ss.id::text,ss.code,ss.status,COALESCE(ss.identity_issuer,''),ss.subject_attribute,
		       (SELECT count(*) FROM scim_users su WHERE su.source_id=ss.id AND su.active AND su.deleted_at IS NULL),
		       (SELECT count(*) FROM directory_groups dg WHERE dg.source_id=ss.id AND dg.deleted_at IS NULL),
		       GREATEST(
		         COALESCE((SELECT max(su.updated_at) FROM scim_users su WHERE su.source_id=ss.id),ss.updated_at),
		         COALESCE((SELECT max(dg.updated_at) FROM directory_groups dg WHERE dg.source_id=ss.id),ss.updated_at)
		       ),ss.created_at,ss.updated_at
		FROM scim_sources ss WHERE ss.tenant_id=$1::uuid ORDER BY ss.code`, tenantID)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value SCIMSourceSummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Status, &value.IdentityIssuer, &value.SubjectAttribute, &value.ActiveUsers, &value.ActiveGroups, &value.LastActivityAt, &value.CreatedAt, &value.UpdatedAt); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Sources = append(result.Sources, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	rows, err = a.pool.Query(ctx, `
		SELECT p.id::text,p.display_name,p.status,COALESCE(su.user_name,''),COALESCE(ss.code,''),COALESCE(ss.status,'')
		FROM principals p
		LEFT JOIN scim_users su ON su.tenant_id=p.tenant_id AND su.principal_id=p.id AND su.deleted_at IS NULL
		LEFT JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id
		WHERE p.tenant_id=$1::uuid AND p.kind='PERSON'
		ORDER BY lower(p.display_name),p.id LIMIT $2`, tenantID, limit)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value PersonSummary
		if err := rows.Scan(&value.ID, &value.DisplayName, &value.Status, &value.UserName, &value.SourceCode, &value.SourceState); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.People = append(result.People, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	rows, err = a.pool.Query(ctx, `
		SELECT dg.id::text,dg.display_name,COALESCE(dg.external_id,''),ss.code,ss.status,count(dgm.scim_user_id)
		FROM directory_groups dg
		JOIN scim_sources ss ON ss.tenant_id=dg.tenant_id AND ss.id=dg.source_id
		LEFT JOIN directory_group_members dgm ON dgm.tenant_id=dg.tenant_id AND dgm.group_id=dg.id
		WHERE dg.tenant_id=$1::uuid AND dg.deleted_at IS NULL
		GROUP BY dg.id,dg.display_name,dg.external_id,ss.code,ss.status
		ORDER BY lower(dg.display_name),dg.id LIMIT $2`, tenantID, limit)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value GroupSummary
		if err := rows.Scan(&value.ID, &value.DisplayName, &value.ExternalID, &value.SourceCode, &value.SourceState, &value.MemberCount); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Groups = append(result.Groups, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	rows, err = a.pool.Query(ctx, `
		SELECT id::text,code,name,capabilities FROM role_templates
		WHERE tenant_id=$1::uuid AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)
		ORDER BY code LIMIT 100`, tenantID)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value RoleTemplateSummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Name, &value.Capabilities); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Roles = append(result.Roles, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	rows, err = a.pool.Query(ctx, `
		SELECT id::text,code,name FROM legal_entities
		WHERE tenant_id=$1::uuid AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)
		ORDER BY code`, tenantID)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value LegalEntitySummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Name); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.LegalEntities = append(result.LegalEntities, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	rows, err = a.pool.Query(ctx, `
		SELECT b.id::text,b.group_id::text,dg.display_name,b.role_template_id::text,rt.code,b.legal_entity_id::text,le.code,b.department_path,b.valid_from,b.valid_until
		FROM directory_group_role_bindings b
		JOIN directory_groups dg ON dg.tenant_id=b.tenant_id AND dg.id=b.group_id
		JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
		JOIN legal_entities le ON le.tenant_id=b.tenant_id AND le.id=b.legal_entity_id
		WHERE b.tenant_id=$1::uuid AND b.legal_entity_id=$2::uuid
		  AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR clock_timestamp()<b.valid_until)
		ORDER BY lower(dg.display_name),rt.code,b.department_path,b.id`, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value GroupRoleBindingSummary
		if err := rows.Scan(&value.ID, &value.GroupID, &value.GroupName, &value.RoleTemplateID, &value.RoleCode, &value.LegalEntityID, &value.LegalEntity, &value.DepartmentPath, &value.ValidFrom, &value.ValidUntil); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Bindings = append(result.Bindings, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminOverview{}, err
	}
	rows.Close()

	if err := a.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM workflow_timers wt WHERE wt.tenant_id=$1::uuid AND wt.timer_type='MATTER_ESCALATION' AND wt.state IN ('READY','CLAIMED')),
		  (SELECT count(*) FROM workflow_tasks wt WHERE wt.tenant_id=$1::uuid AND wt.status='ESCALATED' AND COALESCE(wt.context->>'escalation_active','')='true'),
		  (SELECT count(*) FROM workflow_events we WHERE we.tenant_id=$1::uuid AND we.event_type='WORK_ESCALATION_UNRESOLVED' AND we.occurred_at>=clock_timestamp()-interval '24 hours'),
		  (SELECT count(*) FROM workflow_timers wt WHERE wt.tenant_id=$1::uuid AND wt.timer_type='MATTER_ESCALATION' AND wt.state='FAILED')`, tenantID).
		Scan(&result.Escalation.PendingTimers, &result.Escalation.EscalatedTasks, &result.Escalation.Unresolved24h, &result.Escalation.FailedTimers); err != nil {
		return AdminOverview{}, err
	}
	return result, nil
}

func (a *PostgresAdministrator) CreateSCIMSource(ctx context.Context, input CreateSCIMSourceInput, tokenHash []byte) (SCIMSourceSummary, error) {
	input, err := normalizeSCIMSourceInput(input)
	if err != nil || len(tokenHash) != 32 {
		return SCIMSourceSummary{}, ErrAdminInvalid
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return SCIMSourceSummary{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return SCIMSourceSummary{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO scim_sources(tenant_id,code,token_hash,identity_issuer,subject_attribute)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,NULLIF($4,''),$5)
		RETURNING id::text`, input.TenantID, input.Code, tokenHash, input.IdentityIssuer, input.SubjectAttribute).Scan(&id)
	if err != nil {
		return SCIMSourceSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminAudit(ctx, tx, input.TenantID, input.ActorID, "SCIM_SOURCE_CREATED", "SCIM_SOURCE", id, map[string]any{"code": input.Code}); err != nil {
		return SCIMSourceSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SCIMSourceSummary{}, err
	}
	return a.scimSourceByID(ctx, input.TenantID, id)
}

func (a *PostgresAdministrator) RotateSCIMSourceToken(ctx context.Context, tenant, sourceID, actorID string, tokenHash []byte) error {
	if len(tokenHash) != 32 {
		return ErrAdminInvalid
	}
	return a.mutateSCIMSource(ctx, tenant, sourceID, actorID, "SCIM_SOURCE_TOKEN_ROTATED", func(ctx context.Context, tx pgx.Tx) (int64, error) {
		tag, err := tx.Exec(ctx, `UPDATE scim_sources SET token_hash=$1,updated_at=clock_timestamp() WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND id::text=$3 AND status='ACTIVE'`, tokenHash, tenant, strings.TrimSpace(sourceID))
		if err != nil {
			return 0, mapAdminPgError(err)
		}
		return tag.RowsAffected(), nil
	})
}

func (a *PostgresAdministrator) RevokeSCIMSource(ctx context.Context, tenant, sourceID, actorID string) error {
	return a.mutateSCIMSource(ctx, tenant, sourceID, actorID, "SCIM_SOURCE_REVOKED", func(ctx context.Context, tx pgx.Tx) (int64, error) {
		tag, err := tx.Exec(ctx, `UPDATE scim_sources SET status='REVOKED',updated_at=clock_timestamp() WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2 AND status='ACTIVE'`, tenant, strings.TrimSpace(sourceID))
		return tag.RowsAffected(), err
	})
}

func (a *PostgresAdministrator) mutateSCIMSource(ctx context.Context, tenant, sourceID, actorID, eventType string, mutate func(context.Context, pgx.Tx) (int64, error)) error {
	tenant, sourceID, actorID = strings.TrimSpace(tenant), strings.TrimSpace(sourceID), strings.TrimSpace(actorID)
	if tenant == "" || sourceID == "" || actorID == "" {
		return ErrAdminInvalid
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, tenant, actorID); err != nil {
		return err
	}
	changed, err := mutate(ctx, tx)
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrAdminNotFound
	}
	if err := recordAdminAudit(ctx, tx, tenant, actorID, eventType, "SCIM_SOURCE", sourceID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *PostgresAdministrator) CreateGroupRoleBinding(ctx context.Context, input CreateGroupRoleBindingInput) (GroupRoleBindingSummary, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.GroupID = strings.TrimSpace(input.GroupID)
	input.RoleTemplateID = strings.TrimSpace(input.RoleTemplateID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	path, err := identity.NormalizeDepartmentPath(input.DepartmentPath)
	if err != nil || input.TenantID == "" || input.GroupID == "" || input.RoleTemplateID == "" || input.LegalEntityID == "" || input.ActorID == "" {
		return GroupRoleBindingSummary{}, ErrAdminInvalid
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return GroupRoleBindingSummary{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return GroupRoleBindingSummary{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO directory_group_role_bindings(tenant_id,group_id,role_template_id,legal_entity_id,department_path,valid_from)
		SELECT t.id,dg.id,rt.id,le.id,$5::text[],clock_timestamp()
		FROM tenants t
		JOIN directory_groups dg ON dg.tenant_id=t.id AND dg.id::text=$2 AND dg.deleted_at IS NULL
		JOIN scim_sources ss ON ss.tenant_id=dg.tenant_id AND ss.id=dg.source_id AND ss.status='ACTIVE'
		JOIN role_templates rt ON rt.tenant_id=t.id AND rt.id::text=$3 AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		JOIN legal_entities le ON le.tenant_id=t.id AND le.id::text=$4 AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		WHERE (t.id::text=$1 OR t.slug=$1)
		RETURNING id::text`, input.TenantID, input.GroupID, input.RoleTemplateID, input.LegalEntityID, path).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return GroupRoleBindingSummary{}, ErrAdminNotFound
	}
	if err != nil {
		return GroupRoleBindingSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminAudit(ctx, tx, input.TenantID, input.ActorID, "DIRECTORY_GROUP_ROLE_BOUND", "DIRECTORY_GROUP_ROLE_BINDING", id, map[string]any{"department_path": path}); err != nil {
		return GroupRoleBindingSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GroupRoleBindingSummary{}, err
	}
	return a.groupRoleBindingByID(ctx, input.TenantID, id)
}

func (a *PostgresAdministrator) RetireGroupRoleBinding(ctx context.Context, tenant, bindingID, actorID string) error {
	tenant, bindingID, actorID = strings.TrimSpace(tenant), strings.TrimSpace(bindingID), strings.TrimSpace(actorID)
	if tenant == "" || bindingID == "" || actorID == "" {
		return ErrAdminInvalid
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, tenant, actorID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE directory_group_role_bindings SET valid_until=clock_timestamp() WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2 AND valid_until IS NULL`, tenant, bindingID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminNotFound
	}
	if err := recordAdminAudit(ctx, tx, tenant, actorID, "DIRECTORY_GROUP_ROLE_RETIRED", "DIRECTORY_GROUP_ROLE_BINDING", bindingID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *PostgresAdministrator) scimSourceByID(ctx context.Context, tenant, id string) (SCIMSourceSummary, error) {
	var value SCIMSourceSummary
	err := a.pool.QueryRow(ctx, `
		SELECT ss.id::text,ss.code,ss.status,COALESCE(ss.identity_issuer,''),ss.subject_attribute,
		       (SELECT count(*) FROM scim_users su WHERE su.source_id=ss.id AND su.active AND su.deleted_at IS NULL),
		       (SELECT count(*) FROM directory_groups dg WHERE dg.source_id=ss.id AND dg.deleted_at IS NULL),
		       GREATEST(COALESCE((SELECT max(su.updated_at) FROM scim_users su WHERE su.source_id=ss.id),ss.updated_at),COALESCE((SELECT max(dg.updated_at) FROM directory_groups dg WHERE dg.source_id=ss.id),ss.updated_at)),
		       ss.created_at,ss.updated_at
		FROM scim_sources ss JOIN tenants t ON t.id=ss.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ss.id::text=$2`, tenant, id).
		Scan(&value.ID, &value.Code, &value.Status, &value.IdentityIssuer, &value.SubjectAttribute, &value.ActiveUsers, &value.ActiveGroups, &value.LastActivityAt, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SCIMSourceSummary{}, ErrAdminNotFound
	}
	return value, err
}

func (a *PostgresAdministrator) groupRoleBindingByID(ctx context.Context, tenant, id string) (GroupRoleBindingSummary, error) {
	var value GroupRoleBindingSummary
	err := a.pool.QueryRow(ctx, `
		SELECT b.id::text,b.group_id::text,dg.display_name,b.role_template_id::text,rt.code,b.legal_entity_id::text,le.code,b.department_path,b.valid_from,b.valid_until
		FROM directory_group_role_bindings b
		JOIN tenants t ON t.id=b.tenant_id
		JOIN directory_groups dg ON dg.tenant_id=b.tenant_id AND dg.id=b.group_id
		JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
		JOIN legal_entities le ON le.tenant_id=b.tenant_id AND le.id=b.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND b.id::text=$2`, tenant, id).
		Scan(&value.ID, &value.GroupID, &value.GroupName, &value.RoleTemplateID, &value.RoleCode, &value.LegalEntityID, &value.LegalEntity, &value.DepartmentPath, &value.ValidFrom, &value.ValidUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return GroupRoleBindingSummary{}, ErrAdminNotFound
	}
	return value, err
}

func ensureAdminActor(ctx context.Context, tx pgx.Tx, tenant, actorID string) error {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM principals p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2 AND p.status='ACTIVE')`, tenant, actorID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrAdminInvalid
	}
	return nil
}

func recordAdminAudit(ctx context.Context, tx pgx.Tx, tenant, actorID, eventType, subjectType, subjectID string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events(tenant_id,actor_id,event_type,subject_type,subject_id,purpose,safe_metadata)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,$5::uuid,'IDENTITY_ACCESS_ADMIN',$6::jsonb)`, tenant, actorID, eventType, subjectType, subjectID, metadata)
	return err
}

func mapAdminPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAdminConflict
	}
	return err
}

var _ Administrator = (*PostgresAdministrator)(nil)
var _ = fmt.Sprintf
var _ = time.Time{}
