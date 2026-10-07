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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LossPeriodRepository struct {
	pool *pgxpool.Pool
}

type lossPeriodAggregate struct {
	EventCount             int
	ContributingLossCount  int
	UnattributedEventCount int
	EventHash              string
	ContributorHash        string
	Currencies             []lossCurrencyAggregate
}

type lossCurrencyAggregate struct {
	Currency           string `json:"currency"`
	GrossMinor         int64  `json:"gross_minor"`
	RecoveryMinor      int64  `json:"recovery_minor"`
	ReversalMinor      int64  `json:"reversal_minor"`
	LossEventCount     int    `json:"loss_event_count"`
	RecoveryEventCount int    `json:"recovery_event_count"`
	ReversalEventCount int    `json:"reversal_event_count"`
}

func NewLossPeriodRepository(pool *pgxpool.Pool) *LossPeriodRepository {
	return &LossPeriodRepository{pool: pool}
}

func (r *LossPeriodRepository) CurrentLossPeriod(
	ctx context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
	periodStart time.Time,
	periodEnd time.Time,
	generatedAt time.Time,
) (LossPeriodBundle, error) {
	if r == nil || r.pool == nil || ctx == nil {
		return LossPeriodBundle{}, ErrLossPeriodInvalid
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	organizationScopeID = strings.TrimSpace(organizationScopeID)
	organizationScopeIDs = normalizeDomainScopeIDs(organizationScopeIDs)
	periodStart = periodStart.UTC()
	periodEnd = periodEnd.UTC()
	generatedAt = generatedAt.UTC()
	if tenantID == "" || legalEntityID == "" || periodStart.IsZero() || periodEnd.IsZero() ||
		generatedAt.IsZero() || periodEnd.Before(periodStart) || generatedAt.Before(periodEnd) {
		return LossPeriodBundle{}, ErrLossPeriodInvalid
	}
	if organizationScopeID == "" && len(organizationScopeIDs) > 0 {
		return LossPeriodBundle{}, ErrLossPeriodInvalid
	}
	if organizationScopeID != "" && !containsString(organizationScopeIDs, organizationScopeID) {
		return LossPeriodBundle{}, ErrLossPeriodInvalid
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return LossPeriodBundle{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	scope, err := resolveLossPeriodScope(
		ctx, tx, tenantID, legalEntityID, organizationScopeID, organizationScopeIDs,
	)
	if err != nil {
		return LossPeriodBundle{}, err
	}
	aggregate, err := loadLossPeriodAggregate(ctx, tx, scope, periodStart, periodEnd)
	if err != nil {
		return LossPeriodBundle{}, err
	}
	sourceID, sourceGeneratedAt, err := retainLossPeriodSnapshot(
		ctx, tx, scope, periodStart, periodEnd, generatedAt, aggregate,
	)
	if err != nil {
		return LossPeriodBundle{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LossPeriodBundle{}, err
	}

	bundle := LossPeriodBundle{
		GeneratedAt:            sourceGeneratedAt,
		PeriodStart:            periodStart,
		PeriodEnd:              periodEnd,
		ScopeID:                scope.LegalEntityID,
		ScopeKind:              "LEGAL_ENTITY",
		SourceID:               sourceID,
		SourceRevision:         LossPeriodSourceRevision,
		DefinitionRevision:     LossPeriodDefinitionRevision,
		EventCount:             aggregate.EventCount,
		ContributingLossCount:  aggregate.ContributingLossCount,
		UnattributedEventCount: aggregate.UnattributedEventCount,
		MixedCurrencies:        len(aggregate.Currencies) > 1,
		Currencies:             make([]LossCurrencyFlow, 0, len(aggregate.Currencies)),
	}
	if organizationScopeID != "" {
		bundle.ScopeID = organizationScopeID
		bundle.ScopeKind = "ORGANIZATION_SCOPE"
	}
	for _, value := range aggregate.Currencies {
		gross, err := NewMoneyValue(value.GrossMinor, value.Currency)
		if err != nil {
			return LossPeriodBundle{}, err
		}
		recovery, err := NewMoneyValue(value.RecoveryMinor, value.Currency)
		if err != nil {
			return LossPeriodBundle{}, err
		}
		reversal, err := NewMoneyValue(value.ReversalMinor, value.Currency)
		if err != nil {
			return LossPeriodBundle{}, err
		}
		net, err := NewMoneyValue(value.GrossMinor-value.RecoveryMinor+value.ReversalMinor, value.Currency)
		if err != nil {
			return LossPeriodBundle{}, err
		}
		bundle.Currencies = append(bundle.Currencies, LossCurrencyFlow{
			Currency:           value.Currency,
			Gross:              gross,
			Recovery:           recovery,
			Reversal:           reversal,
			Net:                net,
			LossEventCount:     value.LossEventCount,
			RecoveryEventCount: value.RecoveryEventCount,
			ReversalEventCount: value.ReversalEventCount,
		})
	}
	if len(bundle.Currencies) == 1 {
		net := bundle.Currencies[0].Net
		bundle.NetLoss = &net
	}
	return bundle, nil
}

func resolveLossPeriodScope(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
) (domainScope, error) {
	if organizationScopeID != "" {
		return resolveScopedDomainScope(ctx, tx, tenantID, legalEntityID, organizationScopeID, organizationScopeIDs)
	}
	var scope domainScope
	err := tx.QueryRow(ctx, `
		SELECT tenant.id::text,entity.id::text
		FROM tenants tenant
		JOIN legal_entities entity ON entity.tenant_id=tenant.id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1)
		  AND (entity.id::text=$2 OR entity.code=$2)
		LIMIT 1`, tenantID, legalEntityID).Scan(&scope.TenantID, &scope.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainScope{}, ErrLossPeriodInvalid
	}
	if err != nil {
		return domainScope{}, fmt.Errorf("resolve Loss period entity: %w", err)
	}
	return scope, nil
}

func loadLossPeriodAggregate(
	ctx context.Context,
	tx pgx.Tx,
	scope domainScope,
	periodStart time.Time,
	periodEnd time.Time,
) (lossPeriodAggregate, error) {
	var rawCurrencies []byte
	var aggregate lossPeriodAggregate
	err := tx.QueryRow(ctx, `
		WITH loss_events AS (
			SELECT loss.id,loss.version,COALESCE(organization.version,0) AS organization_version,
			       loss.title,loss.organization_scope_id,loss.currency,loss.gross_amount_minor
			FROM operational_losses loss
			LEFT JOIN organization_scopes organization
			  ON organization.tenant_id=loss.tenant_id
			 AND organization.legal_entity_id=loss.legal_entity_id
			 AND organization.id=loss.organization_scope_id
			WHERE loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.status='ACTIVE'
			  AND loss.occurred_at>=$3
			  AND loss.occurred_at<=$4
			  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))
		), recovery_events AS (
			SELECT recovery.loss_id AS id,
			       loss.version,
			       COALESCE(organization.version,0) AS organization_version,
			       loss.title,
			       loss.organization_scope_id,
			       loss.currency,
			       recovery.kind,
			       recovery.amount_minor
			FROM operational_loss_recoveries recovery
			JOIN operational_losses loss
			  ON loss.tenant_id=recovery.tenant_id
			 AND loss.legal_entity_id=recovery.legal_entity_id
			 AND loss.id=recovery.loss_id
			LEFT JOIN organization_scopes organization
			  ON organization.tenant_id=loss.tenant_id
			 AND organization.legal_entity_id=loss.legal_entity_id
			 AND organization.id=loss.organization_scope_id
			WHERE loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.status='ACTIVE'
			  AND recovery.recovered_at>=$3
			  AND recovery.recovered_at<=$4
			  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))
		), contributors AS (
			SELECT id,version,organization_version FROM loss_events
			UNION
			SELECT id,version,organization_version FROM recovery_events
		), currencies AS (
			SELECT currency FROM loss_events
			UNION
			SELECT currency FROM recovery_events
		), loss_totals AS (
			SELECT currency,
			       sum(gross_amount_minor)::bigint AS gross_minor,
			       count(*)::bigint AS loss_event_count
			FROM loss_events
			GROUP BY currency
		), recovery_totals AS (
			SELECT currency,
			       COALESCE(sum(amount_minor) FILTER (WHERE kind='RECOVERY'),0)::bigint AS recovery_minor,
			       COALESCE(sum(amount_minor) FILTER (WHERE kind='REVERSAL'),0)::bigint AS reversal_minor,
			       count(*) FILTER (WHERE kind='RECOVERY')::bigint AS recovery_event_count,
			       count(*) FILTER (WHERE kind='REVERSAL')::bigint AS reversal_event_count
			FROM recovery_events
			GROUP BY currency
		), currency_flows AS (
			SELECT currency.currency,
			       COALESCE(loss.gross_minor,0)::bigint AS gross_minor,
			       COALESCE(recovery.recovery_minor,0)::bigint AS recovery_minor,
			       COALESCE(recovery.reversal_minor,0)::bigint AS reversal_minor,
			       COALESCE(loss.loss_event_count,0)::bigint AS loss_event_count,
			       COALESCE(recovery.recovery_event_count,0)::bigint AS recovery_event_count,
			       COALESCE(recovery.reversal_event_count,0)::bigint AS reversal_event_count
			FROM currencies currency
			LEFT JOIN loss_totals loss USING(currency)
			LEFT JOIN recovery_totals recovery USING(currency)
		)
		SELECT
		  (SELECT count(*) FROM loss_events),
		  (SELECT count(*) FROM contributors),
		  (SELECT count(*) FROM loss_events WHERE organization_scope_id IS NULL),
		  (SELECT md5(COALESCE(string_agg(id::text||':'||version::text||':'||organization_version::text,',' ORDER BY id),'')) FROM loss_events),
		  (SELECT md5(COALESCE(string_agg(id::text||':'||version::text||':'||organization_version::text,',' ORDER BY id),'')) FROM contributors),
		  COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'currency',currency,
				'gross_minor',gross_minor,
				'recovery_minor',recovery_minor,
				'reversal_minor',reversal_minor,
				'loss_event_count',loss_event_count,
				'recovery_event_count',recovery_event_count,
				'reversal_event_count',reversal_event_count
			) ORDER BY currency)
			FROM currency_flows
		  ),'[]'::jsonb)`,
		scope.TenantID, scope.LegalEntityID, periodStart, periodEnd,
		scope.OrganizationScopeID != "", scope.OrganizationScopeIDs,
	).Scan(
		&aggregate.EventCount,
		&aggregate.ContributingLossCount,
		&aggregate.UnattributedEventCount,
		&aggregate.EventHash,
		&aggregate.ContributorHash,
		&rawCurrencies,
	)
	if err != nil {
		return lossPeriodAggregate{}, fmt.Errorf("load Loss period aggregate: %w", err)
	}
	if err := json.Unmarshal(rawCurrencies, &aggregate.Currencies); err != nil {
		return lossPeriodAggregate{}, fmt.Errorf("decode Loss period currency flows: %w", err)
	}
	sort.SliceStable(aggregate.Currencies, func(i, j int) bool {
		return aggregate.Currencies[i].Currency < aggregate.Currencies[j].Currency
	})
	return aggregate, nil
}

func retainLossPeriodSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	scope domainScope,
	periodStart time.Time,
	periodEnd time.Time,
	generatedAt time.Time,
	aggregate lossPeriodAggregate,
) (string, time.Time, error) {
	fingerprint, err := lossPeriodFingerprint(scope, periodStart, periodEnd, aggregate)
	if err != nil {
		return "", time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM metric_runtime_membership_sets
		WHERE definition_revision=$1
		  AND tenant_id=$2::uuid
		  AND legal_entity_id=$3::uuid
		  AND organization_scope_id IS NOT DISTINCT FROM NULLIF($4,'')::uuid
		  AND request_fingerprint=$5
		  AND expires_at<=clock_timestamp()`,
		LossPeriodDefinitionRevision, scope.TenantID, scope.LegalEntityID, scope.OrganizationScopeID, fingerprint,
	); err != nil {
		return "", time.Time{}, fmt.Errorf("expire matching Loss period snapshot: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM metric_runtime_membership_sets
		WHERE source_id IN (
			SELECT source_id
			FROM metric_runtime_membership_sets
			WHERE definition_revision=$1
			  AND expires_at<=clock_timestamp()
			ORDER BY expires_at,source_id
			LIMIT 25
		)`, LossPeriodDefinitionRevision); err != nil {
		return "", time.Time{}, fmt.Errorf("expire old Loss period snapshots: %w", err)
	}

	var sourceID string
	var sourceGeneratedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO metric_runtime_membership_sets(
			tenant_id,legal_entity_id,organization_scope_id,definition_revision,
			source_revision,request_fingerprint,generated_at,period_start,period_end,expires_at
		)
		VALUES(
			$1::uuid,$2::uuid,NULLIF($3,'')::uuid,$4,$5,$6,$7::timestamptz,$8::timestamptz,$9::timestamptz,
			$7::timestamptz+interval '24 hours'
		)
		ON CONFLICT DO NOTHING
		RETURNING source_id::text,generated_at`,
		scope.TenantID, scope.LegalEntityID, scope.OrganizationScopeID,
		LossPeriodDefinitionRevision, LossPeriodSourceRevision, fingerprint,
		generatedAt, periodStart, periodEnd,
	).Scan(&sourceID, &sourceGeneratedAt)
	inserted := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT source_id::text,generated_at
			FROM metric_runtime_membership_sets
			WHERE tenant_id=$1::uuid
			  AND legal_entity_id=$2::uuid
			  AND organization_scope_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid
			  AND definition_revision=$4
			  AND request_fingerprint=$5
			  AND expires_at>clock_timestamp()
			LIMIT 1`,
			scope.TenantID, scope.LegalEntityID, scope.OrganizationScopeID,
			LossPeriodDefinitionRevision, fingerprint,
		).Scan(&sourceID, &sourceGeneratedAt)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("resolve Loss period snapshot: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE metric_runtime_membership_sets
			SET expires_at=GREATEST(expires_at,$2::timestamptz+interval '24 hours')
			WHERE source_id=$1::uuid`, sourceID, generatedAt); err != nil {
			return "", time.Time{}, fmt.Errorf("extend Loss period snapshot: %w", err)
		}
	} else if err != nil {
		return "", time.Time{}, fmt.Errorf("retain Loss period snapshot: %w", err)
	}
	if !inserted {
		return sourceID, sourceGeneratedAt.UTC(), nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO metric_runtime_memberships(
			source_id,metric_id,definition_revision,member_id,target_type,target_id,
			organization_scope_id,target_title,state
		)
		SELECT $7::uuid,$8,$9,loss.id,'LOSS',loss.id,loss.organization_scope_id,loss.title,'OCCURRED'
		FROM operational_losses loss
		WHERE loss.tenant_id=$1::uuid
		  AND loss.legal_entity_id=$2::uuid
		  AND loss.status='ACTIVE'
		  AND loss.occurred_at>=$3
		  AND loss.occurred_at<=$4
		  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))`,
		scope.TenantID, scope.LegalEntityID, periodStart, periodEnd,
		scope.OrganizationScopeID != "", scope.OrganizationScopeIDs,
		sourceID, LossPeriodMetricEvents, LossPeriodDefinitionRevision,
	); err != nil {
		return "", time.Time{}, fmt.Errorf("retain Loss event members: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		WITH contributors AS (
			SELECT loss.id,loss.title,loss.organization_scope_id,true AS gross,false AS recovery,false AS reversal
			FROM operational_losses loss
			WHERE loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.status='ACTIVE'
			  AND loss.occurred_at>=$3
			  AND loss.occurred_at<=$4
			  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))
			UNION ALL
			SELECT loss.id,loss.title,loss.organization_scope_id,false,
			       recovery.kind='RECOVERY',recovery.kind='REVERSAL'
			FROM operational_loss_recoveries recovery
			JOIN operational_losses loss
			  ON loss.tenant_id=recovery.tenant_id
			 AND loss.legal_entity_id=recovery.legal_entity_id
			 AND loss.id=recovery.loss_id
			WHERE loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.status='ACTIVE'
			  AND recovery.recovered_at>=$3
			  AND recovery.recovered_at<=$4
			  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))
		), rolled AS (
			SELECT id,title,organization_scope_id,
			       bool_or(gross) AS has_gross,
			       bool_or(recovery) AS has_recovery,
			       bool_or(reversal) AS has_reversal
			FROM contributors
			GROUP BY id,title,organization_scope_id
		)
		INSERT INTO metric_runtime_memberships(
			source_id,metric_id,definition_revision,member_id,target_type,target_id,
			organization_scope_id,target_title,state
		)
		SELECT $7::uuid,$8,$9,id,'LOSS',id,organization_scope_id,title,
		       CASE
		         WHEN has_gross AND (has_recovery OR has_reversal) THEN 'GROSS_AND_RECOVERY_ACTIVITY'
		         WHEN has_recovery AND has_reversal THEN 'RECOVERY_AND_REVERSAL'
		         WHEN has_reversal THEN 'REVERSAL'
		         WHEN has_recovery THEN 'RECOVERY'
		         ELSE 'GROSS'
		       END
		FROM rolled`,
		scope.TenantID, scope.LegalEntityID, periodStart, periodEnd,
		scope.OrganizationScopeID != "", scope.OrganizationScopeIDs,
		sourceID, LossPeriodMetricNet, LossPeriodDefinitionRevision,
	); err != nil {
		return "", time.Time{}, fmt.Errorf("retain net Loss members: %w", err)
	}
	return sourceID, sourceGeneratedAt.UTC(), nil
}

func lossPeriodFingerprint(
	scope domainScope,
	periodStart time.Time,
	periodEnd time.Time,
	aggregate lossPeriodAggregate,
) (string, error) {
	payload := struct {
		OrganizationScopeID  string                  `json:"organization_scope_id"`
		OrganizationScopeIDs []string                `json:"organization_scope_ids"`
		PeriodStart          string                  `json:"period_start"`
		PeriodEnd            string                  `json:"period_end"`
		EventCount           int                     `json:"event_count"`
		ContributorCount     int                     `json:"contributor_count"`
		UnattributedCount    int                     `json:"unattributed_count"`
		EventHash            string                  `json:"event_hash"`
		ContributorHash      string                  `json:"contributor_hash"`
		Currencies           []lossCurrencyAggregate `json:"currencies"`
	}{
		OrganizationScopeID:  scope.OrganizationScopeID,
		OrganizationScopeIDs: append([]string(nil), scope.OrganizationScopeIDs...),
		PeriodStart:          periodStart.UTC().Format(time.RFC3339Nano),
		PeriodEnd:            periodEnd.UTC().Format(time.RFC3339Nano),
		EventCount:           aggregate.EventCount,
		ContributorCount:     aggregate.ContributingLossCount,
		UnattributedCount:    aggregate.UnattributedEventCount,
		EventHash:            aggregate.EventHash,
		ContributorHash:      aggregate.ContributorHash,
		Currencies:           append([]lossCurrencyAggregate(nil), aggregate.Currencies...),
	}
	sort.Strings(payload.OrganizationScopeIDs)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode Loss period fingerprint: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

var _ LossPeriodReader = (*LossPeriodRepository)(nil)
