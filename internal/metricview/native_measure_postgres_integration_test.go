//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNativeMeasureDatabaseAcceptsSignedMoneyAndRejectsCurrencyOnCount(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	tenantID := mustNativeMeasureID(t)
	entityID := mustNativeMeasureID(t)
	moneyObservationID := mustNativeMeasureID(t)
	moneySourceID := mustNativeMeasureID(t)
	now := time.Now().UTC().Truncate(time.Second)
	bucketDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if _, err := tx.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Native Measure Test')`,
		tenantID, "native-measure-"+tenantID[len(tenantID)-8:]); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'Native Measure Nigeria','NG',$4)`,
		entityID, tenantID, "NM-"+entityID[len(entityID)-8:], now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO metric_definitions(
			metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
			drill_workspace,drill_filter,drill_consistency
		) VALUES(
			'native_money_test','native-money-test-v1','Native money test','MONEY','PERIOD_FLOW',
			'NO_CONDITION','SUM_SAME_CURRENCY','losses','native-money-test','CURRENT_STATE'
		)`); err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO metric_observation_daily_rollups(
			tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date,
			observation_id,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,
			value,condition,currency,member_count,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,'native_money_test','native-money-test-v1',$3::date,
			$4::uuid,$5::uuid,'native-money-test-v1','{}'::jsonb,
			$6,$3::date,$6,$6,
			-500,'NEUTRAL','NGN',2,'CURRENT','COMPLETE',2,0,0
		)`,
		tenantID, entityID, bucketDate, moneyObservationID, moneySourceID, now); err != nil {
		t.Fatal(err)
	}

	var value int64
	var currency string
	var memberCount int64
	if err := tx.QueryRow(ctx, `
		SELECT value,currency,member_count
		FROM metric_observation_daily_rollups
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		  AND metric_id='native_money_test' AND definition_revision='native-money-test-v1'`,
		tenantID, entityID).Scan(&value, &currency, &memberCount); err != nil {
		t.Fatal(err)
	}
	if value != -500 || currency != "NGN" || memberCount != 2 {
		t.Fatalf("money row value=%d currency=%q members=%d", value, currency, memberCount)
	}

	if _, err := tx.Exec(ctx, "SAVEPOINT invalid_count_measure"); err != nil {
		t.Fatal(err)
	}
	_, invalidErr := tx.Exec(ctx, `
		INSERT INTO metric_observation_daily_rollups(
			tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date,
			observation_id,source_id,source_revision,source_high_water,
			generated_at,period_start,period_end,posture_as_of,
			value,condition,currency,member_count,freshness,completeness,population,excluded,unknown
		) VALUES(
			$1::uuid,$2::uuid,'critical_high_open',$3,$4::date,
			$5::uuid,$6::uuid,'count-test','{}'::jsonb,
			$7,$4::date,$7,$7,
			1,'CLEAR','NGN',1,'CURRENT','COMPLETE',1,0,0
		)`,
		tenantID, entityID, HomeDefinitionRevision, bucketDate,
		mustNativeMeasureID(t), mustNativeMeasureID(t), now)
	if invalidErr == nil {
		t.Fatal("COUNT metric with currency unexpectedly persisted")
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT invalid_count_measure"); err != nil {
		t.Fatal(err)
	}
}

func mustNativeMeasureID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
