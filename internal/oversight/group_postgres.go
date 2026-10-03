//go:build postgres

package oversight

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	groupRefreshInterval = 5 * time.Minute
	groupRetentionPeriod = 90 * 24 * time.Hour
	groupChildStaleAfter = 15 * time.Minute
)

type GroupMaintainer struct {
	Repository *PostgresRepository
}

func (r *PostgresRepository) LatestGroup(ctx context.Context, tenantID string) (GroupProjection, error) {
	tenantID = strings.TrimSpace(tenantID)
	if r == nil || r.pool == nil || tenantID == "" {
		return GroupProjection{}, ErrInvalid
	}
	var value GroupProjection
	err := r.pool.QueryRow(ctx, `
		SELECT run.id::text,tenant.slug,run.generated_at,run.refresh_slot,run.projection_version,
		       run.active_child_count,run.captured_child_count,run.missing_child_count,run.stale_child_count
		FROM group_oversight_runs run
		JOIN tenants tenant ON tenant.id=run.tenant_id
		WHERE tenant.id::text=$1 OR tenant.slug=$1
		ORDER BY run.generated_at DESC,run.id DESC
		LIMIT 1`, tenantID).
		Scan(&value.ID, &value.TenantID, &value.GeneratedAt, &value.RefreshSlot, &value.ProjectionVersion,
			&value.ActiveChildCount, &value.CapturedChildCount, &value.MissingChildCount, &value.StaleChildCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return GroupProjection{}, ErrNotFound
	}
	if err != nil {
		return GroupProjection{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT legal_entity_id::text,legal_entity_code,legal_entity_name,jurisdiction,state,
		       COALESCE(child_snapshot_id::text,''),child_generated_at,COALESCE(child_projection_version,''),
		       coverage_population,coverage_excluded,coverage_unknown,counts,source_high_water
		FROM group_oversight_child_facts
		WHERE run_id=$1::uuid
		ORDER BY lower(legal_entity_name),legal_entity_id`, value.ID)
	if err != nil {
		return GroupProjection{}, err
	}
	defer rows.Close()

	value.Children = make([]GroupChildFact, 0, value.ActiveChildCount)
	for rows.Next() {
		child, scanErr := scanGroupChild(rows)
		if scanErr != nil {
			return GroupProjection{}, scanErr
		}
		value.Children = append(value.Children, child)
	}
	if err := rows.Err(); err != nil {
		return GroupProjection{}, err
	}
	return value, nil
}

func (m *GroupMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Repository == nil || m.Repository.pool == nil || ctx == nil {
		return 0, ErrInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	now = now.UTC()
	rows, err := m.Repository.pool.Query(ctx, `
		SELECT tenant.id::text
		FROM tenants tenant
		WHERE (
			SELECT count(*)
			FROM legal_entities entity
			WHERE entity.tenant_id=tenant.id
			  AND entity.valid_from<=$1
			  AND (entity.valid_until IS NULL OR $1<entity.valid_until)
		) >= 2
		  AND NOT EXISTS (
			SELECT 1
			FROM group_oversight_runs run
			WHERE run.tenant_id=tenant.id
			  AND run.projection_version=$2
			  AND run.generated_at>$1-interval '5 minutes'
		)
		ORDER BY tenant.id
		LIMIT $3`, now, GroupProjectionVersion, limit)
	if err != nil {
		return 0, err
	}
	tenantIDs := make([]string, 0, limit)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			rows.Close()
			return 0, err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	completed := 0
	for _, tenantID := range tenantIDs {
		if ctx.Err() != nil {
			return completed, ctx.Err()
		}
		projection, err := m.Repository.buildGroupProjection(ctx, tenantID, now)
		if err != nil {
			return completed, err
		}
		inserted, err := m.Repository.storeGroupProjection(ctx, projection)
		if err != nil {
			return completed, err
		}
		if inserted {
			completed++
		}
	}
	_, cleanupErr := m.Repository.pool.Exec(ctx, `DELETE FROM group_oversight_runs WHERE generated_at<$1`, now.Add(-groupRetentionPeriod))
	return completed, cleanupErr
}

func (r *PostgresRepository) buildGroupProjection(ctx context.Context, tenantID string, now time.Time) (GroupProjection, error) {
	now = now.UTC()
	rows, err := r.pool.Query(ctx, `
		WITH selected_tenant AS (
			SELECT id,slug
			FROM tenants
			WHERE id::text=$1 OR slug=$1
			LIMIT 1
		)
		SELECT tenant.id::text,entity.id::text,entity.code,entity.name,COALESCE(entity.jurisdiction,''),
		       snapshot.id::text,snapshot.generated_at,snapshot.projection_version,
		       snapshot.source_high_water,snapshot.coverage_population,snapshot.coverage_excluded,
		       snapshot.coverage_unknown,snapshot.payload
		FROM selected_tenant tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id
		LEFT JOIN LATERAL (
			SELECT value.id,value.generated_at,value.projection_version,value.source_high_water,
			       value.coverage_population,value.coverage_excluded,value.coverage_unknown,value.payload
			FROM oversight_snapshots value
			WHERE value.tenant_id=tenant.id
			  AND value.legal_entity_id=entity.id
			  AND value.generated_at<=$2
			ORDER BY value.generated_at DESC,value.id DESC
			LIMIT 1
		) snapshot ON true
		WHERE entity.valid_from<=$2
		  AND (entity.valid_until IS NULL OR $2<entity.valid_until)
		ORDER BY entity.id`, tenantID, now)
	if err != nil {
		return GroupProjection{}, err
	}
	defer rows.Close()

	value := GroupProjection{
		TenantID: tenantID, GeneratedAt: now, RefreshSlot: now.Truncate(groupRefreshInterval),
		ProjectionVersion: GroupProjectionVersion, Children: []GroupChildFact{},
	}
	for rows.Next() {
		child, resolvedTenantID, scanErr := scanLatestGroupSource(rows, now)
		if scanErr != nil {
			return GroupProjection{}, scanErr
		}
		value.TenantID = resolvedTenantID
		value.ActiveChildCount++
		switch child.State {
		case GroupChildMissing:
			value.MissingChildCount++
		default:
			value.CapturedChildCount++
			if child.State == GroupChildStale {
				value.StaleChildCount++
			}
		}
		value.Children = append(value.Children, child)
	}
	if err := rows.Err(); err != nil {
		return GroupProjection{}, err
	}
	if value.TenantID == "" || value.ActiveChildCount == 0 {
		return GroupProjection{}, ErrNotFound
	}
	return value, nil
}

func (r *PostgresRepository) storeGroupProjection(ctx context.Context, value GroupProjection) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var runID string
	err = tx.QueryRow(ctx, `
		INSERT INTO group_oversight_runs(
			tenant_id,refresh_slot,generated_at,projection_version,
			active_child_count,captured_child_count,missing_child_count,stale_child_count
		) VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(tenant_id,projection_version,refresh_slot) DO NOTHING
		RETURNING id::text`,
		value.TenantID, value.RefreshSlot, value.GeneratedAt, value.ProjectionVersion,
		value.ActiveChildCount, value.CapturedChildCount, value.MissingChildCount, value.StaleChildCount).
		Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for _, child := range value.Children {
		counts, err := json.Marshal(child.Counts)
		if err != nil {
			return false, err
		}
		highWater, err := json.Marshal(child.SourceHighWater)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO group_oversight_child_facts(
				run_id,tenant_id,legal_entity_id,legal_entity_code,legal_entity_name,jurisdiction,state,
				child_snapshot_id,child_generated_at,child_projection_version,
				coverage_population,coverage_excluded,coverage_unknown,counts,source_high_water
			) VALUES(
				$1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,
				NULLIF($8,'')::uuid,$9,NULLIF($10,''),
				$11,$12,$13,$14::jsonb,$15::jsonb
			)`,
			runID, value.TenantID, child.LegalEntityID, child.LegalEntityCode, child.LegalEntityName, child.Jurisdiction, child.State,
			child.ChildSnapshotID, child.ChildGeneratedAt, child.ChildProjectionVersion,
			nullableCoveragePopulation(child), child.Coverage.Excluded, child.Coverage.Unknown, counts, highWater)
		if err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func scanLatestGroupSource(rows pgx.Rows, now time.Time) (GroupChildFact, string, error) {
	var tenantID, entityID, code, name, jurisdiction string
	var snapshotID, projectionVersion sql.NullString
	var generatedAt sql.NullTime
	var highWater, payload []byte
	var population, excluded, unknown sql.NullInt64
	if err := rows.Scan(
		&tenantID, &entityID, &code, &name, &jurisdiction,
		&snapshotID, &generatedAt, &projectionVersion, &highWater,
		&population, &excluded, &unknown, &payload,
	); err != nil {
		return GroupChildFact{}, "", err
	}
	child := GroupChildFact{
		LegalEntityID: entityID, LegalEntityCode: code, LegalEntityName: name, Jurisdiction: jurisdiction,
		State: GroupChildMissing, SourceHighWater: map[string]time.Time{},
	}
	if !snapshotID.Valid {
		return child, tenantID, nil
	}
	child.ChildSnapshotID = snapshotID.String
	child.ChildGeneratedAt = &generatedAt.Time
	child.ChildProjectionVersion = projectionVersion.String
	child.Coverage.Population = int(population.Int64)
	if excluded.Valid {
		child.Coverage.Excluded = intPtr(int(excluded.Int64))
	}
	if unknown.Valid {
		child.Coverage.Unknown = intPtr(int(unknown.Int64))
	}
	if len(highWater) > 0 {
		if err := json.Unmarshal(highWater, &child.SourceHighWater); err != nil {
			return GroupChildFact{}, "", fmt.Errorf("decode group child high-water marks: %w", err)
		}
	}
	var metadata struct {
		Counts Counts `json:"counts"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &metadata); err != nil {
			return GroupChildFact{}, "", fmt.Errorf("decode group child snapshot: %w", err)
		}
	}
	child.Counts = metadata.Counts
	child.State = GroupChildAvailable
	if child.ChildProjectionVersion != ProjectionVersion || now.Sub(generatedAt.Time.UTC()) > groupChildStaleAfter {
		child.State = GroupChildStale
	}
	return child, tenantID, nil
}

func scanGroupChild(rows pgx.Rows) (GroupChildFact, error) {
	var state string
	var generatedAt sql.NullTime
	var population, excluded, unknown sql.NullInt64
	var countsJSON, sourceHighWaterJSON []byte
	var child GroupChildFact
	if err := rows.Scan(
		&child.LegalEntityID, &child.LegalEntityCode, &child.LegalEntityName, &child.Jurisdiction, &state,
		&child.ChildSnapshotID, &generatedAt, &child.ChildProjectionVersion,
		&population, &excluded, &unknown, &countsJSON, &sourceHighWaterJSON,
	); err != nil {
		return GroupChildFact{}, err
	}
	child.State = GroupChildState(state)
	if generatedAt.Valid {
		child.ChildGeneratedAt = &generatedAt.Time
	}
	if population.Valid {
		child.Coverage.Population = int(population.Int64)
	}
	if excluded.Valid {
		child.Coverage.Excluded = intPtr(int(excluded.Int64))
	}
	if unknown.Valid {
		child.Coverage.Unknown = intPtr(int(unknown.Int64))
	}
	child.SourceHighWater = map[string]time.Time{}
	if len(countsJSON) > 0 {
		if err := json.Unmarshal(countsJSON, &child.Counts); err != nil {
			return GroupChildFact{}, fmt.Errorf("decode group child counts: %w", err)
		}
	}
	if len(sourceHighWaterJSON) > 0 {
		if err := json.Unmarshal(sourceHighWaterJSON, &child.SourceHighWater); err != nil {
			return GroupChildFact{}, fmt.Errorf("decode group child high-water marks: %w", err)
		}
	}
	return child, nil
}

func nullableCoveragePopulation(child GroupChildFact) any {
	if child.State == GroupChildMissing {
		return nil
	}
	return child.Coverage.Population
}
