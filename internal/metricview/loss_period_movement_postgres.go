//go:build postgres

package metricview

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func lossCurrencyFlows(values []lossCurrencyAggregate) ([]LossCurrencyFlow, *MoneyValue, error) {
	result := make([]LossCurrencyFlow, 0, len(values))
	for _, value := range values {
		gross, err := NewMoneyValue(value.GrossMinor, value.Currency)
		if err != nil {
			return nil, nil, err
		}
		recovery, err := NewMoneyValue(value.RecoveryMinor, value.Currency)
		if err != nil {
			return nil, nil, err
		}
		reversal, err := NewMoneyValue(value.ReversalMinor, value.Currency)
		if err != nil {
			return nil, nil, err
		}
		net, err := NewMoneyValue(value.GrossMinor-value.RecoveryMinor+value.ReversalMinor, value.Currency)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, LossCurrencyFlow{
			Currency: value.Currency, Gross: gross, Recovery: recovery, Reversal: reversal, Net: net,
			LossEventCount: value.LossEventCount,
			RecoveryEventCount: value.RecoveryEventCount,
			ReversalEventCount: value.ReversalEventCount,
		})
	}
	if len(result) == 1 {
		net := result[0].Net
		return result, &net, nil
	}
	return result, nil, nil
}

func buildLossPeriodComparison(
	current lossPeriodAggregate,
	previous lossPeriodAggregate,
	periodStart time.Time,
	periodEnd time.Time,
) (LossPeriodComparison, error) {
	currencies, netLoss, err := lossCurrencyFlows(previous.Currencies)
	if err != nil {
		return LossPeriodComparison{}, err
	}
	comparison := LossPeriodComparison{
		PeriodStart: periodStart,
		PeriodEnd: periodEnd,
		EventCount: previous.EventCount,
		ContributingLossCount: previous.ContributingLossCount,
		MixedCurrencies: len(previous.Currencies) > 1,
		NetLoss: netLoss,
		Currencies: currencies,
		EventDelta: current.EventCount - previous.EventCount,
		Direction: TrendUnknown,
		ComparisonQuality: ComparisonLimited,
	}
	currentCurrency, currentNet, currentComparable := comparableLossNet(current.Currencies)
	previousCurrency, previousNet, previousComparable := comparableLossNet(previous.Currencies)
	switch {
	case len(current.Currencies) == 0 && len(previous.Currencies) == 0:
		comparison.Direction = TrendUnchanged
		comparison.ComparisonQuality = ComparisonComplete
	case currentComparable && len(previous.Currencies) == 0:
		delta, err := NewMoneyValue(currentNet, currentCurrency)
		if err != nil {
			return LossPeriodComparison{}, err
		}
		comparison.NetDelta = &delta
		comparison.Direction = lossFlowDirection(currentNet)
		comparison.ComparisonQuality = ComparisonComplete
	case previousComparable && len(current.Currencies) == 0:
		delta, err := NewMoneyValue(-previousNet, previousCurrency)
		if err != nil {
			return LossPeriodComparison{}, err
		}
		comparison.NetDelta = &delta
		comparison.Direction = lossFlowDirection(-previousNet)
		comparison.ComparisonQuality = ComparisonComplete
	case currentComparable && previousComparable && currentCurrency == previousCurrency:
		deltaValue := currentNet - previousNet
		delta, err := NewMoneyValue(deltaValue, currentCurrency)
		if err != nil {
			return LossPeriodComparison{}, err
		}
		comparison.NetDelta = &delta
		comparison.Direction = lossFlowDirection(deltaValue)
		comparison.ComparisonQuality = ComparisonComplete
	}
	return comparison, nil
}

func comparableLossNet(values []lossCurrencyAggregate) (string, int64, bool) {
	if len(values) != 1 {
		return "", 0, false
	}
	value := values[0]
	return value.Currency, value.GrossMinor - value.RecoveryMinor + value.ReversalMinor, true
}

