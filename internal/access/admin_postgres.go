//go:build postgres

package access

import (
	"context"
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/organization"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdministrator struct{ pool *pgxpool.Pool }

func NewPostgresAdministrator(pool *pgxpool.Pool) *PostgresAdministrator {
	return &PostgresAdministrator{pool: pool}
}

func (a *PostgresAdministrator) OperationalStatus(ctx context.Context, tenant string, limit int) (OperationalStatus, error) {
	tenant = strings.TrimSpace(tenant)
	if tenant == "" {
		return OperationalStatus{}, ErrAdminInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var tenantID string
	if err := a.pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1`, tenant).Scan(&tenantID); errors.Is(err, pgx.ErrNoRows) {
		return OperationalStatus{}, ErrAdminNotFound
	} else if err != nil {
		return OperationalStatus{}, err
	}

	result := OperationalStatus{}
	rows, err := a.pool.Query(ctx, `
		SELECT ss.id::text,ss.code,ss.status,COALESCE(ss.identity_issuer,''),ss.subject_attribute,
		       (SELECT count(*) FROM scim_users su WHERE su.source_id=ss.id AND su.active AND su.deleted_at IS NULL),
		       (SELECT count(*) FROM directory_groups dg WHERE dg.source_id=ss.id AND dg.deleted_at IS NULL),
		       GREATEST(
		         COALESCE((SELECT max(su.updated_at) FROM scim_users su WHERE su.source_id=ss.id),ss.updated_at),
		         COALESCE((SELECT max(dg.updated_at) FROM directory_groups dg WHERE dg.source_id=ss.id),ss.updated_at)
		       ),ss.created_at,ss.updated_at
		FROM scim_sources ss
		WHERE ss.tenant_id=$1::uuid AND upper(ss.status)<>'ACTIVE'
		ORDER BY ss.code LIMIT $2`, tenantID, limit)
	if err != nil {
		return OperationalStatus{}, err
	}
	for rows.Next() {
		var value SCIMSourceSummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Status, &value.IdentityIssuer, &value.SubjectAttribute, &value.ActiveUsers, &value.ActiveGroups, &value.LastActivityAt, &value.CreatedAt, &value.UpdatedAt); err != nil {
			rows.Close()
			return OperationalStatus{}, err
		}
		result.SourceExceptions = append(result.SourceExceptions, value)
	}
	if err := closeRows(rows); err != nil {
		return OperationalStatus{}, err
	}

	if err := a.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM workflow_timers wt WHERE wt.tenant_id=$1::uuid AND wt.timer_type='MATTER_ESCALATION' AND wt.state IN ('READY','CLAIMED')),
		  (SELECT count(*) FROM workflow_tasks wt WHERE wt.tenant_id=$1::uuid AND wt.status='ESCALATED' AND COALESCE(wt.context->>'escalation_active','')='true'),
		  (SELECT count(*) FROM workflow_events we WHERE we.tenant_id=$1::uuid AND we.event_type='WORK_ESCALATION_UNRESOLVED' AND we.occurred_at>=clock_timestamp()-interval '24 hours'),
		  (SELECT count(*) FROM workflow_timers wt WHERE wt.tenant_id=$1::uuid AND wt.timer_type='MATTER_ESCALATION' AND wt.state='FAILED')`, tenantID).
		Scan(&result.Escalation.PendingTimers, &result.Escalation.EscalatedTasks, &result.Escalation.Unresolved24h, &result.Escalation.FailedTimers); err != nil {
		return OperationalStatus{}, err
	}
	return result, nil
}

func (a *PostgresAdministrator) Overview(ctx context.Context, tenant, legalEntity string, limit int) (AdminOverview, error) {
	tenant, legalEntity = strings.TrimSpace(tenant), strings.TrimSpace(legalEntity)
	if tenant == "" || legalEntity == "" {
		return AdminOverview{}, ErrAdminInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	tenantID, entityID, err := a.scopeIDs(ctx, tenant, legalEntity)
	if err != nil {
		return AdminOverview{}, err
	}

	result := AdminOverview{}
	organizationPage, err := organization.NewPostgresRepository(a.pool).List(ctx, tenantID, entityID, 500)
	if err != nil {
		return AdminOverview{}, err
	}
	result.OrganizationScopes = organizationPage.Items
	result.OrganizationScopesTruncated = organizationPage.Truncated

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
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

	rows, err = a.pool.Query(ctx, `
		SELECT op.id::text,op.code,op.title,COALESCE(op.function_name,''),op.department_path,COALESCE(op.organization_scope_id::text,''),
		       COALESCE(parent.id::text,''),COALESCE(parent.code,''),COALESCE(parent.title,''),
		       COALESCE(occupant.id::text,''),COALESCE(occupant.display_name,''),COALESCE(occupant.status,''),
		       COALESCE(array_agg(DISTINCT rt.code ORDER BY rt.code) FILTER (WHERE rt.id IS NOT NULL),ARRAY[]::text[]),
		       op.valid_from,op.valid_until,op.version
		FROM org_positions op
		LEFT JOIN org_positions parent ON parent.tenant_id=op.tenant_id AND parent.id=op.parent_position_id
		LEFT JOIN principals occupant ON occupant.tenant_id=op.tenant_id AND occupant.id=op.occupant_principal_id
		LEFT JOIN position_role_bindings prb ON prb.tenant_id=op.tenant_id AND prb.position_id=op.id
		  AND prb.valid_from<=clock_timestamp() AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
		LEFT JOIN role_templates rt ON rt.tenant_id=prb.tenant_id AND rt.id=prb.role_template_id
		  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		WHERE op.tenant_id=$1::uuid AND op.legal_entity_id=$2::uuid
		  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
		GROUP BY op.id,parent.id,parent.code,parent.title,occupant.id,occupant.display_name,occupant.status
		ORDER BY cardinality(op.department_path),op.department_path,op.code LIMIT $3`, tenantID, entityID, limit)
	if err != nil {
		return AdminOverview{}, err
	}
	for rows.Next() {
		var value PositionSummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Title, &value.FunctionName, &value.DepartmentPath, &value.OrganizationScopeID, &value.ParentPositionID, &value.ParentPositionCode, &value.ParentPositionTitle, &value.OccupantPrincipalID, &value.OccupantName, &value.OccupantStatus, &value.RoleCodes, &value.ValidFrom, &value.ValidUntil, &value.Version); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Positions = append(result.Positions, value)
	}
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

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
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

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
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

	result.Roles, err = a.organizationRoleTemplates(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}

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
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

	rows, err = a.pool.Query(ctx, `
		SELECT b.id::text,b.group_id::text,dg.display_name,b.role_template_id::text,rt.code,
		       b.legal_entity_id::text,le.code,b.department_path,COALESCE(b.organization_scope_id::text,''),b.valid_from,b.valid_until
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
		if err := rows.Scan(&value.ID, &value.GroupID, &value.GroupName, &value.RoleTemplateID, &value.RoleCode, &value.LegalEntityID, &value.LegalEntity, &value.DepartmentPath, &value.OrganizationScopeID, &value.ValidFrom, &value.ValidUntil); err != nil {
			rows.Close()
			return AdminOverview{}, err
		}
		result.Bindings = append(result.Bindings, value)
	}
	if err := closeRows(rows); err != nil {
		return AdminOverview{}, err
	}

	result.OrganizationScopeRevisions, err = a.organizationScopeRevisions(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	result.OrganizationPositionRevisions, err = a.organizationPositionRevisions(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	result.OrganizationPositionHistory, err = a.organizationPositionHistory(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	result.OrganizationPositionRoleRevisions, err = a.organizationPositionRoleRevisions(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	result.DataBoundary, err = a.legalEntityDataBoundary(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}
	result.DataBoundaryRevisions, err = a.legalEntityDataBoundaryRevisions(ctx, tenantID, entityID)
	if err != nil {
		return AdminOverview{}, err
	}

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
	if err := tx.QueryRow(ctx, `
		INSERT INTO scim_sources(tenant_id,code,token_hash,identity_issuer,subject_attribute)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,NULLIF($4,''),$5)
		RETURNING id::text`, input.TenantID, input.Code, tokenHash, input.IdentityIssuer, input.SubjectAttribute).Scan(&id); err != nil {
		return SCIMSourceSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "SCIM_SOURCE_CREATED", "SCIM_SOURCE", id); err != nil {
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
		return tag.RowsAffected(), err
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
	if err := recordAdminDecision(ctx, tx, tenant, actorID, eventType, "SCIM_SOURCE", sourceID); err != nil {
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
	if err := tx.QueryRow(ctx, `
		INSERT INTO directory_group_role_bindings(tenant_id,group_id,role_template_id,legal_entity_id,department_path,valid_from)
		SELECT t.id,dg.id,rt.id,le.id,$5::text[],clock_timestamp()
		FROM tenants t
		JOIN directory_groups dg ON dg.tenant_id=t.id AND dg.id::text=$2 AND dg.deleted_at IS NULL
		JOIN scim_sources ss ON ss.tenant_id=dg.tenant_id AND ss.id=dg.source_id AND ss.status='ACTIVE'
		JOIN role_templates rt ON rt.tenant_id=t.id AND rt.id::text=$3 AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		JOIN legal_entities le ON le.tenant_id=t.id AND le.id::text=$4 AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		WHERE (t.id::text=$1 OR t.slug=$1)
		RETURNING id::text`, input.TenantID, input.GroupID, input.RoleTemplateID, input.LegalEntityID, path).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupRoleBindingSummary{}, ErrAdminNotFound
		}
		return GroupRoleBindingSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "DIRECTORY_GROUP_ROLE_BOUND", "DIRECTORY_GROUP_ROLE_BINDING", id); err != nil {
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
	if err := recordAdminDecision(ctx, tx, tenant, actorID, "DIRECTORY_GROUP_ROLE_RETIRED", "DIRECTORY_GROUP_ROLE_BINDING", bindingID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *PostgresAdministrator) ProposeOrganizationScope(ctx context.Context, input ProposeOrganizationScopeInput) (OrganizationScopeRevisionSummary, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.ScopeID = strings.TrimSpace(input.ScopeID)
	input.ParentScopeID = strings.TrimSpace(input.ParentScopeID)
	input.Code = normalizeOrganizationScopeCode(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.ActorID == "" {
		return OrganizationScopeRevisionSummary{}, ErrAdminInvalid
	}
	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}

	var (
		scopeID       = input.ScopeID
		baseVersion   = input.ExpectedVersion
		parentScopeID = input.ParentScopeID
		code          = input.Code
		name          = input.Name
		kind          = input.Kind
		currentPath   []string
	)
	switch input.Operation {
	case OrganizationScopeCreate:
		if scopeID != "" || baseVersion != 0 || code == "" || name == "" || !validManagedOrganizationScopeKind(kind) {
			return OrganizationScopeRevisionSummary{}, ErrAdminInvalid
		}
		if _, err := organizationScopeParentPath(ctx, tx, tenantID, entityID, parentScopeID); err != nil {
			return OrganizationScopeRevisionSummary{}, err
		}
		var conflict bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM organization_scopes s
				WHERE s.tenant_id=$1::uuid AND s.legal_entity_id=$2::uuid
				  AND upper(s.code)=upper($3) AND s.status='ACTIVE' AND s.valid_until IS NULL
				UNION ALL
				SELECT 1 FROM organization_scope_revisions r
				WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid
				  AND upper(r.proposed_code)=upper($3) AND r.operation='CREATE' AND r.status='PENDING'
			)`, tenantID, entityID, code).Scan(&conflict); err != nil {
			return OrganizationScopeRevisionSummary{}, err
		}
		if conflict {
			return OrganizationScopeRevisionSummary{}, ErrAdminConflict
		}
		if err := tx.QueryRow(ctx, `SELECT uuidv7()::text`).Scan(&scopeID); err != nil {
			return OrganizationScopeRevisionSummary{}, err
		}
	case OrganizationScopeUpdate, OrganizationScopeMove, OrganizationScopeRetire:
		if scopeID == "" || baseVersion <= 0 {
			return OrganizationScopeRevisionSummary{}, ErrAdminInvalid
		}
		var currentParent, currentCode, currentName string
		var currentKind organization.ScopeKind
		var currentVersion int64
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(parent_scope_id::text,''),code,name,kind,department_path,version
			FROM organization_scopes
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND status='ACTIVE' AND valid_from<=clock_timestamp()
			  AND (valid_until IS NULL OR clock_timestamp()<valid_until)
			FOR UPDATE`, tenantID, entityID, scopeID).
			Scan(&currentParent, &currentCode, &currentName, &currentKind, &currentPath, &currentVersion)
		if errors.Is(err, pgx.ErrNoRows) {
			return OrganizationScopeRevisionSummary{}, ErrAdminNotFound
		}
		if err != nil {
			return OrganizationScopeRevisionSummary{}, err
		}
		if currentVersion != baseVersion {
			return OrganizationScopeRevisionSummary{}, ErrAdminConflict
		}
		code = currentCode
		switch input.Operation {
		case OrganizationScopeUpdate:
			parentScopeID = currentParent
			if name == "" || !validManagedOrganizationScopeKind(kind) {
				return OrganizationScopeRevisionSummary{}, ErrAdminInvalid
			}
		case OrganizationScopeMove:
			name, kind = currentName, currentKind
			parentPath, err := organizationScopeParentPath(ctx, tx, tenantID, entityID, parentScopeID)
			if err != nil {
				return OrganizationScopeRevisionSummary{}, err
			}
			if parentScopeID == scopeID || hasOrganizationPathPrefix(parentPath, currentPath) {
				return OrganizationScopeRevisionSummary{}, ErrAdminConflict
			}
		case OrganizationScopeRetire:
			parentScopeID, name, kind = currentParent, currentName, currentKind
		}
	default:
		return OrganizationScopeRevisionSummary{}, ErrAdminInvalid
	}

	var revisionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_scope_revisions(
			tenant_id,legal_entity_id,scope_id,operation,base_version,
			proposed_parent_scope_id,proposed_code,proposed_name,proposed_kind,maker_id
		)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10::uuid)
		RETURNING id::text`,
		tenantID, entityID, scopeID, input.Operation, baseVersion,
		parentScopeID, code, name, string(kind), input.ActorID,
	).Scan(&revisionID)
	if err != nil {
		return OrganizationScopeRevisionSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_SCOPE_CHANGE_PROPOSED", "ORGANIZATION_SCOPE_REVISION", revisionID); err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}
	return a.organizationScopeRevisionByID(ctx, tenantID, entityID, revisionID)
}

