package thirdparty

import (
	"context"
	"sort"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (r *MemoryRepository) ReserveApprovedVendorBrand(_ context.Context, record VendorBrandMutationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.vendorVisibleInScope(record.Scope, record.VendorID) {
		return ErrNotFound
	}
	key := record.TenantID + "\x00" + record.VendorID + "\x00" + record.IdempotencyKey
	if receipt, ok := r.vendorBrandReceipts[key]; ok {
		_, err := vendorBrandReceiptVersion(receipt, VendorBrandApproveCommand, record.ExpectedVersion)
		return err
	}
	if existing, ok := r.vendorBrandReservations[key]; ok {
		if existing.ArtifactKey != record.Asset.ArtifactKey || existing.SourceDigest != record.Asset.SourceDigest || existing.ExpectedVersion != record.ExpectedVersion {
			return ErrVersionConflict
		}
		return nil
	}
	r.vendorBrandReservations[key] = VendorBrandUploadReservation{TenantID: record.TenantID, VendorID: record.VendorID, IdempotencyKey: record.IdempotencyKey, ArtifactKey: record.Asset.ArtifactKey, SourceDigest: record.Asset.SourceDigest, State: "RESERVED", ExpectedVersion: record.ExpectedVersion, CreatedAt: record.OccurredAt, UpdatedAt: record.OccurredAt}
	return nil
}

func (r *MemoryRepository) PutApprovedVendorBrand(_ context.Context, record VendorBrandMutationRecord) (VendorBrandAsset, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.vendorVisibleInScope(record.Scope, record.VendorID) {
		return VendorBrandAsset{}, 0, ErrNotFound
	}
	receiptKey := record.TenantID + "\x00" + record.VendorID + "\x00" + record.IdempotencyKey
	if receipt, ok := r.vendorBrandReceipts[receiptKey]; ok {
		if receipt.Command != VendorBrandApproveCommand || receipt.ExpectedVersion != record.ExpectedVersion {
			return VendorBrandAsset{}, 0, ErrVersionConflict
		}
		for _, a := range r.vendorBrandAssets {
			if a.TenantID == record.TenantID && a.VendorID == record.VendorID && a.SourceKind == VendorBrandAssetApprovedOverride && a.State == VendorBrandAssetCurrent {
				return a, receipt.ResultVersion, nil
			}
		}
		return record.Asset, receipt.ResultVersion, nil
	}
	currentVersion := r.nextVendorBrandEventVersion(record.TenantID, record.VendorID) - 1
	if currentVersion != record.ExpectedVersion {
		return VendorBrandAsset{}, currentVersion, ErrBrandVersionConflict
	}
	for key, a := range r.vendorBrandAssets {
		if a.TenantID == record.TenantID && a.VendorID == record.VendorID && a.SourceKind == VendorBrandAssetApprovedOverride && a.State == VendorBrandAssetCurrent {
			a.State = VendorBrandAssetSuperseded
			a.UpdatedAt = record.OccurredAt
			a.Version++
			r.vendorBrandAssets[key] = a
		}
	}
	r.vendorBrandAssets[record.Asset.ID] = record.Asset
	version := currentVersion + 1
	event := VendorBrandEvent{TenantID: record.TenantID, VendorID: record.VendorID, AssetID: record.Asset.ID, AssetVersion: record.Asset.Version, EventType: VendorBrandApprovedEvent, ArtifactKey: record.Asset.ArtifactKey, SourceDigest: record.Asset.SourceDigest, OccurredAt: record.OccurredAt, EventVersion: version}
	r.vendorBrandEvents = append(r.vendorBrandEvents, event)
	r.vendorBrandOutbox = append(r.vendorBrandOutbox, event)
	r.vendorBrandReceipts[receiptKey] = VendorBrandReceipt{TenantID: record.TenantID, VendorID: record.VendorID, IdempotencyKey: record.IdempotencyKey, Command: VendorBrandApproveCommand, ExpectedVersion: record.ExpectedVersion, ResultVersion: version}
	reservation := r.vendorBrandReservations[receiptKey]
	reservation.State = "COMMITTED"
	reservation.UpdatedAt = record.OccurredAt
	r.vendorBrandReservations[receiptKey] = reservation
	return record.Asset, version, nil
}

func (r *MemoryRepository) RemoveApprovedVendorBrand(_ context.Context, record VendorBrandMutationRecord) (VendorBrandAsset, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.vendorVisibleInScope(record.Scope, record.VendorID) {
		return VendorBrandAsset{}, 0, ErrNotFound
	}
	receiptKey := record.TenantID + "\x00" + record.VendorID + "\x00" + record.IdempotencyKey
	if receipt, ok := r.vendorBrandReceipts[receiptKey]; ok {
		if receipt.Command != VendorBrandRemoveCommand || receipt.ExpectedVersion != record.ExpectedVersion {
			return VendorBrandAsset{}, 0, ErrVersionConflict
		}
		return receipt.Asset, receipt.ResultVersion, nil
	}
	currentVersion := r.nextVendorBrandEventVersion(record.TenantID, record.VendorID) - 1
	if currentVersion != record.ExpectedVersion {
		return VendorBrandAsset{}, currentVersion, ErrBrandVersionConflict
	}
	var removed VendorBrandAsset
	for key, a := range r.vendorBrandAssets {
		if a.TenantID == record.TenantID && a.VendorID == record.VendorID && a.SourceKind == VendorBrandAssetApprovedOverride && a.State == VendorBrandAssetCurrent {
			a.State = VendorBrandAssetSuperseded
			a.UpdatedAt = record.OccurredAt
			a.Version++
			r.vendorBrandAssets[key] = a
			removed = a
		}
	}
	if removed.ID == "" {
		return VendorBrandAsset{}, currentVersion, ErrVendorBrandOverrideNotFound
	}
	version := currentVersion + 1
	event := VendorBrandEvent{TenantID: record.TenantID, VendorID: record.VendorID, AssetID: removed.ID, AssetVersion: removed.Version, EventType: VendorBrandRemovedEvent, OccurredAt: record.OccurredAt, EventVersion: version}
	r.vendorBrandEvents = append(r.vendorBrandEvents, event)
	r.vendorBrandOutbox = append(r.vendorBrandOutbox, event)
	r.vendorBrandReceipts[receiptKey] = VendorBrandReceipt{TenantID: record.TenantID, VendorID: record.VendorID, IdempotencyKey: record.IdempotencyKey, Command: VendorBrandRemoveCommand, ExpectedVersion: record.ExpectedVersion, ResultVersion: version, Asset: removed}
	return removed, version, nil
}

func (r *MemoryRepository) CurrentVendorBrandVersion(_ context.Context, scope Scope, vendorID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return 0, ErrNotFound
	}
	return r.nextVendorBrandEventVersion(scope.TenantID, vendorID) - 1, nil
}

