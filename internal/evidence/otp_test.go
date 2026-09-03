package evidence

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestOTPIssueUsesSixDigitsHMACAndTenMinuteExpiry(t *testing.T) {
	now := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	key := repeatedRecipientKey(0xa1)
	service := testOTPService(now, key, "042731")
	route := otpRoute(AccessDirectEmailOTP, "recipient-a", now.Add(time.Hour))
	recipient := accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	challenge, issued, err := service.Issue(route, recipient, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(issued.Code) != OTPCodeDigits || issued.Code != "042731" || !challenge.ExpiresAt.Equal(now.Add(OTPValidity)) {
		t.Fatalf("unexpected OTP issue result: challenge=%+v issued=%+v", challenge, issued)
	}
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(challenge.ID + "|" + issued.Code))
	if !hmac.Equal(challenge.Digest, mac.Sum(nil)) {
		t.Fatal("OTP digest did not bind challengeID|code")
	}
	if challenge.MaxAttempts != OTPMaxAttempts || challenge.MaxResends != OTPMaxResends || challenge.Attempts != 0 || challenge.Resends != 0 {
		t.Fatalf("unexpected OTP limits: %+v", challenge)
	}
}

func TestOTPVerifyIsSingleUseAndReturnsOnlyGenericFailures(t *testing.T) {
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	service := testOTPService(now, repeatedRecipientKey(0xb2), "123456")
	route := otpRoute(AccessDirectEmailOTP, "recipient-a", now.Add(time.Hour))
	recipient := accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	challenge, issued, err := service.Issue(route, recipient, now)
	if err != nil {
		t.Fatal(err)
	}
	verification, err := service.Verify(route, &challenge, issued.Code, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if verification.routeID != route.ID || verification.recipientID != recipient.ID || challenge.ConsumedAt == nil {
		t.Fatalf("unexpected verification result: %+v challenge=%+v", verification, challenge)
	}

	failures := []struct {
		name      string
		route     AccessRoute
		challenge *OTPChallenge
		code      string
		now       time.Time
	}{
		{name: "single-use", route: route, challenge: &challenge, code: issued.Code, now: now.Add(2 * time.Minute)},
		{name: "unknown", route: route, challenge: nil, code: issued.Code, now: now.Add(2 * time.Minute)},
		{name: "expired", route: route, challenge: expiredOTPChallenge(service, route, recipient, now), code: "654321", now: now.Add(11 * time.Minute)},
		{name: "revoked-route", route: revokedOTPRoute(route, now.Add(time.Minute)), challenge: activeOTPChallenge(service, route, recipient, now, "654321"), code: "654321", now: now.Add(2 * time.Minute)},
	}
	for _, failure := range failures {
		t.Run(failure.name, func(t *testing.T) {
			_, err := service.Verify(failure.route, failure.challenge, failure.code, failure.now)
			if !errors.Is(err, ErrAccessVerificationFailed) || err.Error() != ErrAccessVerificationFailed.Error() {
				t.Fatalf("failure was distinguishable: %v", err)
			}
		})
	}
}

func TestOTPAttemptCapCountsMalformedCodes(t *testing.T) {
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	service := testOTPService(now, repeatedRecipientKey(0xc3), "777777")
	route := otpRoute(AccessDirectEmailOTP, "recipient-a", now.Add(time.Hour))
	recipient := accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	challenge, issued, err := service.Issue(route, recipient, now)
	if err != nil {
		t.Fatal(err)
	}
	attempts := []string{"000000", "not-six", "12345x", "111111", "222222"}
	for index, code := range attempts {
		if _, err := service.Verify(route, &challenge, code, now.Add(time.Duration(index+1)*time.Second)); !errors.Is(err, ErrAccessVerificationFailed) {
			t.Fatalf("attempt %d returned %v", index+1, err)
		}
		if challenge.Attempts != index+1 {
			t.Fatalf("attempt %d was not counted: %+v", index+1, challenge)
		}
	}
	if _, err := service.Verify(route, &challenge, issued.Code, now.Add(time.Minute)); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("exhausted challenge accepted the correct code: %v", err)
	}
	if challenge.Attempts != OTPMaxAttempts {
		t.Fatalf("attempt counter exceeded its cap: %d", challenge.Attempts)
	}
}

