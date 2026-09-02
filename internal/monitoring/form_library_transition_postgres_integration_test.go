//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresReusableFormCanEnterApprovalWithoutProgramID(t *testing.T) {
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
		tenantID    = "9d222222-2222-7222-8222-222222222221"
		entityID    = "9d222222-2222-7222-8222-222222222222"
		principalID = "9d222222-2222-7222-8222-222222222223"
		formID      = "9d222222-2222-7222-8222-222222222224"
		tenantSlug  = "reusable-form-transition-test"
	)
	cleanupReusableFormTransition(ctx, pool, tenantID)
	defer cleanupReusableFormTransition(context.Background(), pool, tenantID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Reusable Form Transition Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($3::uuid,$1::uuid,'FORM-REVIEW','Form Review Entity','NG');
		INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($4::uuid,$1::uuid,'PERSON','Form Maker')`,
		tenantID, tenantSlug, entityID, principalID); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresRepository(pool), nil)
	service.ConfigureCommandGuard(guard)
	service.newID = func() (string, error) { return formID, nil }
	service.now = func() time.Time { return time.Date(2026, 9, 2, 22, 0, 0, 0, time.UTC) }
	actorCtx := formActorContext(tenantSlug, entityID, principalID)

	draft, err := service.CreateLibraryForm(actorCtx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	if draft.ProgramID != "" {
		t.Fatalf("reusable form unexpectedly acquired a Program: %#v", draft)
	}
	pending, err := service.TransitionLibraryForm(actorCtx, draft.ID, TransitionInput{ExpectedVersion: draft.Version, To: LifecyclePendingApproval})
	if err != nil {
		t.Fatalf("reusable form could not enter approval without a Program ID: %v", err)
	}
	if pending.Status != LifecyclePendingApproval || pending.SubmittedBy != principalID || pending.Version != draft.Version+1 {
		t.Fatalf("pending reusable form = %#v", pending)
	}
}

func cleanupReusableFormTransition(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
