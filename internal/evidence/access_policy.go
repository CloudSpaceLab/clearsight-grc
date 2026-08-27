package evidence

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	accessRouteSelectorBytes = 32
	sharedRouteRedemptions   = 20
)

var (
	ErrDistributionAccessUnavailable = errors.New("distribution access is unavailable")
	ErrAccessVerificationFailed      = errors.New("access verification failed")
)

// AccessRoute is the internal durable route model. SelectorHash and RecipientID
// are intentionally not public projections.
type AccessRoute struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	LegalEntityID  string       `json:"legal_entity_id"`
	DistributionID string       `json:"distribution_id"`
	RecipientID    string       `json:"-"`
	Policy         AccessPolicy `json:"policy"`
	SelectorHash   []byte       `json:"-"`
	AudienceHint   string       `json:"audience_hint,omitempty"`
	ExpiresAt      time.Time    `json:"expires_at"`
	MaxRedemptions int          `json:"max_redemptions"`
	Redemptions    int          `json:"redemptions"`
	RevokedAt      *time.Time   `json:"revoked_at,omitempty"`
	CreatedBy      string       `json:"created_by"`
	CreatedAt      time.Time    `json:"created_at"`
}

type AccessRouteInput struct {
	TenantID       string
	LegalEntityID  string
	DistributionID string
	RecipientID    string
	Policy         AccessPolicy
	AudienceHint   string
	RouteExpiresAt time.Time
	Deadline       time.Time
	MaxRedemptions int
	CreatedBy      string
}

// IssuedAccessRoute exposes the selector exactly once to the trusted caller.
// Formatted output redacts it to avoid accidental log disclosure.
type IssuedAccessRoute struct {
	RouteID   string       `json:"route_id"`
	Selector  string       `json:"selector"`
	Policy    AccessPolicy `json:"policy"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type MaskedRecipient struct {
	SelectorID   string `json:"selector_id"`
	Hint         string `json:"hint"`
	ContactLabel string `json:"contact_label,omitempty"`
}

type AccessStart struct {
	Policy     AccessPolicy      `json:"policy"`
	Recipients []MaskedRecipient `json:"recipients,omitempty"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

// AccessGrant is an internal capability seed. RecipientID is never a public
// selector and must only be copied into a bounded server-side session.
type AccessGrant struct {
	RouteID        string          `json:"-"`
	TenantID       string          `json:"-"`
	DistributionID string          `json:"distribution_id"`
	RecipientID    string          `json:"-"`
	Assurance      AccessAssurance `json:"assurance"`
	ExpiresAt      time.Time       `json:"expires_at"`
}

type AccessPolicyEngine struct {
	hmacKey    [32]byte
	configured bool
	random     io.Reader
	now        func() time.Time
}

func NewAccessPolicyEngine(hmacKey [32]byte) *AccessPolicyEngine {
	return &AccessPolicyEngine{hmacKey: hmacKey, configured: securityKeyConfigured(hmacKey), random: rand.Reader, now: time.Now}
}

func (route AccessRoute) String() string {
	return fmt.Sprintf("AccessRoute{id:%q distribution_id:%q policy:%q expires_at:%s recipient:protected selector:protected}", route.ID, route.DistributionID, route.Policy, route.ExpiresAt.UTC().Format(time.RFC3339))
}

func (route AccessRoute) GoString() string {
	return route.String()
}

func (value IssuedAccessRoute) String() string {
	return fmt.Sprintf("IssuedAccessRoute{route_id:%q policy:%q expires_at:%s selector:protected}", value.RouteID, value.Policy, value.ExpiresAt.UTC().Format(time.RFC3339))
}

func (value IssuedAccessRoute) GoString() string {
	return value.String()
}

func (grant AccessGrant) String() string {
	return fmt.Sprintf("AccessGrant{distribution_id:%q assurance:%q expires_at:%s route:protected recipient:protected}", grant.DistributionID, grant.Assurance, grant.ExpiresAt.UTC().Format(time.RFC3339))
}

func (grant AccessGrant) GoString() string {
	return grant.String()
}

var (
	_ fmt.Stringer   = AccessRoute{}
	_ fmt.GoStringer = AccessRoute{}
	_ fmt.Stringer   = IssuedAccessRoute{}
	_ fmt.GoStringer = IssuedAccessRoute{}
	_ fmt.Stringer   = AccessGrant{}
	_ fmt.GoStringer = AccessGrant{}
)
