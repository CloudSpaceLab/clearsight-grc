//go:build postgres

package metricview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const DomainScopedSourceRevision = "enterprise-domain-scoped-v1"

func (r *DomainRepository) CurrentDomainMetrics(
	ctx context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
	at time.Time,
) (DomainBundle, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	organizationScopeID = strings.TrimSpace(organizationScopeID)
	organizationScopeIDs = normalizeDomainScopeIDs(organizationScopeIDs)
	if tenantID == "" || legalEntityID == "" || organizationScopeID == "" || len(organizationScopeIDs) == 0 || at.IsZero() {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	if !containsString(organizationScopeIDs, organizationScopeID) {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return DomainBundle{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	scope, err := resolveScopedDomainScope(ctx, tx, tenantID, legalEntityID, organizationScopeID, organizationScopeIDs)
	if err != nil {
		return DomainBundle{}, err
	}
	at = at.UTC()
	results, err := loadDomainMetricResults(ctx, tx, scope, at)
	if err != nil {
		return DomainBundle{}, err
	}
	highWater, err := domainSourceHighWater(ctx, tx, scope)
	if err != nil {
		return DomainBundle{}, err
	}
	sourceID, err := retainScopedDomainMembers(ctx, tx, scope, at, highWater, results)
	if err != nil {
		return DomainBundle{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DomainBundle{}, err
	}

	bundle := DomainBundle{
		GeneratedAt:        at,
		PostureAsOf:        at,
		ScopeID:            organizationScopeID,
		ScopeKind:          "ORGANIZATION_SCOPE",
		SourceID:           sourceID,
		SourceRevision:     DomainScopedSourceRevision,
		DefinitionRevision: DomainDefinitionRevision,
		Items:              make([]Metric, 0, len(results)),
	}
	zero := 0
	for _, result := range results {
		condition := ConditionClear
		if len(result.Members) > 0 {
			condition = ConditionAttention
		}
		unknown := result.Unknown
		completeness := CompletenessComplete
		if unknown > 0 {
			completeness = CompletenessPartial
		}
		bundle.Items = append(bundle.Items, Metric{
			ID:                 result.Definition.ID,
			Label:              result.Definition.Label,
			Value:              len(result.Members),
			Unit:               result.Definition.Unit,
			Condition:          condition,
			Freshness:          oversight.FreshnessCurrent,
			Completeness:       completeness,
			Population:         result.Population,
			Excluded:           &zero,
			Unknown:            &unknown,
			GeneratedAt:        at,
			SourceRevision:     DomainScopedSourceRevision,
			DefinitionRevision: result.Definition.Revision,
			Basis:              result.Definition.Basis,
			Drill:              result.Definition.Drill,
		})
	}
	return bundle, nil
}

func loadDomainMetricResults(ctx context.Context, tx pgx.Tx, scope domainScope, at time.Time) ([]domainMetricResult, error) {
	loaders := [...]func(context.Context, pgx.Tx, domainScope, time.Time) (domainMetricResult, error){
		loadOutsideAppetiteMetric,
		loadIndicatorBreachMetric,
		loadAssuranceFailureMetric,
		loadLossWithoutIssueMetric,
	}
	results := make([]domainMetricResult, 0, len(loaders))
	for _, load := range loaders {
		result, err := load(ctx, tx, scope, at)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func resolveScopedDomainScope(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
) (domainScope, error) {
	var scope domainScope
	err := tx.QueryRow(ctx, `
		SELECT tenant.id::text,entity.id::text
		FROM tenants tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		LIMIT 1`, tenantID, legalEntityID).Scan(&scope.TenantID, &scope.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainScope{}, ErrDomainMetricsInvalid
	}
	if err != nil {
		return domainScope{}, fmt.Errorf("resolve scoped domain entity: %w", err)
	}

	var matched int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM organization_scopes organization_scope
		WHERE organization_scope.tenant_id=$1::uuid
		  AND organization_scope.legal_entity_id=$2::uuid
		  AND organization_scope.id=ANY($3::uuid[])`,
		scope.TenantID, scope.LegalEntityID, organizationScopeIDs,
	).Scan(&matched); err != nil {
		return domainScope{}, fmt.Errorf("verify scoped domain organizations: %w", err)
	}
	if matched != len(organizationScopeIDs) {
		return domainScope{}, ErrDomainMetricsInvalid
	}
	scope.OrganizationScopeID = organizationScopeID
	scope.OrganizationScopeIDs = organizationScopeIDs
	return scope, nil
}

func retainScopedDomainMembers(
	ctx context.Context,
	tx pgx.Tx,
	scope domainScope,
	at time.Time,
	highWater map[string]time.Time,
	results []domainMetricResult,
) (string, error) {
	fingerprint, err := scopedDomainFingerprint(scope, highWater, results)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM metric_runtime_membership_sets
		WHERE source_id IN (
			SELECT source_id
			FROM metric_runtime_membership_sets
			WHERE expires_at<=clock_timestamp()
			ORDER BY expires_at,source_id
			LIMIT 25
		)`); err != nil {
		return "", fmt.Errorf("expire scoped domain memberships: %w", err)
	}

	var sourceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO metric_runtime_membership_sets(
			tenant_id,legal_entity_id,organization_scope_id,definition_revision,
			source_revision,request_fingerprint,generated_at,period_start,period_end,expires_at
		)
		VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$7,$7,
			clock_timestamp()+interval '24 hours'
		)
		ON CONFLICT DO NOTHING
		RETURNING source_id::text`,
		scope.TenantID, scope.LegalEntityID, scope.OrganizationScopeID,
		DomainDefinitionRevision, DomainScopedSourceRevision, fingerprint, at,
	).Scan(&sourceID)

	inserted := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT source_id::text
			FROM metric_runtime_membership_sets
			WHERE tenant_id=$1::uuid
			  AND legal_entity_id=$2::uuid
			  AND organization_scope_id=$3::uuid
			  AND definition_revision=$4
			  AND request_fingerprint=$5
			  AND expires_at>clock_timestamp()
			LIMIT 1`,
			scope.TenantID, scope.LegalEntityID, scope.OrganizationScopeID,
			DomainDefinitionRevision, fingerprint,
		).Scan(&sourceID)
		if err != nil {
			return "", fmt.Errorf("resolve scoped domain membership source: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE metric_runtime_membership_sets
			SET expires_at=GREATEST(expires_at,clock_timestamp()+interval '24 hours')
			WHERE source_id=$1::uuid`, sourceID); err != nil {
			return "", fmt.Errorf("extend scoped domain membership source: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("retain scoped domain membership source: %w", err)
	}
	if !inserted {
		return sourceID, nil
	}

	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(sourceID); err != nil || !sourceUUID.Valid {
		return "", ErrDomainMetricsInvalid
	}
	rows := make([][]any, 0)
	for _, result := range results {
		for _, member := range result.Members {
			var memberUUID, targetUUID pgtype.UUID
			if err := memberUUID.Scan(member.MemberID); err != nil || !memberUUID.Valid {
				return "", ErrDomainMetricsInvalid
			}
			if err := targetUUID.Scan(member.TargetID); err != nil || !targetUUID.Valid {
				return "", ErrDomainMetricsInvalid
			}
			organizationUUID, err := optionalDomainUUID(member.OrganizationScopeID)
			if err != nil {
				return "", ErrDomainMetricsInvalid
			}
			rows = append(rows, []any{
				sourceUUID,
				result.Definition.ID,
				DomainDefinitionRevision,
				memberUUID,
				member.TargetType,
				targetUUID,
				organizationUUID,
				member.Title,
				member.State,
			})
		}
	}
	if len(rows) > 0 {
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"metric_runtime_memberships"},
			[]string{"source_id", "metric_id", "definition_revision", "member_id", "target_type", "target_id", "organization_scope_id", "target_title", "state"},
			pgx.CopyFromRows(rows),
		); err != nil {
			return "", fmt.Errorf("retain scoped domain metric members: %w", err)
		}
	}
	return sourceID, nil
}

func scopedDomainFingerprint(scope domainScope, highWater map[string]time.Time, results []domainMetricResult) (string, error) {
	type fingerprintResult struct {
		MetricID   string   `json:"metric_id"`
		Population int      `json:"population"`
		Unknown    int      `json:"unknown"`
		Members    []string `json:"members"`
	}
	payload := struct {
		OrganizationScopeID  string              `json:"organization_scope_id"`
		OrganizationScopeIDs []string            `json:"organization_scope_ids"`
		HighWater            map[string]string   `json:"high_water"`
		Results              []fingerprintResult `json:"results"`
	}{
		OrganizationScopeID:  scope.OrganizationScopeID,
		OrganizationScopeIDs: append([]string(nil), scope.OrganizationScopeIDs...),
		HighWater:            make(map[string]string, len(highWater)),
		Results:              make([]fingerprintResult, 0, len(results)),
	}
	sort.Strings(payload.OrganizationScopeIDs)
	for key, value := range highWater {
		payload.HighWater[key] = value.UTC().Format(time.RFC3339Nano)
	}
	for _, result := range results {
		item := fingerprintResult{
			MetricID:   result.Definition.ID,
			Population: result.Population,
			Unknown:    result.Unknown,
			Members:    make([]string, 0, len(result.Members)),
		}
		for _, member := range result.Members {
			item.Members = append(item.Members, member.MemberID+":"+member.OrganizationScopeID+":"+member.State)
		}
		sort.Strings(item.Members)
		payload.Results = append(payload.Results, item)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode scoped domain fingerprint: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func optionalDomainUUID(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil || !parsed.Valid {
		return nil, ErrDomainMetricsInvalid
	}
	return parsed, nil
}

func normalizeDomainScopeIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ ScopedDomainReader = (*DomainRepository)(nil)
