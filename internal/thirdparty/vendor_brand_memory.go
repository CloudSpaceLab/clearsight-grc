package thirdparty

import (
	"context"
	"sort"
)

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
		ActorPrincipalID: actorID, EventType: eventType, WebsiteDomain: vendor.WebsiteDomain, OccurredAt: vendor.UpdatedAt,
	}
	r.vendorIdentityEvents = append(r.vendorIdentityEvents, event)
	r.vendorIdentityOutbox = append(r.vendorIdentityOutbox, event)
}

func vendorBrandJobKey(tenantID, vendorID string) string { return tenantID + "\x00" + vendorID }
