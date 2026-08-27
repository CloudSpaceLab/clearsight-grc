package evidence

import (
	"context"
	"fmt"
	"time"
)

// recipientAddressRevealer is deliberately separate from the creation-time
// protector so read paths cannot accidentally gain plaintext access. Only the
// access ceremony is given this capability.
type recipientAddressRevealer interface {
	RevealRecipientAddress(context.Context, string, string, string, protectedRecipientAddress) (string, error)
}

// OTPDelivery is the narrow boundary allowed to receive both the transient OTP
// and the decrypted recipient address. Neither value is returned by public
// service methods or stored in durable state.
type OTPDelivery interface {
	DeliverDistributionOTP(context.Context, DistributionOTPDelivery) error
}

type DistributionOTPDelivery struct {
	Address        string    `json:"-"`
	Code           string    `json:"-"`
	ChallengeID    string    `json:"challenge_id"`
	DistributionID string    `json:"distribution_id"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (value DistributionOTPDelivery) String() string {
	return fmt.Sprintf("DistributionOTPDelivery{challenge_id:%q distribution_id:%q expires_at:%s address:protected code:protected}", value.ChallengeID, value.DistributionID, value.ExpiresAt.UTC().Format(time.RFC3339))
}

func (value DistributionOTPDelivery) GoString() string { return value.String() }

// OTPSendReceipt is safe for public callers. The code and full address remain
// inside the protected delivery boundary.
type OTPSendReceipt struct {
	ChallengeID string    `json:"challenge_id"`
	Hint        string    `json:"hint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// DistributionAccessSession is the bounded capability produced only by the new
// distribution access ceremony. Legacy invitation sessions remain unchanged.
type DistributionAccessSession struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"-"`
	LegalEntityID  string          `json:"-"`
	DistributionID string          `json:"distribution_id"`
	RecipientID    string          `json:"-"`
	RequestID      string          `json:"request_id"`
	RouteID        string          `json:"-"`
	AudienceHint   string          `json:"audience_hint"`
	Assurance      AccessAssurance `json:"assurance"`
	TokenHash      []byte          `json:"-"`
	ExpiresAt      time.Time       `json:"expires_at"`
	RevokedAt      *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type RedeemedDistributionSession struct {
	SessionID      string          `json:"session_id"`
	SessionToken   string          `json:"session_token"`
	DistributionID string          `json:"distribution_id"`
	RequestID      string          `json:"request_id"`
	AudienceHint   string          `json:"audience_hint"`
	Assurance      AccessAssurance `json:"assurance"`
	ExpiresAt      time.Time       `json:"expires_at"`
}

func (value RedeemedDistributionSession) String() string {
	return fmt.Sprintf("RedeemedDistributionSession{session_id:%q distribution_id:%q request_id:%q assurance:%q expires_at:%s token:protected}", value.SessionID, value.DistributionID, value.RequestID, value.Assurance, value.ExpiresAt.UTC().Format(time.RFC3339))
}

func (value RedeemedDistributionSession) GoString() string { return value.String() }

type otpChallengeSnapshot struct {
	Challenge OTPChallenge
	Found     bool
}

type accessSessionCommit struct {
	Route               AccessRoute
	Recipient           DistributionRecipient
	Session             DistributionAccessSession
	Challenge           *OTPChallenge
	ExpectedAttempts    int
	ExpectedResends     int
	ExpectedDigest      []byte
	ExpectedRedemptions int
}

// DistributionAccessStore owns the durable, concurrency-sensitive pieces of
// the external access ceremony. Implementations must use atomic compare-and-
// mutate semantics for challenge updates and session commits.
type DistributionAccessStore interface {
	GetDistribution(context.Context, string, string, string) (DistributionBundle, error)
	GetRequest(context.Context, string, string) (Request, error)
	CreateAccessRoutes(context.Context, []AccessRoute) error
	AccessRouteBySelectorHash(context.Context, []byte) (AccessRoute, error)
	AccessRouteByID(context.Context, string, string, string, string) (AccessRoute, error)
	ProtectedRecipientForAccess(context.Context, AccessRoute, string) (DistributionRecipient, protectedRecipientAddress, error)
	ActiveOTPChallenge(context.Context, AccessRoute, string, time.Time) (otpChallengeSnapshot, error)
	OTPChallengeByID(context.Context, AccessRoute, string, time.Time) (OTPChallenge, error)
	CreateOTPChallenge(context.Context, OTPChallenge) error
	UpdateOTPChallenge(context.Context, OTPChallenge, int, int, []byte) error
	CommitAccessSession(context.Context, accessSessionCommit) error
	RotateAccessRoute(context.Context, AccessRoute, AccessRoute, time.Time) error
	RevokeAccessRoute(context.Context, AccessRoute, time.Time) error
	DistributionSessionByTokenHash(context.Context, []byte, time.Time) (DistributionAccessSession, error)
}

var (
	_ fmt.Stringer   = DistributionOTPDelivery{}
	_ fmt.GoStringer = DistributionOTPDelivery{}
	_ fmt.Stringer   = RedeemedDistributionSession{}
	_ fmt.GoStringer = RedeemedDistributionSession{}
)
