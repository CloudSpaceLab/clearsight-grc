package thirdparty

import "time"

type VendorStatus string

const (
	VendorActive   VendorStatus = "ACTIVE"
	VendorInactive VendorStatus = "INACTIVE"
)

type RelationshipStatus string

const (
	RelationshipProposed    RelationshipStatus = "PROPOSED"
	RelationshipUnderReview RelationshipStatus = "UNDER_REVIEW"
	RelationshipActive      RelationshipStatus = "ACTIVE"
	RelationshipRestricted  RelationshipStatus = "RESTRICTED"
	RelationshipSuspended   RelationshipStatus = "SUSPENDED"
	RelationshipExiting     RelationshipStatus = "EXITING"
	RelationshipTerminated  RelationshipStatus = "TERMINATED"
)

type Criticality string

const (
	CriticalityStandard  Criticality = "STANDARD"
	CriticalityImportant Criticality = "IMPORTANT"
	CriticalityCritical  Criticality = "CRITICAL"
)

type PrivacyRole string

const (
	PrivacyNone            PrivacyRole = "NONE"
	PrivacyProcessor       PrivacyRole = "PROCESSOR"
	PrivacyJointController PrivacyRole = "JOINT_CONTROLLER"
)

type Actor struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type Vendor struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenant_id"`
	LegalName       string        `json:"legal_name"`
	TradingName     string        `json:"trading_name,omitempty"`
	RegistrationRef string        `json:"registration_ref,omitempty"`
	Jurisdiction    string        `json:"jurisdiction,omitempty"`
	SourceID        string        `json:"source_id,omitempty"`
	ExternalRef     string        `json:"external_ref,omitempty"`
	WebsiteDomain   WebsiteDomain `json:"website_domain,omitempty"`
	Status          VendorStatus  `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Version         int64         `json:"version"`
}

type Relationship struct {
	ID                       string             `json:"id"`
	TenantID                 string             `json:"tenant_id"`
	LegalEntityID            string             `json:"legal_entity_id"`
	VendorID                 string             `json:"vendor_id"`
	ServiceName              string             `json:"service_name"`
	BusinessOwnerPrincipalID string             `json:"business_owner_principal_id"`
	Criticality              Criticality        `json:"criticality"`
	PrivacyRole              PrivacyRole        `json:"privacy_role"`
	Status                   RelationshipStatus `json:"status"`
	EffectiveFrom            *time.Time         `json:"effective_from,omitempty"`
	RenewalAt                *time.Time         `json:"renewal_at,omitempty"`
	SourceID                 string             `json:"source_id,omitempty"`
	ExternalRef              string             `json:"external_ref,omitempty"`
	CreatedAt                time.Time          `json:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at"`
	Version                  int64              `json:"version"`
}

type Aggregate struct {
	Vendor       Vendor                  `json:"vendor"`
	Relationship Relationship            `json:"relationship"`
	Brand        VendorBrandPresentation `json:"brand"`
}

type CreateRelationshipInput struct {
	ExistingRelationshipID string      `json:"existing_relationship_id,omitempty"`
	LegalName              string      `json:"legal_name"`
	TradingName            string      `json:"trading_name,omitempty"`
	RegistrationRef        string      `json:"registration_ref,omitempty"`
	Jurisdiction           string      `json:"jurisdiction,omitempty"`
	SourceID               string      `json:"source_id,omitempty"`
	ExternalRef            string      `json:"external_ref,omitempty"`
	WebsiteDomain          string      `json:"website_domain,omitempty"`
	ServiceName            string      `json:"service_name"`
	Criticality            Criticality `json:"criticality"`
	PrivacyRole            PrivacyRole `json:"privacy_role"`
	EffectiveFrom          *time.Time  `json:"effective_from,omitempty"`
	RenewalAt              *time.Time  `json:"renewal_at,omitempty"`
}

type CreateRecord struct {
	Vendor       Vendor
	Relationship Relationship
	ReuseVendor  bool
	ActorID      string
	BrandJob     *VendorBrandJob
}

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type ListInput struct {
	Search string `json:"search,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ListFilter struct {
	Scope
	Search string
	Cursor string
	Limit  int
}

type RelationshipPage struct {
	Items      []Aggregate `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type UpdateRelationshipInput struct {
	ExpectedVersion int64       `json:"expected_version"`
	ServiceName     string      `json:"service_name"`
	Criticality     Criticality `json:"criticality"`
	PrivacyRole     PrivacyRole `json:"privacy_role"`
	EffectiveFrom   *time.Time  `json:"effective_from,omitempty"`
	RenewalAt       *time.Time  `json:"renewal_at,omitempty"`
}

type UpdateRecord struct {
	Scope
	ID              string
	ExpectedVersion int64
	Relationship    Relationship
	ActorID         string
}

type UpdateVendorIdentityInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	LegalName       string `json:"legal_name"`
	TradingName     string `json:"trading_name,omitempty"`
	RegistrationRef string `json:"registration_ref,omitempty"`
	Jurisdiction    string `json:"jurisdiction,omitempty"`
	WebsiteDomain   string `json:"website_domain,omitempty"`
}

type UpdateVendorIdentityRecord struct {
	Scope
	ID              string
	ExpectedVersion int64
	Vendor          Vendor
	ActorID         string
	BrandJob        *VendorBrandJob
}
