//go:build postgres

package access

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type organizationPositionRoleScanner interface {
	Scan(...any) error
}

const organizationRoleTemplateSelect = `
	WITH current_entity AS (
		SELECT id,code
		FROM legal_entities
		WHERE tenant_id=$1::uuid AND id=$2::uuid
		  AND valid_from<=clock_timestamp()
		  AND (valid_until IS NULL OR clock_timestamp()<valid_until)
	)
	SELECT rt.id::text,rt.code,rt.name,rt.capabilities,rt.version,
	       array_remove(ARRAY[
	         CASE WHEN EXISTS (
	           SELECT 1
	           FROM responsibility_assignments assignment
	           WHERE assignment.tenant_id=rt.tenant_id
	             AND assignment.role_template_id=rt.id
	             AND (assignment.legal_entity_id IS NULL OR assignment.legal_entity_id=$2::uuid)
	             AND assignment.valid_from<=clock_timestamp()
	             AND (assignment.valid_until IS NULL OR clock_timestamp()<assignment.valid_until)
	         ) THEN 'RESPONSIBILITY_ASSIGNMENT' END,
	         CASE WHEN EXISTS (
	           SELECT 1
	           FROM authority_grants grant_row
	           WHERE grant_row.tenant_id=rt.tenant_id
	             AND grant_row.role_template_id=rt.id
	             AND (grant_row.legal_entity_id IS NULL OR grant_row.legal_entity_id=$2::uuid)
	             AND grant_row.valid_from<=clock_timestamp()
	             AND (grant_row.valid_until IS NULL OR clock_timestamp()<grant_row.valid_until)
	         ) THEN 'AUTHORITY_GRANT' END,
	         CASE WHEN EXISTS (
	           SELECT 1
	           FROM effective_authority_routes route
	           CROSS JOIN current_entity entity
	           WHERE route.tenant_id=rt.tenant_id
	             AND route.selector_kind IN ('ROLE','ROLE_ID')
	             AND route.selector_ref IN (rt.code,rt.id::text)
	             AND route.legal_entity_ref IN ('*',entity.id::text,entity.code)
	             AND route.valid_from<=clock_timestamp()
	             AND (route.valid_until IS NULL OR clock_timestamp()<route.valid_until)
	         ) THEN 'AUTHORITY_ROUTE' END,
	         CASE WHEN EXISTS (
	           SELECT 1
	           FROM segregation_rules rule
	           WHERE rule.tenant_id=rt.tenant_id
	             AND rule.prohibited_role_code=rt.code
	             AND rule.status='ACTIVE'
	             AND rule.valid_from<=clock_timestamp()
	             AND (rule.valid_until IS NULL OR clock_timestamp()<rule.valid_until)
	         ) THEN 'SEGREGATION_RULE' END,
	         CASE WHEN EXISTS (
	           SELECT 1
	           FROM routing_policies policy
	           JOIN routing_policy_versions version
	             ON version.policy_id=policy.id AND version.version=policy.current_version
	           CROSS JOIN LATERAL jsonb_array_elements(COALESCE(version.definition->'escalations','[]'::jsonb)) escalation(value)
	           CROSS JOIN LATERAL jsonb_array_elements(COALESCE(escalation.value->'steps','[]'::jsonb)) step(value)
	           WHERE policy.tenant_id=rt.tenant_id
	             AND policy.status='ACTIVE'
	             AND (policy.legal_entity_id IS NULL OR policy.legal_entity_id=$2::uuid)
	             AND (version.effective_from IS NULL OR version.effective_from<=clock_timestamp())
	             AND (version.effective_until IS NULL OR clock_timestamp()<version.effective_until)
	             AND (
	               COALESCE(step.value->'source_roles','[]'::jsonb) ? rt.code
	               OR COALESCE(step.value->'targets'->'roles','[]'::jsonb) ? rt.code
	             )
	         ) THEN 'ESCALATION_ROUTE' END
	       ],NULL)::text[] AS lock_reasons
	FROM role_templates rt
	WHERE rt.tenant_id=$1::uuid
	  AND rt.valid_from<=clock_timestamp()
	  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
`

func (a *PostgresAdministrator) organizationRoleTemplates(ctx context.Context, tenantID, entityID string) ([]RoleTemplateSummary, error) {
	rows, err := a.pool.Query(ctx, organizationRoleTemplateSelect+`
		ORDER BY rt.code
		LIMIT 100`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]RoleTemplateSummary, 0)
	for rows.Next() {
		value, scanErr := scanOrganizationRoleTemplate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func organizationRoleTemplate(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, roleID string, forUpdate bool) (RoleTemplateSummary, error) {
	query := organizationRoleTemplateSelect + `
		AND rt.id=$3::uuid
		LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE OF rt`
	}
	value, err := scanOrganizationRoleTemplate(q.QueryRow(ctx, query, tenantID, entityID, roleID))
	if errors.Is(err, pgx.ErrNoRows) {
		return RoleTemplateSummary{}, ErrAdminNotFound
	}
	return value, err
}

