package attention

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type digestRepoStub struct {
	candidates  []DigestCandidate
	claims      []EmailDeliveryRecord
	records     []EmailDeliveryRecord
	claimResult bool
}

func (s *digestRepoStub) DueDigests(context.Context, time.Time, int) ([]DigestCandidate, error) {
	return s.candidates, nil
}

func (s *digestRepoStub) ClaimDigest(_ context.Context, _ DigestCandidate, record EmailDeliveryRecord) (bool, error) {
	s.claims = append(s.claims, record)
	return s.claimResult, nil
}

func (s *digestRepoStub) RecordDigest(_ context.Context, _ DigestCandidate, record EmailDeliveryRecord) error {
	s.records = append(s.records, record)
	return nil
}

func TestDailyDigestUsesOneClaimAndGenericCounts(t *testing.T) {
	now := time.Date(2026, 10, 5, 7, 0, 3, 0, time.UTC)
	repo := &digestRepoStub{
		candidates: []DigestCandidate{{
			TenantID: "bank", PrincipalID: "person", LocalDate: now,
			BrandName: "ClearSight", RecipientName: "Risk Officer", RecipientAddress: "risk@example.test",
			MaterialChanges: 8, AssignedWork: 3, DueSoon: 1, Worsened: 1, Cleared: 2,
		}},
		claimResult: true,
	}
	delivery := &criticalEmailDeliveryStub{receipt: evidence.InvitationDeliveryReceipt{
		Status: evidence.InvitationDelivered, DeliveredAt: &now,
	}}
	maintainer := NewDigestMaintainer(repo, delivery, "https://clearsight.example.test")
	processed, err := maintainer.Maintain(context.Background(), now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || delivery.calls != 1 || len(repo.claims) != 1 || len(repo.records) != 1 {
		t.Fatalf("processed=%d calls=%d claims=%d records=%d", processed, delivery.calls, len(repo.claims), len(repo.records))
	}
	if repo.claims[0].Status != "DELIVERY_STARTED" || repo.records[0].Status != "DELIVERED" {
		t.Fatalf("claim=%#v record=%#v", repo.claims[0], repo.records[0])
	}
}

func TestDailyDigestMissingContactIsFinalWithoutSMTP(t *testing.T) {
	now := time.Date(2026, 10, 5, 7, 0, 3, 0, time.UTC)
	repo := &digestRepoStub{
		candidates: []DigestCandidate{{
			TenantID: "bank", PrincipalID: "person", LocalDate: now,
			BrandName: "ClearSight", RecipientName: "Risk Officer",
		}},
		claimResult: true,
	}
	delivery := &criticalEmailDeliveryStub{}
	maintainer := NewDigestMaintainer(repo, delivery, "https://clearsight.example.test")
	if _, err := maintainer.Maintain(context.Background(), now, 10); err != nil {
		t.Fatal(err)
	}
	if delivery.calls != 0 || repo.claims[0].Status != "CONTACT_UNAVAILABLE" {
		t.Fatalf("calls=%d claim=%#v", delivery.calls, repo.claims)
	}
}


func TestDailyDigestSkipsEmptySummary(t *testing.T) {
	now := time.Date(2026, 10, 5, 7, 0, 3, 0, time.UTC)
	repo := &digestRepoStub{
		candidates: []DigestCandidate{{
			TenantID: "bank", PrincipalID: "person", LocalDate: now,
			BrandName: "ClearSight", RecipientName: "Risk Officer", RecipientAddress: "risk@example.test",
		}},
		claimResult: true,
	}
	delivery := &criticalEmailDeliveryStub{}
	maintainer := NewDigestMaintainer(repo, delivery, "https://clearsight.example.test")
	processed, err := maintainer.Maintain(context.Background(), now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || delivery.calls != 0 || len(repo.claims) != 0 || len(repo.records) != 0 {
		t.Fatalf("processed=%d calls=%d claims=%d records=%d", processed, delivery.calls, len(repo.claims), len(repo.records))
	}
}
