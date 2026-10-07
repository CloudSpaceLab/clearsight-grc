//go:build postgres && postgresintegration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPersistedITExceptionsReconcileIntoCurrentERMRisksRepeatSafely(t *testing.T) {
	pool, _, seed := sampleTestSetup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	insertPersistedITException(t, ctx, pool, seed.TenantID, seed.LegalEntityID, "it-risk-exceptions-row-2", "072", "Azure Infrastructure_2025", "User Access Management", "Identity and Access Risk", "Unauthorized access to sensitive corporate resources.", "Azure portal")
	insertPersistedITException(t, ctx, pool, seed.TenantID, seed.LegalEntityID, "it-risk-exceptions-row-3", "082", "Azure Infrastructure_2025", "User Access Management", "Identity and Access Risk", "Unauthorized or unmanaged devices can access corporate resources.", "Azure portal")

	var matterCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM matters WHERE trigger_type='SOURCE_REGISTER_IMPORT'`).Scan(&matterCount); err != nil {
		t.Fatal(err)
	}
	if matterCount != 2 {
		t.Fatalf("source Matter count=%d want 2", matterCount)
	}

	first, err := reconcilePersistedSourceRisks(ctx, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Risks != 2 || first.RiskAssessments != 2 {
		t.Fatalf("first reconciliation=%+v", first)
	}

	rows, err := pool.Query(ctx, `
		SELECT r.code,r.name,r.category,r.impact,r.status,r.version,
		       a.dimensions->>'risk_level',a.appetite_position,a.assessed_at,a.assessed_by IS NULL
		FROM risks r
		JOIN risk_assessments a
		  ON a.tenant_id=r.tenant_id
		 AND a.legal_entity_id=r.legal_entity_id
		 AND a.risk_id=r.id
		 AND a.risk_version=r.version
		WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid
		ORDER BY r.code`, seed.TenantID, seed.LegalEntityID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	expectedCodes := []string{"072", "082"}
	index := 0
	sourceDate := time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC)
	for rows.Next() {
		if index >= len(expectedCodes) {
			t.Fatal("unexpected extra ERM Risk")
		}
		var code, name, category, impact, status, rating, appetite string
		var version int64
		var assessedAt time.Time
		var assessorUnknown bool
		if err = rows.Scan(&code, &name, &category, &impact, &status, &version, &rating, &appetite, &assessedAt, &assessorUnknown); err != nil {
			t.Fatal(err)
		}
		if code != expectedCodes[index] || name != "User Access Management" || category != "Identity and Access Risk" ||
			impact == "" || status != "ACTIVE" || version != 3 || rating != "High" || appetite != "UNKNOWN" ||
			!assessedAt.Equal(sourceDate) || !assessorUnknown {
			t.Fatalf("risk %d changed: code=%s name=%q category=%q impact=%q status=%s version=%d rating=%s appetite=%s assessed=%s assessorUnknown=%v",
				index, code, name, category, impact, status, version, rating, appetite, assessedAt, assessorUnknown)
		}
		index++
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if index != 2 {
		t.Fatalf("ERM Risk count=%d want 2", index)
	}

	var appetiteCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM risk_appetite_statements`).Scan(&appetiteCount); err != nil {
		t.Fatal(err)
	}
	if appetiteCount != 0 {
		t.Fatalf("source severity created %d appetite statements", appetiteCount)
	}

	second, err := reconcilePersistedSourceRisks(ctx, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if second.Risks != 2 || second.RiskAssessments != 0 {
		t.Fatalf("repeat reconciliation=%+v", second)
	}

	var riskCount, assessmentCount, afterMatterCount int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM risks),(SELECT count(*) FROM risk_assessments),(SELECT count(*) FROM matters WHERE trigger_type='SOURCE_REGISTER_IMPORT')`).Scan(&riskCount, &assessmentCount, &afterMatterCount); err != nil {
		t.Fatal(err)
	}
	if riskCount != 2 || assessmentCount != 2 || afterMatterCount != matterCount {
		t.Fatalf("repeat changed populations: risks=%d assessments=%d matters=%d", riskCount, assessmentCount, afterMatterCount)
	}
}

func insertPersistedITException(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, entityID, recordKey, riskID, assessment, description, category, implication, affectedArea string) {
	t.Helper()
	fields := []sourceRecordField{
		{Label: "RISK ID", Value: riskID},
		{Label: "RISK ASSESSMENT", Value: assessment},
		{Label: "RISK DESCRIPTION", Value: description},
		{Label: "APPLICATION/ SERVICES AFFECTED", Value: affectedArea},
		{Label: "RISK CATEGORY", Value: category},
		{Label: "RISK/ IMPLICATIONS", Value: implication},
		{Label: "RISK LEVEL", Value: "High"},
		{Label: "RISK ASSESSMENT PUBLICATION DATE", Value: "2025-10-29"},
	}
	facts := sourceJSON(map[string]any{
		"source_file":     "Sample IT Risk Exception Register (1).xlsx",
		"source_sha256":   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"source_sheet":    "Sheet1",
		"source_range":    "Sheet1!A2:R2",
		"source_rating":   "High",
		"source_owner":    "CISO",
		"source_assessor": "",
		"source_fields":   fields,
	})
	scope := sourceJSON(map[string]any{"sample": true, "seed_package": sourceRecordPackage, "source_group": "it-risk-exceptions"})
	query := `
		INSERT INTO matters(
			id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,
			trigger_type,trigger_key,known_facts,missing_facts,contradictions,created_at,updated_at,version
		) VALUES(
			uuidv7(),$1::uuid,$2::uuid,$3,'RISK_SITUATION','ASSESSMENT',4,$4,$5,$6::jsonb,
			'SOURCE_REGISTER_IMPORT',$7,$8::jsonb,'[]'::jsonb,'[]'::jsonb,clock_timestamp(),clock_timestamp(),1
		)`
	if _, err := pool.Exec(ctx, query, tenantID, entityID, "ITR-"+riskID, description, implication, string(scope), sourceRecordPackage+":"+recordKey, string(facts)); err != nil {
		t.Fatal(err)
	}
}
