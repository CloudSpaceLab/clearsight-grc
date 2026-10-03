//go:build postgres && postgresintegration

package oversight

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresLatestManyReturnsExactLatestChildSnapshots(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID   = "8d111111-1111-7111-8111-111111111111"
		entityA    = "8d111111-1111-7111-8111-111111111112"
		entityB    = "8d111111-1111-7111-8111-111111111113"
		oldA       = "8d111111-1111-7111-8111-111111111114"
		currentA   = "8d111111-1111-7111-8111-111111111115"
		currentB   = "8d111111-1111-7111-8111-111111111116"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM oversight_snapshots WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'group-snapshot-test','Group Snapshot Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($2::uuid,$1::uuid,'A','Alpha','NG',$7),
			($3::uuid,$1::uuid,'B','Beta','GH',$7);
		INSERT INTO oversight_snapshots(
			id,tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,
			projection_version,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload
		) VALUES
			($4::uuid,$1::uuid,$2::uuid,$7,$8,$7,$7,'oversight-v5','{}'::jsonb,2,0,0,'{"counts":{"critical_high":1}}'::jsonb),
			($5::uuid,$1::uuid,$2::uuid,$7,$8,$8,$8,'oversight-v5','{}'::jsonb,4,0,0,'{"counts":{"critical_high":2}}'::jsonb),
			($6::uuid,$1::uuid,$3::uuid,$7,$8,$8,$8,'oversight-v5','{}'::jsonb,6,0,0,'{"counts":{"critical_high":3}}'::jsonb)
	`, tenantID, entityA, entityB, oldA, currentA, currentB, now.Add(-2*time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	values, err := repo.LatestMany(ctx, "group-snapshot-test", []string{entityA, entityB})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 {
		t.Fatalf("snapshots=%#v", values)
	}
	byEntity := map[string]Snapshot{}
	for _, value := range values {
		byEntity[value.LegalEntityID] = value
	}
	if byEntity[entityA].SnapshotID != currentA || byEntity[entityA].Counts.CriticalHigh != 2 ||
		byEntity[entityB].SnapshotID != currentB || byEntity[entityB].Counts.CriticalHigh != 3 {
		t.Fatalf("latest child snapshots=%#v", byEntity)
	}

	service := NewService(repo)
	service.Now = func() time.Time { return now.Add(-time.Hour) }
	decorated, err := service.GetMany(ctx, "group-snapshot-test", []string{entityA, entityB})
	if err != nil {
		t.Fatal(err)
	}
	group, err := BuildGroupSnapshot("tenant-group", "Group Snapshot Test", []GroupEntity{
		{ID: entityA, Name: "Alpha"}, {ID: entityB, Name: "Beta"},
	}, decorated)
	if err != nil {
		t.Fatal(err)
	}
	if group.Counts.CriticalHigh != 5 || len(group.Contributors) != 2 ||
		group.Contributors[0].SnapshotID != currentA || group.Contributors[1].SnapshotID != currentB {
		t.Fatalf("group provenance=%#v", group)
	}
}