func (a *PostgresAdministrator) ApproveOrganizationScope(ctx context.Context, input DecideOrganizationScopeInput) error {
	return a.decideOrganizationScope(ctx, input, true)
}

func (a *PostgresAdministrator) RejectOrganizationScope(ctx context.Context, input DecideOrganizationScopeInput) error {
	return a.decideOrganizationScope(ctx, input, false)
}

func (a *PostgresAdministrator) decideOrganizationScope(ctx context.Context, input DecideOrganizationScopeInput, approve bool) error {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.RevisionID = strings.TrimSpace(input.RevisionID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.TenantID == "" || input.LegalEntityID == "" || input.RevisionID == "" || input.ActorID == "" || input.Rationale == "" {
		return ErrAdminInvalid
	}
	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return err
	}

	var revision OrganizationScopeRevisionSummary
	err = tx.QueryRow(ctx, `
		SELECT id::text,scope_id::text,operation,base_version,COALESCE(proposed_parent_scope_id::text,''),
		       proposed_code,proposed_name,proposed_kind,maker_id::text,status,created_at
		FROM organization_scope_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		FOR UPDATE`, tenantID, entityID, input.RevisionID).
		Scan(&revision.ID, &revision.ScopeID, &revision.Operation, &revision.BaseVersion, &revision.ProposedParentScopeID,
			&revision.ProposedCode, &revision.ProposedName, &revision.ProposedKind, &revision.MakerID, &revision.Status, &revision.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdminNotFound
	}
	if err != nil {
		return err
	}
	if revision.Status != "PENDING" {
		return ErrAdminConflict
	}
	if revision.MakerID == input.ActorID {
		return ErrAdminMakerChecker
	}

	if !approve {
		tag, err := tx.Exec(ctx, `
			UPDATE organization_scope_revisions
			SET status='REJECTED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
			tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_SCOPE_CHANGE_REJECTED", "ORGANIZATION_SCOPE_REVISION", revision.ID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	if err := applyOrganizationScopeRevision(ctx, tx, tenantID, entityID, revision); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE organization_scope_revisions
		SET status='APPLIED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp(),applied_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
		tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	eventType := map[OrganizationScopeOperation]string{
		OrganizationScopeCreate: "ORGANIZATION_SCOPE_CREATED",
		OrganizationScopeUpdate: "ORGANIZATION_SCOPE_UPDATED",
		OrganizationScopeMove:   "ORGANIZATION_SCOPE_MOVED",
		OrganizationScopeRetire: "ORGANIZATION_SCOPE_RETIRED",
	}[revision.Operation]
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, eventType, "ORGANIZATION_SCOPE", revision.ScopeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func applyOrganizationScopeRevision(ctx context.Context, tx pgx.Tx, tenantID, entityID string, revision OrganizationScopeRevisionSummary) error {
	switch revision.Operation {
	case OrganizationScopeCreate:
		parentPath, err := organizationScopeParentPath(ctx, tx, tenantID, entityID, revision.ProposedParentScopeID)
		if err != nil {
			return err
		}
		path := append(append([]string(nil), parentPath...), revision.ProposedCode)
		_, err = tx.Exec(ctx, `
			INSERT INTO organization_scopes(
				id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
			)
			VALUES($1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8::text[],'MANAGED','ACTIVE',clock_timestamp())`,
			revision.ScopeID, tenantID, entityID, revision.ProposedParentScopeID,
			revision.ProposedCode, revision.ProposedName, string(revision.ProposedKind), path)
		return mapAdminPgError(err)
	case OrganizationScopeUpdate:
		tag, err := tx.Exec(ctx, `
			UPDATE organization_scopes
			SET name=$4,kind=$5,origin='MANAGED',version=version+1,updated_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND status='ACTIVE' AND version=$6`,
			tenantID, entityID, revision.ScopeID, revision.ProposedName, string(revision.ProposedKind), revision.BaseVersion)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		return nil
	case OrganizationScopeMove:
		var oldPath []string
		var version int64
		err := tx.QueryRow(ctx, `
			SELECT department_path,version
			FROM organization_scopes
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND status='ACTIVE'
			FOR UPDATE`, tenantID, entityID, revision.ScopeID).Scan(&oldPath, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdminNotFound
		}
		if err != nil {
			return err
		}
		if version != revision.BaseVersion || len(oldPath) == 0 {
			return ErrAdminConflict
		}
		parentPath, err := organizationScopeParentPath(ctx, tx, tenantID, entityID, revision.ProposedParentScopeID)
		if err != nil {
			return err
		}
		if revision.ProposedParentScopeID == revision.ScopeID || hasOrganizationPathPrefix(parentPath, oldPath) {
			return ErrAdminConflict
		}
		newPath := append(append([]string(nil), parentPath...), oldPath[len(oldPath)-1])
		_, err = tx.Exec(ctx, `
			UPDATE organization_scopes
			SET department_path=$4::text[] || department_path[$5:cardinality(department_path)],
			    version=version+1,updated_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
			  AND cardinality(department_path)>=$6
			  AND department_path[1:$6]=$7::text[]`,
			tenantID, entityID, revision.ScopeID, newPath, len(oldPath)+1, len(oldPath), oldPath)
		if err != nil {
			return mapAdminPgError(err)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE organization_scopes
			SET parent_scope_id=NULLIF($4,'')::uuid,origin='MANAGED'
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
			tenantID, entityID, revision.ScopeID, revision.ProposedParentScopeID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		if _, err := tx.Exec(ctx, `
			UPDATE org_positions p
			SET department_path=s.department_path
			FROM organization_scopes s
			WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid
			  AND p.organization_scope_id=s.id
			  AND s.tenant_id=p.tenant_id AND s.legal_entity_id=p.legal_entity_id
			  AND p.valid_until IS NULL
			  AND p.department_path IS DISTINCT FROM s.department_path`, tenantID, entityID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE directory_group_role_bindings b
			SET department_path=s.department_path
			FROM organization_scopes s
			WHERE b.tenant_id=$1::uuid AND b.legal_entity_id=$2::uuid
			  AND b.organization_scope_id=s.id
			  AND s.tenant_id=b.tenant_id AND s.legal_entity_id=b.legal_entity_id
			  AND b.valid_until IS NULL
			  AND b.department_path IS DISTINCT FROM s.department_path`, tenantID, entityID); err != nil {
			return err
		}
		return nil
	case OrganizationScopeRetire:
		impact, err := organizationScopeImpact(ctx, tx, tenantID, entityID, revision.ScopeID)
		if err != nil {
			return err
		}
		if impact.ChildScopes > 0 || impact.Positions > 0 || impact.AccessMappings > 0 || impact.OpenMatters > 0 {
			return ErrAdminConflict
		}
		tag, err := tx.Exec(ctx, `
			UPDATE organization_scopes
			SET status='RETIRED',valid_until=clock_timestamp(),origin='MANAGED',version=version+1,updated_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND status='ACTIVE' AND version=$4`,
			tenantID, entityID, revision.ScopeID, revision.BaseVersion)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		return nil
	default:
		return ErrAdminInvalid
	}
}

