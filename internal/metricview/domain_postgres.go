//go:build postgres

package metricview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DomainSnapshotInterval  = 5 * time.Minute
	DomainSnapshotFreshness = 15 * time.Minute
)

type DomainRepository struct {
	pool *pgxpool.Pool
}

type DomainMaintainer struct {
	Repository *ObservationRepository
}

type domainScope struct {
	TenantID             string
	LegalEntityID        string
	OrganizationScopeID  string
	OrganizationScopeIDs []string
}

type domainMetricMember struct {
	MemberID   string
	TargetType string
	TargetID   string
	Title      string
	State      string
}

type domainMetricResult struct {
	Definition Definition
	Population int
	Unknown    int
	Members    []domainMetricMember
}

func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

func NewDomainMaintainer(repository *ObservationRepository) *DomainMaintainer {
	return &DomainMaintainer{Repository: repository}
}

func (r *DomainRepository) LatestDomainMetrics(ctx context.Context, tenantID, legalEntityID string) (DomainBundle, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	tenantID, legalEntityID = strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return DomainBundle{}, ErrDomainMetricsInvalid
	}
	var bundle DomainBundle
	err := r.pool.QueryRow(ctx, `
		SELECT source.id::text,source.generated_at,source.source_revision,source.definition_revision,entity.id::text
		FROM domain_metric_snapshots source
		JOIN tenants tenant ON tenant.id=source.tenant_id
		JOIN legal_entities entity ON entity.tenant_id=source.tenant_id AND entity.id=source.legal_entity_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		  AND source.definition_revision=$3
		ORDER BY source.generated_at DESC,source.id DESC
		LIMIT 1`, tenantID, legalEntityID, DomainDefinitionRevision).Scan(
		&bundle.SourceID, &bundle.GeneratedAt, &bundle.SourceRevision, &bundle.DefinitionRevision, &bundle.ScopeID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DomainBundle{}, ErrDomainMetricsNotFound
	}
	if err != nil {
		return DomainBundle{}, fmt.Errorf("load latest domain metric source: %w", err)
	}
	bundle.PostureAsOf = bundle.GeneratedAt.UTC()
	bundle.ScopeKind = "LEGAL_ENTITY"
	rows, err := r.pool.Query(ctx, `
		SELECT metric_id,value,condition,freshness,completeness,population,excluded,unknown
		FROM metric_observations
		WHERE source_kind=$1
		  AND source_id=$2::uuid
		  AND definition_revision=$3
		ORDER BY metric_id`, ObservationSourceDomainSnapshot, bundle.SourceID, DomainDefinitionRevision)
	if err != nil {
		return DomainBundle{}, fmt.Errorf("load latest domain metrics: %w", err)
	}
	defer rows.Close()
	bundle.Items = make([]Metric, 0, len(domainDefinitions))
	for rows.Next() {
		var item Metric
		if err := rows.Scan(&item.ID, &item.Value, &item.Condition, &item.Freshness, &item.Completeness, &item.Population, &item.Excluded, &item.Unknown); err != nil {
			return DomainBundle{}, fmt.Errorf("scan latest domain metric: %w", err)
		}
		definition, ok := DomainDefinition(item.ID)
		if !ok {
			return DomainBundle{}, ErrDefinitionMismatch
		}
		item.Label = definition.Label
		item.Unit = definition.Unit
		item.Basis = definition.Basis
		item.Drill = definition.Drill
		item.DefinitionRevision = definition.Revision
		item.SourceRevision = bundle.SourceRevision
		item.GeneratedAt = bundle.GeneratedAt.UTC()
		if time.Since(bundle.GeneratedAt.UTC()) > DomainSnapshotFreshness {
			item.Freshness = oversight.FreshnessStale
		}
		bundle.Items = append(bundle.Items, item)
	}
	if err := rows.Err(); err != nil {
		return DomainBundle{}, err
	}
	if len(bundle.Items) != len(domainDefinitions) {
		return DomainBundle{}, ErrDomainMetricsNotFound
	}
	return bundle, nil
}

var _ DomainReader = (*DomainRepository)(nil)
