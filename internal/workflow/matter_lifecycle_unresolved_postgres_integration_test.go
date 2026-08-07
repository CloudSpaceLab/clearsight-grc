//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMatterLifecycleProjectorBlocksWhenCanonicalLegalEntityIsUnresolved(t *testing.T) {
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
		tenantID    = "98888888-8888-7888-8888-888888888801"
		entityID    = "98888888-8888-7888-8888-888888888802"
		principalA  = "98888888-8888-7888-8888-888888888803"
		principalB  = "98888888-8888-7888-8888-888888888804"
		programID   = "98888888-8888-7888-8888-888888888805"
		matterID    = "98888888-8888-7888-8888-888888888806"
		responseAck = "98888888-8888-7888-8888-888888888807"
		responseFix = "98888888-8888-7888-8888-888888888808"
		responseAmb = "98888888-8888-7888-8888-888888888809"
		ackRouteA   = "98888888-8888-7888-8888-888888888810"
		fixRouteA   = "98888888-8888-7888-8888-888888888811"
		fixRouteB   = "98888888-8888-7888-8888-888888888812"
		eventID     = "98888888-8888-7888-8888-888888888813"
	)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Date(2026, 8, 7, 21, 0, 0, 0, time.UTC)
	seedMatterLifecycleWork(t, ctx, pool, tenantID, entityID, principalA, principalB, programID, matterID, responseAck, responseFix, responseAmb, ackRouteA, fixRouteA, fixRouteB, now)

	// Remove the only canonical Program link. Even though routing assignments and
	// principals exist, asynchronous work must not guess a legal entity.
	if _, err := pool.Exec(ctx, `DELETE FROM matter_links WHERE tenant_id=$1::uuid AND matter_id=$2::uuid`, tenantID, matterID); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	projector := &MatterLifecycleProjector{
		Repo:       repo,
		Continuity: continuity.NewService(continuity.NewCurrentPostgresRepository(pool)),
		Authority:  authority.NewEffectivePostgresService(pool),
	}
	publishLifecycleTestEvent(t, ctx, projector, eventID, "lifecycle-work-test", matterID, continuity.EventResponsePackageStateChanged, now)

	var status Status
	var principalID, routingState, legalEntityID string
	if err := pool.QueryRow(ctx, `
		SELECT wt.status,COALESCE(wt.principal_id::text,''),
		       COALESCE(wt.context->>'routing_state',''),COALESCE(wt.context->>'legal_entity_id','')
		FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		WHERE wi.tenant_id=$1::uuid
		  AND wi.kind='MATTER_RESPONSE'
		  AND wi.subject_id=$2::uuid`, tenantID, responseFix).Scan(&status, &principalID, &routingState, &legalEntityID); err != nil {
		t.Fatal(err)
	}
	if status != StatusBlocked || principalID != "" || routingState != "LEGAL_ENTITY_UNRESOLVED" || legalEntityID != "" {
		t.Fatalf("unresolved entity scope was not fail-closed: status=%s principal=%q routing=%q entity=%q", status, principalID, routingState, legalEntityID)
	}

	service := NewService(repo)
	visible, err := service.List(ctx, ListFilter{
		TenantID:                "lifecycle-work-test",
		PrincipalID:             principalA,
		SupportedMatterWorkOnly: true,
		ActiveOnly:              true,
		VisibleMatterWorkOnly:   true,
		Limit:                   20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range visible {
		if task.Context["response_id"] == responseFix {
			t.Fatalf("unassigned unresolved work leaked into an actor queue: %#v", task)
		}
	}
}