func (a *PostgresAdministrator) organizationScopeRevisions(ctx context.Context, tenantID, entityID string) ([]OrganizationScopeRevisionSummary, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT r.id::text,r.scope_id::text,r.operation,r.base_version,COALESCE(r.proposed_parent_scope_id::text,''),
		       r.proposed_code,r.proposed_name,r.proposed_kind,r.maker_id::text,COALESCE(r.checker_id::text,''),
		       r.status,r.rationale,r.created_at,r.decided_at,r.applied_at
		FROM organization_scope_revisions r
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.status='PENDING'
		ORDER BY r.created_at,r.id
		LIMIT 100`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]OrganizationScopeRevisionSummary, 0)
	for rows.Next() {
		var value OrganizationScopeRevisionSummary
		if err := rows.Scan(
			&value.ID, &value.ScopeID, &value.Operation, &value.BaseVersion, &value.ProposedParentScopeID,
			&value.ProposedCode, &value.ProposedName, &value.ProposedKind, &value.MakerID, &value.CheckerID,
			&value.Status, &value.Rationale, &value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
		); err != nil {
			return nil, err
		}
		value.Impact, err = organizationScopeImpact(ctx, a.pool, tenantID, entityID, value.ScopeID)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (a *PostgresAdministrator) organizationScopeRevisionByID(ctx context.Context, tenantID, entityID, revisionID string) (OrganizationScopeRevisionSummary, error) {
	var value OrganizationScopeRevisionSummary
	err := a.pool.QueryRow(ctx, `
		SELECT r.id::text,r.scope_id::text,r.operation,r.base_version,COALESCE(r.proposed_parent_scope_id::text,''),
		       r.proposed_code,r.proposed_name,r.proposed_kind,r.maker_id::text,COALESCE(r.checker_id::text,''),
		       r.status,r.rationale,r.created_at,r.decided_at,r.applied_at
		FROM organization_scope_revisions r
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`,
		tenantID, entityID, revisionID).
		Scan(&value.ID, &value.ScopeID, &value.Operation, &value.BaseVersion, &value.ProposedParentScopeID,
			&value.ProposedCode, &value.ProposedName, &value.ProposedKind, &value.MakerID, &value.CheckerID,
			&value.Status, &value.Rationale, &value.CreatedAt, &value.DecidedAt, &value.AppliedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationScopeRevisionSummary{}, ErrAdminNotFound
	}
	if err != nil {
		return OrganizationScopeRevisionSummary{}, err
	}
	value.Impact, err = organizationScopeImpact(ctx, a.pool, tenantID, entityID, value.ScopeID)
	return value, err
}

type organizationScopeQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func organizationScopeImpact(ctx context.Context, q organizationScopeQuerier, tenantID, entityID, scopeID string) (OrganizationScopeImpact, error) {
	var value OrganizationScopeImpact
	err := q.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM organization_scopes child
		   WHERE child.tenant_id=$1::uuid AND child.legal_entity_id=$2::uuid
		     AND child.parent_scope_id=$3::uuid AND child.status='ACTIVE' AND child.valid_until IS NULL),
		  (SELECT count(*) FROM org_positions p
		   WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid
		     AND p.organization_scope_id=$3::uuid AND p.valid_until IS NULL),
		  (SELECT count(*) FROM directory_group_role_bindings b
		   WHERE b.tenant_id=$1::uuid AND b.legal_entity_id=$2::uuid
		     AND b.organization_scope_id=$3::uuid AND b.valid_until IS NULL),
		  (SELECT count(*) FROM matters m
		   WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid
		     AND m.organization_scope_id=$3::uuid AND m.status<>'CLOSED')`,
		tenantID, entityID, scopeID).
		Scan(&value.ChildScopes, &value.Positions, &value.AccessMappings, &value.OpenMatters)
	return value, err
}

