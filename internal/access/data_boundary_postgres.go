//go:build postgres

package access

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (a *PostgresAdministrator) ProposeLegalEntityDataBoundary(ctx context.Context, input ProposeLegalEntityDataBoundaryInput) (LegalEntityDataBoundaryRevision, error) {
	input, err := NormalizeLegalEntityDataBoundaryInput(input)
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}

	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT boundary.version
			FROM legal_entity_data_boundaries boundary
			WHERE boundary.tenant_id=entity.tenant_id AND boundary.legal_entity_id=entity.id
		),0)
		FROM legal_entities entity
		WHERE entity.tenant_id=$1::uuid AND entity.id=$2::uuid
		  AND entity.valid_from<=clock_timestamp()
		  AND (entity.valid_until IS NULL OR clock_timestamp()<entity.valid_until)
		FOR UPDATE`, tenantID, entityID).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return LegalEntityDataBoundaryRevision{}, ErrAdminNotFound
	}
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	if currentVersion != input.ExpectedVersion {
		return LegalEntityDataBoundaryRevision{}, ErrAdminConflict
	}

	var revisionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO legal_entity_data_boundary_revisions(
			tenant_id,legal_entity_id,base_version,
			proposed_residency_region,proposed_detail_transfer_mode,proposed_destination_regions,maker_id
		)
		VALUES($1::uuid,$2::uuid,$3,$4,$5,$6::text[],$7::uuid)
		RETURNING id::text`,
		tenantID, entityID, currentVersion,
		input.ResidencyRegion, input.DetailTransferMode, input.AllowedDestinationRegions, input.ActorID,
	).Scan(&revisionID)
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, mapAdminPgError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	return a.legalEntityDataBoundaryRevisionByID(ctx, tenantID, entityID, revisionID)
}

func (a *PostgresAdministrator) ApproveLegalEntityDataBoundary(ctx context.Context, input DecideLegalEntityDataBoundaryInput) error {
	return a.decideLegalEntityDataBoundary(ctx, input, true)
}

func (a *PostgresAdministrator) RejectLegalEntityDataBoundary(ctx context.Context, input DecideLegalEntityDataBoundaryInput) error {
	return a.decideLegalEntityDataBoundary(ctx, input, false)
}

func (a *PostgresAdministrator) decideLegalEntityDataBoundary(ctx context.Context, input DecideLegalEntityDataBoundaryInput, approve bool) error {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.RevisionID = strings.TrimSpace(input.RevisionID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.TenantID == "" || input.LegalEntityID == "" || input.RevisionID == "" ||
		input.ActorID == "" || input.Rationale == "" {
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

	var revision LegalEntityDataBoundaryRevision
	err = tx.QueryRow(ctx, `
		SELECT id::text,legal_entity_id::text,base_version,
		       proposed_residency_region,proposed_detail_transfer_mode,proposed_destination_regions,
		       maker_id::text,status,created_at
		FROM legal_entity_data_boundary_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		FOR UPDATE`, tenantID, entityID, input.RevisionID).
		Scan(
			&revision.ID, &revision.LegalEntityID, &revision.BaseVersion,
			&revision.ProposedResidencyRegion, &revision.ProposedDetailTransferMode, &revision.ProposedDestinationRegions,
			&revision.MakerID, &revision.Status, &revision.CreatedAt,
		)
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
			UPDATE legal_entity_data_boundary_revisions
			SET status='REJECTED',checker_id=$4::uuid,rationale=$5,decided_at=clock_timestamp()
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
			tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		return tx.Commit(ctx)
	}

	if err := applyLegalEntityDataBoundaryRevision(ctx, tx, tenantID, entityID, revision); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE legal_entity_data_boundary_revisions
		SET status='APPLIED',checker_id=$4::uuid,rationale=$5,
		    decided_at=clock_timestamp(),applied_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='PENDING'`,
		tenantID, entityID, revision.ID, input.ActorID, input.Rationale)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	return tx.Commit(ctx)
}

func applyLegalEntityDataBoundaryRevision(ctx context.Context, tx pgx.Tx, tenantID, entityID string, revision LegalEntityDataBoundaryRevision) error {
	if revision.BaseVersion == 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO legal_entity_data_boundaries(
				tenant_id,legal_entity_id,residency_region,detail_transfer_mode,allowed_destination_regions,version
			)
			VALUES($1::uuid,$2::uuid,$3,$4,$5::text[],1)
			ON CONFLICT(tenant_id,legal_entity_id) DO NOTHING`,
			tenantID, entityID, revision.ProposedResidencyRegion,
			revision.ProposedDetailTransferMode, revision.ProposedDestinationRegions)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrAdminConflict
		}
		return nil
	}

	tag, err := tx.Exec(ctx, `
		UPDATE legal_entity_data_boundaries
		SET residency_region=$4,detail_transfer_mode=$5,allowed_destination_regions=$6::text[],
		    version=version+1,updated_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$3`,
		tenantID, entityID, revision.BaseVersion,
		revision.ProposedResidencyRegion, revision.ProposedDetailTransferMode, revision.ProposedDestinationRegions)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAdminConflict
	}
	return nil
}

