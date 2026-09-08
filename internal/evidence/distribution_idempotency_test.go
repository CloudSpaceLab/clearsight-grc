package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestVendorRequestRetryKeepsOneDistributionAndRejectsChangedPayload(t *testing.T) {
	now := time.Now().UTC()
	store := NewMemoryDistributionStore(NewMemoryRepository(nil, nil), stubDistributionFormReader{form: activeDistributionForm()}, stubRecipientProtector{})
	input := CreateDistributionInput{TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectType: "VENDOR", SubjectID: "00000000-0000-0000-0000-000000000101", Title: "Security review", Purpose: "Confirm controls", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 10, Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: "owner", Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test"}}, IdempotencyKey: "batch:relationship"}
	first, err := store.CreateDistribution(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateDistribution(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Distribution.ID != second.Distribution.ID || len(store.distributions) != 1 || len(store.events) != 1 {
		t.Fatal("retry created duplicate request")
	}
	store.protector = stubRecipientProtector{err: errors.New("recipient protection unavailable")}
	store.forms = stubDistributionFormReader{err: errors.New("form is retired")}
	if retry, err := store.CreateDistribution(context.Background(), input); err != nil || retry.Distribution.ID != first.Distribution.ID {
		t.Fatalf("committed receipt must recover without creation dependencies: %+v %v", retry, err)
	}
	input.Recipients[0].Address = "different@example.test"
	if _, err = store.CreateDistribution(context.Background(), input); !errors.Is(err, ErrDistributionConflict) {
		t.Fatalf("changed recipient accepted: %v", err)
	}
}
