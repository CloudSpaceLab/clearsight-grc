package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderIdentity  = "X-ClearSight-Identity"
	HeaderTimestamp = "X-ClearSight-Identity-Timestamp"
	HeaderSignature = "X-ClearSight-Identity-Signature"
)

type Authenticator interface {
	Authenticate(*http.Request) (Actor, bool, error)
}

type SignedAuthenticator struct {
	secret  []byte
	maxSkew time.Duration
	now     func() time.Time
}

func NewSignedAuthenticator(secret string, maxSkew time.Duration) (*SignedAuthenticator, error) {
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, fmt.Errorf("identity signing secret must contain at least 32 characters")
	}
	if maxSkew <= 0 || maxSkew > 10*time.Minute {
		return nil, fmt.Errorf("identity maximum clock skew must be between 1 second and 10 minutes")
	}
	return &SignedAuthenticator{secret: []byte(secret), maxSkew: maxSkew, now: time.Now}, nil
}

func (a *SignedAuthenticator) Authenticate(r *http.Request) (Actor, bool, error) {
	envelope := strings.TrimSpace(r.Header.Get(HeaderIdentity))
	timestamp := strings.TrimSpace(r.Header.Get(HeaderTimestamp))
	signature := strings.TrimSpace(r.Header.Get(HeaderSignature))
	if envelope == "" && timestamp == "" && signature == "" {
		return Actor{}, false, nil
	}
	if envelope == "" || timestamp == "" || signature == "" {
		return Actor{}, false, ErrInvalidIdentity
	}
	issuedUnix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return Actor{}, false, ErrInvalidIdentity
	}
	issued := time.Unix(issuedUnix, 0).UTC()
	now := a.now().UTC()
	if issued.Before(now.Add(-a.maxSkew)) || issued.After(now.Add(a.maxSkew)) {
		return Actor{}, false, ErrExpiredIdentity
	}
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return Actor{}, false, ErrInvalidIdentity
	}
	message := strings.Join([]string{r.Method, r.URL.EscapedPath(), timestamp, envelope}, "\n")
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(message))
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return Actor{}, false, ErrInvalidIdentity
	}
	payload, err := base64.RawURLEncoding.DecodeString(envelope)
	if err != nil {
		return Actor{}, false, ErrInvalidIdentity
	}
	var actor Actor
	if err := json.Unmarshal(payload, &actor); err != nil {
		return Actor{}, false, ErrInvalidIdentity
	}
	if actor.IssuedAt.IsZero() {
		actor.IssuedAt = issued
	}
	if err := actor.Valid(now); err != nil {
		return Actor{}, false, err
	}
	return actor, true, nil
}

type DevelopmentAuthenticator struct {
	TenantID      string
	PrincipalID   string
	LegalEntityID string
	now           func() time.Time
}

func NewDevelopmentAuthenticator(tenantID, principalID, legalEntityID string) *DevelopmentAuthenticator {
	return &DevelopmentAuthenticator{TenantID: tenantID, PrincipalID: principalID, LegalEntityID: legalEntityID, now: time.Now}
}

func (a *DevelopmentAuthenticator) Authenticate(r *http.Request) (Actor, bool, error) {
	tenant := strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Tenant"))
	principal := strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Principal"))
	entity := strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Legal-Entity"))
	if tenant == "" {
		tenant = a.TenantID
	}
	if principal == "" {
		principal = a.PrincipalID
	}
	if entity == "" {
		entity = a.LegalEntityID
	}
	if tenant == "" || principal == "" || entity == "" {
		return Actor{}, false, nil
	}
	now := a.now().UTC()
	return Actor{TenantID: tenant, PrincipalID: principal, LegalEntityID: entity, Kind: "PERSON", AuthenticationMethod: "DEVELOPMENT", AssuranceLevel: "DEMO", SessionID: "development", IssuedAt: now, ExpiresAt: now.Add(time.Hour)}, true, nil
}
