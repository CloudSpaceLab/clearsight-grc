//go:build postgres

package access

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type organizationPositionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (a *PostgresAdministrator) ProposeOrganizationPosition(ctx context.Context, input ProposeOrganizationPositionInput) (OrganizationPositionRevisionSummary, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.PositionID = strings.TrimSpace(input.PositionID)
	input.Code = normalizeOrganizationPositionCode(input.Code)
	input.Title = strings.TrimSpace(input.Title)
	input.FunctionName = strings.TrimSpace(input.FunctionName)
	input.OrganizationScopeID = strings.TrimSpace(input.OrganizationScopeID)
	input.ParentPositionID = strings.TrimSpace(input.ParentPositionID)
	input.OccupantPrincipalID = strings.TrimSpace(input.OccupantPrincipalID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.ActorID == "" {
		return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
	}

	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}

	var (
		positionID  string
		baseVersion int64
		base        OrganizationPositionState
		proposed    OrganizationPositionState
	)
	switch input.Operation {
	case OrganizationPositionCreate:
		if input.PositionID != "" || input.ExpectedVersion != 0 || input.Code == "" || input.Title == "" {
			return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
		}
		if err := tx.QueryRow(ctx, `SELECT uuidv7()::text`).Scan(&positionID); err != nil {
			return OrganizationPositionRevisionSummary{}, err
		}
		proposed = OrganizationPositionState{
			Code: input.Code, Title: input.Title, FunctionName: input.FunctionName,
			OrganizationScopeID: input.OrganizationScopeID, ParentPositionID: input.ParentPositionID,
			OccupantPrincipalID: input.OccupantPrincipalID,
		}
	case OrganizationPositionUpdate, OrganizationPositionRetire:
		if input.PositionID == "" || input.ExpectedVersion < 1 {
			return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
		}
		positionID = input.PositionID
		base, baseVersion, err = organizationPositionState(ctx, tx, tenantID, entityID, positionID, true)
		if err != nil {
			return OrganizationPositionRevisionSummary{}, err
		}
		if baseVersion != input.ExpectedVersion {
			return OrganizationPositionRevisionSummary{}, ErrAdminConflict
		}
		if input.Operation == OrganizationPositionRetire {
			proposed = base
		} else {
			if input.Title == "" {
				return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
			}
			proposed = OrganizationPositionState{
				Code: base.Code, Title: input.Title, FunctionName: input.FunctionName,
				OrganizationScopeID: input.OrganizationScopeID, ParentPositionID: input.ParentPositionID,
				OccupantPrincipalID: input.OccupantPrincipalID,
			}
		}
	default:
		return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
	}
	if len(proposed.Title) > 240 || len(proposed.FunctionName) > 240 {
		return OrganizationPositionRevisionSummary{}, ErrAdminInvalid
	}
	if err := validateOrganizationPositionState(ctx, tx, tenantID, entityID, positionID, proposed); err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}

	var revisionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_position_revisions(
			tenant_id,legal_entity_id,position_id,operation,base_version,
			base_code,base_title,base_function_name,base_organization_scope_id,base_parent_position_id,base_occupant_principal_id,
			proposed_code,proposed_title,proposed_function_name,proposed_organization_scope_id,proposed_parent_position_id,proposed_occupant_principal_id,
			maker_id
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4,$5,
			$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,NULLIF($11,'')::uuid,
			$12,$13,$14,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,NULLIF($17,'')::uuid,
			$18::uuid
		)
		RETURNING id::text`,
		tenantID, entityID, positionID, input.Operation, baseVersion,
		base.Code, base.Title, base.FunctionName, base.OrganizationScopeID, base.ParentPositionID, base.OccupantPrincipalID,
		proposed.Code, proposed.Title, proposed.FunctionName, proposed.OrganizationScopeID, proposed.ParentPositionID, proposed.OccupantPrincipalID,
		input.ActorID,
	).Scan(&revisionID)
	if err != nil {
		return OrganizationPositionRevisionSummary{}, mapAdminPgError(err)
	}
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_POSITION_CHANGE_PROPOSED", "ORGANIZATION_POSITION_REVISION", revisionID); err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}
	return a.organizationPositionRevisionByID(ctx, tenantID, entityID, revisionID)
}

func (a *PostgresAdministrator) ApproveOrganizationPosition(ctx context.Context, input DecideOrganizationPositionInput) error {
	return a.decideOrganizationPosition(ctx, input, true)
}

func (a *PostgresAdministrator) RejectOrganizationPosition(ctx context.Context, input DecideOrganizationPositionInput) error {
	return a.decideOrganizationPosition(ctx, input, false)
}

func (a *PostgresAdministrator) decideOrganizationPosition(ctx context.Context, input DecideOrganizationPositionInput, approve bool) error {
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

	revision, err := organizationPositionRevision(ctx, tx, tenantID, entityID, input.RevisionID, true)
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
			UPDATE organization_position_revisions
			SET status='REJECTED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
			tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, "ORGANIZATION_POSITION_CHANGE_REJECTED", "ORGANIZATION_POSITION_REVISION", revision.ID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	if err := applyOrganizationPositionRevision(ctx, tx, tenantID, entityID, revision); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE organization_position_revisions
		SET status='APPLIED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp(),applied_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
		tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	eventType := map[OrganizationPositionOperation]string{
		OrganizationPositionCreate: "ORGANIZATION_POSITION_CREATED",
		OrganizationPositionUpdate: "ORGANIZATION_POSITION_UPDATED",
		OrganizationPositionRetire: "ORGANIZATION_POSITION_RETIRED",
	}[revision.Operation]
	if err := recordAdminDecision(ctx, tx, input.TenantID, input.ActorID, eventType, "ORGANIZATION_POSITION", revision.PositionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func applyOrganizationPositionRevision(ctx context.Context, tx pgx.Tx, tenantID, entityID string, revision OrganizationPositionRevisionSummary) error {
	switch revision.Operation {
	case OrganizationPositionCreate:
		if err := validateOrganizationPositionState(ctx, tx, tenantID, entityID, revision.PositionID, revision.Proposed); err != nil {
			return err
		}
		departmentPath, err := organizationPositionDepartmentPath(ctx, tx, tenantID, entityID, revision.Proposed.OrganizationScopeID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO org_positions(
				id,tenant_id,legal_entity_id,code,title,function_name,organization_scope_id,
				parent_position_id,occupant_principal_id,department_path,valid_from,version
			) VALUES(
				$1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,''),
				NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,$10,clock_timestamp(),1
			)`,
			revision.PositionID, tenantID, entityID, revision.Proposed.Code, revision.Proposed.Title,
			revision.Proposed.FunctionName, revision.Proposed.OrganizationScopeID,
			revision.Proposed.ParentPositionID, revision.Proposed.OccupantPrincipalID, departmentPath)
		return mapOrganizationPositionPgError(err)
	case OrganizationPositionUpdate:
		current, version, err := organizationPositionState(ctx, tx, tenantID, entityID, revision.PositionID, true)
		if err != nil {
			return err
		}
		if version != revision.BaseVersion || current != revision.Base {
			return ErrAdminConflict
		}
		if err := validateOrganizationPositionState(ctx, tx, tenantID, entityID, revision.PositionID, revision.Proposed); err != nil {
			return err
		}
		departmentPath, err := organizationPositionDepartmentPath(ctx, tx, tenantID, entityID, revision.Proposed.OrganizationScopeID)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE org_positions
			SET title=$4,function_name=NULLIF($5,''),organization_scope_id=NULLIF($6,'')::uuid,
			    parent_position_id=NULLIF($7,'')::uuid,occupant_principal_id=NULLIF($8,'')::uuid,
			    department_path=$9,version=version+1,recorded_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND valid_until IS NULL AND version=$10`,
			tenantID, entityID, revision.PositionID, revision.Proposed.Title, revision.Proposed.FunctionName,
			revision.Proposed.OrganizationScopeID, revision.Proposed.ParentPositionID, revision.Proposed.OccupantPrincipalID,
			departmentPath, revision.BaseVersion)
		if err != nil {
			return mapOrganizationPositionPgError(err)
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		return nil
	case OrganizationPositionRetire:
		current, version, err := organizationPositionState(ctx, tx, tenantID, entityID, revision.PositionID, true)
		if err != nil {
			return err
		}
		if version != revision.BaseVersion || current != revision.Base {
			return ErrAdminConflict
		}
		impact, err := organizationPositionImpact(ctx, tx, tenantID, entityID, revision.PositionID)
		if err != nil {
			return err
		}
		if impact.ChildPositions > 0 || impact.ResponsibilityAssignments > 0 || impact.AuthorityGrants > 0 {
			return ErrAdminConflict
		}
		var retiredAt string
		if err := tx.QueryRow(ctx, `SELECT clock_timestamp()::text`).Scan(&retiredAt); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE org_positions
			SET valid_until=$4::timestamptz,version=version+1,recorded_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
			  AND valid_until IS NULL AND version=$5`,
			tenantID, entityID, revision.PositionID, retiredAt, revision.BaseVersion)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		if _, err := tx.Exec(ctx, `
			UPDATE position_role_bindings
			SET valid_until=$3::timestamptz
			WHERE tenant_id=$1::uuid AND position_id=$2::uuid
			  AND valid_until IS NULL`, tenantID, revision.PositionID, retiredAt); err != nil {
			return err
		}
		return nil
	default:
		return ErrAdminInvalid
	}
}