func (r *MemoryRepository) VendorBrandCommandReceipt(_ context.Context, scope Scope, vendorID, idempotencyKey string) (VendorBrandReceipt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return VendorBrandReceipt{}, ErrNotFound
	}
	receipt, ok := r.vendorBrandReceipts[scope.TenantID+"\x00"+vendorID+"\x00"+idempotencyKey]
	if !ok {
		return VendorBrandReceipt{}, ErrNotFound
	}
	return receipt, nil
}

func (r *MemoryRepository) ClaimExpiredVendorBrandReservations(_ context.Context, now, cutoff time.Time, lease time.Duration, limit int) ([]VendorBrandUploadReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit < 1 {
		limit = 25
	}
	items := []VendorBrandUploadReservation{}
	for key, item := range r.vendorBrandReservations {
		claimable := item.State == "RESERVED" && !item.UpdatedAt.After(cutoff) || item.State == "CLEANING" && item.LeaseExpiresAt != nil && !item.LeaseExpiresAt.After(now)
		if !claimable {
			continue
		}
		token, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		expires := now.Add(lease)
		item.State = "CLEANING"
		item.LeaseToken = token
		item.LeaseExpiresAt = &expires
		item.UpdatedAt = now
		r.vendorBrandReservations[key] = item
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}
func (r *MemoryRepository) VendorBrandArtifactReference(_ context.Context, item VendorBrandUploadReservation) (VendorBrandArtifactReference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	receiptKey := item.TenantID + "\x00" + item.VendorID + "\x00" + item.IdempotencyKey
	expectedAssetID := vendorBrandReservationAssetID(item.TenantID, item.VendorID, item.IdempotencyKey)
	receipt, hasReceipt := r.vendorBrandReceipts[receiptKey]
	for _, a := range r.vendorBrandAssets {
		if a.TenantID == item.TenantID && a.ArtifactKey == item.ArtifactKey {
			if a.ID == expectedAssetID && a.VendorID == item.VendorID && a.SourceDigest == item.SourceDigest && hasReceipt && receipt.Command == VendorBrandApproveCommand && receipt.ExpectedVersion == item.ExpectedVersion {
				return VendorBrandArtifactCommitted, nil
			}
			return VendorBrandArtifactProtected, nil
		}
	}
	for _, other := range r.vendorBrandReservations {
		if other.TenantID == item.TenantID && other.ArtifactKey == item.ArtifactKey && (other.VendorID != item.VendorID || other.IdempotencyKey != item.IdempotencyKey) {
			return VendorBrandArtifactProtected, nil
		}
	}
	return VendorBrandArtifactUnreferenced, nil
}
func (r *MemoryRepository) CompleteVendorBrandReservationCleanup(_ context.Context, item VendorBrandUploadReservation, reference VendorBrandArtifactReference, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := item.TenantID + "\x00" + item.VendorID + "\x00" + item.IdempotencyKey
	current, ok := r.vendorBrandReservations[key]
	if !ok || current.State != "CLEANING" || current.LeaseToken != item.LeaseToken {
		return ErrVersionConflict
	}
	if reference == VendorBrandArtifactCommitted {
		current.State = "COMMITTED"
		current.LeaseToken = ""
		current.LeaseExpiresAt = nil
		current.UpdatedAt = at
		r.vendorBrandReservations[key] = current
	} else {
		delete(r.vendorBrandReservations, key)
	}
	return nil
}

func (r *MemoryRepository) UpdateVendorIdentity(_ context.Context, record UpdateVendorIdentityRecord) (Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.vendors[record.ID]
	if !ok || current.TenantID != record.TenantID || !r.vendorVisibleInScope(record.Scope, record.ID) {
		return Vendor{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return Vendor{}, ErrVersionConflict
	}
	updated := record.Vendor
	updated.ID = current.ID
	updated.TenantID = current.TenantID
	updated.SourceID = current.SourceID
	updated.ExternalRef = current.ExternalRef
	updated.Status = current.Status
	updated.CreatedAt = current.CreatedAt
	updated.Version = current.Version + 1
	r.vendors[current.ID] = updated
	if record.BrandJob != nil {
		job := *record.BrandJob
		key := vendorBrandJobKey(updated.TenantID, updated.ID)
		if existing, exists := r.vendorBrandJobs[key]; exists {
			job.ID = existing.ID
			job.CreatedAt = existing.CreatedAt
			job.Version = existing.Version + 1
		}
		job.TenantID = updated.TenantID
		job.VendorID = updated.ID
		job.VendorVersion = updated.Version
		r.vendorBrandJobs[key] = job
	}
	r.appendVendorIdentityAudit(updated, record.ActorID, VendorIdentityUpdatedEvent)
	return updated, nil
}

func (r *MemoryRepository) GetVendor(_ context.Context, scope Scope, vendorID string) (Vendor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	vendor, ok := r.vendors[vendorID]
	if !ok || vendor.TenantID != scope.TenantID || !r.vendorVisibleInScope(scope, vendorID) {
		return Vendor{}, ErrNotFound
	}
	return vendor, nil
}

func (r *MemoryRepository) GetVendorBrandJob(_ context.Context, scope Scope, vendorID string) (VendorBrandJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return VendorBrandJob{}, ErrNotFound
	}
	job, ok := r.vendorBrandJobs[vendorBrandJobKey(scope.TenantID, vendorID)]
	if !ok {
		return VendorBrandJob{}, ErrNotFound
	}
	return job, nil
}

func (r *MemoryRepository) ListVendorBrandAssets(_ context.Context, scope Scope, vendorID string) ([]VendorBrandAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return nil, ErrNotFound
	}
	values := make([]VendorBrandAsset, 0)
	for _, asset := range r.vendorBrandAssets {
		if asset.TenantID == scope.TenantID && asset.VendorID == vendorID {
			values = append(values, asset)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].UpdatedAt.After(values[j].UpdatedAt)
	})
	return values, nil
}

