package thirdparty

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu                      sync.RWMutex
	vendors                 map[string]Vendor
	relationships           map[string]Relationship
	vendorBrandAssets       map[string]VendorBrandAsset
	vendorBrandJobs         map[string]VendorBrandJob
	vendorBrandEvents       []VendorBrandEvent
	vendorBrandOutbox       []VendorBrandEvent
	vendorBrandReceipts     map[string]VendorBrandReceipt
	vendorBrandReservations map[string]VendorBrandUploadReservation
	vendorIdentityEvents    []VendorIdentityEvent
	vendorIdentityOutbox    []VendorIdentityEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		vendors: map[string]Vendor{}, relationships: map[string]Relationship{},
		vendorBrandAssets: map[string]VendorBrandAsset{}, vendorBrandJobs: map[string]VendorBrandJob{}, vendorBrandReceipts: map[string]VendorBrandReceipt{}, vendorBrandReservations: map[string]VendorBrandUploadReservation{},
	}
}

func (r *MemoryRepository) CreateRelationship(_ context.Context, record CreateRecord) (Aggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	createdVendor := false
	if record.ReuseVendor {
		existing, ok := r.vendors[record.Vendor.ID]
		if !ok || existing.TenantID != record.Vendor.TenantID {
			return Aggregate{}, ErrNotFound
		}
		record.Vendor = existing
		record.Relationship.VendorID = existing.ID
	} else if record.Vendor.SourceID != "" && record.Vendor.ExternalRef != "" {
		found := false
		for _, existing := range r.vendors {
			if existing.TenantID == record.Vendor.TenantID && existing.SourceID == record.Vendor.SourceID && existing.ExternalRef == record.Vendor.ExternalRef {
				record.Vendor = existing
				record.Relationship.VendorID = existing.ID
				found = true
				break
			}
		}
		createdVendor = !found
	} else {
		createdVendor = true
	}
	if !record.ReuseVendor && createdVendor {
		r.vendors[record.Vendor.ID] = record.Vendor
		if record.BrandJob != nil {
			job := *record.BrandJob
			r.vendorBrandJobs[vendorBrandJobKey(job.TenantID, job.VendorID)] = job
		}
		r.appendVendorIdentityAudit(record.Vendor, record.ActorID, VendorIdentityCreatedEvent)
	}
	r.relationships[record.Relationship.ID] = record.Relationship
	return Aggregate{Vendor: record.Vendor, Relationship: record.Relationship}, nil
}

func (r *MemoryRepository) UpdateRelationship(_ context.Context, record UpdateRecord) (Aggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.relationships[record.ID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return Aggregate{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		return Aggregate{}, ErrVersionConflict
	}
	vendor, ok := r.vendors[current.VendorID]
	if !ok || vendor.TenantID != record.TenantID {
		return Aggregate{}, ErrNotFound
	}
	record.Relationship.ID, record.Relationship.TenantID = current.ID, current.TenantID
	record.Relationship.LegalEntityID, record.Relationship.VendorID = current.LegalEntityID, current.VendorID
	record.Relationship.BusinessOwnerPrincipalID = current.BusinessOwnerPrincipalID
	record.Relationship.SourceID, record.Relationship.ExternalRef = current.SourceID, current.ExternalRef
	record.Relationship.Status, record.Relationship.CreatedAt = current.Status, current.CreatedAt
	record.Relationship.Version = current.Version + 1
	r.relationships[current.ID] = record.Relationship
	return Aggregate{Vendor: vendor, Relationship: record.Relationship}, nil
}

func (r *MemoryRepository) GetRelationship(_ context.Context, scope Scope, relationshipID string) (Aggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	relationship, ok := r.relationships[relationshipID]
	if !ok || relationship.TenantID != scope.TenantID || relationship.LegalEntityID != scope.LegalEntityID {
		return Aggregate{}, ErrNotFound
	}
	vendor, ok := r.vendors[relationship.VendorID]
	if !ok || vendor.TenantID != scope.TenantID {
		return Aggregate{}, ErrNotFound
	}
	return Aggregate{Vendor: vendor, Relationship: relationship}, nil
}

func (r *MemoryRepository) ListRelationships(_ context.Context, filter ListFilter) (RelationshipPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query := strings.ToLower(strings.TrimSpace(filter.Search))
	items := make([]Aggregate, 0)
	for _, relationship := range r.relationships {
		if relationship.TenantID != filter.TenantID || relationship.LegalEntityID != filter.LegalEntityID {
			continue
		}
		vendor, ok := r.vendors[relationship.VendorID]
		if !ok || vendor.TenantID != filter.TenantID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(vendor.LegalName+" "+vendor.TradingName+" "+vendor.RegistrationRef+" "+vendor.SourceID+" "+vendor.ExternalRef+" "+relationship.ServiceName+" "+relationship.ExternalRef), query) {
			continue
		}
		items = append(items, Aggregate{Vendor: vendor, Relationship: relationship})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Relationship.UpdatedAt.Equal(items[j].Relationship.UpdatedAt) {
			return items[i].Relationship.ID > items[j].Relationship.ID
		}
		return items[i].Relationship.UpdatedAt.After(items[j].Relationship.UpdatedAt)
	})
	if filter.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(filter.Cursor)
		if err != nil {
			return RelationshipPage{}, ErrInvalid
		}
		start := 0
		for start < len(items) {
			value := items[start].Relationship
			if value.UpdatedAt.Before(cursorTime) || (value.UpdatedAt.Equal(cursorTime) && value.ID < cursorID) {
				break
			}
			start++
		}
		items = items[start:]
	}
	page := RelationshipPage{Items: []Aggregate{}}
	if len(items) > filter.Limit {
		page.Items = append(page.Items, items[:filter.Limit]...)
		last := page.Items[len(page.Items)-1].Relationship
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	} else {
		page.Items = append(page.Items, items...)
	}
	return page, nil
}

func encodeCursor(updatedAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d|%s", updatedAt.UnixNano(), id)))
}

func decodeCursor(value string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, "", err
	}
	var unixNano int64
	var id string
	if _, err = fmt.Sscanf(string(raw), "%d|%s", &unixNano, &id); err != nil || id == "" {
		return time.Time{}, "", ErrInvalid
	}
	return time.Unix(0, unixNano).UTC(), id, nil
}
