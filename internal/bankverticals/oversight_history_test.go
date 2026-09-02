package bankverticals

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestInstallSampleCreatesReconstructableOversightHistory(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	repository := continuity.NewMemoryRepository()
	base := continuity.NewServiceWithClock(repository, func() time.Time { return now })
	service := NewService(base, newReferenceEvidenceService(now, "bank-ng"))
	configureReferenceTimeline(service, repository)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now

	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, config.LegalEntityID)
	matters, err := base.ListMatters(ctx, config.TenantID, "", 200)
	if err != nil {
		t.Fatal(err)
	}

	vendorReviews := 0
	additionalTypes := map[continuity.MatterType]bool{}
	owners := map[string]bool{}
	reopened := false
	overdue := false
	for _, aggregate := range matters {
		var scope map[string]any
		if json.Unmarshal(aggregate.Matter.Scope, &scope) != nil || scope["journey_code"] != referenceOversightJourneyCode {
			continue
		}
		if scope["sample"] != true || scope["reference_data"] != true {
			t.Fatalf("reference history scope = %#v", scope)
		}
		if aggregate.Matter.Status != continuity.MatterClosed || aggregate.Matter.ClosedAt == nil {
			t.Fatalf("reference history %s is not closed: %#v", aggregate.Matter.Title, aggregate.Matter)
		}
		if aggregate.Matter.CreatedAt.Equal(*aggregate.Matter.ClosedAt) {
			t.Fatalf("reference history %s has no elapsed lifecycle", aggregate.Matter.Title)
		}
		if aggregate.Matter.Type == continuity.MatterVendorReview {
			vendorReviews++
		} else {
			additionalTypes[aggregate.Matter.Type] = true
		}
		owners[aggregate.Matter.OwnerPrincipalID] = true
		reopened = reopened || aggregate.Matter.ReopenCount > 0
		overdue = overdue || (aggregate.Matter.DueAt != nil && aggregate.Matter.ClosedAt.After(*aggregate.Matter.DueAt))
		for _, action := range aggregate.Actions {
			if action.ImplementedAt == nil || action.ImplementedAt.Equal(action.CreatedAt) {
				t.Fatalf("reference action %s has no elapsed implementation history", action.Title)
			}
			for _, result := range aggregate.VerificationResults {
				if result.ReviewerPrincipalID == action.OwnerPrincipalID {
					t.Fatalf("outcome reviewer %s is not independent of action owner", result.ReviewerPrincipalID)
				}
			}
		}
	}
	if vendorReviews < 5 || len(additionalTypes) < 2 || len(owners) < 2 || !reopened || !overdue {
		t.Fatalf("reference cohort vendor=%d additional=%d owners=%d reopened=%t overdue=%t", vendorReviews, len(additionalTypes), len(owners), reopened, overdue)
	}
}

func TestInstallSampleRejectsShallowReferenceHistoryAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	repository := continuity.NewMemoryRepository()
	base := continuity.NewServiceWithClock(repository, func() time.Time { return now })
	service := NewService(base, newReferenceEvidenceService(now, "bank-ng"))
	configureReferenceTimeline(service, repository)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), config.TenantID, config.LegalEntityID)

	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	versions := map[string]int64{}
	for _, spec := range referenceOversightHistories(config) {
		aggregate, err := base.MatterByTriggerKey(ctx, config.TenantID, "reference:oversight:"+spec.Key)
		if err != nil {
			t.Fatal(err)
		}
		if reason := incompleteReferenceHistoryReason(aggregate, spec); reason != "" {
			t.Fatalf("reference history %s incomplete: %s", spec.Key, reason)
		}
		versions[spec.Key] = aggregate.Matter.Version
	}

	if _, err := service.InstallSample(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	for _, spec := range referenceOversightHistories(config) {
		aggregate, err := base.MatterByTriggerKey(ctx, config.TenantID, "reference:oversight:"+spec.Key)
		if err != nil {
			t.Fatal(err)
		}
		if aggregate.Matter.Version != versions[spec.Key] {
			t.Fatalf("reference history %s version changed on repeat install: got %d want %d", spec.Key, aggregate.Matter.Version, versions[spec.Key])
		}
	}
}

func configureReferenceTimeline(service *Service, repository continuity.Repository) {
	service.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
		return continuity.NewServiceWithClock(repository, func() time.Time { return at })
	})
}
