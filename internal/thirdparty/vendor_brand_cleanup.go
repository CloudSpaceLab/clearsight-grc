package thirdparty

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type VendorBrandReservationCleaner struct {
	Repository VendorBrandReservationRepository
	Store      evidence.ObjectStore
	Retention  time.Duration
	Lease      time.Duration
}

func (c *VendorBrandReservationCleaner) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if c == nil || c.Repository == nil || c.Store == nil {
		return 0, errors.New("vendor brand reservation cleanup is not configured")
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	retention := c.Retention
	if retention <= 0 {
		retention = 24 * time.Hour
	}
	lease := c.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	items, err := c.Repository.ClaimExpiredVendorBrandReservations(ctx, now.UTC(), now.UTC().Add(-retention), lease, limit)
	if err != nil {
		return 0, err
	}
	var failures []error
	for _, item := range items {
		referenced, checkErr := c.Repository.VendorBrandArtifactReferenced(ctx, item)
		if checkErr != nil {
			failures = append(failures, checkErr)
			continue
		}
		if !referenced {
			if deleteErr := c.Store.Delete(ctx, item.ArtifactKey); deleteErr != nil {
				failures = append(failures, deleteErr)
				continue
			}
		}
		if completeErr := c.Repository.CompleteVendorBrandReservationCleanup(ctx, item, referenced, now.UTC()); completeErr != nil {
			failures = append(failures, completeErr)
		}
	}
	return len(items), errors.Join(failures...)
}
