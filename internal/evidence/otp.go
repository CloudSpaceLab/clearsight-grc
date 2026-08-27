package evidence

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	OTPCodeDigits   = 6
	OTPValidity     = 10 * time.Minute
	OTPMaxAttempts  = 5
	OTPMaxResends   = 3
	otpCodeVariants = 1_000_000
)

// OTPChallenge stores only a keyed digest. The plaintext code is transient and
// available solely in IssuedOTP for delivery through a protected adapter.
type OTPChallenge struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	LegalEntityID  string     `json:"legal_entity_id"`
	DistributionID string     `json:"distribution_id"`
	RouteID        string     `json:"route_id"`
	RecipientID    string     `json:"-"`
	Digest         []byte     `json:"-"`
	ExpiresAt      time.Time  `json:"expires_at"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
	Resends        int        `json:"resends"`
	MaxResends     int        `json:"max_resends"`
	ConsumedAt     *time.Time `json:"consumed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (challenge OTPChallenge) String() string {
	return fmt.Sprintf("OTPChallenge{id:%q route_id:%q expires_at:%s attempts:%d/%d resends:%d/%d recipient:protected digest:protected}",
		challenge.ID, challenge.RouteID, challenge.ExpiresAt.UTC().Format(time.RFC3339), challenge.Attempts,
		challenge.MaxAttempts, challenge.Resends, challenge.MaxResends)
}

func (challenge OTPChallenge) GoString() string {
	return challenge.String()
}

