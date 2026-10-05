package attention

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

const DigestWorkClass = "notification-daily-digest"

type DigestCandidate struct {
	TenantID         string
	PrincipalID      string
	LocalDate        time.Time
	BrandName        string
	RecipientName    string
	RecipientAddress string
	MaterialChanges  int
	AssignedWork     int
	DueSoon          int
	Worsened         int
	Cleared          int
}

type DigestRepository interface {
	DueDigests(context.Context, time.Time, int) ([]DigestCandidate, error)
	ClaimDigest(context.Context, DigestCandidate, EmailDeliveryRecord) (bool, error)
	RecordDigest(context.Context, DigestCandidate, EmailDeliveryRecord) error
}

type DigestMaintainer struct {
	repository     DigestRepository
	delivery       governedEmailDelivery
	applicationURL string
}

func NewDigestMaintainer(repository DigestRepository, delivery governedEmailDelivery, applicationURL string) *DigestMaintainer {
	return &DigestMaintainer{
		repository: repository, delivery: delivery,
		applicationURL: strings.TrimRight(strings.TrimSpace(applicationURL), "/"),
	}
}

func (m *DigestMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.repository == nil || m.delivery == nil || m.applicationURL == "" {
		return 0, fmt.Errorf("daily digest delivery is unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	candidates, err := m.repository.DueDigests(ctx, now.UTC(), limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	var failures []error
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}
		if err := m.deliver(ctx, now.UTC(), candidate); err != nil {
			failures = append(failures, err)
		}
		processed++
	}
	return processed, errors.Join(failures...)
}

func (m *DigestMaintainer) deliver(ctx context.Context, now time.Time, candidate DigestCandidate) error {
	address := strings.TrimSpace(candidate.RecipientAddress)
	if !canonicalMailbox(address) {
		claimed, err := m.repository.ClaimDigest(ctx, candidate, EmailDeliveryRecord{
			Status: "CONTACT_UNAVAILABLE", AttemptedAt: now,
		})
		if err != nil || !claimed {
			return err
		}
		return nil
	}
	fingerprint := emailFingerprint(address)
	claimed, err := m.repository.ClaimDigest(ctx, candidate, EmailDeliveryRecord{
		Status: "DELIVERY_STARTED", RecipientFingerprint: fingerprint, AttemptedAt: now,
	})
	if err != nil || !claimed {
		return err
	}

	request, err := evidence.BuildDailyDigestRequest(address, evidence.DailyDigestContext{
		BrandName: candidate.BrandName, RecipientName: candidate.RecipientName,
		HomeURL: m.applicationURL + "/#home",
		MaterialChanges: candidate.MaterialChanges, AssignedWork: candidate.AssignedWork,
		DueSoon: candidate.DueSoon, Worsened: candidate.Worsened, Cleared: candidate.Cleared,
	})
	if err != nil {
		_ = m.repository.RecordDigest(ctx, candidate, EmailDeliveryRecord{
			Status: "PERMANENT_FAILURE", RecipientFingerprint: fingerprint,
			FailureCode: "INVALID_MESSAGE", AttemptedAt: now,
		})
		return nil
	}

	receipt, deliveryErr := m.delivery.DeliverGoverned(ctx, request)
	record := emailRecordFromReceipt(receipt, deliveryErr, fingerprint, now)
	if err := m.repository.RecordDigest(ctx, candidate, record); err != nil {
		return err
	}
	if record.Status == "TEMPORARY_FAILURE" {
		return errors.New("daily digest delivery should be retried")
	}
	return nil
}