func organizationScopeParentPath(ctx context.Context, tx pgx.Tx, tenantID, entityID, parentScopeID string) ([]string, error) {
	parentScopeID = strings.TrimSpace(parentScopeID)
	if parentScopeID == "" {
		return []string{}, nil
	}
	var path []string
	err := tx.QueryRow(ctx, `
		SELECT department_path
		FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND status='ACTIVE' AND valid_from<=clock_timestamp()
		  AND (valid_until IS NULL OR clock_timestamp()<valid_until)`,
		tenantID, entityID, parentScopeID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminNotFound
	}
	return path, err
}

func normalizeOrganizationScopeCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return ""
		}
	}
	if len(value) > 80 {
		return ""
	}
	return value
}

func validManagedOrganizationScopeKind(kind organization.ScopeKind) bool {
	switch kind {
	case organization.ScopeKindBranch, organization.ScopeKindDepartment, organization.ScopeKindFunction,
		organization.ScopeKindBusinessUnit, organization.ScopeKindCriticalService:
		return true
	default:
		return false
	}
}

func hasOrganizationPathPrefix(candidate, prefix []string) bool {
	if len(prefix) == 0 || len(candidate) < len(prefix) {
		return false
	}
	for index := range prefix {
		if !strings.EqualFold(candidate[index], prefix[index]) {
			return false
		}
	}
	return true
}