func scanOrganizationRoleTemplate(row organizationPositionRoleScanner) (RoleTemplateSummary, error) {
	var value RoleTemplateSummary
	if err := row.Scan(&value.ID, &value.Code, &value.Name, &value.Capabilities, &value.Version, &value.OrganizationLockReasons); err != nil {
		return RoleTemplateSummary{}, err
	}
	value.OrganizationEditable = len(value.OrganizationLockReasons) == 0
	return value, nil
}

func (a *PostgresAdministrator) ProposeOrganizationPositionRole(ctx context.Context, input ProposeOrganizationPositionRoleInput) (OrganizationPositionRoleRevisionSummary, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.PositionID = strings.TrimSpace(input.PositionID)
	input.RoleTemplateID = strings.TrimSpace(input.RoleTemplateID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.PositionID == "" || input.RoleTemplateID == "" ||
		input.ActorID == "" || input.ExpectedPositionVersion < 1 {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminInvalid
	}
	if input.Operation != OrganizationPositionRoleAdd && input.Operation != OrganizationPositionRoleRemove {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminInvalid
	}
	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	_, version, err := organizationPositionState(ctx, tx, tenantID, entityID, input.PositionID, true)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	if version != input.ExpectedPositionVersion {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
	}
	role, err := organizationRoleTemplate(ctx, tx, tenantID, entityID, input.RoleTemplateID, true)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	if !role.OrganizationEditable {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
	}
	bindingID, bindingCount, err := organizationPositionRoleBinding(ctx, tx, tenantID, entityID, input.PositionID, input.RoleTemplateID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	switch input.Operation {
	case OrganizationPositionRoleAdd:
		if bindingCount != 0 {
			return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
		}
		bindingID = ""
	case OrganizationPositionRoleRemove:
		if bindingCount != 1 || bindingID == "" {
			return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
		}
	}

	var revisionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_position_role_revisions(
			tenant_id,legal_entity_id,position_id,role_template_id,operation,
			base_position_version,base_role_version,base_binding_id,
			role_code,role_name,capabilities,maker_id
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,
			$6,$7,NULLIF($8,'')::uuid,
			$9,$10,$11,$12::uuid
		)
		RETURNING id::text`,
		tenantID, entityID, input.PositionID, role.ID, input.Operation,
		version, role.Version, bindingID, role.Code, role.Name, role.Capabilities, input.ActorID,
	).Scan(&revisionID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_POSITION_ROLE_CHANGE_PROPOSED", "ORGANIZATION_POSITION_ROLE_REVISION", revisionID); err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	return a.organizationPositionRoleRevisionByID(ctx, tenantID, entityID, revisionID)
}

func (a *PostgresAdministrator) ApproveOrganizationPositionRole(ctx context.Context, input DecideOrganizationPositionRoleInput) error {
	return a.decideOrganizationPositionRole(ctx, input, true)
}

func (a *PostgresAdministrator) RejectOrganizationPositionRole(ctx context.Context, input DecideOrganizationPositionRoleInput) error {
	return a.decideOrganizationPositionRole(ctx, input, false)
}

func (a *PostgresAdministrator) decideOrganizationPositionRole(ctx context.Context, input DecideOrganizationPositionRoleInput, approve bool) error {
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
	revision, err := organizationPositionRoleRevision(ctx, tx, tenantID, entityID, input.RevisionID, true)
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
			UPDATE organization_position_role_revisions
			SET status='REJECTED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
			tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_POSITION_ROLE_CHANGE_REJECTED", "ORGANIZATION_POSITION_ROLE_REVISION", revision.ID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	_, positionVersion, err := organizationPositionState(ctx, tx, tenantID, entityID, revision.PositionID, true)
	if err != nil {
		return err
	}
	if positionVersion != revision.BasePositionVersion {
		return ErrAdminConflict
	}
	role, err := organizationRoleTemplate(ctx, tx, tenantID, entityID, revision.RoleTemplateID, true)
	if err != nil {
		return err
	}
	if role.Version != revision.BaseRoleVersion || role.Code != revision.RoleCode || !role.OrganizationEditable {
		return ErrAdminConflict
	}
	bindingID, bindingCount, err := organizationPositionRoleBinding(ctx, tx, tenantID, entityID, revision.PositionID, revision.RoleTemplateID)
	if err != nil {
		return err
	}
	switch revision.Operation {
	case OrganizationPositionRoleAdd:
		if bindingCount != 0 {
			return ErrAdminConflict
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO position_role_bindings(
				tenant_id,position_id,role_template_id,scope,priority,valid_from
			) VALUES(
				$1::uuid,$2::uuid,$3::uuid,jsonb_build_object('legal_entity_id',$4::text),0,clock_timestamp()
			)
			RETURNING id::text`,
			tenantID, revision.PositionID, revision.RoleTemplateID, entityID).Scan(&bindingID); err != nil {
			return mapAdminPgError(err)
		}
	case OrganizationPositionRoleRemove:
		if bindingCount != 1 || bindingID == "" || bindingID != revision.BaseBindingID {
			return ErrAdminConflict
		}
		tag, err := tx.Exec(ctx, `
			UPDATE position_role_bindings
			SET valid_until=clock_timestamp()
			WHERE tenant_id=$1::uuid AND id=$2::uuid AND position_id=$3::uuid AND role_template_id=$4::uuid
			  AND valid_until IS NULL`,
			tenantID, bindingID, revision.PositionID, revision.RoleTemplateID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
	default:
		return ErrAdminInvalid
	}
	tag, err := tx.Exec(ctx, `
		UPDATE org_positions
		SET version=version+1,recorded_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND valid_until IS NULL AND version=$4`,
		tenantID, entityID, revision.PositionID, revision.BasePositionVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	tag, err = tx.Exec(ctx, `
		UPDATE organization_position_role_revisions
		SET status='APPLIED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp(),applied_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
		tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	eventType := "ORGANIZATION_POSITION_ROLE_ADDED"
	if revision.Operation == OrganizationPositionRoleRemove {
		eventType = "ORGANIZATION_POSITION_ROLE_REMOVED"
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, eventType, "ORGANIZATION_POSITION_ROLE_BINDING", bindingID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func organizationPositionRoleBinding(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, positionID, roleID string) (string, int, error) {
	var bindingID string
	var count int
	err := q.QueryRow(ctx, `
		SELECT COALESCE(min(binding.id::text),''),count(*)
		FROM position_role_bindings binding
		WHERE binding.tenant_id=$1::uuid
		  AND binding.position_id=$2::uuid
		  AND binding.role_template_id=$3::uuid
		  AND binding.valid_from<=clock_timestamp()
		  AND (binding.valid_until IS NULL OR clock_timestamp()<binding.valid_until)
		  AND (
		    NOT (binding.scope ? 'legal_entity_id')
		    OR binding.scope->>'legal_entity_id' IN ('*',$4)
		  )`, tenantID, positionID, roleID, entityID).Scan(&bindingID, &count)
	return bindingID, count, err
}

const organizationPositionRoleRevisionSelect = `
	SELECT revision.id::text,revision.position_id::text,revision.role_template_id::text,revision.operation,
	       revision.base_position_version,revision.base_role_version,COALESCE(revision.base_binding_id::text,''),
	       revision.role_code,revision.role_name,revision.capabilities,
	       revision.maker_id::text,COALESCE(revision.checker_id::text,''),revision.status,revision.rationale,
	       revision.created_at,revision.decided_at,revision.applied_at
	FROM organization_position_role_revisions revision
`

func (a *PostgresAdministrator) organizationPositionRoleRevisions(ctx context.Context, tenantID, entityID string) ([]OrganizationPositionRoleRevisionSummary, error) {
	rows, err := a.pool.Query(ctx, organizationPositionRoleRevisionSelect+`
		WHERE revision.tenant_id=$1::uuid AND revision.legal_entity_id=$2::uuid AND revision.status='PENDING'
		ORDER BY revision.created_at,revision.id
		LIMIT 100`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]OrganizationPositionRoleRevisionSummary, 0)
	for rows.Next() {
		value, scanErr := scanOrganizationPositionRoleRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (a *PostgresAdministrator) organizationPositionRoleRevisionByID(ctx context.Context, tenantID, entityID, revisionID string) (OrganizationPositionRoleRevisionSummary, error) {
	value, err := scanOrganizationPositionRoleRevision(a.pool.QueryRow(ctx, organizationPositionRoleRevisionSelect+`
		WHERE revision.tenant_id=$1::uuid AND revision.legal_entity_id=$2::uuid AND revision.id=$3::uuid`,
		tenantID, entityID, revisionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminNotFound
	}
	return value, err
}

func organizationPositionRoleRevision(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, revisionID string, forUpdate bool) (OrganizationPositionRoleRevisionSummary, error) {
	query := organizationPositionRoleRevisionSelect + `
		WHERE revision.tenant_id=$1::uuid AND revision.legal_entity_id=$2::uuid AND revision.id=$3::uuid`
	if forUpdate {
		query += ` FOR UPDATE OF revision`
	}
	value, err := scanOrganizationPositionRoleRevision(q.QueryRow(ctx, query, tenantID, entityID, revisionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminNotFound
	}
	return value, err
}

func scanOrganizationPositionRoleRevision(row organizationPositionRoleScanner) (OrganizationPositionRoleRevisionSummary, error) {
	var value OrganizationPositionRoleRevisionSummary
	if err := row.Scan(
		&value.ID, &value.PositionID, &value.RoleTemplateID, &value.Operation,
		&value.BasePositionVersion, &value.BaseRoleVersion, &value.BaseBindingID,
		&value.RoleCode, &value.RoleName, &value.Capabilities,
		&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale,
		&value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
	); err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	return value, nil
}
