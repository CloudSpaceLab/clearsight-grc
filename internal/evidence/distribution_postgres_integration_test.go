//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresTestRecipientProtector struct{}

func (postgresTestRecipientProtector) ProtectRecipientAddress(_ context.Context, _, _, _, _ string) (protectedRecipientAddress, error) {
	return protectedRecipientAddress{
		Hash:       []byte("01234567890123456789012345678901"),
		Ciphertext: []byte("sealed-recipient-address"),
		KeyID:      "recipient-test-v1",
	}, nil
}

func TestPostgresDistributionCreatesCanonicalTORequestsAndOneWorkspace(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID          = "9d111111-1111-7111-8111-111111111111"
		entityID          = "9d111111-1111-7111-8111-111111111112"
		actorID           = "9d111111-1111-7111-8111-111111111113"
		internalRecipient = "9d111111-1111-7111-8111-111111111114"
		formID            = "9d111111-1111-7111-8111-111111111115"
		subjectID         = "9d111111-1111-7111-8111-111111111116"
	)
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	setupDistributionFixture(t, ctx, pool, tenantID, entityID, actorID, internalRecipient, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)

	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-test", LegalEntityID: entityID,
		FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID,
		Title: "Vendor resilience review", Purpose: "Confirm current resilience controls.",
		AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 8,
		Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{
			{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: internalRecipient, ContactLabel: "Control owner"},
			{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "v***@example.test", ContactLabel: "Vendor assurance"},
			{Role: RecipientCC, Type: RecipientExternalAudience, Address: "observer@example.test", AudienceHint: "o***@example.test", ContactLabel: "Observer"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Distribution.FormTemplateID != formID || bundle.Distribution.FormTemplateVersion != 1 {
		t.Fatalf("distribution lost exact form revision: %+v", bundle.Distribution)
	}
	if len(bundle.Recipients) != 3 || bundle.Workspace.DistributionID != bundle.Distribution.ID {
		t.Fatalf("unexpected distribution bundle: %+v", bundle)
	}

	var requestCount, ccRequests, workspaceCount, eventCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_requests WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, tenantID, bundle.Distribution.ID).Scan(&requestCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND role='CC' AND request_id IS NOT NULL`, tenantID, bundle.Distribution.ID).Scan(&ccRequests); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_response_workspaces WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, tenantID, bundle.Distribution.ID).Scan(&workspaceCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, tenantID, bundle.Distribution.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='FORM_DISTRIBUTION' AND aggregate_id=$2::uuid`, tenantID, bundle.Distribution.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if requestCount != 2 || ccRequests != 0 || workspaceCount != 1 || eventCount != 1 || outboxCount != 1 {
		t.Fatalf("requests/cc-request/workspace/event/outbox = %d/%d/%d/%d/%d, want 2/0/1/1/1", requestCount, ccRequests, workspaceCount, eventCount, outboxCount)
	}

	var collectionIntent, targetKey, targetSubject, cachePolicy string
	if err := pool.QueryRow(ctx, `
		SELECT fields->0->>'collection_intent',fields->0->'record_target'->>'key',
		       fields->0->'record_target'->>'required_subject_type',fields->0->>'browser_cache_policy'
		FROM capture_requests
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid
		ORDER BY id LIMIT 1`, tenantID, bundle.Distribution.ID).Scan(&collectionIntent, &targetKey, &targetSubject, &cachePolicy); err != nil {
		t.Fatal(err)
	}
	if collectionIntent != "CONFIRM_OR_CORRECT" || targetKey != "registered_address" || targetSubject != "VENDOR" || cachePolicy != "NO_BROWSER_CACHE" {
		t.Fatalf("governed collection semantics were not retained: %q %q %q %q", collectionIntent, targetKey, targetSubject, cachePolicy)
	}

	var rawAddressStored bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM capture_distribution_recipients
			WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid
			  AND to_jsonb(capture_distribution_recipients)::text ILIKE '%vendor@example.test%'
		)`, tenantID, bundle.Distribution.ID).Scan(&rawAddressStored); err != nil {
		t.Fatal(err)
	}
	if rawAddressStored {
		t.Fatal("raw external recipient address reached durable distribution state")
	}
}

func TestPostgresDistributionRollsBackWhenOutboxFails(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID          = "9d222222-2222-7222-8222-222222222221"
		entityID          = "9d222222-2222-7222-8222-222222222222"
		actorID           = "9d222222-2222-7222-8222-222222222223"
		internalRecipient = "9d222222-2222-7222-8222-222222222224"
		formID            = "9d222222-2222-7222-8222-222222222225"
		subjectID         = "9d222222-2222-7222-8222-222222222226"
	)
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	setupDistributionFixture(t, ctx, pool, tenantID, entityID, actorID, internalRecipient, formID, now)
	cleanupDistributionFailureTrigger(ctx, pool)
	defer func() {
		cleanupDistributionFailureTrigger(context.Background(), pool)
		cleanupDistributionTenant(context.Background(), pool, tenantID)
	}()
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION distribution_outbox_failure_test() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.aggregate_type='FORM_DISTRIBUTION' THEN RAISE EXCEPTION 'forced distribution outbox failure'; END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER distribution_outbox_failure_test BEFORE INSERT ON outbox_events
		FOR EACH ROW EXECUTE FUNCTION distribution_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}

	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	_, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-rollback", LegalEntityID: entityID,
		FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Rollback test", Purpose: "Prove atomic persistence.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5,
		Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: internalRecipient}},
	})
	if err == nil {
		t.Fatal("expected forced outbox failure")
	}

	for table, predicate := range map[string]string{
		"capture_form_distributions":      "tenant_id=$1::uuid",
		"capture_distribution_recipients": "tenant_id=$1::uuid",
		"capture_response_workspaces":     "tenant_id=$1::uuid",
		"capture_distribution_events":     "tenant_id=$1::uuid",
		"capture_requests":                "tenant_id=$1::uuid AND distribution_id IS NOT NULL",
	} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE " + predicate
		if err := pool.QueryRow(ctx, query, tenantID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("failed distribution left %d rows in %s", count, table)
		}
	}
}

func TestPostgresDistributionRejectsCrossEntityActiveRevision(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID          = "9d333333-3333-7333-8333-333333333331"
		entityID          = "9d333333-3333-7333-8333-333333333332"
		otherEntityID     = "9d333333-3333-7333-8333-333333333333"
		actorID           = "9d333333-3333-7333-8333-333333333334"
		internalRecipient = "9d333333-3333-7333-8333-333333333335"
		formID            = "9d333333-3333-7333-8333-333333333336"
		subjectID         = "9d333333-3333-7333-8333-333333333337"
	)
	now := time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC)
	setupDistributionFixture(t, ctx, pool, tenantID, entityID, actorID, internalRecipient, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'OTHER','Other entity','NG',$3)`, otherEntityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	_, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-cross-scope", LegalEntityID: otherEntityID,
		FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Cross scope", Purpose: "Must fail closed.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5,
		Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: internalRecipient}},
	})
	if err == nil {
		t.Fatal("expected cross-entity exact revision rejection")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_form_distributions WHERE tenant_id=$1::uuid`, tenantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cross-entity rejection left %d distribution rows", count)
	}
}

func TestPostgresDistributionRejectsDirectWrongEntityTampering(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID          = "9d444444-4444-7444-8444-444444444441"
		entityID          = "9d444444-4444-7444-8444-444444444442"
		otherEntityID     = "9d444444-4444-7444-8444-444444444443"
		actorID           = "9d444444-4444-7444-8444-444444444444"
		internalRecipient = "9d444444-4444-7444-8444-444444444445"
		formID            = "9d444444-4444-7444-8444-444444444446"
		subjectID         = "9d444444-4444-7444-8444-444444444447"
	)
	now := time.Date(2026, 8, 27, 22, 0, 0, 0, time.UTC)
	setupDistributionFixture(t, ctx, pool, tenantID, entityID, actorID, internalRecipient, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'OTHER','Other entity','NG',$3)`, otherEntityID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-tamper", LegalEntityID: entityID,
		FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Tamper regression", Purpose: "Prove entity scope cannot drift.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5,
		Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: internalRecipient}},
	})
	if err != nil {
		t.Fatal(err)
	}

	attempts := []struct {
		name         string
		query        string
		wantSQLState string
	}{
		{name: "distribution root", query: `UPDATE capture_form_distributions SET legal_entity_id=$1::uuid WHERE tenant_id=$2::uuid AND id=$3::uuid`, wantSQLState: "23503"},
		{name: "canonical request", query: `UPDATE capture_requests SET legal_entity_id=$1::uuid WHERE tenant_id=$2::uuid AND distribution_id=$3::uuid`, wantSQLState: "23514"},
		{name: "recipient binding", query: `UPDATE capture_distribution_recipients SET legal_entity_id=$1::uuid WHERE tenant_id=$2::uuid AND distribution_id=$3::uuid`, wantSQLState: "23503"},
		{name: "shared workspace", query: `UPDATE capture_response_workspaces SET legal_entity_id=$1::uuid WHERE tenant_id=$2::uuid AND distribution_id=$3::uuid`, wantSQLState: "23503"},
	}
	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, attempt.query, otherEntityID, tenantID, bundle.Distribution.ID)
			requirePostgresState(t, err, attempt.wantSQLState)
		})
	}

	var wrongEntityRows int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM capture_form_distributions WHERE tenant_id=$1::uuid AND id=$2::uuid AND legal_entity_id=$3::uuid) +
			(SELECT count(*) FROM capture_requests WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND legal_entity_id=$3::uuid) +
			(SELECT count(*) FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND legal_entity_id=$3::uuid) +
			(SELECT count(*) FROM capture_response_workspaces WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND legal_entity_id=$3::uuid)`,
		tenantID, bundle.Distribution.ID, otherEntityID).Scan(&wrongEntityRows); err != nil {
		t.Fatal(err)
	}
	if wrongEntityRows != 0 {
		t.Fatalf("direct SQL tampering moved %d aggregate rows into the wrong entity", wrongEntityRows)
	}
}

func requirePostgresState(t *testing.T, err error, wantSQLState string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected database guard to reject wrong-entity tampering")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != wantSQLState {
		t.Fatalf("expected PostgreSQL SQLSTATE %s, got %T: %v", wantSQLState, err, err)
	}
}

func distributionTestPool(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, ctx
}

func setupDistributionFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, entityID, actorID, internalRecipient, formID string, now time.Time) {
	t.Helper()
	cleanupDistributionTenant(ctx, pool, tenantID)
	slug := "distribution-test"
	switch tenantID {
	case "9d222222-2222-7222-8222-222222222221":
		slug = "distribution-rollback"
	case "9d333333-3333-7333-8333-333333333331":
		slug = "distribution-cross-scope"
	case "9d444444-4444-7444-8444-444444444441":
		slug = "distribution-tamper"
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$6,'Distribution Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Distribution Bank','NG',$7);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($3::uuid,$1::uuid,'PERSON','Distribution Actor','ACTIVE',$7),
			($4::uuid,$1::uuid,'PERSON','Distribution Recipient','ACTIVE',$7);
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES(
			$5::uuid,$1::uuid,$2::uuid,'VENDOR-RESILIENCE','Vendor resilience','Confirm current resilience controls.',
			'{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
			'[{"id":"general","title":"General"}]'::jsonb,
			'[{"id":"registered_address","section_id":"general","label":"Registered address","type":"short_text","required":true,"collection_intent":"CONFIRM_OR_CORRECT","record_target":{"key":"registered_address","required_subject_type":"VENDOR"},"browser_cache_policy":"NO_BROWSER_CACHE"}]'::jsonb,
			'ACTIVE',true,$7,1,$3::uuid,$7,$7
		)`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, actorID, internalRecipient, formID, slug, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func cleanupDistributionTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}

func cleanupDistributionFailureTrigger(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DROP TRIGGER IF EXISTS distribution_outbox_failure_test ON outbox_events`)
	_, _ = pool.Exec(ctx, `DROP FUNCTION IF EXISTS distribution_outbox_failure_test()`)
}