func lossFlowDirection(delta int64) TrendDirection {
	switch {
	case delta < 0:
		return TrendImproved
	case delta > 0:
		return TrendWorsened
	default:
		return TrendUnchanged
	}
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

func loadLossFlowPoints(
	ctx context.Context,
	tx pgx.Tx,
	scope domainScope,
	periodStart time.Time,
	periodEnd time.Time,
	resolution LossFlowResolution,
	aggregate lossPeriodAggregate,
) ([]LossFlowPoint, error) {
	truncation := "day"
	if resolution == LossFlowResolutionWeek {
		truncation = "week"
	} else if resolution != LossFlowResolutionDay {
		return nil, ErrLossPeriodInvalid
	}

	rows, err := tx.Query(ctx, `
		WITH flows AS (
			SELECT loss.id AS loss_id,
			       loss.occurred_at AS flow_at,
			       loss.currency,
			       loss.gross_amount_minor::bigint AS gross_minor,
			       0::bigint AS recovery_minor,
			       0::bigint AS reversal_minor,
			       1::bigint AS loss_event_count,
			       0::bigint AS recovery_event_count,
			       0::bigint AS reversal_event_count
			FROM operational_losses loss
			WHERE loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.status='ACTIVE'
			  AND loss.occurred_at>=$3
			  AND loss.occurred_at<=$4
			  AND (NOT $5::boolean OR loss.organization_scope_id=ANY($6::uuid[]))
			UNION ALL
			SELECT loss.id,
			       recovery.recovered_at,
			       loss.currency,
			       0::bigint,
			       CASE WHEN recovery.kind='RECOVERY' THEN recovery.amount_minor ELSE 0 END::bigint,
			       CASE WHEN recovery.kind='REVERSAL' THEN recovery.amount_minor ELSE 0 END::bigint,
			       0::bigint,
			       CASE WHEN recovery.kind='RECOVERY' THEN 1 ELSE 0 END::bigint,
			       CASE WHEN recovery.kind='REVERSAL' THEN 1 ELSE 0 END::bigint
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
		), bucketed AS (
			SELECT (date_trunc($7,flow_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') AS bucket_start,
			       currency,
			       sum(gross_minor)::bigint AS gross_minor,
			       sum(recovery_minor)::bigint AS recovery_minor,
			       sum(reversal_minor)::bigint AS reversal_minor,
			       sum(loss_event_count)::bigint AS loss_event_count,
			       sum(recovery_event_count)::bigint AS recovery_event_count,
			       sum(reversal_event_count)::bigint AS reversal_event_count
			FROM flows
			GROUP BY bucket_start,currency
		), contributors AS (
			SELECT (date_trunc($7,flow_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC') AS bucket_start,
			       count(DISTINCT loss_id)::bigint AS contributor_count
			FROM flows
			GROUP BY bucket_start
		)
		SELECT bucket.bucket_start,
		       bucket.currency,
		       bucket.gross_minor,
		       bucket.recovery_minor,
		       bucket.reversal_minor,
		       bucket.loss_event_count,
		       bucket.recovery_event_count,
		       bucket.reversal_event_count,
		       contributor.contributor_count
		FROM bucketed bucket
		JOIN contributors contributor USING(bucket_start)
		ORDER BY bucket.bucket_start,bucket.currency`,
		scope.TenantID, scope.LegalEntityID, periodStart, periodEnd,
		scope.OrganizationScopeID != "", scope.OrganizationScopeIDs, truncation,
	)
	if err != nil {
		return nil, fmt.Errorf("load Loss flow points: %w", err)
	}
	defer rows.Close()

	byStart := make(map[int64]*LossFlowPoint)
	for rows.Next() {
		var bucketStart time.Time
		var currency string
		var grossMinor, recoveryMinor, reversalMinor int64
		var lossEvents, recoveryEvents, reversalEvents, contributors int
		if err := rows.Scan(
			&bucketStart, &currency,
			&grossMinor, &recoveryMinor, &reversalMinor,
			&lossEvents, &recoveryEvents, &reversalEvents, &contributors,
		); err != nil {
			return nil, fmt.Errorf("scan Loss flow point: %w", err)
		}
		bucketStart = bucketStart.UTC()
		key := bucketStart.UnixNano()
		point := byStart[key]
		if point == nil {
			point = &LossFlowPoint{
				Start: bucketStart,
				ContributingLossCount: contributors,
				Currencies: make([]LossCurrencyFlow, 0, 1),
			}
			byStart[key] = point
		}
		gross, err := NewMoneyValue(grossMinor, currency)
		if err != nil {
			return nil, err
		}
		recovery, err := NewMoneyValue(recoveryMinor, currency)
		if err != nil {
			return nil, err
		}
		reversal, err := NewMoneyValue(reversalMinor, currency)
		if err != nil {
			return nil, err
		}
		net, err := NewMoneyValue(grossMinor-recoveryMinor+reversalMinor, currency)
		if err != nil {
			return nil, err
		}
		point.LossEventCount += lossEvents
		point.Currencies = append(point.Currencies, LossCurrencyFlow{
			Currency: currency, Gross: gross, Recovery: recovery, Reversal: reversal, Net: net,
			LossEventCount: lossEvents,
			RecoveryEventCount: recoveryEvents,
			ReversalEventCount: reversalEvents,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Loss flow points: %w", err)
	}

	first := lossFlowBucketStart(periodStart, resolution)
	result := make([]LossFlowPoint, 0)
	for bucketStart := first; !bucketStart.After(periodEnd); bucketStart = lossFlowNextBucket(bucketStart, resolution) {
		next := lossFlowNextBucket(bucketStart, resolution)
		point := LossFlowPoint{
			Start: lossTimeMax(bucketStart, periodStart),
			End: lossTimeMin(next.Add(-time.Nanosecond), periodEnd),
			Currencies: []LossCurrencyFlow{},
		}
		if stored := byStart[bucketStart.UnixNano()]; stored != nil {
			point.LossEventCount = stored.LossEventCount
			point.ContributingLossCount = stored.ContributingLossCount
			point.Currencies = stored.Currencies
			point.MixedCurrencies = len(point.Currencies) > 1
			if len(point.Currencies) == 1 {
				net := point.Currencies[0].Net
				point.NetLoss = &net
			}
		} else if len(aggregate.Currencies) == 1 {
			zero, err := NewMoneyValue(0, aggregate.Currencies[0].Currency)
			if err != nil {
				return nil, err
			}
			point.NetLoss = &zero
		}
		result = append(result, point)
	}
	return result, nil
}

func lossFlowBucketStart(value time.Time, resolution LossFlowResolution) time.Time {
	day := time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if resolution == LossFlowResolutionDay {
		return day
	}
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

func lossFlowNextBucket(value time.Time, resolution LossFlowResolution) time.Time {
	if resolution == LossFlowResolutionWeek {
		return value.AddDate(0, 0, 7)
	}
	return value.AddDate(0, 0, 1)
}

func lossTimeMin(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func lossTimeMax(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func lossFlowPointsMatchAggregate(points []LossFlowPoint, aggregate lossPeriodAggregate) bool {
	eventCount := 0
	type totals struct {
		gross, recovery, reversal                  int64
		lossEvents, recoveryEvents, reversalEvents int
	}
	byCurrency := make(map[string]*totals)
	for _, point := range points {
		eventCount += point.LossEventCount
		for _, currency := range point.Currencies {
			value := byCurrency[currency.Currency]
			if value == nil {
				value = &totals{}
				byCurrency[currency.Currency] = value
			}
			gross, err1 := parseMoneyMinorUnits(currency.Gross)
			recovery, err2 := parseMoneyMinorUnits(currency.Recovery)
			reversal, err3 := parseMoneyMinorUnits(currency.Reversal)
			if err1 != nil || err2 != nil || err3 != nil {
				return false
			}
			value.gross += gross
			value.recovery += recovery
			value.reversal += reversal
			value.lossEvents += currency.LossEventCount
			value.recoveryEvents += currency.RecoveryEventCount
			value.reversalEvents += currency.ReversalEventCount
		}
	}
	if eventCount != aggregate.EventCount || len(byCurrency) != len(aggregate.Currencies) {
		return false
	}
	for _, currency := range aggregate.Currencies {
		value := byCurrency[currency.Currency]
		if value == nil ||
			value.gross != currency.GrossMinor ||
			value.recovery != currency.RecoveryMinor ||
			value.reversal != currency.ReversalMinor ||
			value.lossEvents != currency.LossEventCount ||
			value.recoveryEvents != currency.RecoveryEventCount ||
			value.reversalEvents != currency.ReversalEventCount {
			return false
		}
	}
	return true
}

