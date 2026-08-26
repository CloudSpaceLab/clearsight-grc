package thirdparty

import (
	"context"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (r *MemoryRepository) GetVendorForBrandDiscovery(_ context.Context, tenantID, vendorID string) (Vendor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	vendor, ok := r.vendors[vendorID]
	if !ok || vendor.TenantID != tenantID {
		return Vendor{}, ErrNotFound
	}
	return vendor, nil
}

func (r *MemoryRepository) ClaimVendorBrandJobs(_ context.Context, workerID string, now time.Time, lease time.Duration, maxAttempts, limit int) ([]VendorBrandJob, error) {
	if workerID == "" || lease <= 0 || maxAttempts < 1 || maxAttempts > 20 {
		return nil, ErrInvalid
	}
	limit = boundedVendorBrandJobLimit(limit)
	now = now.UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	candidates := make([]VendorBrandJob, 0)
	for key, job := range r.vendorBrandJobs {
		claimable := job.State == VendorBrandJobReady || (job.State == VendorBrandJobLeased && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(now))
		if !claimable || job.AvailableAt.After(now) {
			continue
		}
		if job.Attempts >= maxAttempts {
			job.State = VendorBrandJobFailed
			job.LeaseToken = ""
			job.LeaseExpiresAt = nil
			job.LastFailureCode = VendorBrandFailureAttemptsExhausted
			job.UpdatedAt = now
			job.Version++
			r.vendorBrandJobs[key] = job
			continue
		}
		candidates = append(candidates, job)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].AvailableAt.Equal(candidates[j].AvailableAt) {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].AvailableAt.Before(candidates[j].AvailableAt)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	for index := range candidates {
		job := candidates[index]
		token, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		expires := now.Add(lease)
		job.State = VendorBrandJobLeased
		job.Attempts++
		job.LeaseToken = token
		job.LeaseExpiresAt = &expires
		job.UpdatedAt = now
		job.Version++
		r.vendorBrandJobs[vendorBrandJobKey(job.TenantID, job.VendorID)] = job
		candidates[index] = job
	}
	return candidates, nil
}

func (r *MemoryRepository) CompleteVendorBrandJob(_ context.Context, claim VendorBrandJob, asset VendorBrandAsset, at time.Time) (VendorBrandAsset, error) {
	at = at.UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	key := vendorBrandJobKey(claim.TenantID, claim.VendorID)
	current, ok := r.vendorBrandJobs[key]
	if !ok || !sameVendorBrandLease(current, claim, at) {
		return VendorBrandAsset{}, ErrVendorBrandJobLeaseLost
	}
	vendor, ok := r.vendors[claim.VendorID]
	if !ok || vendor.TenantID != claim.TenantID || vendor.Version != claim.VendorVersion || vendor.WebsiteDomain != claim.WebsiteDomain {
		return VendorBrandAsset{}, ErrVendorBrandJobStale
	}
	for assetKey, existing := range r.vendorBrandAssets {
		if existing.TenantID == claim.TenantID && existing.VendorID == claim.VendorID && existing.SourceKind == VendorBrandAssetDiscovered && existing.State == VendorBrandAssetCurrent {
			existing.State = VendorBrandAssetSuperseded
			existing.UpdatedAt = at
			existing.Version++
			r.vendorBrandAssets[assetKey] = existing
		}
	}
	r.vendorBrandAssets[asset.ID] = asset
	current.State = VendorBrandJobCompleted
	current.LeaseToken = ""
	current.LeaseExpiresAt = nil
	current.LastFailureCode = ""
	current.UpdatedAt = at
	current.Version++
	r.vendorBrandJobs[key] = current
	event := VendorBrandEvent{TenantID: claim.TenantID, VendorID: claim.VendorID, AssetID: asset.ID, AssetVersion: asset.Version, VendorVersion: claim.VendorVersion, EventType: VendorBrandDiscoveredEvent, ArtifactKey: asset.ArtifactKey, SourceDigest: asset.SourceDigest, OccurredAt: at}
	r.vendorBrandEvents = append(r.vendorBrandEvents, event)
	r.vendorBrandOutbox = append(r.vendorBrandOutbox, event)
	return asset, nil
}

func (r *MemoryRepository) CancelVendorBrandJob(_ context.Context, claim VendorBrandJob, code string, at time.Time) error {
	if !validVendorBrandFailureCode(code) {
		return ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := vendorBrandJobKey(claim.TenantID, claim.VendorID)
	current, ok := r.vendorBrandJobs[key]
	if !ok || !sameVendorBrandLease(current, claim, at) {
		return ErrVendorBrandJobLeaseLost
	}
	current.State = VendorBrandJobCancelled
	current.WebsiteDomain = ""
	current.LeaseToken = ""
	current.LeaseExpiresAt = nil
	current.LastFailureCode = code
	current.UpdatedAt = at.UTC()
	current.Version++
	r.vendorBrandJobs[key] = current
	return nil
}

func (r *MemoryRepository) FailVendorBrandJob(_ context.Context, claim VendorBrandJob, maxAttempts int, code string, at, availableAt time.Time) (VendorBrandJob, error) {
	if maxAttempts < 1 || maxAttempts > 20 || !validVendorBrandFailureCode(code) || availableAt.Before(at) {
		return VendorBrandJob{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := vendorBrandJobKey(claim.TenantID, claim.VendorID)
	current, ok := r.vendorBrandJobs[key]
	if !ok || !sameVendorBrandLease(current, claim, at) {
		return VendorBrandJob{}, ErrVendorBrandJobLeaseLost
	}
	if current.Attempts >= maxAttempts {
		current.State = VendorBrandJobFailed
	} else {
		current.State = VendorBrandJobReady
		current.AvailableAt = availableAt.UTC()
	}
	current.LeaseToken = ""
	current.LeaseExpiresAt = nil
	current.LastFailureCode = code
	current.UpdatedAt = at.UTC()
	current.Version++
	r.vendorBrandJobs[key] = current
	return current, nil
}

func sameVendorBrandLease(current, claim VendorBrandJob, at time.Time) bool {
	return current.State == VendorBrandJobLeased && current.LeaseToken != "" && current.LeaseToken == claim.LeaseToken && current.LeaseExpiresAt != nil && current.LeaseExpiresAt.After(at.UTC())
}

var _ VendorBrandWorkerRepository = (*MemoryRepository)(nil)
