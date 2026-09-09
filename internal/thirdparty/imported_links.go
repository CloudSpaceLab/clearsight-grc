package thirdparty

import "context"

func (r *MemoryAssessmentRepository) StoreImportedLinks(_ context.Context, links []RelationshipLink, versions map[string]int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.relationshipLinkRepo.mu.Lock()
	defer r.relationshipLinkRepo.mu.Unlock()
	for _, link := range links {
		rel, ok := r.relationships[link.RelationshipID]
		if !ok || rel.TenantID != link.TenantID || rel.LegalEntityID != link.LegalEntityID {
			return ErrNotFound
		}
		if rel.Version != versions[rel.ID] {
			return ErrVersionConflict
		}
		vendor, exists := r.vendors[rel.VendorID]
		if !exists || vendor.Status != VendorActive || rel.Status == RelationshipTerminated {
			return ErrNotFound
		}
		if _, exists := r.relationshipLinkRepo.links[link.ID]; exists {
			return ErrVersionConflict
		}
	}
	for _, link := range links {
		r.relationshipLinkRepo.links[link.ID] = link
	}
	return nil
}