func (a *PostgresAdministrator) organizationPositionRevisions(ctx context.Context, tenantID, entityID string) ([]OrganizationPositionRevisionSummary, error) {
	rows, err := a.pool.Query(ctx, organizationPositionRevisionSelect+`
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.status='PENDING'
		ORDER BY r.created_at,r.id
		LIMIT 100`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]OrganizationPositionRevisionSummary, 0)
	for rows.Next() {
		value, scanErr := scanOrganizationPositionRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		value.Impact, err = organizationPositionImpact(ctx, a.pool, tenantID, entityID, value.PositionID)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (a *PostgresAdministrator) organizationPositionRevisionByID(ctx context.Context, tenantID, entityID, revisionID string) (OrganizationPositionRevisionSummary, error) {
	row := a.pool.QueryRow(ctx, organizationPositionRevisionSelect+`
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`, tenantID, entityID, revisionID)
	value, err := scanOrganizationPositionRevision(row)
	if err != nil {
		return OrganizationPositionRevisionSummary{}, err
	}
	value.Impact, err = organizationPositionImpact(ctx, a.pool, tenantID, entityID, value.PositionID)
	return value, err
}

func organizationPositionRevision(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, revisionID string, lock bool) (OrganizationPositionRevisionSummary, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	row := q.QueryRow(ctx, organizationPositionRevisionSelect+`
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`+suffix, tenantID, entityID, revisionID)
	return scanOrganizationPositionRevision(row)
}

const organizationPositionRevisionSelect = `
	SELECT r.id::text,r.position_id::text,r.operation,r.base_version,
	       r.base_code,r.base_title,r.base_function_name,COALESCE(r.base_organization_scope_id::text,''),COALESCE(r.base_parent_position_id::text,''),COALESCE(r.base_occupant_principal_id::text,''),
	       r.proposed_code,r.proposed_title,r.proposed_function_name,COALESCE(r.proposed_organization_scope_id::text,''),COALESCE(r.proposed_parent_position_id::text,''),COALESCE(r.proposed_occupant_principal_id::text,''),
	       r.maker_id::text,COALESCE(r.checker_id::text,''),r.status,r.rationale,r.created_at,r.decided_at,r.applied_at
	FROM organization_position_revisions r`

type organizationPositionScanner interface {
	Scan(...any) error
}

func scanOrganizationPositionRevision(row organizationPositionScanner) (OrganizationPositionRevisionSummary, error) {
	var value OrganizationPositionRevisionSummary
	err := row.Scan(
		&value.ID, &value.PositionID, &value.Operation, &value.BaseVersion,
		&value.Base.Code, &value.Base.Title, &value.Base.FunctionName, &value.Base.OrganizationScopeID, &value.Base.ParentPositionID, &value.Base.OccupantPrincipalID,
		&value.Proposed.Code, &value.Proposed.Title, &value.Proposed.FunctionName, &value.Proposed.OrganizationScopeID, &value.Proposed.ParentPositionID, &value.Proposed.OccupantPrincipalID,
		&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale, &value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationPositionRevisionSummary{}, ErrAdminNotFound
	}
	return value, err
}

func organizationPositionState(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, positionID string, lock bool) (OrganizationPositionState, int64, error) {
	query := `
		SELECT code,title,COALESCE(function_name,''),COALESCE(organization_scope_id::text,''),
		       COALESCE(parent_position_id::text,''),COALESCE(occupant_principal_id::text,''),version
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`
	if lock {
		query += " FOR UPDATE"
	}
	var value OrganizationPositionState
	var version int64
	err := q.QueryRow(ctx, query, tenantID, entityID, positionID).Scan(
		&value.Code, &value.Title, &value.FunctionName, &value.OrganizationScopeID,
		&value.ParentPositionID, &value.OccupantPrincipalID, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationPositionState{}, 0, ErrAdminNotFound
	}
	return value, version, err
}

func validateOrganizationPositionState(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, positionID string, value OrganizationPositionState) error {
	if value.Code == "" || value.Title == "" {
		return ErrAdminInvalid
	}
	if value.OrganizationScopeID != "" {
		var ok bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM organization_scopes
				WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
				  AND status='ACTIVE' AND valid_from<=clock_timestamp()
				  AND (valid_until IS NULL OR clock_timestamp()<valid_until)
			)`, tenantID, entityID, value.OrganizationScopeID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrAdminInvalid
		}
	}
	if value.ParentPositionID != "" {
		if value.ParentPositionID == positionID {
			return ErrAdminConflict
		}
		var valid, cycle bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM org_positions
				WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
				  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)
			)`, tenantID, entityID, value.ParentPositionID).Scan(&valid); err != nil {
			return err
		}
		if !valid {
			return ErrAdminInvalid
		}
		if err := q.QueryRow(ctx, `
			WITH RECURSIVE chain(id,parent_position_id,path,depth) AS (
				SELECT p.id,p.parent_position_id,ARRAY[p.id],1
				FROM org_positions p
				WHERE p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid
				UNION ALL
				SELECT p.id,p.parent_position_id,chain.path || p.id,chain.depth+1
				FROM chain
				JOIN org_positions p ON p.id=chain.parent_position_id
				  AND p.tenant_id=$1::uuid AND p.legal_entity_id=$2::uuid
				WHERE chain.depth<64 AND NOT p.id=ANY(chain.path)
			)
			SELECT EXISTS(SELECT 1 FROM chain WHERE id=$4::uuid)`,
			tenantID, entityID, value.ParentPositionID, positionID).Scan(&cycle); err != nil {
			return err
		}
		if cycle {
			return ErrAdminConflict
		}
	}
	if value.OccupantPrincipalID != "" {
		var ok bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM principals
				WHERE tenant_id=$1::uuid AND id=$2::uuid AND kind='PERSON' AND status='ACTIVE'
				  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)
			)`, tenantID, value.OccupantPrincipalID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrAdminInvalid
		}
	}
	return nil
}

func organizationPositionDepartmentPath(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, scopeID string) ([]string, error) {
	if strings.TrimSpace(scopeID) == "" {
		return []string{}, nil
	}
	var path []string
	err := q.QueryRow(ctx, `
		SELECT department_path
		FROM organization_scopes
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		  AND status='ACTIVE' AND valid_from<=clock_timestamp()
		  AND (valid_until IS NULL OR clock_timestamp()<valid_until)`,
		tenantID, entityID, scopeID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAdminInvalid
	}
	return path, err
}

func organizationPositionImpact(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, positionID string) (OrganizationPositionImpact, error) {
	var value OrganizationPositionImpact
	err := q.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM org_positions child
		   WHERE child.tenant_id=$1::uuid AND child.legal_entity_id=$2::uuid
		     AND child.parent_position_id=$3::uuid
		     AND child.valid_from<=clock_timestamp() AND (child.valid_until IS NULL OR clock_timestamp()<child.valid_until)),
		  (SELECT count(*) FROM responsibility_assignments assignment
		   WHERE assignment.tenant_id=$1::uuid AND assignment.legal_entity_id=$2::uuid
		     AND assignment.position_id=$3::uuid
		     AND assignment.valid_from<=clock_timestamp() AND (assignment.valid_until IS NULL OR clock_timestamp()<assignment.valid_until)),
		  (SELECT count(*) FROM authority_grants grant_row
		   WHERE grant_row.tenant_id=$1::uuid AND grant_row.legal_entity_id=$2::uuid
		     AND grant_row.position_id=$3::uuid
		     AND grant_row.valid_from<=clock_timestamp() AND (grant_row.valid_until IS NULL OR clock_timestamp()<grant_row.valid_until)),
		  (SELECT count(*) FROM position_role_bindings binding
		   WHERE binding.tenant_id=$1::uuid AND binding.position_id=$3::uuid
		     AND binding.valid_from<=clock_timestamp() AND (binding.valid_until IS NULL OR clock_timestamp()<binding.valid_until))`,
		tenantID, entityID, positionID).
		Scan(&value.ChildPositions, &value.ResponsibilityAssignments, &value.AuthorityGrants, &value.ActiveRoleBindings)
	return value, err
}

func normalizeOrganizationPositionCode(value string) string {
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

func mapOrganizationPositionPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "P0001" {
		return ErrAdminConflict
	}
	return mapAdminPgError(err)
}