func (r *MemoryRepository) GetVendorBrandAsset(_ context.Context, scope Scope, vendorID, token string) (VendorBrandAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return VendorBrandAsset{}, ErrNotFound
	}
	for _, asset := range r.vendorBrandAssets {
		if asset.TenantID == scope.TenantID && asset.VendorID == vendorID && brandAssetToken(asset) == token {
			return asset, nil
		}
	}
	return VendorBrandAsset{}, ErrNotFound
}

func (r *MemoryRepository) GetVendorBrandProjection(_ context.Context, scope Scope, vendorID string) (VendorBrandProjection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.vendorVisibleInScope(scope, vendorID) {
		return VendorBrandProjection{}, ErrNotFound
	}
	return r.vendorBrandProjectionLocked(scope.TenantID, vendorID), nil
}

func (r *MemoryRepository) GetVendorBrandProjections(_ context.Context, scope Scope, vendorIDs []string) (map[string]VendorBrandProjection, error) {
	if len(vendorIDs) > 100 {
		return nil, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make(map[string]VendorBrandProjection, len(vendorIDs))
	for _, vendorID := range vendorIDs {
		if r.vendorVisibleInScope(scope, vendorID) {
			values[vendorID] = r.vendorBrandProjectionLocked(scope.TenantID, vendorID)
		}
	}
	return values, nil
}

func (r *MemoryRepository) vendorBrandProjectionLocked(tenantID, vendorID string) VendorBrandProjection {
	value := VendorBrandProjection{VendorID: vendorID}
	vendor := r.vendors[vendorID]
	for _, asset := range r.vendorBrandAssets {
		if asset.TenantID != tenantID || asset.VendorID != vendorID || asset.State != VendorBrandAssetCurrent {
			continue
		}
		candidate := asset
		if asset.SourceKind == VendorBrandAssetApprovedOverride && (value.CurrentApproved == nil || asset.UpdatedAt.After(value.CurrentApproved.UpdatedAt)) {
			value.CurrentApproved = &candidate
		}
		if asset.SourceKind == VendorBrandAssetDiscovered && asset.SourceDomain == vendor.WebsiteDomain && (value.CurrentDiscovered == nil || asset.UpdatedAt.After(value.CurrentDiscovered.UpdatedAt)) {
			value.CurrentDiscovered = &candidate
		}
	}
	if job, ok := r.vendorBrandJobs[vendorBrandJobKey(tenantID, vendorID)]; ok {
		value.JobState = job.State
	}
	for _, event := range r.vendorBrandEvents {
		if event.TenantID == tenantID && event.VendorID == vendorID && event.EventVersion > value.EventVersion {
			value.EventVersion = event.EventVersion
		}
	}
	return value
}

func (r *MemoryRepository) vendorVisibleInScope(scope Scope, vendorID string) bool {
	for _, relationship := range r.relationships {
		if relationship.TenantID == scope.TenantID && relationship.LegalEntityID == scope.LegalEntityID && relationship.VendorID == vendorID {
			return true
		}
	}
	return false
}

func (r *MemoryRepository) appendVendorIdentityAudit(vendor Vendor, actorID, eventType string) {
	event := VendorIdentityEvent{
		TenantID: vendor.TenantID, VendorID: vendor.ID, VendorVersion: vendor.Version,
		ActorPrincipalID: actorID, EventType: eventType,
		LegalName: vendor.LegalName, TradingName: vendor.TradingName, RegistrationRef: vendor.RegistrationRef,
		Jurisdiction: vendor.Jurisdiction, WebsiteDomain: vendor.WebsiteDomain, Status: vendor.Status, OccurredAt: vendor.UpdatedAt,
	}
	r.vendorIdentityEvents = append(r.vendorIdentityEvents, event)
	r.vendorIdentityOutbox = append(r.vendorIdentityOutbox, event)
}

func vendorBrandJobKey(tenantID, vendorID string) string { return tenantID + "\x00" + vendorID }
