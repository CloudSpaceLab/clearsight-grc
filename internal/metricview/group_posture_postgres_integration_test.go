//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGroupPosturePostgresPreservesMissingRiskSourcesAndAuthorizedChildren(t *testing.T) {
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

	newID := func() string {
		t.Helper()
		value, idErr := platformid.NewUUIDv7()
		if idErr != nil {
			t.Fatal(idErr)
		}
		return value
	}
	tenantID := newID()
	entityA := newID()
	entityB := newID()
	excludedEntity := newID()
	now := time.Now().UTC().Truncate(time.Second)
	validFrom := now.Add(-48 * time.Hour)

	if _, err := pool.Exec(ctx,
		`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Group posture test')`,
		tenantID, "group-posture-"+tenantID[len(tenantID)-8:],
	); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES
		  ($1::uuid,$4::uuid,'GPA','Group A','NG',$5),
		  ($2::uuid,$4::uuid,'GPB','Group B','GH',$5),
		  ($3::uuid,$4::uuid,'GPC','Excluded Group C','NG',$5)`,
		entityA, entityB, excludedEntity, tenantID, validFrom,
	); err != nil {
		t.Fatal(err)
	}

	repository := NewGroupPostureRepository(pool)
	active, err := repository.ActiveGroupEntities(ctx, tenantID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 3 {
		t.Fatalf("active Group entities=%#v", active)
	}

	bundle, err := repository.GroupPosture(
		ctx, tenantID, []string{entityB, entityA}, now.Add(-24*time.Hour), now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Children) != 2 || bundle.RiskCoverage.AuthorizedChildren != 2 ||
		bundle.RiskCoverage.MissingChildren != 2 || bundle.RiskCoverage.IncludedChildren != 0 ||
		bundle.RiskCoverage.Complete || bundle.LossCoverage.AuthorizedChildren != 2 ||
		bundle.LossCoverage.IncludedChildren != 2 || !bundle.LossCoverage.Complete ||
		bundle.Counts != (GroupPostureCounts{}) {
		t.Fatalf("Group posture=%#v", bundle)
	}
	for _, child := range bundle.Children {
		if child.LegalEntityID == excludedEntity || child.RiskState != "MISSING" ||
			child.Completeness != CompletenessUnknown || child.Counts != (GroupPostureCounts{}) {
			t.Fatalf("Group child=%#v", child)
		}
	}
}
