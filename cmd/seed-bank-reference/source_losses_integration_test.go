//go:build postgres && postgresintegration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"testing/fstest"
	"time"
)

func TestOpsLossOnlyInstallerCreatesEightLossesAndNoSyntheticRecoveries(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	group := syntheticOpsLossGroup()
	for index := 0; index < 8; index++ {
		for month := 1; month <= 12; month++ {
			row := syntheticOpsLossRecord(index, fmt.Sprintf("%02d", month))
			number := 2 + index + 8*(month-1)
			row.SourceRange = fmt.Sprintf("Sheet1!A%d:R%d", number, number)
			group.Records = append(group.Records, row)
		}
	}
	data, err := json.Marshal(sourceRecordManifest{Version: 1, Groups: []sourceRecordGroup{group}})
	if err != nil {
		t.Fatal(err)
	}
	prior := sourceRecordFiles
	sourceRecordFiles = fstest.MapFS{"source_records_ops_loss.json": &fstest.MapFile{Data: data}}
	t.Cleanup(func() { sourceRecordFiles = prior })

	first, err := installSourceLossesOnly(ctx, cfg, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Losses != 8 || first.LossesCreated != 8 {
		t.Fatalf("96 monthly rows must create 8 canonical events: %+v", first)
	}
	second, err := installSourceLossesOnly(ctx, cfg, pool, seed)
	if err != nil || second.Losses != 8 || second.LossesCreated != 0 {
		t.Fatalf("repeat must be read-only: %+v err=%v", second, err)
	}
	var losses, recoveries int
	var net, gross int64
	if err = pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM operational_losses WHERE code LIKE 'OPSL-%'),
		  (SELECT count(*) FROM operational_loss_recoveries),
		  (SELECT COALESCE(sum(gross_amount_minor),0) FROM operational_losses WHERE code LIKE 'OPSL-%'),
		  (SELECT COALESCE(sum(gross_amount_minor),0) FROM operational_losses WHERE code LIKE 'OPSL-%')
	`).Scan(&losses, &recoveries, &gross, &net); err != nil {
		t.Fatal(err)
	}
	if losses != 8 || recoveries != 0 || gross <= 0 || net != gross {
		t.Fatalf("unexpected Loss ledger counts/amounts: losses=%d recoveries=%d", losses, recoveries)
	}
	var currency string
	var occurred, discovered time.Time
	if err = pool.QueryRow(ctx, `
		SELECT currency, occurred_at, discovered_at
		FROM operational_losses WHERE code LIKE 'OPSL-%'
		ORDER BY code LIMIT 1
	`).Scan(&currency, &occurred, &discovered); err != nil {
		t.Fatal(err)
	}
	if currency != "NGN" ||
		!occurred.Equal(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)) ||
		!discovered.Equal(time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("loss occurrence/recognition dates or currency were replaced by import time")
	}

	if _, err = pool.Exec(ctx, `
		UPDATE operational_losses SET title='Operator edited title'
		WHERE code=(SELECT min(code) FROM operational_losses WHERE code LIKE 'OPSL-%')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err = installSourceLossesOnly(ctx, cfg, pool, seed); err == nil {
		t.Fatal("rerun should reject edited canonical Loss instead of overwriting it")
	}
}

func TestOpsLossOnlyInstallerRejectsWrongTenantBeforeWriting(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	prior := sourceRecordFiles
	sourceRecordFiles = fstest.MapFS{}
	t.Cleanup(func() { sourceRecordFiles = prior })
	seed.LegalEntityID = "00000000-0000-4000-8000-000000000099"
	if _, err := installSourceLossesOnly(context.Background(), cfg, pool, seed); err == nil {
		t.Fatal("must never import source Loss into another legal entity")
	}
}


func TestOpsLossOnlyInstallerRejectsIncompleteSourceWithoutWrites(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	records := make([]sourceRecord, 0, 7)
	for index := 0; index < 7; index++ {
		records = append(records, syntheticOpsLossRecord(index, "Jan"))
	}
	group := syntheticOpsLossGroup()
	group.Records = records
	data, err := json.Marshal(sourceRecordManifest{Version: 1, Groups: []sourceRecordGroup{group}})
	if err != nil {
		t.Fatal(err)
	}
	prior := sourceRecordFiles
	sourceRecordFiles = fstest.MapFS{"source_records_ops_loss.json": &fstest.MapFile{Data: data}}
	t.Cleanup(func() { sourceRecordFiles = prior })

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var before, after int
	query := `SELECT count(*) FROM operational_losses
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code LIKE 'OPSL-%'`
	if err := pool.QueryRow(ctx, query, seed.TenantID, seed.LegalEntityID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = installSourceLossesOnly(ctx, cfg, pool, seed); err == nil {
		t.Fatal("truncated historical Loss source unexpectedly passed")
	}
	if err := pool.QueryRow(ctx, query, seed.TenantID, seed.LegalEntityID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("incomplete source created %d partial financial records", after-before)
	}
}