func TestOTPResendRotatesDigestAndCapsAtThree(t *testing.T) {
	now := time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC)
	service := testOTPService(now, repeatedRecipientKey(0xd4), "100001", "100002", "100003", "100004", "100005")
	route := otpRoute(AccessSharedEmailOTP, "", now.Add(12*time.Minute))
	recipient := accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	challenge, issued, err := service.Issue(route, recipient, now)
	if err != nil {
		t.Fatal(err)
	}
	previousCode := issued.Code
	previousDigest := append([]byte(nil), challenge.Digest...)
	for resend := 1; resend <= OTPMaxResends; resend++ {
		issued, err = service.Resend(route, &challenge, now.Add(time.Duration(resend)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if issued.Code == previousCode || bytes.Equal(challenge.Digest, previousDigest) || challenge.Resends != resend {
			t.Fatalf("resend %d did not rotate protected code state: %+v %+v", resend, issued, challenge)
		}
		previousCode = issued.Code
		previousDigest = append(previousDigest[:0], challenge.Digest...)
	}
	if !challenge.ExpiresAt.Equal(route.ExpiresAt) {
		t.Fatalf("resend expiry was not clamped to route expiry: %s != %s", challenge.ExpiresAt, route.ExpiresAt)
	}
	if _, err := service.Resend(route, &challenge, now.Add(4*time.Minute)); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("fourth resend was not rejected generically: %v", err)
	}
	if challenge.Resends != OTPMaxResends {
		t.Fatalf("resend counter exceeded cap: %d", challenge.Resends)
	}
}

func TestOTPProtectedFieldsNeverSerializeOrFormat(t *testing.T) {
	now := time.Date(2026, 8, 28, 4, 0, 0, 0, time.UTC)
	service := testOTPService(now, repeatedRecipientKey(0xe5), "314159")
	route := otpRoute(AccessDirectEmailOTP, "recipient-secret", now.Add(time.Hour))
	recipient := accessRecipient("recipient-secret", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	challenge, issued, err := service.Issue(route, recipient, now)
	if err != nil {
		t.Fatal(err)
	}
	challengeJSON, err := json.Marshal(challenge)
	if err != nil {
		t.Fatal(err)
	}
	issuedJSON, err := json.Marshal(issued)
	if err != nil {
		t.Fatal(err)
	}
	for name, output := range map[string]string{
		"challenge-json": string(challengeJSON),
		"issued-json":    string(issuedJSON),
		"challenge-log":  fmt.Sprintf("%#v", challenge),
		"issued-log":     fmt.Sprintf("%#v", issued),
	} {
		if strings.Contains(output, issued.Code) || strings.Contains(output, recipient.ID) || strings.Contains(output, fmt.Sprintf("%x", challenge.Digest)) {
			t.Fatalf("%s leaked protected OTP data: %s", name, output)
		}
	}
}

func TestOTPRejectsUnconfiguredHMACKey(t *testing.T) {
	now := time.Date(2026, 8, 28, 4, 30, 0, 0, time.UTC)
	service := NewOTPService([32]byte{})
	route := otpRoute(AccessDirectEmailOTP, "recipient-a", now.Add(time.Hour))
	recipient := accessRecipient("recipient-a", RecipientTo, RecipientExternalAudience, DistributionRecipientPending, "Owner")
	if _, _, err := service.Issue(route, recipient, now); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("zero HMAC key enabled OTP issuance: %v", err)
	}
}

func testOTPService(now time.Time, key [32]byte, codes ...string) *OTPService {
	service := NewOTPService(key)
	service.now = func() time.Time { return now }
	index := 0
	service.generateCode = func() (string, error) {
		if index >= len(codes) {
			return "", errors.New("test code sequence exhausted")
		}
		code := codes[index]
		index++
		return code, nil
	}
	return service
}

func otpRoute(policy AccessPolicy, recipientID string, expiresAt time.Time) AccessRoute {
	return AccessRoute{
		ID: "route-a", TenantID: "tenant-a", LegalEntityID: "entity-a", DistributionID: "distribution-a",
		RecipientID: recipientID, Policy: policy, ExpiresAt: expiresAt,
		CreatedBy: "actor-a", CreatedAt: expiresAt.Add(-time.Hour),
	}
}

func activeOTPChallenge(service *OTPService, route AccessRoute, recipient DistributionRecipient, now time.Time, code string) *OTPChallenge {
	return &OTPChallenge{
		ID: "challenge-active", TenantID: route.TenantID, LegalEntityID: route.LegalEntityID,
		DistributionID: route.DistributionID, RouteID: route.ID, RecipientID: recipient.ID,
		Digest: service.digest("challenge-active", code), ExpiresAt: now.Add(OTPValidity),
		MaxAttempts: OTPMaxAttempts, MaxResends: OTPMaxResends, CreatedAt: now,
	}
}

func expiredOTPChallenge(service *OTPService, route AccessRoute, recipient DistributionRecipient, now time.Time) *OTPChallenge {
	challenge := activeOTPChallenge(service, route, recipient, now, "654321")
	challenge.ExpiresAt = now.Add(5 * time.Minute)
	return challenge
}

func revokedOTPRoute(route AccessRoute, revokedAt time.Time) AccessRoute {
	revokedAt = revokedAt.UTC()
	route.RevokedAt = &revokedAt
	return route
}
