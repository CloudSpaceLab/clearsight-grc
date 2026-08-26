//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresRepository) RelationshipExists(ctx context.Context, scope Scope, relationshipID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM third_party_relationships rel JOIN tenants t ON t.id=rel.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND rel.legal_entity_id::text=$2 AND rel.id::text=$3
	)`, scope.TenantID, scope.LegalEntityID, relationshipID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) TargetAvailable(ctx context.Context, scope Scope, targetType LinkTargetType, targetID string) (bool, error) {
	return targetAvailable(ctx, r.pool, scope, targetType, targetID)
}

type linkQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func targetAvailable(ctx context.Context, queryer linkQueryer, scope Scope, targetType LinkTargetType, targetID string) (bool, error) {
	var exists bool
	var err error
	switch targetType {
	case LinkTargetProgram:
		err = queryer.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM programs p JOIN tenants t ON t.id=p.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$3
			  AND (p.legal_entity_id IS NULL OR p.legal_entity_id::text=$2)
		)`, scope.TenantID, scope.LegalEntityID, targetID).Scan(&exists)
	case LinkTargetMatter:
		err = queryer.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM matters m JOIN tenants t ON t.id=m.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND m.id::text=$3
			  AND NOT EXISTS (
				SELECT 1 FROM matter_links ml JOIN programs p ON p.id=ml.program_id AND p.tenant_id=ml.tenant_id
				WHERE ml.tenant_id=m.tenant_id AND ml.matter_id=m.id
				  AND p.legal_entity_id IS NOT NULL AND p.legal_entity_id::text<>$2
			  )
		)`, scope.TenantID, scope.LegalEntityID, targetID).Scan(&exists)
	default:
		return false, ErrInvalid
	}
	return exists, err
}

func (r *PostgresRepository) CreateRelationshipLink(ctx context.Context, value RelationshipLink) (RelationshipLink, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("begin vendor relationship link: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, value.TenantID)
	if err != nil {
		return RelationshipLink{}, err
	}
	var relationshipExists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_relationships WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND id::text=$3)`, tenantID, value.LegalEntityID, value.RelationshipID).Scan(&relationshipExists)
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("verify vendor relationship link scope: %w", err)
	}
	available, err := targetAvailable(ctx, tx, Scope{TenantID: tenantID, LegalEntityID: value.LegalEntityID}, value.TargetType, value.TargetID)
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("verify vendor relationship link target: %w", err)
	}
	if !relationshipExists || !available {
		return RelationshipLink{}, ErrNotFound
	}
	table, targetColumn, err := relationshipLinkTable(value.TargetType)
	if err != nil {
		return RelationshipLink{}, err
	}
	query := fmt.Sprintf(`INSERT INTO %s(id,tenant_id,legal_entity_id,relationship_id,%s,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'ACTIVE',$8::uuid,1,$9,$9)`, table, targetColumn)
	_, err = tx.Exec(ctx, query, value.ID, tenantID, value.LegalEntityID, value.RelationshipID, value.TargetID, value.PurposeCode, value.PurposeLabel, value.CreatedBy, value.CreatedAt)
	if isUniqueViolation(err) {
		return RelationshipLink{}, ErrVersionConflict
	}
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("store vendor relationship link: %w", err)
	}
	eventID, err := appendRelationshipLinkEvent(ctx, tx, tenantID, value, "VendorRelationshipLinked")
	if err != nil {
		return RelationshipLink{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, relationshipLinkCommitProof(eventID, value, "VendorRelationshipLinked")); err != nil {
		return RelationshipLink{}, fmt.Errorf("commit vendor relationship link: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) GetRelationshipLink(ctx context.Context, scope Scope, linkID string) (RelationshipLink, error) {
	return getRelationshipLink(ctx, r.pool, scope, linkID, false)
}

func getRelationshipLink(ctx context.Context, queryer linkQueryer, scope Scope, linkID string, lock bool) (RelationshipLink, error) {
	// The update uses an expected version in a serializable transaction. The
	// UNION projection cannot be portably row-locked across both typed tables.
	_ = lock
	query := relationshipLinkUnion + ` WHERE tenant_ref=$1 AND legal_entity_id::text=$2 AND id::text=$3`
	value, err := scanRelationshipLink(queryer.QueryRow(ctx, query, scope.TenantID, scope.LegalEntityID, linkID))
	if errors.Is(err, pgx.ErrNoRows) {
		return RelationshipLink{}, ErrNotFound
	}
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("get vendor relationship link: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) EndRelationshipLink(ctx context.Context, scope Scope, linkID string, expected int64, actor, reason string, now time.Time) (RelationshipLink, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("begin vendor relationship unlink: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := getRelationshipLink(ctx, tx, scope, linkID, true)
	if err != nil {
		return RelationshipLink{}, err
	}
	if current.Version != expected {
		return RelationshipLink{}, ErrVersionConflict
	}
	if current.State != RelationshipLinkActive {
		return RelationshipLink{}, ErrInvalid
	}
	table, _, _ := relationshipLinkTable(current.TargetType)
	var lockedVersion int64
	var lockedState RelationshipLinkState
	lockQuery := fmt.Sprintf(`SELECT version,state FROM %s WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id::text=$3 FOR UPDATE`, table)
	if err := tx.QueryRow(ctx, lockQuery, linkID, scope.TenantID, scope.LegalEntityID).Scan(&lockedVersion, &lockedState); errors.Is(err, pgx.ErrNoRows) {
		return RelationshipLink{}, ErrNotFound
	} else if err != nil {
		return RelationshipLink{}, fmt.Errorf("lock vendor relationship link: %w", err)
	}
	if lockedVersion != expected {
		return RelationshipLink{}, ErrVersionConflict
	}
	if lockedState != RelationshipLinkActive {
		return RelationshipLink{}, ErrInvalid
	}
	var activeWork bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_work_requests WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id::text=$2 AND COALESCE(program_link_id,matter_link_id)::text=$3 AND state NOT IN ('ACCEPTED','CANCELLED'))`, scope.TenantID, scope.LegalEntityID, linkID).Scan(&activeWork); err != nil {
		return RelationshipLink{}, fmt.Errorf("check active vendor work: %w", err)
	}
	if activeWork {
		return RelationshipLink{}, ErrVersionConflict
	}
	query := fmt.Sprintf(`UPDATE %s SET state='ENDED',ended_by_principal_id=$4::uuid,end_reason=$5,ended_at=$6,updated_at=$6,version=version+1
		WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id::text=$3 AND version=$7 RETURNING version`, table)
	var version int64
	err = tx.QueryRow(ctx, query, linkID, scope.TenantID, scope.LegalEntityID, actor, reason, now, expected).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return RelationshipLink{}, ErrVersionConflict
	}
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("end vendor relationship link: %w", err)
	}
	current.State, current.EndedBy, current.EndReason, current.EndedAt, current.UpdatedAt, current.Version = RelationshipLinkEnded, actor, reason, &now, now, version
	tenantID, err := resolveTenant(ctx, tx, scope.TenantID)
	if err != nil {
		return RelationshipLink{}, err
	}
	eventID, err := appendRelationshipLinkEvent(ctx, tx, tenantID, current, "VendorRelationshipLinkEnded")
	if err != nil {
		return RelationshipLink{}, err
	}
	if err := r.commitThirdPartyEvents(ctx, tx, relationshipLinkCommitProof(eventID, current, "VendorRelationshipLinkEnded")); err != nil {
		return RelationshipLink{}, fmt.Errorf("commit vendor relationship unlink: %w", err)
	}
	return current, nil
}

func (r *PostgresRepository) ListRelationshipLinks(ctx context.Context, scope Scope, input RelationshipLinkListInput) (RelationshipLinkPage, error) {
	args := []any{scope.TenantID, scope.LegalEntityID, input.RelationshipID, string(input.TargetType), input.TargetID, input.IncludeEnded, input.VisiblePrincipalID}
	cursorClause := ""
	if input.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(input.Cursor)
		if err != nil {
			return RelationshipLinkPage{}, ErrInvalid
		}
		args = append(args, cursorTime, cursorID)
		cursorClause = " AND (updated_at,id) < ($8,$9::uuid)"
	}
	args = append(args, input.Limit+1)
	limitArg := len(args)
	rows, err := r.pool.Query(ctx, relationshipLinkUnion+` WHERE tenant_ref=$1 AND legal_entity_id::text=$2
		AND ($3='' OR relationship_id::text=$3) AND ($4='' OR target_type=$4) AND ($5='' OR target_id::text=$5)
		AND ($6 OR state='ACTIVE')
		AND (target_type<>'MATTER' OR EXISTS (
			SELECT 1 FROM matters m JOIN tenants mt ON mt.id=m.tenant_id
			WHERE (mt.id::text=$1 OR mt.slug=$1) AND m.id=link_rows.target_id AND `+matterVisibilitySQL("m", 7)+`
		))`+cursorClause+` ORDER BY updated_at DESC,id DESC LIMIT $`+fmt.Sprint(limitArg), args...)
	if err != nil {
		return RelationshipLinkPage{}, fmt.Errorf("list vendor relationship links: %w", err)
	}
	defer rows.Close()
	items := make([]RelationshipLink, 0, input.Limit+1)
	for rows.Next() {
		value, scanErr := scanRelationshipLink(rows)
		if scanErr != nil {
			return RelationshipLinkPage{}, scanErr
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return RelationshipLinkPage{}, err
	}
	page := RelationshipLinkPage{Items: items}
	if len(items) > input.Limit {
		page.Items = items[:input.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func matterVisibilitySQL(alias string, principalArg int) string {
	return fmt.Sprintf(`CASE
		WHEN NOT (%[1]s.scope ? 'access') THEN true
		WHEN jsonb_typeof(%[1]s.scope->'access')<>'string' THEN false
		WHEN upper(btrim(%[1]s.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		WHEN upper(btrim(%[1]s.scope->>'access'))='RESTRICTED' THEN
			CASE
				WHEN jsonb_typeof(%[1]s.scope->'allowed_principal_ids')<>'array' THEN false
				ELSE NOT EXISTS (
					SELECT 1 FROM jsonb_array_elements(%[1]s.scope->'allowed_principal_ids') entry(value)
					WHERE jsonb_typeof(entry.value)<>'string'
				) AND EXISTS (
					SELECT 1 FROM jsonb_array_elements_text(%[1]s.scope->'allowed_principal_ids') nonblank(value)
					WHERE btrim(nonblank.value)<>''
				) AND EXISTS (
					SELECT 1 FROM jsonb_array_elements_text(%[1]s.scope->'allowed_principal_ids') allowed(value)
					WHERE btrim(allowed.value)=$%[2]d
				)
			END
		ELSE false
	END`, alias, principalArg)
}

func relationshipLinkTable(targetType LinkTargetType) (string, string, error) {
	switch targetType {
	case LinkTargetProgram:
		return "third_party_relationship_program_links", "program_id", nil
	case LinkTargetMatter:
		return "third_party_relationship_matter_links", "matter_id", nil
	default:
		return "", "", ErrInvalid
	}
}

const relationshipLinkUnion = `SELECT * FROM (
	SELECT l.id::text,t.slug AS tenant_ref,l.legal_entity_id,l.relationship_id,'PROGRAM'::text AS target_type,l.program_id AS target_id,l.purpose_code,l.purpose_label,l.state,
	       l.created_by_principal_id::text,COALESCE(l.ended_by_principal_id::text,''),l.end_reason,l.version,l.created_at,l.updated_at,l.ended_at
	FROM third_party_relationship_program_links l JOIN tenants t ON t.id=l.tenant_id
	UNION ALL
	SELECT l.id::text,t.slug AS tenant_ref,l.legal_entity_id,l.relationship_id,'MATTER'::text AS target_type,l.matter_id AS target_id,l.purpose_code,l.purpose_label,l.state,
	       l.created_by_principal_id::text,COALESCE(l.ended_by_principal_id::text,''),l.end_reason,l.version,l.created_at,l.updated_at,l.ended_at
	FROM third_party_relationship_matter_links l JOIN tenants t ON t.id=l.tenant_id
) link_rows`

func scanRelationshipLink(row rowScanner) (RelationshipLink, error) {
	var value RelationshipLink
	var targetType string
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &targetType, &value.TargetID,
		&value.PurposeCode, &value.PurposeLabel, &value.State, &value.CreatedBy, &value.EndedBy, &value.EndReason,
		&value.Version, &value.CreatedAt, &value.UpdatedAt, &value.EndedAt)
	value.TargetType = LinkTargetType(targetType)
	return value, err
}

func appendRelationshipLinkEvent(ctx context.Context, tx pgx.Tx, tenantID string, value RelationshipLink, eventType string) (string, error) {
	actor := value.CreatedBy
	if value.EndedBy != "" {
		actor = value.EndedBy
	}
	var eventID string
	err := tx.QueryRow(ctx, `INSERT INTO third_party_relationship_link_events(tenant_id,legal_entity_id,link_id,relationship_id,target_type,target_id,link_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9,jsonb_build_object('purpose_code',$10::text,'state',$11::text,'reason',$12::text),$13)
		RETURNING id::text`,
		tenantID, value.LegalEntityID, value.ID, value.RelationshipID, value.TargetType, value.TargetID, value.Version, actor, eventType, value.PurposeCode, value.State, value.EndReason, value.UpdatedAt).Scan(&eventID)
	if err != nil {
		return "", fmt.Errorf("append vendor relationship link event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'VENDOR_RELATIONSHIP_LINK',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'relationship_id',$5::text,'target_type',$6::text,'target_id',$7::text,'state',$8::text),$9,$9)`,
		tenantID, value.ID, eventType, value.Version, value.RelationshipID, value.TargetType, value.TargetID, value.State, value.UpdatedAt)
	if err != nil {
		return "", fmt.Errorf("append vendor relationship link outbox event: %w", err)
	}
	return eventID, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var _ RelationshipLinkRepository = (*PostgresRepository)(nil)
