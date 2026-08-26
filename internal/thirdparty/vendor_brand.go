package thirdparty

import (
	"errors"
	"net/netip"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const (
	VendorIdentityObjectType    = "VENDOR"
	VendorIdentityUpdateCommand = "thirdparty.vendor.identity.update"

	VendorIdentityCreatedEvent = "VendorIdentityCreated"
	VendorIdentityUpdatedEvent = "VendorIdentityUpdated"
)

var (
	ErrVendorIdentityAuthorityUnavailable = errors.New("vendor identity authority service is unavailable")
	ErrVendorIdentityMismatch             = errors.New("vendor identity command identity does not match verified context")
)

// WebsiteDomain is the canonical ASCII DNS hostname used to discover a
// vendor's public website icon. It never contains a URL scheme or authority
// component.
type WebsiteDomain string

type VendorBrandAssetSource string

const (
	VendorBrandAssetDiscovered       VendorBrandAssetSource = "DISCOVERED"
	VendorBrandAssetApprovedOverride VendorBrandAssetSource = "APPROVED_OVERRIDE"
)

type VendorBrandAssetState string

const (
	VendorBrandAssetCurrent    VendorBrandAssetState = "CURRENT"
	VendorBrandAssetSuperseded VendorBrandAssetState = "SUPERSEDED"
)

type VendorBrandJobState string

const (
	VendorBrandJobReady     VendorBrandJobState = "READY"
	VendorBrandJobLeased    VendorBrandJobState = "LEASED"
	VendorBrandJobCompleted VendorBrandJobState = "COMPLETED"
	VendorBrandJobFailed    VendorBrandJobState = "FAILED"
	VendorBrandJobCancelled VendorBrandJobState = "CANCELLED"
)

const VendorBrandDiscoveryJobType = "DISCOVER_ICON"

type VendorBrandAsset struct {
	ID                    string                 `json:"id"`
	TenantID              string                 `json:"tenant_id"`
	VendorID              string                 `json:"vendor_id"`
	SourceKind            VendorBrandAssetSource `json:"source_kind"`
	State                 VendorBrandAssetState  `json:"state"`
	SourceDomain          WebsiteDomain          `json:"source_domain,omitempty"`
	ArtifactKey           string                 `json:"artifact_key"`
	SourceDigest          string                 `json:"source_digest,omitempty"`
	MediaType             string                 `json:"media_type"`
	PixelWidth            int                    `json:"pixel_width"`
	PixelHeight           int                    `json:"pixel_height"`
	ByteSize              int64                  `json:"byte_size"`
	RetrievedAt           *time.Time             `json:"retrieved_at,omitempty"`
	NextRefreshAt         *time.Time             `json:"next_refresh_at,omitempty"`
	ApprovedByPrincipalID string                 `json:"approved_by_principal_id,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	Version               int64                  `json:"version"`
}

type VendorBrandJob struct {
	ID              string              `json:"id"`
	TenantID        string              `json:"tenant_id"`
	VendorID        string              `json:"vendor_id"`
	VendorVersion   int64               `json:"vendor_version"`
	JobType         string              `json:"job_type"`
	WebsiteDomain   WebsiteDomain       `json:"website_domain,omitempty"`
	State           VendorBrandJobState `json:"state"`
	Attempts        int                 `json:"attempts"`
	AvailableAt     time.Time           `json:"available_at"`
	LeaseToken      string              `json:"lease_token,omitempty"`
	LeaseExpiresAt  *time.Time          `json:"lease_expires_at,omitempty"`
	LastFailureCode string              `json:"last_failure_code,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	Version         int64               `json:"version"`
}

type VendorIdentityEvent struct {
	TenantID         string
	VendorID         string
	VendorVersion    int64
	ActorPrincipalID string
	EventType        string
	LegalName        string
	TradingName      string
	RegistrationRef  string
	Jurisdiction     string
	WebsiteDomain    WebsiteDomain
	Status           VendorStatus
	OccurredAt       time.Time
}

// NormalizeWebsiteDomain converts an internationalized DNS hostname to its
// lowercase IDNA lookup form and rejects URL and IP-address input.
func NormalizeWebsiteDomain(value string) (WebsiteDomain, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 253 || strings.ContainsAny(value, "/\\@:#?[]") || strings.HasSuffix(value, ".") {
		return "", ErrInvalid
	}
	if _, err := netip.ParseAddr(value); err == nil {
		return "", ErrInvalid
	}
	value, err := idna.Lookup.ToASCII(value)
	if err != nil {
		return "", ErrInvalid
	}
	value = strings.ToLower(value)
	if value == "" || len(value) > 253 {
		return "", ErrInvalid
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalid
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
				return "", ErrInvalid
			}
		}
	}
	return WebsiteDomain(value), nil
}
