//go:build postgres && postgresintegration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestScoringSamplePresentationRecognizesExistingHistory(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Session-local tables isolate the read contract from stored bank records.
	_, err = pool.Exec(ctx, `CREATE TEMP TABLE capture_form_distributions (id uuid,tenant_id uuid,legal_entity_id uuid,form_template_id uuid,form_template_version bigint,subject_type text,subject_id uuid,title text);
	CREATE TEMP TABLE capture_response_revisions (distribution_id uuid,tenant_id uuid,legal_entity_id uuid,is_current boolean);`)
	if err != nil {
		t.Fatal(err)
	}
	seed := bankverticals.SeedConfig{TenantID: sampleTestTenant, LegalEntityID: sampleTestEntity}
	formID, subjectID := "00000000-0000-4000-8000-000000000010", "00000000-0000-4000-8000-000000000011"
	for _, mode := range []string{"legacy", "new", "mixed", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			if _, err := pool.Exec(ctx, `TRUNCATE pg_temp.capture_form_distributions, pg_temp.capture_response_revisions`); err != nil {
				t.Fatal(err)
			}
			for i, label := range []string{"good", "borderline", "poor"} {
				title := scoringSampleTitle(label)
				if mode == "legacy" || mode == "mixed" && i == 1 {
					title = "Scoring acceptance — " + label
				}
				id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i+20)
				if _, err := pool.Exec(ctx, `INSERT INTO capture_form_distributions VALUES ($1,$2,$3,$4,1,'PROGRAM',$5,$6)`, id, seed.TenantID, seed.LegalEntityID, formID, subjectID, title); err != nil {
					t.Fatal(err)
				}
				if _, err := pool.Exec(ctx, `INSERT INTO capture_response_revisions VALUES ($1,$2,$3,true)`, id, seed.TenantID, seed.LegalEntityID); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "duplicate" {
				if _, err := pool.Exec(ctx, `INSERT INTO capture_form_distributions SELECT '00000000-0000-4000-8000-000000000030',tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,'Scoring acceptance — good' FROM capture_form_distributions LIMIT 1;
				INSERT INTO capture_response_revisions SELECT '00000000-0000-4000-8000-000000000030',tenant_id,legal_entity_id,true FROM capture_response_revisions LIMIT 1;`); err != nil {
					t.Fatal(err)
				}
			}
			seeded, err := scoringAcceptanceAlreadySeeded(ctx, pool, seed, formID, 1, subjectID)
			if err != nil || !seeded {
				t.Fatalf("existing %s sample history not recognized: %v, %v", mode, seeded, err)
			}
		})
	}
}
