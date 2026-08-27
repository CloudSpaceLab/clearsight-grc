package evidence

import (
	"context"
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
	Address      string    `json:"-"`
	Code         string    `json:"-"`
	ChallengeID  string    `json:"challenge_id"`
	DistributionID string  `json:"distribution_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OTPSendReceipt is safe for public callers. The code and full address remain
// inside the protected delivery boundary.
type OTPSendReceipt struct {
	ChallengeID string    `json:"challenge_id"`
	Hint        string    `json:"hint"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type otpChallengeSnapshot struct {
	Challenge OTPChallenge
	Found     bool
}

type accessSessionCommit struct {
	Route              AccessRoute
	Recipient          DistributionRecipient
	Session            Session
	Challenge          *OTPChallenge
	ExpectedAttempts   int
	ExpectedResends    int
	ExpectedDigest     []byte
	ExpectedRedemptions int
}

// DistributionAccessStore owns the durable, concurrency-sensitive pieces of
// the external access ceremony. Implementations must use atomic compare-and-
// mutate semantics for challenge updates and session commits.
type DistributionAccessStore interface {
	GetDistribution(context.Context, string, string, string) (DistributionBundle, error)
	CreateAccessRoutes(context.Context, []AccessRoute) error
	AccessRouteBySelectorHash(context.Context, []byte) (AccessRoute, error)
	AccessRouteByID(context.Context, string, string, string, string) (AccessRoute, error)
	ProtectedRecipientForAccess(context.Context, AccessRoute, string) (DistributionRecipient, protectedRecipientAddress, error)
	ActiveOTPChallenge(context.Context, AccessRoute, string, time.Time) (otpChallengeSnapshot, error)
	CreateOTPChallenge(context.Context, OTPChallenge) error
	UpdateOTPChallenge(context.Context, OTPChallenge, int, int, []byte) error
	CommitAccessSession(context.Context, accessSessionCommit) error
	RotateAccessRoute(context.Context, AccessRoute, AccessRoute, time.Time) error
	RevokeAccessRoute(context.Context, AccessRoute, time.Time) error
	DistributionSessionByTokenHash(context.Context, []byte, time.Time) (Session, error)
}
