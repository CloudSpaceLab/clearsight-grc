//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"testing"
	"time"
)

func TestPostgresAmendRollsBackWhenReceiptCannotBeBuilt(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, actor, recipient, form, subject := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	setupDistributionFixture(t, ctx, pool, tenant, entity, actor, recipient, form, now)
	t.Cleanup(func() { cleanupDistributionTenant(context.Background(), pool, tenant) })
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	created, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "VENDOR", SubjectID: subject, Title: "Review", Purpose: "Check controls", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 8, Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: actor, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: recipient}}})
	if err != nil {
		t.Fatal(err)
	}
	firstDeadline := now.Add(84 * time.Hour)
	amended, err := store.AmendDistribution(ctx, tenant, entity, created.Distribution.ID, AmendDistributionInput{ExpectedVersion: created.Distribution.Version, Deadline: &firstDeadline, ActorID: actor}, now)
	if err != nil || amended.Bundle.Distribution.Version != created.Distribution.Version+1 || !amended.Bundle.Distribution.Deadline.Equal(firstDeadline) || amended.Bundle.Workspace.ID != created.Workspace.ID || len(amended.Bundle.Recipients) != 1 || amended.Bundle.Recipients[0].PrincipalID != recipient {
		t.Fatalf("amendment receipt: %+v %v", amended, err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM capture_response_workspaces WHERE distribution_id=$1::uuid`, created.Distribution.ID); err != nil {
		t.Fatal(err)
	}
	deadline := now.Add(96 * time.Hour)
	if _, err = store.AmendDistribution(ctx, tenant, entity, created.Distribution.ID, AmendDistributionInput{ExpectedVersion: amended.Bundle.Distribution.Version, Deadline: &deadline, ActorID: actor}, now); err == nil {
		t.Fatal("missing receipt accepted")
	}
	var version int64
	var events, outbox int
	if err = pool.QueryRow(ctx, `SELECT d.version,(SELECT count(*) FROM capture_distribution_events e WHERE e.distribution_id=d.id AND e.event_type='FORM_DISTRIBUTION_AMENDED'),(SELECT count(*) FROM outbox_events o WHERE o.aggregate_id=d.id AND o.event_type='FORM_DISTRIBUTION_AMENDED') FROM capture_form_distributions d WHERE d.id=$1::uuid`, created.Distribution.ID).Scan(&version, &events, &outbox); err != nil {
		t.Fatal(err)
	}
	if version != amended.Bundle.Distribution.Version || events != 1 || outbox != 1 {
		t.Fatalf("failed receipt committed version=%d events=%d outbox=%d", version, events, outbox)
	}
}
