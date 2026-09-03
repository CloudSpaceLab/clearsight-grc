//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAccessOTPDelivery struct {
	address string
	code    string
}

func (delivery *postgresAccessOTPDelivery) DeliverDistributionOTP(_ context.Context, value DistributionOTPDelivery) error {
	delivery.address = value.Address
	delivery.code = value.Code
	return nil
}

func TestPostgresDistributionAccessPersistsProtectedVerifiedSessionAndRevokesOnRotation(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID  = "9d555555-5555-7555-8555-555555555551"
		entityID  = "9d555555-5555-7555-8555-555555555552"
		actorID   = "9d555555-5555-7555-8555-555555555553"
		formID    = "9d555555-5555-7555-8555-555555555554"
		subjectID = "9d555555-5555-7555-8555-555555555555"
	)
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	setupDistributionAccessFixture(t, ctx, pool, tenantID, entityID, actorID, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)

	var recipientKey [32]byte
	var accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index] = 0x5a
		accessKey[index] = 0x6b
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-access-integration", LegalEntityID: "ACCESS", FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "External resilience review", Purpose: "Verify protected access.",
		AccessPolicy: AccessSharedEmailOTP, EstimatedMinutes: 5,
		Deadline: now.Add(4 * time.Hour), RouteExpiresAt: now.Add(3 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{
			Role: RecipientTo, Type: RecipientExternalAudience, Address: "assurance@example.test",
			AudienceHint: "a***@example.test", ContactLabel: "Assurance owner",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Distribution.TenantID != tenantID || bundle.Distribution.LegalEntityID != entityID || bundle.Recipients[0].TenantID != tenantID || bundle.Recipients[0].LegalEntityID != entityID {
		t.Fatalf("distribution aliases were not canonicalized before protection: %+v %+v", bundle.Distribution, bundle.Recipients[0])
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN',updated_at=$2 WHERE id=$1::uuid`, bundle.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}

	delivery := &postgresAccessOTPDelivery{}
	service, err := NewDistributionAccessService(store, keyring, delivery, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }

	routes, err := service.IssueDistributionAccessRoutes(ctx, tenantID, entityID, bundle.Distribution.ID, actorID)
	if err != nil || len(routes) != 1 {
		t.Fatalf("issue shared access route: routes=%+v err=%v", routes, err)
	}
	start, err := service.StartDistributionAccess(ctx, routes[0].Selector)
	if err != nil || len(start.Recipients) != 1 {
		t.Fatalf("start shared access: %+v %v", start, err)
	}
	receipt, err := service.SendOTP(ctx, routes[0].Selector, start.Recipients[0].SelectorID)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.address != "assurance@example.test" || len(delivery.code) != OTPCodeDigits || receipt.ChallengeID == "" {
		t.Fatalf("protected delivery boundary received unexpected values: address=%q code_len=%d receipt=%+v", delivery.address, len(delivery.code), receipt)
	}

	var storedAddress bool
	var digest []byte
	var resends int
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM capture_distribution_recipients
			WHERE distribution_id=$1::uuid AND to_jsonb(capture_distribution_recipients)::text ILIKE '%assurance@example.test%'
		), c.code_hash, c.resends
		FROM capture_otp_challenges c WHERE c.id=$2::uuid`, bundle.Distribution.ID, receipt.ChallengeID).Scan(&storedAddress, &digest, &resends); err != nil {
		t.Fatal(err)
	}
	if storedAddress || resends != 0 || strings.Contains(strings.ToLower(hex.EncodeToString(digest)), strings.ToLower(hex.EncodeToString([]byte(delivery.code)))) {
		t.Fatalf("protected address/OTP persistence invariant failed: address=%t resends=%d", storedAddress, resends)
	}

	redeemed, err := service.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, delivery.code)
	if err != nil || redeemed.Assurance != AssuranceEmailVerified || redeemed.SessionToken == "" {
		t.Fatalf("verify OTP: %+v %v", redeemed, err)
	}
	session, request, err := service.SessionRequest(ctx, redeemed.SessionToken)
	if err != nil || session.Assurance != AssuranceEmailVerified || request.ID != redeemed.RequestID {
		t.Fatalf("recover verified session: session=%+v request=%+v err=%v", session, request, err)
	}

	var assurance AccessAssurance
	var consumed bool
	if err := pool.QueryRow(ctx, `
		SELECT s.assurance,(c.consumed_at IS NOT NULL)
		FROM capture_distribution_sessions s
		JOIN capture_otp_challenges c ON c.id=$2::uuid
		WHERE s.id=$1::uuid`, redeemed.SessionID, receipt.ChallengeID).Scan(&assurance, &consumed); err != nil {
		t.Fatal(err)
	}
	if assurance != AssuranceEmailVerified || !consumed {
		t.Fatalf("durable assurance/challenge consumption mismatch: %q consumed=%t", assurance, consumed)
	}

	replacement, err := service.RotateDistributionAccessRoute(ctx, tenantID, entityID, bundle.Distribution.ID, routes[0].RouteID, actorID)
	if err != nil || replacement.Selector == "" || replacement.Selector == routes[0].Selector {
		t.Fatalf("rotate route: %+v %v", replacement, err)
	}
	if _, _, err := service.SessionRequest(ctx, redeemed.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("route rotation left old session usable: %v", err)
	}
}

func TestPostgresDirectMagicLinkReopensUntilExpiryOrRevocation(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID  = "9d666666-6666-7666-8666-666666666661"
		entityID  = "9d666666-6666-7666-8666-666666666662"
		actorID   = "9d666666-6666-7666-8666-666666666663"
		formID    = "9d666666-6666-7666-8666-666666666664"
		subjectID = "9d666666-6666-7666-8666-666666666665"
	)
	now := time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC)
	setupDistributionAccessFixture(t, ctx, pool, tenantID, entityID, actorID, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)

	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index] = 0x6c
		accessKey[index] = 0x7d
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-access-integration", LegalEntityID: "ACCESS", FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Direct evidence request", Purpose: "Verify reusable access before expiry.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5,
		Deadline: now.Add(4 * time.Hour), RouteExpiresAt: now.Add(3 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{
			Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test",
			AudienceHint: "o***@example.test", ContactLabel: "Evidence owner",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN',updated_at=$2 WHERE id=$1::uuid`, bundle.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}

	service, err := NewDistributionAccessService(store, keyring, &postgresAccessOTPDelivery{}, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
	routes, err := service.IssueDistributionAccessRoutes(ctx, tenantID, entityID, bundle.Distribution.ID, actorID)
	if err != nil || len(routes) != 1 {
		t.Fatalf("issue direct access route: routes=%+v err=%v", routes, err)
	}
	first, err := service.RedeemDirectRoute(ctx, routes[0].Selector)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.RedeemDirectRoute(ctx, routes[0].Selector)
	if err != nil {
		t.Fatalf("unexpired selector could not be reopened: %v", err)
	}
	if first.SessionID == second.SessionID || first.SessionToken == second.SessionToken {
		t.Fatalf("reopening reused a bearer session: first=%s second=%s", first.SessionID, second.SessionID)
	}
	if err := service.RevokeDistributionAccessRoute(ctx, tenantID, entityID, bundle.Distribution.ID, routes[0].RouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RedeemDirectRoute(ctx, routes[0].Selector); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("revoked selector remained usable: %v", err)
	}
}

func setupDistributionAccessFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, entityID, actorID, formID string, now time.Time) {
	t.Helper()
	cleanupDistributionTenant(ctx, pool, tenantID)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'distribution-access-integration','Distribution Access Integration');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($2::uuid,$1::uuid,'ACCESS','Distribution Access Entity','NG',$5);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($3::uuid,$1::uuid,'PERSON','Distribution Access Actor','ACTIVE',$5);
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES(
			$4::uuid,$1::uuid,$2::uuid,'ACCESS-FORM','Access form','Protected distribution access integration.',
			'{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
			'[{"id":"general","title":"General"}]'::jsonb,
			'[{"id":"registered_address","section_id":"general","label":"Registered address","type":"short_text","required":true,"collection_intent":"CONFIRM_OR_CORRECT","record_target":{"key":"registered_address","required_subject_type":"VENDOR"},"browser_cache_policy":"NO_BROWSER_CACHE"}]'::jsonb,
			'ACTIVE',true,$5,1,$3::uuid,$5,$5
		)`, pgx.QueryExecModeSimpleProtocol, tenantID, entityID, actorID, formID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
}
