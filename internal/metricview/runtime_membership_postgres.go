//go:build postgres

package metricview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const runtimeMembershipTTL = "24 hours"

func (r *MembershipRepository) RetainRuntimeSnapshot(ctx context.Context, snapshot oversight.Snapshot) (string, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return "", ErrMetricMembershipInvalid
	}
	if err := validateRuntimeMetricMembers(snapshot); err != nil {
		return "", err
	}
	fingerprint, err := runtimeSnapshotFingerprint(snapshot)
	if err != nil {
		return "", err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM metric_runtime_membership_sets WHERE expires_at<=clock_timestamp()`); err != nil {
		return "", fmt.Errorf("expire runtime metric memberships: %w", err)
	}

	var sourceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO metric_runtime_membership_sets(
			tenant_id,legal_entity_id,organization_scope_id,definition_revision,
			source_revision,request_fingerprint,generated_at,period_start,period_end,expires_at
		)
		SELECT tenant.id,entity.id,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,clock_timestamp()+interval '`+runtimeMembershipTTL+`'
		FROM tenants tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND (
		    $3='' OR EXISTS (
		      SELECT 1
		      FROM organization_scopes scope
		      WHERE scope.id=$3::uuid
		        AND scope.tenant_id=tenant.id
		        AND scope.legal_entity_id=entity.id
		    )
		  )
		ON CONFLICT DO NOTHING
		RETURNING source_id::text`,
		strings.TrimSpace(snapshot.TenantID),
		strings.TrimSpace(snapshot.LegalEntityID),
		strings.TrimSpace(snapshot.OrganizationScopeID),
		HomeDefinitionRevision,
		strings.TrimSpace(snapshot.ProjectionVersion),
		fingerprint,
		snapshot.GeneratedAt.UTC(),
		snapshot.PeriodStart.UTC(),
		snapshot.PeriodEnd.UTC(),
	).Scan(&sourceID)
	switch {
	case err == nil:
		var sourceUUID pgtype.UUID
		if scanErr := sourceUUID.Scan(sourceID); scanErr != nil || !sourceUUID.Valid {
			return "", ErrMetricMembershipInvalid
		}
		rows := make([][]any, 0, len(snapshot.MetricMembers))
		for _, member := range snapshot.MetricMembers {
			var memberUUID, targetUUID pgtype.UUID
			if scanErr := memberUUID.Scan(member.MemberID); scanErr != nil || !memberUUID.Valid {
				return "", ErrMetricMembershipInvalid
			}
			if scanErr := targetUUID.Scan(member.TargetID); scanErr != nil || !targetUUID.Valid {
				return "", ErrMetricMembershipInvalid
			}
			rows = append(rows, []any{
				sourceUUID,
				member.MetricID,
				HomeDefinitionRevision,
				memberUUID,
				member.TargetType,
				targetUUID,
				member.TargetTitle,
				member.State,
			})
		}
		if len(rows) > 0 {
			if _, err := tx.CopyFrom(
				ctx,
				pgx.Identifier{"metric_runtime_memberships"},
				[]string{"source_id", "metric_id", "definition_revision", "member_id", "target_type", "target_id", "target_title", "state"},
				pgx.CopyFromRows(rows),
			); err != nil {
				return "", fmt.Errorf("retain runtime metric members: %w", err)
			}
		}
	case err == pgx.ErrNoRows:
		err = tx.QueryRow(ctx, `
			SELECT membership.source_id::text
			FROM metric_runtime_membership_sets membership
			JOIN tenants tenant ON tenant.id=membership.tenant_id
			JOIN legal_entities entity
			  ON entity.tenant_id=membership.tenant_id
			 AND entity.id=membership.legal_entity_id
			WHERE (tenant.id::text=$1 OR tenant.slug=$1)
			  AND (entity.id::text=$2 OR entity.code=$2)
			  AND membership.organization_scope_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid
			  AND membership.definition_revision=$4
			  AND membership.request_fingerprint=$5
			  AND membership.expires_at>clock_timestamp()
			LIMIT 1`,
			strings.TrimSpace(snapshot.TenantID),
			strings.TrimSpace(snapshot.LegalEntityID),
			strings.TrimSpace(snapshot.OrganizationScopeID),
			HomeDefinitionRevision,
			fingerprint,
		).Scan(&sourceID)
		if err != nil {
			return "", fmt.Errorf("resolve retained runtime metric membership: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE metric_runtime_membership_sets
			SET expires_at=GREATEST(expires_at,clock_timestamp()+interval '`+runtimeMembershipTTL+`')
			WHERE source_id=$1::uuid`, sourceID); err != nil {
			return "", fmt.Errorf("extend runtime metric membership: %w", err)
		}
	default:
		return "", fmt.Errorf("retain runtime metric membership source: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return sourceID, nil
}

func validateRuntimeMetricMembers(snapshot oversight.Snapshot) error {
	if strings.TrimSpace(snapshot.TenantID) == "" ||
		strings.TrimSpace(snapshot.LegalEntityID) == "" ||
		strings.TrimSpace(snapshot.ProjectionVersion) == "" ||
		snapshot.GeneratedAt.IsZero() ||
		snapshot.PeriodStart.IsZero() ||
		snapshot.PeriodEnd.IsZero() ||
		snapshot.PeriodStart.After(snapshot.PeriodEnd) {
		return ErrMetricMembershipInvalid
	}

	counts := map[string]int{
		"critical_high_open": 0,
		"overdue_open":       0,
		"routing_gaps":       0,
		"outcome_failures":   0,
	}
	seen := make(map[string]struct{}, len(snapshot.MetricMembers))
	for _, member := range snapshot.MetricMembers {
		if _, ok := counts[member.MetricID]; !ok ||
			strings.TrimSpace(member.MemberID) == "" ||
			strings.TrimSpace(member.TargetID) == "" ||
			strings.TrimSpace(member.TargetTitle) == "" ||
			(member.TargetType != "MATTER" && member.TargetType != "PROGRAM") {
			return ErrMetricMembershipInvalid
		}
		key := member.MetricID + ":" + member.MemberID
		if _, duplicate := seen[key]; duplicate {
			return ErrMetricMembershipInvalid
		}
		seen[key] = struct{}{}
		counts[member.MetricID]++
	}
	expected := map[string]int{
		"critical_high_open": snapshot.Counts.CriticalHigh,
		"overdue_open":       snapshot.Counts.Overdue,
		"routing_gaps":       snapshot.Counts.RoutingFailures,
		"outcome_failures":   snapshot.Counts.OutcomeFailures,
	}
	for metricID, value := range expected {
		if counts[metricID] != value {
			return ErrMetricMembershipInvalid
		}
	}
	return nil
}

func runtimeSnapshotFingerprint(snapshot oversight.Snapshot) (string, error) {
	payload := struct {
		TenantID             string                   `json:"tenant_id"`
		LegalEntityID        string                   `json:"legal_entity_id"`
		OrganizationScopeID  string                   `json:"organization_scope_id,omitempty"`
		GeneratedAt          string                   `json:"generated_at"`
		PeriodStart          string                   `json:"period_start"`
		PeriodEnd            string                   `json:"period_end"`
		ProjectionVersion    string                   `json:"projection_version"`
		Counts               oversight.Counts         `json:"counts"`
		SourceHighWater      map[string]string         `json:"source_high_water"`
		Members              []oversight.MetricMember `json:"members"`
	}{
		TenantID:            strings.TrimSpace(snapshot.TenantID),
		LegalEntityID:       strings.TrimSpace(snapshot.LegalEntityID),
		OrganizationScopeID: strings.TrimSpace(snapshot.OrganizationScopeID),
		GeneratedAt:         snapshot.GeneratedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		PeriodStart:         snapshot.PeriodStart.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		PeriodEnd:           snapshot.PeriodEnd.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		ProjectionVersion:   strings.TrimSpace(snapshot.ProjectionVersion),
		Counts:              snapshot.Counts,
		SourceHighWater:     make(map[string]string, len(snapshot.SourceHighWater)),
		Members:             snapshot.MetricMembers,
	}
	for key, value := range snapshot.SourceHighWater {
		payload.SourceHighWater[key] = value.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode runtime metric membership fingerprint: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

var _ MembershipWriter = (*MembershipRepository)(nil)
