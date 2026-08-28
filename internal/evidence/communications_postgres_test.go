//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCommunicationGovernancePersistsActivationFallbackAndRollback(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenantID := mustCommunicationTestID(t)
	entityID := mustCommunicationTestID(t)
	makerID := mustCommunicationTestID(t)
	checkerID := mustCommunicationTestID(t)
	tenantSlug := "communication-" + tenantID[len(tenantID)-12:]
	now := time.Date(2026, 8, 28, 7, 0, 0, 0, time.UTC)
	setupCommunicationFixture(t, ctx, pool, tenantID, tenantSlug, entityID, makerID, checkerID, now)
	t.Cleanup(func() { cleanupCommunicationFixture(context.Background(), pool, tenantID) })

	store := NewPostgresCommunicationStore(NewPostgresRepository(pool))
	service := NewCommunicationService(store)
	service.now = func() time.Time { return now }

	profile, err := service.CreateProfileRevision(ctx, CreateCommunicationProfileInput{
		TenantID: tenantSlug, LegalEntityID: "COMMS", DefaultLocale: "en-NG", BankName: "Example Bank",
		SupportContact: "grc@example.test", EffectiveFrom: now.Add(-time.Hour), MakerID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err = service.TransitionProfile(ctx, tenantSlug, "COMMS", profile.Version, CommunicationTransitionInput{
		ExpectedVersion: profile.Version, To: CommunicationPendingApproval, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err = service.TransitionProfile(ctx, tenantSlug, "COMMS", profile.Version, CommunicationTransitionInput{
		ExpectedVersion: profile.Version, To: CommunicationActive, ActorID: checkerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status != CommunicationActive || profile.CheckerID != checkerID || profile.TenantID != tenantID || profile.LegalEntityID != entityID {
		t.Fatalf("unexpected canonical active profile: %+v", profile)
	}

	first := createAndActivatePostgresCommunicationTemplate(t, ctx, service, tenantSlug, "COMMS", now, makerID, checkerID, "{{bank_name}}: {{form_title}}")
	second := createAndActivatePostgresCommunicationTemplate(t, ctx, service, tenantSlug, "COMMS", now, makerID, checkerID, "Action required: {{form_title}}")
	if first.Version != 1 || second.Version != 2 {
		t.Fatalf("unexpected template versions: first=%d second=%d", first.Version, second.Version)
	}

	persistedFirst, err := service.store.GetTemplate(ctx, tenantSlug, "COMMS", CommunicationInvitation, "en-NG", first.Version)
	if err != nil {
		t.Fatal(err)
	}
	if persistedFirst.Status != CommunicationRetired || second.Status != CommunicationActive {
		t.Fatalf("overlapping activation did not retire prior revision: first=%s second=%s", persistedFirst.Status, second.Status)
	}
	resolved, err := service.ResolveTemplate(ctx, tenantSlug, "COMMS", CommunicationInvitation, "fr-FR", now)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Version != second.Version || resolved.Locale != "en-NG" {
		t.Fatalf("default-locale resolution did not select latest active revision: %+v", resolved)
	}

	rollback, err := service.RollbackTemplate(ctx, tenantSlug, "COMMS", CommunicationInvitation, "en-NG", first.Version, makerID)
	if err != nil {
		t.Fatal(err)
	}
	if rollback.Version != 3 || rollback.RollbackOriginVersion != first.Version || rollback.Status != CommunicationDraft || rollback.SubjectTemplate != first.SubjectTemplate || !communicationNodesEqual(rollback.Document, first.Document) {
		t.Fatalf("exact rollback lineage was not persisted: %+v", rollback)
	}

	var profileCount, templateCount, activeCount, retiredCount, auditCount, outboxCount, exposedPayloads int
	if err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM form_communication_profiles WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid),
		  (SELECT count(*) FROM form_communication_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid),
		  (SELECT count(*) FROM form_communication_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action='INVITATION' AND locale='en-NG' AND status='ACTIVE'),
		  (SELECT count(*) FROM form_communication_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action='INVITATION' AND locale='en-NG' AND status='RETIRED'),
		  (SELECT count(*) FROM audit_events WHERE tenant_id=$1::uuid AND subject_type IN ('FORM_COMMUNICATION_PROFILE','FORM_COMMUNICATION_TEMPLATE')),
		  (SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type IN ('FORM_COMMUNICATION_PROFILE','FORM_COMMUNICATION_TEMPLATE')),
		  (SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type IN ('FORM_COMMUNICATION_PROFILE','FORM_COMMUNICATION_TEMPLATE') AND (payload::text LIKE '%Action required:%' OR payload::text LIKE '%secure_form_link%'))`,
		tenantID, entityID,
	).Scan(&profileCount, &templateCount, &activeCount, &retiredCount, &auditCount, &outboxCount, &exposedPayloads); err != nil {
		t.Fatal(err)
	}
	if profileCount != 1 || templateCount != 3 || activeCount != 1 || retiredCount != 1 {
		t.Fatalf("unexpected persisted communication counts: profiles=%d templates=%d active=%d retired=%d", profileCount, templateCount, activeCount, retiredCount)
	}
	if auditCount < 11 || outboxCount < 11 || exposedPayloads != 0 {
		t.Fatalf("governance records are incomplete or exposed copy: audit=%d outbox=%d exposed=%d", auditCount, outboxCount, exposedPayloads)
	}
}

func createAndActivatePostgresCommunicationTemplate(t *testing.T, ctx context.Context, service *CommunicationService, tenantID, legalEntityID string, now time.Time, makerID, checkerID, subject string) CommunicationTemplate {
	t.Helper()
	base := validCommunicationTemplate(CommunicationInvitation, "en-NG")
	value, err := service.CreateTemplateRevision(ctx, CreateCommunicationTemplateInput{
		TenantID: tenantID, LegalEntityID: legalEntityID, Action: CommunicationInvitation, Locale: "en-NG",
		SubjectTemplate: subject, Document: base.Document, EffectiveFrom: now.Add(-time.Hour), MakerID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.TransitionTemplate(ctx, tenantID, legalEntityID, value.Action, value.Locale, value.Version, CommunicationTransitionInput{
		ExpectedVersion: value.Version, To: CommunicationPendingApproval, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err = service.TransitionTemplate(ctx, tenantID, legalEntityID, value.Action, value.Locale, value.Version, CommunicationTransitionInput{
		ExpectedVersion: value.Version, To: CommunicationActive, ActorID: checkerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustCommunicationTestID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func setupCommunicationFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, tenantSlug, entityID, makerID, checkerID string, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Communication Integration');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($3::uuid,$1::uuid,'COMMS','Communication Entity','NG',$6);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($4::uuid,$1::uuid,'PERSON','Communication Maker','ACTIVE',$6),
			($5::uuid,$1::uuid,'PERSON','Communication Checker','ACTIVE',$6);`,
		pgx.QueryExecModeSimpleProtocol, tenantID, tenantSlug, entityID, makerID, checkerID, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func cleanupCommunicationFixture(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	for _, statement := range []string{
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM audit_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM form_delivery_attempts WHERE tenant_id=$1::uuid`,
		`DELETE FROM form_communication_templates WHERE tenant_id=$1::uuid`,
		`DELETE FROM form_communication_profiles WHERE tenant_id=$1::uuid`,
		`DELETE FROM form_brand_assets WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = pool.Exec(ctx, statement, tenantID)
	}
}

func TestCommunicationGovernanceOutboxMetadataDoesNotContainTemplateCopy(t *testing.T) {
	if strings.Contains(strings.ToLower("legal_entity_id action locale version status"), "subject_template") {
		t.Fatal("safe metadata contract unexpectedly includes template copy")
	}
}