type IssuedOTP struct {
	ChallengeID string    `json:"challenge_id"`
	Code        string    `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (issued IssuedOTP) String() string {
	return fmt.Sprintf("IssuedOTP{challenge_id:%q expires_at:%s code:protected}", issued.ChallengeID, issued.ExpiresAt.UTC().Format(time.RFC3339))
}

func (issued IssuedOTP) GoString() string {
	return issued.String()
}

type OTPVerification struct {
	challengeID    string
	routeID        string
	recipientID    string
	distributionID string
}

type OTPService struct {
	hmacKey      [32]byte
	configured   bool
	now          func() time.Time
	generateCode func() (string, error)
}

func NewOTPService(hmacKey [32]byte) *OTPService {
	service := &OTPService{hmacKey: hmacKey, configured: securityKeyConfigured(hmacKey), now: time.Now}
	service.generateCode = secureOTPCode
	return service
}

func (service *OTPService) Issue(route AccessRoute, recipient DistributionRecipient, now time.Time) (OTPChallenge, IssuedOTP, error) {
	now = service.normalizeNow(now)
	if service == nil || !service.configured || !otpRouteRecipientEligible(route, recipient, now) {
		return OTPChallenge{}, IssuedOTP{}, ErrAccessVerificationFailed
	}
	challengeID, err := id.NewUUIDv7()
	if err != nil {
		return OTPChallenge{}, IssuedOTP{}, ErrAccessVerificationFailed
	}
	code, err := service.nextCode()
	if err != nil {
		return OTPChallenge{}, IssuedOTP{}, ErrAccessVerificationFailed
	}
	expiresAt := now.Add(OTPValidity)
	if route.ExpiresAt.Before(expiresAt) {
		expiresAt = route.ExpiresAt
	}
	if !expiresAt.After(now) {
		return OTPChallenge{}, IssuedOTP{}, ErrAccessVerificationFailed
	}
	challenge := OTPChallenge{
		ID: challengeID, TenantID: route.TenantID, LegalEntityID: route.LegalEntityID,
		DistributionID: route.DistributionID, RouteID: route.ID, RecipientID: recipient.ID,
		Digest: service.digest(challengeID, code), ExpiresAt: expiresAt,
		MaxAttempts: OTPMaxAttempts, MaxResends: OTPMaxResends, CreatedAt: now,
	}
	return challenge, IssuedOTP{ChallengeID: challenge.ID, Code: code, ExpiresAt: challenge.ExpiresAt}, nil
}

func (service *OTPService) Resend(route AccessRoute, challenge *OTPChallenge, now time.Time) (IssuedOTP, error) {
	now = service.normalizeNow(now)
	if service == nil || !service.configured || !otpChallengeUsable(route, challenge, now) || challenge.Resends >= challenge.MaxResends {
		service.consumeDummyOTP(challenge, "000000")
		return IssuedOTP{}, ErrAccessVerificationFailed
	}
	code, err := service.nextCode()
	if err != nil {
		return IssuedOTP{}, ErrAccessVerificationFailed
	}
	expiresAt := now.Add(OTPValidity)
	if route.ExpiresAt.Before(expiresAt) {
		expiresAt = route.ExpiresAt
	}
	if !expiresAt.After(now) {
		return IssuedOTP{}, ErrAccessVerificationFailed
	}
	challenge.Digest = service.digest(challenge.ID, code)
	challenge.ExpiresAt = expiresAt
	challenge.Resends++
	return IssuedOTP{ChallengeID: challenge.ID, Code: code, ExpiresAt: challenge.ExpiresAt}, nil
}

func (service *OTPService) Verify(route AccessRoute, challenge *OTPChallenge, code string, now time.Time) (OTPVerification, error) {
	now = service.normalizeNow(now)
	challengeID := "unknown-challenge"
	if challenge != nil && challenge.ID != "" {
		challengeID = challenge.ID
	}
	supplied := service.digest(challengeID, code)
	usable := service != nil && service.configured && otpChallengeUsable(route, challenge, now)
	expected := service.digest("unknown-challenge", "000000")
	if usable {
		expected = challenge.Digest
	}
	matched := subtle.ConstantTimeCompare(expected, supplied) == 1
	if !usable || !validOTPCode(code) || !matched {
		if usable && challenge.Attempts < challenge.MaxAttempts {
			challenge.Attempts++
		}
		return OTPVerification{}, ErrAccessVerificationFailed
	}
	consumedAt := now
	challenge.ConsumedAt = &consumedAt
	return OTPVerification{
		challengeID: challenge.ID, routeID: challenge.RouteID, recipientID: challenge.RecipientID,
		distributionID: challenge.DistributionID,
	}, nil
}

func otpRouteRecipientEligible(route AccessRoute, recipient DistributionRecipient, now time.Time) bool {
	if route.Policy != AccessDirectEmailOTP && route.Policy != AccessSharedEmailOTP {
		return false
	}
	if !accessRouteActive(route, now) {
		return false
	}
	eligible := eligibleAccessRecipients(route, []DistributionRecipient{recipient})
	return len(eligible) == 1
}

func otpChallengeUsable(route AccessRoute, challenge *OTPChallenge, now time.Time) bool {
	return challenge != nil && challenge.ID != "" && len(challenge.Digest) == sha256.Size && challenge.ConsumedAt == nil &&
		challenge.Attempts < challenge.MaxAttempts && challenge.MaxAttempts >= 1 && challenge.MaxAttempts <= OTPMaxAttempts &&
		challenge.Resends <= challenge.MaxResends && challenge.MaxResends >= 1 && challenge.MaxResends <= OTPMaxResends &&
		challenge.CreatedAt.After(time.Time{}) && !now.Before(challenge.CreatedAt) && challenge.ExpiresAt.After(now) &&
		challenge.TenantID == route.TenantID && challenge.LegalEntityID == route.LegalEntityID &&
		challenge.DistributionID == route.DistributionID && challenge.RouteID == route.ID &&
		strings.TrimSpace(challenge.RecipientID) != "" && accessRouteOpen(route, now) &&
		(route.Policy == AccessSharedEmailOTP || route.RecipientID == challenge.RecipientID)
}

func validOTPCode(code string) bool {
	if len(code) != OTPCodeDigits {
		return false
	}
	for _, value := range []byte(code) {
		if value < '0' || value > '9' {
			return false
		}
	}
	return true
}

func secureOTPCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(otpCodeVariants))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func (service *OTPService) nextCode() (string, error) {
	if service == nil || service.generateCode == nil {
		return "", ErrAccessVerificationFailed
	}
	code, err := service.generateCode()
	if err != nil || !validOTPCode(code) {
		return "", ErrAccessVerificationFailed
	}
	return code, nil
}

func (service *OTPService) digest(challengeID, code string) []byte {
	var key [32]byte
	if service != nil {
		key = service.hmacKey
	}
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(challengeID + "|" + code))
	return mac.Sum(nil)
}

func (service *OTPService) consumeDummyOTP(challenge *OTPChallenge, code string) {
	challengeID := "unknown-challenge"
	if challenge != nil && challenge.ID != "" {
		challengeID = challenge.ID
	}
	service.compareDummyDigest(service.digest(challengeID, code))
}

func (service *OTPService) compareDummyDigest(supplied []byte) {
	dummy := service.digest("unknown-challenge", "000000")
	_ = subtle.ConstantTimeCompare(dummy, supplied)
}

func (service *OTPService) normalizeNow(now time.Time) time.Time {
	if !now.IsZero() {
		return now.UTC()
	}
	if service != nil && service.now != nil {
		return service.now().UTC()
	}
	return time.Now().UTC()
}

var (
	_ fmt.Stringer   = OTPChallenge{}
	_ fmt.GoStringer = OTPChallenge{}
	_ fmt.Stringer   = IssuedOTP{}
	_ fmt.GoStringer = IssuedOTP{}
)