func (a *PostgresAdministrator) GetLegalEntityDataBoundary(ctx context.Context, tenantID, entityID string) (LegalEntityDataBoundary, error) {
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	if tenantID == "" || entityID == "" {
		return LegalEntityDataBoundary{}, ErrAdminInvalid
	}
	resolvedTenantID, resolvedEntityID, err := a.scopeIDs(ctx, tenantID, entityID)
	if err != nil {
		return LegalEntityDataBoundary{}, err
	}
	return a.legalEntityDataBoundary(ctx, resolvedTenantID, resolvedEntityID)
}

func (a *PostgresAdministrator) legalEntityDataBoundary(ctx context.Context, tenantID, entityID string) (LegalEntityDataBoundary, error) {
	var value LegalEntityDataBoundary
	var updatedAt *time.Time
	err := a.pool.QueryRow(ctx, `
		SELECT entity.id::text,entity.code,entity.name,COALESCE(entity.jurisdiction,''),
		       COALESCE(boundary.residency_region,''),
		       COALESCE(boundary.detail_transfer_mode,'AGGREGATE_ONLY'),
		       COALESCE(boundary.allowed_destination_regions,ARRAY[]::text[]),
		       COALESCE(boundary.version,0),boundary.updated_at
		FROM legal_entities entity
		LEFT JOIN legal_entity_data_boundaries boundary
		  ON boundary.tenant_id=entity.tenant_id AND boundary.legal_entity_id=entity.id
		WHERE entity.tenant_id=$1::uuid AND entity.id=$2::uuid
		  AND entity.valid_from<=clock_timestamp()
		  AND (entity.valid_until IS NULL OR clock_timestamp()<entity.valid_until)`,
		tenantID, entityID).
		Scan(
			&value.LegalEntityID, &value.LegalEntityCode, &value.LegalEntityName, &value.Jurisdiction,
			&value.ResidencyRegion, &value.DetailTransferMode, &value.AllowedDestinationRegions,
			&value.Version, &updatedAt,
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return LegalEntityDataBoundary{}, ErrAdminNotFound
	}
	if err != nil {
		return LegalEntityDataBoundary{}, err
	}
	value.Configured = value.Version > 0
	value.UpdatedAt = updatedAt
	if value.AllowedDestinationRegions == nil {
		value.AllowedDestinationRegions = []string{}
	}
	return value, nil
}

func (a *PostgresAdministrator) legalEntityDataBoundaryRevisions(ctx context.Context, tenantID, entityID string) ([]LegalEntityDataBoundaryRevision, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT id::text,legal_entity_id::text,base_version,
		       proposed_residency_region,proposed_detail_transfer_mode,proposed_destination_regions,
		       maker_id::text,COALESCE(checker_id::text,''),status,rationale,created_at,decided_at,applied_at
		FROM legal_entity_data_boundary_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		ORDER BY created_at DESC,id DESC
		LIMIT 20`, tenantID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]LegalEntityDataBoundaryRevision, 0, 20)
	for rows.Next() {
		var value LegalEntityDataBoundaryRevision
		if err := rows.Scan(
			&value.ID, &value.LegalEntityID, &value.BaseVersion,
			&value.ProposedResidencyRegion, &value.ProposedDetailTransferMode, &value.ProposedDestinationRegions,
			&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale,
			&value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
		); err != nil {
			return nil, err
		}
		if value.ProposedDestinationRegions == nil {
			value.ProposedDestinationRegions = []string{}
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (a *PostgresAdministrator) legalEntityDataBoundaryRevisionByID(ctx context.Context, tenantID, entityID, revisionID string) (LegalEntityDataBoundaryRevision, error) {
	var value LegalEntityDataBoundaryRevision
	err := a.pool.QueryRow(ctx, `
		SELECT id::text,legal_entity_id::text,base_version,
		       proposed_residency_region,proposed_detail_transfer_mode,proposed_destination_regions,
		       maker_id::text,COALESCE(checker_id::text,''),status,rationale,created_at,decided_at,applied_at
		FROM legal_entity_data_boundary_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, revisionID).
		Scan(
			&value.ID, &value.LegalEntityID, &value.BaseVersion,
			&value.ProposedResidencyRegion, &value.ProposedDetailTransferMode, &value.ProposedDestinationRegions,
			&value.MakerID, &value.CheckerID, &value.Status, &value.Rationale,
			&value.CreatedAt, &value.DecidedAt, &value.AppliedAt,
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return LegalEntityDataBoundaryRevision{}, ErrAdminNotFound
	}
	if err != nil {
		return LegalEntityDataBoundaryRevision{}, err
	}
	if value.ProposedDestinationRegions == nil {
		value.ProposedDestinationRegions = []string{}
	}
	return value, nil
}

var _ DataBoundaryReader = (*PostgresAdministrator)(nil)
var _ DataBoundaryAdministrator = (*PostgresAdministrator)(nil)
