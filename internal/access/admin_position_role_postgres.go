//go:build postgres

package access

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type workspaceRoleQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type workspaceRoleState struct {
	Summary RoleTemplateSummary
	Version int64
}

func (a *PostgresAdministrator) ProposeOrganizationPositionRole(ctx context.Context, input ProposeOrganizationPositionRoleInput) (OrganizationPositionRoleRevisionSummary, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.PositionID = strings.TrimSpace(input.PositionID)
	input.RoleTemplateID = strings.TrimSpace(input.RoleTemplateID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.PositionID == "" || input.RoleTemplateID == "" ||
		input.ExpectedPositionVersion < 1 || input.ActorID == "" ||
		(input.Operation != OrganizationPositionRoleAdd && input.Operation != OrganizationPositionRoleRetire) {
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

	positionVersion, err := activeOrganizationPositionVersion(ctx, tx, tenantID, entityID, input.PositionID, true)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	if positionVersion != input.ExpectedPositionVersion {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
	}
	role, err := organizationWorkspaceRole(ctx, tx, tenantID, input.RoleTemplateID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	if !role.Summary.WorkspaceEditable {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminInvalid
	}
	bindingID, bindingCount, err := activePositionRoleBinding(ctx, tx, tenantID, input.PositionID, input.RoleTemplateID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, err
	}
	switch input.Operation {
	case OrganizationPositionRoleAdd:
		if bindingCount != 0 {
			return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
		}
	case OrganizationPositionRoleRetire:
		if bindingCount != 1 || bindingID == "" {
			return OrganizationPositionRoleRevisionSummary{}, ErrAdminConflict
		}
	}

	var revisionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_position_role_revisions(
			tenant_id,legal_entity_id,position_id,role_template_id,role_template_version,
			operation,base_position_version,role_code,role_name,capabilities,maker_id
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11::uuid
		)
		RETURNING id::text`,
		tenantID, entityID, input.PositionID, input.RoleTemplateID, role.Version,
		input.Operation, positionVersion, role.Summary.Code, role.Summary.Name, role.Summary.Capabilities, input.ActorID,
	).Scan(&revisionID)
	if err != nil {
		return OrganizationPositionRoleRevisionSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID,
		"ORGANIZATION_POSITION_ROLE_CHANGE_PROPOSED", "ORGANIZATION_POSITION_ROLE_REVISION", revisionID); err != nil {
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
		if _, err := tx.Exec(ctx, `
			UPDATE organization_position_role_revisions
			SET status='REJECTED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
			tenantID, entityID, input.RevisionID, input.ActorID, input.Rationale); err != nil {
			return err
		}
		if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID,
			"ORGANIZATION_POSITION_ROLE_CHANGE_REJECTED", "ORGANIZATION_POSITION_ROLE_REVISION", input.RevisionID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	positionVersion, err := activeOrganizationPositionVersion(ctx, tx, tenantID, entityID, revision.PositionID, true)
	if err != nil {
		return err
	}
	if positionVersion != revision.BasePositionVersion {
		return ErrAdminConflict
	}
	role, err := organizationWorkspaceRole(ctx, tx, tenantID, revision.RoleTemplateID)
	if err != nil {
		return err
	}
	if role.Version != revision.RoleTemplateVersion || !role.Summary.WorkspaceEditable {
		return ErrAdminConflict
	}
	bindingID, bindingCount, err := activePositionRoleBinding(ctx, tx, tenantID, revision.PositionID, revision.RoleTemplateID)
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
			RETURNING id::text`, tenantID, revision.PositionID, revision.RoleTemplateID, entityID).Scan(&bindingID); err != nil {
			return mapAdminPgError(err)
		}
	case OrganizationPositionRoleRetire:
		if bindingCount != 1 || bindingID == "" {
			return ErrAdminConflict
		}
		tag, err := tx.Exec(ctx, `
			UPDATE position_role_bindings
			SET valid_until=clock_timestamp()
			WHERE tenant_id=$1::uuid AND id=$2::uuid
			  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`,
			tenantID, bindingID)
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
	if _, err := tx.Exec(ctx, `
		UPDATE organization_position_role_revisions
		SET status='APPLIED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp(),applied_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
		tenantID, entityID, input.RevisionID, input.ActorID, input.Rationale); err != nil {
		return err
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID,
		"ORGANIZATION_POSITION_ROLE_CHANGE_APPLIED", "ORGANIZATION_POSITION_ROLE_REVISION", input.RevisionID); err != nil {
		return err
	}
	eventType := "ORGANIZATION_POSITION_ROLE_BOUND"
	if revision.Operation == OrganizationPositionRoleRetire {
		eventType = "ORGANIZATION_POSITION_ROLE_RETIRED"
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID,
		eventType, "ORGANIZATION_POSITION_ROLE_BINDING", bindingID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func activeOrganizationPositionVersion(ctx context.Context, q workspaceRoleQuerier, tenantID, entityID, positionID string, lock bool) (int64, error) {
	query := `
		SELECT version
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`
	if lock {
		query += " FOR UPDATE"
	}
	var version int64
	if err := q.QueryRow(ctx, query, tenantID, entityID, positionID).Scan(&version); errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAdminNotFound
	} else if err != nil {
		return 0, err
	}
	return version, nil
}

func organizationWorkspaceRole(ctx context.Context, q workspaceRoleQuerier, tenantID, roleTemplateID string) (workspaceRoleState, error) {
	var (
		value            workspaceRoleState
		declared         int
		assignments      int
		authorityGrants  int
		routingPolicies  int
		segregationRules int
	)
	err := q.QueryRow(ctx, `
		SELECT rt.id::text,rt.code,rt.name,rt.capabilities,rt.version,
		       cardinality(rt.responsibilities),
		       (SELECT count(*) FROM responsibility_assignments assignment
		        WHERE assignment.tenant_id=rt.tenant_id AND assignment.role_template_id=rt.id
		          AND assignment.valid_from<=clock_timestamp()
		          AND (assignment.valid_until IS NULL OR clock_timestamp()<assignment.valid_until)),
		       (SELECT count(*) FROM authority_grants grant_row
		        WHERE grant_row.tenant_id=rt.tenant_id AND grant_row.role_template_id=rt.id
		          AND grant_row.valid_from<=clock_timestamp()
		          AND (grant_row.valid_until IS NULL OR clock_timestamp()<grant_row.valid_until)),
		       (SELECT count(*) FROM routing_policies policy
		        JOIN routing_policy_versions version
		          ON version.policy_id=policy.id AND version.version=policy.current_version
		        WHERE policy.tenant_id=rt.tenant_id AND policy.status='ACTIVE'
		          AND (
		            strpos(version.definition::text,to_jsonb(rt.code)::text)>0
		            OR strpos(version.definition::text,to_jsonb(rt.id::text)::text)>0
		          )),
		       (SELECT count(*) FROM segregation_rules rule
		        WHERE rule.tenant_id=rt.tenant_id AND rule.status='ACTIVE'
		          AND rule.prohibited_role_code=rt.code
		          AND rule.valid_from<=clock_timestamp()
		          AND (rule.valid_until IS NULL OR clock_timestamp()<rule.valid_until))
		FROM role_templates rt
		WHERE rt.tenant_id=$1::uuid AND rt.id=$2::uuid
		  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)`,
		tenantID, roleTemplateID,
	).Scan(
		&value.Summary.ID, &value.Summary.Code, &value.Summary.Name, &value.Summary.Capabilities, &value.Version,
		&declared, &assignments, &authorityGrants, &routingPolicies, &segregationRules,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return workspaceRoleState{}, ErrAdminNotFound
	}
	if err != nil {
		return workspaceRoleState{}, err
	}
	value.Summary.MaterialReferences = WorkspaceRoleMaterialReferences{
		DeclaredResponsibilities: declared, ResponsibilityAssignments: assignments,
		AuthorityGrants: authorityGrants, RoutingPolicies: routingPolicies, SegregationRules: segregationRules,
	}
	value.Summary.WorkspaceLockReason = workspaceRoleLockReason(value.Summary)
	value.Summary.WorkspaceEditable = value.Summary.WorkspaceLockReason == ""
	return value, nil
}

func workspaceRoleLockReason(role RoleTemplateSummary) string {
	refs := role.MaterialReferences
	switch {
	case len(role.Capabilities) == 0:
		return "No workspace capabilities"
	case refs.DeclaredResponsibilities > 0:
		return "Defines business responsibilities"
	case refs.ResponsibilityAssignments > 0:
		return "Used by responsibility routing"
	case refs.AuthorityGrants > 0:
		return "Used by decision authority"
	case refs.RoutingPolicies > 0:
		return "Used by routing or escalation"
	case refs.SegregationRules > 0:
		return "Used by segregation rules"
	default:
		return ""
	}
}

func activePositionRoleBinding(ctx context.Context, q workspaceRoleQuerier, tenantID, positionID, roleTemplateID string) (string, int, error) {
	var ids []string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(array_agg(binding.id::text ORDER BY binding.id),ARRAY[]::text[])
		FROM position_role_bindings binding
		WHERE binding.tenant_id=$1::uuid AND binding.position_id=$2::uuid AND binding.role_template_id=$3::uuid
		  AND binding.valid_from<=clock_timestamp() AND (binding.valid_until IS NULL OR clock_timestamp()<binding.valid_until)`,
		tenantID, positionID, roleTemplateID).Scan(&ids)
	if err != nil {
		return "", 0, err
	}
	if len(ids) == 0 {
		return "", 0, nil
	}
	return ids[0], len(ids), nil
}

const organizationPositionRoleRevisionSelect = `
	SELECT r.id::text,r.position_id::text,r.role_template_id::text,r.role_template_version,
	       r.operation,r.base_position_version,r.role_code,r.role_name,r.capabilities,
	       r.maker_id::text,COALESCE(r.checker_id::text,''),r.status,r.rationale,
	       r.created_at,r.decided_at,r.applied_at
	FROM organization_position_role_revisions r`

func organizationPositionRoleRevision(ctx context.Context, q workspaceRoleQuerier, tenantID, entityID, revisionID string, lock bool) (OrganizationPositionRoleRevisionSummary, error) {
	query := organizationPositionRoleRevisionSelect + `
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`
	if lock {
		query += " FOR UPDATE"
	}
	var value OrganizationPositionRoleRevisionSummary
	err := q.QueryRow(ctx, query, tenantID, entityID, revisionID).Scan(
		&value.ID, &value.PositionID, &value.RoleTemplateID, &value.RoleTemplateVersion,
		&value.Operation, &value.BasePositionVersion, &value.RoleCode, &value.RoleName, &value.Capabilities,
		&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale,
		&value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationPositionRoleRevisionSummary{}, ErrAdminNotFound
	}
	return value, err
}

func (a *PostgresAdministrator) organizationPositionRoleRevisionByID(ctx context.Context, tenantID, entityID, revisionID string) (OrganizationPositionRoleRevisionSummary, error) {
	return organizationPositionRoleRevision(ctx, a.pool, tenantID, entityID, revisionID, false)
}

func (a *PostgresAdministrator) organizationPositionRoleRevisions(ctx context.Context, tenantID, entityID string) ([]OrganizationPositionRoleRevisionSummary, error) {
	rows, err := a.pool.Query(ctx, organizationPositionRoleRevisionSelect+`
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.status='PENDING'
		ORDER BY r.created_at,r.id
		LIMIT 100`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]OrganizationPositionRoleRevisionSummary, 0)
	for rows.Next() {
		var value OrganizationPositionRoleRevisionSummary
		if err := rows.Scan(
			&value.ID, &value.PositionID, &value.RoleTemplateID, &value.RoleTemplateVersion,
			&value.Operation, &value.BasePositionVersion, &value.RoleCode, &value.RoleName, &value.Capabilities,
			&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale,
			&value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
		); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
