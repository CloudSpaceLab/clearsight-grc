//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMagicLinkExpiryMigrationPreservesIndependentLinksAndOTPReplacement(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID  = "9d777777-7777-7777-8777-777777777771"
		entityID  = "9d777777-7777-7777-8777-777777777772"
		actorID   = "9d777777-7777-7777-8777-777777777773"
		formID    = "9d777777-7777-7777-8777-777777777774"
		subjectID = "9d777777-7777-7777-8777-777777777775"
	)
	now := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	setupDistributionAccessFixture(t, ctx, pool, tenantID, entityID, actorID, formID, now)
	up, err := os.ReadFile("../../migrations/000075_magic_link_expiry_semantics.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000075_magic_link_expiry_semantics.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	latestSchema := true
	defer func() {
		cleanupDistributionTenant(context.Background(), pool, tenantID)
		if !latestSchema {
			if _, restoreErr := pool.Exec(context.Background(), string(up)); restoreErr != nil {
				t.Errorf("restore migration 75: %v", restoreErr)
			}
		}
	}()

	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index] = 0x7e
		accessKey[index] = 0x8f
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	service, err := NewDistributionAccessService(store, keyring, &postgresAccessOTPDelivery{}, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }

	magic, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: tenantID, LegalEntityID: entityID, FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Independent invitation expiry", Purpose: "Verify each emailed link against its printed expiry.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5, Deadline: now.Add(4 * time.Hour), RouteExpiresAt: now.Add(3 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Evidence owner"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN',updated_at=$2 WHERE id=$1::uuid`, magic.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}
	first, err := service.IssueDistributionAccessRoutes(ctx, tenantID, entityID, magic.Distribution.ID, actorID)
	if err != nil || len(first) != 1 {
		t.Fatalf("issue first magic route: %+v %v", first, err)
	}
	second, err := service.RotateDistributionAccessRoute(ctx, tenantID, entityID, magic.Distribution.ID, first[0].RouteID, actorID)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, string(down)); err == nil {
		connection.Release()
		t.Fatal("rollback accepted multiple unrevoked magic links")
	}
	if _, err := connection.Exec(ctx, "ROLLBACK"); err != nil {
		connection.Release()
		t.Fatalf("clear refused rollback transaction: %v", err)
	}
	var otpIndex, legacyIndex bool
	if err := connection.QueryRow(ctx, `SELECT to_regclass('capture_access_routes_active_direct_otp_uq') IS NOT NULL,to_regclass('capture_access_routes_active_direct_uq') IS NOT NULL`).Scan(&otpIndex, &legacyIndex); err != nil {
		connection.Release()
		t.Fatal(err)
	}
	connection.Release()
	if !otpIndex || legacyIndex {
		t.Fatalf("failed rollback was not atomic: otp_index=%t legacy_index=%t", otpIndex, legacyIndex)
	}
	if err := service.RevokeDistributionAccessRoute(ctx, tenantID, entityID, magic.Distribution.ID, first[0].RouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("clean rollback failed: %v", err)
	}
	latestSchema = false
	if _, err := service.RotateDistributionAccessRoute(ctx, tenantID, entityID, magic.Distribution.ID, second.RouteID, actorID); err == nil {
		t.Fatal("legacy direct-route constraint allowed a second active magic link")
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply migration 75: %v", err)
	}
	latestSchema = true
	if _, err := service.RotateDistributionAccessRoute(ctx, tenantID, entityID, magic.Distribution.ID, second.RouteID, actorID); err != nil {
		t.Fatalf("reapplied migration did not restore independent magic links: %v", err)
	}

	otpSubjectID := "9d777777-7777-7777-8777-777777777776"
	otp, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: tenantID, LegalEntityID: entityID, FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: otpSubjectID, Title: "Verified invitation replacement", Purpose: "Verify one active OTP route per recipient.",
		AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(4 * time.Hour), RouteExpiresAt: now.Add(3 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "verified@example.test", AudienceHint: "v***@example.test", ContactLabel: "Evidence owner"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN',updated_at=$2 WHERE id=$1::uuid`, otp.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}
	otpRoutes, err := service.IssueDistributionAccessRoutes(ctx, tenantID, entityID, otp.Distribution.ID, actorID)
	if err != nil || len(otpRoutes) != 1 {
		t.Fatalf("issue OTP route: %+v %v", otpRoutes, err)
	}
	active, err := store.ListActiveAccessRoutes(ctx, tenantID, entityID, otp.Distribution.ID, now)
	if err != nil || len(active) != 1 {
		t.Fatalf("load OTP route: %+v %v", active, err)
	}
	duplicate, _, err := service.engine.IssueRoute(AccessRouteInput{
		TenantID: tenantID, LegalEntityID: entityID, DistributionID: otp.Distribution.ID, RecipientID: active[0].RecipientID,
		Policy: AccessDirectEmailOTP, AudienceHint: active[0].AudienceHint, RouteExpiresAt: otp.Distribution.RouteExpiresAt, Deadline: otp.Distribution.Deadline, CreatedBy: actorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccessRoutes(ctx, []AccessRoute{duplicate}); err == nil {
		t.Fatal("migration allowed two active direct OTP routes for one recipient")
	}
}