func (a *PostgresAdministrator) scopeIDs(ctx context.Context, tenant, legalEntity string) (string, string, error) {
	var tenantID, entityID string
	if err := a.pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1`, tenant).Scan(&tenantID); errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrAdminNotFound
	} else if err != nil {
		return "", "", err
	}
	if err := a.pool.QueryRow(ctx, `SELECT id::text FROM legal_entities WHERE tenant_id=$1::uuid AND (id::text=$2 OR code=$2) AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until) LIMIT 1`, tenantID, legalEntity).Scan(&entityID); errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrAdminNotFound
	} else if err != nil {
		return "", "", err
	}
	return tenantID, entityID, nil
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

func recordAdminDecision(ctx context.Context, tx pgx.Tx, tenant, actorID, eventType, objectType, objectID string) error {
	fromState, toState := adminDecisionStates(eventType)
	_, err := tx.Exec(ctx, `
		INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_type,actor_id,rationale)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3::uuid,$4,$5,'PRINCIPAL',$6::uuid,$7)`,
		tenant, objectType, objectID, fromState, toState, actorID, eventType)
	return err
}

func adminDecisionStates(eventType string) (string, string) {
	switch eventType {
	case "SCIM_SOURCE_CREATED", "DIRECTORY_GROUP_ROLE_BOUND", "ORGANIZATION_SCOPE_CREATED", "ORGANIZATION_POSITION_CREATED":
		return "NONE", "ACTIVE"
	case "ORGANIZATION_SCOPE_CHANGE_PROPOSED", "ORGANIZATION_POSITION_CHANGE_PROPOSED",
		"ORGANIZATION_POSITION_ROLE_CHANGE_PROPOSED", "LEGAL_ENTITY_DATA_BOUNDARY_CHANGE_PROPOSED":
		return "NONE", "PENDING"
	case "ORGANIZATION_SCOPE_CHANGE_REJECTED", "ORGANIZATION_POSITION_CHANGE_REJECTED",
		"ORGANIZATION_POSITION_ROLE_CHANGE_REJECTED", "LEGAL_ENTITY_DATA_BOUNDARY_CHANGE_REJECTED":
		return "PENDING", "REJECTED"
	case "LEGAL_ENTITY_DATA_BOUNDARY_APPLIED":
		return "PENDING", "ACTIVE"
	case "SCIM_SOURCE_REVOKED":
		return "ACTIVE", "REVOKED"
	case "DIRECTORY_GROUP_ROLE_RETIRED", "ORGANIZATION_SCOPE_RETIRED", "ORGANIZATION_POSITION_RETIRED",
		"ORGANIZATION_POSITION_ROLE_REMOVED":
		return "ACTIVE", "RETIRED"
	case "ORGANIZATION_POSITION_ROLE_ADDED":
		return "NONE", "ACTIVE"
	default:
		return "ACTIVE", "ACTIVE"
	}
}

func mapAdminPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAdminConflict
	}
	return err
}

func closeRows(rows pgx.Rows) error {
	err := rows.Err()
	rows.Close()
	return err
}

var _ Administrator = (*PostgresAdministrator)(nil)
