package thirdparty

import (
	"context"
	"errors"
)

var (
	ErrInvalid         = errors.New("invalid third-party relationship")
	ErrNotFound        = errors.New("third-party relationship not found")
	ErrVersionConflict = errors.New("third-party relationship version conflict")
)

type Repository interface {
	VendorBrandRepository
	CreateRelationship(context.Context, CreateRecord) (Aggregate, error)
	UpdateRelationship(context.Context, UpdateRecord) (Aggregate, error)
	GetRelationship(context.Context, Scope, string) (Aggregate, error)
	ListRelationships(context.Context, ListFilter) (RelationshipPage, error)
}

type VendorBrandRepository interface {
	UpdateVendorIdentity(context.Context, UpdateVendorIdentityRecord) (Vendor, error)
	GetVendor(context.Context, Scope, string) (Vendor, error)
	GetVendorBrandJob(context.Context, Scope, string) (VendorBrandJob, error)
	ListVendorBrandAssets(context.Context, Scope, string) ([]VendorBrandAsset, error)
	GetVendorBrandAsset(context.Context, Scope, string, string) (VendorBrandAsset, error)
	GetVendorBrandProjection(context.Context, Scope, string) (VendorBrandProjection, error)
	GetVendorBrandProjections(context.Context, Scope, []string) (map[string]VendorBrandProjection, error)
}
