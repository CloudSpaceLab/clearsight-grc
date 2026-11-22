package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type recordingOTPDelivery struct {
	values []DistributionOTPDelivery
	err    error
}

func (delivery *recordingOTPDelivery) DeliverDistributionOTP(_ context.Context, value DistributionOTPDelivery) error {
	if delivery.err != nil {
		return delivery.err
	}
	delivery.values = append(delivery.values, value)
	return nil
}

func TestDistributionAccessDirectMagicLinkReissueKeepsPriorLinkUntilExplicitRevocation(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectMagicLink, []DistributionRecipientInput{{
		Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner",
	}})

	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue direct route: %+v %v", issued, err)
	}
	start, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector)
	if err != nil || start.Policy != AccessDirectMagicLink || len(start.Recipients) != 0 {
		t.Fatalf("unexpected direct start: %+v %v", start, err)
	}
	redeemed, err := fixture.access.RedeemDirectRoute(context.Background(), issued[0].Selector)
	if err != nil || redeemed.Assurance != AssuranceLinkPossession || redeemed.SessionToken == "" {
		t.Fatalf("unexpected direct redemption: %+v %v", redeemed, err)
	}
	if strings.Contains(redeemed.String(), redeemed.SessionToken) {
		t.Fatal("redeemed session formatting leaked bearer token")
	}
	session, request, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken)
	if err != nil || session.DistributionID != fixture.distribution.ID || request.ID != redeemed.RequestID {
		t.Fatalf("session request failed: %+v %+v %v", session, request, err)
	}
	second, err := fixture.access.RedeemDirectRoute(context.Background(), issued[0].Selector)
	if err != nil {
		t.Fatalf("unexpired direct selector could not be reopened: %v", err)
	}
	if second.SessionID == redeemed.SessionID || second.SessionToken == redeemed.SessionToken {
		t.Fatalf("reopening reused a prior session: first=%s second=%s", redeemed.SessionID, second.SessionID)
	}

	replacement, err := fixture.access.RotateDistributionAccessRoute(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, issued[0].RouteID, "actor-a")
	if err != nil || replacement.Selector == "" || replacement.Selector == issued[0].Selector {
		t.Fatalf("route reissue failed: %+v %v", replacement, err)
	}
	if _, _, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken); err != nil {
		t.Fatalf("reissue invalidated an unexpired session: %v", err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector); err != nil {
		t.Fatalf("reissue invalidated the prior unexpired link: %v", err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), replacement.Selector); err != nil {
		t.Fatalf("replacement link was not usable: %v", err)
	}
	if err := fixture.access.RevokeDistributionAccessRoute(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, issued[0].RouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("explicitly revoked prior link remained usable: %v", err)
	}
	if err := fixture.access.RevokeDistributionAccessRoute(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, replacement.RouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), replacement.Selector); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("revoked replacement remained usable: %v", err)
	}
}

func TestDistributionAccessSessionUsesRecipientIdentityInsteadOfDisplayHint(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectMagicLink, []DistributionRecipientInput{{
		Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner",
	}})

	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue direct route: %+v %v", issued, err)
	}
	redeemed, err := fixture.access.RedeemDirectRoute(context.Background(), issued[0].Selector)
	if err != nil {
		t.Fatal(err)
	}

	fixture.distributions.mu.Lock()
	recipient := fixture.distributions.recipients[fixture.distribution.ID][0]
	fixture.distributions.repo.mu.Lock()
	request := fixture.distributions.repo.requests[recipient.safe.RequestID]
	request.Recipient.AudienceHint = "masked address updated"
	fixture.distributions.repo.requests[request.ID] = request
	fixture.distributions.repo.mu.Unlock()
	fixture.distributions.mu.Unlock()

	session, loaded, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken)
	if err != nil || session.RequestID != request.ID || loaded.ID != request.ID {
		t.Fatalf("stable recipient identity was rejected after display hint changed: session=%+v request=%+v err=%v", session, loaded, err)
	}

	fixture.distributions.repo.mu.Lock()
	request = fixture.distributions.repo.requests[request.ID]
	request.Recipient.AudienceHash = append([]byte(nil), request.Recipient.AudienceHash...)
	request.Recipient.AudienceHash[0] ^= 0xff
	fixture.distributions.repo.requests[request.ID] = request
	fixture.distributions.repo.mu.Unlock()
	if _, _, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("session survived a recipient identity mismatch: %v", err)
	}
}

func TestDistributionAccessDirectMagicLinkUsesEachLinksPrintedExpiry(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectMagicLink, []DistributionRecipientInput{{
		Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner",
	}})

	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue direct route: %+v %v", issued, err)
	}
	shorterExpiry := fixture.now.Add(10 * time.Minute)
	fixture.distributions.mu.Lock()
	distribution := fixture.distributions.distributions[fixture.distribution.ID]
	distribution.RouteExpiresAt = shorterExpiry
	distribution.Version++
	fixture.distributions.distributions[distribution.ID] = distribution
	fixture.distributions.mu.Unlock()
	replacement, err := fixture.access.RotateDistributionAccessRoute(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, issued[0].RouteID, "actor-a")
	if err != nil {
		t.Fatal(err)
	}
	if !replacement.ExpiresAt.Equal(shorterExpiry) || !issued[0].ExpiresAt.After(replacement.ExpiresAt) {
		t.Fatalf("test setup did not create independently expiring links: first=%s second=%s", issued[0].ExpiresAt, replacement.ExpiresAt)
	}

	*fixture.now = fixture.now.Add(15 * time.Minute)
	if _, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector); err != nil {
		t.Fatalf("first link expired before the date printed in its email: %v", err)
	}
	if _, err := fixture.access.StartDistributionAccess(context.Background(), replacement.Selector); !errors.Is(err, ErrDistributionAccessUnavailable) {
		t.Fatalf("shorter-lived replacement remained available after its own expiry: %v", err)
	}
}

func TestDistributionAccessSharedOTPIsMaskedSingleUseAndResendBounded(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessSharedEmailOTP, []DistributionRecipientInput{
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "alpha@example.test", AudienceHint: "a***@example.test", ContactLabel: "Alpha"},
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "beta@example.test", AudienceHint: "b***@example.test", ContactLabel: "Beta"},
		{Role: RecipientCC, Type: RecipientExternalAudience, Address: "observer@example.test", AudienceHint: "o***@example.test", ContactLabel: "Observer"},
	})
	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue shared route: %+v %v", issued, err)
	}
	start, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector)
	if err != nil || len(start.Recipients) != 2 {
		t.Fatalf("shared start did not expose the two eligible TO recipients: %+v %v", start, err)
	}
	for _, masked := range start.Recipients {
		if masked.SelectorID == "" || strings.Contains(masked.SelectorID, "recipient") || masked.Hint == "" {
			t.Fatalf("unsafe masked recipient projection: %+v", masked)
		}
	}

	receipt, err := fixture.access.SendOTP(context.Background(), issued[0].Selector, start.Recipients[0].SelectorID)
	if err != nil || receipt.ChallengeID == "" || len(fixture.delivery.values) != 1 {
		t.Fatalf("send OTP failed: %+v %v deliveries=%d", receipt, err, len(fixture.delivery.values))
	}
	delivered := fixture.delivery.values[0]
	if delivered.Address != "alpha@example.test" || delivered.Code == "" || strings.Contains(delivered.String(), delivered.Address) || strings.Contains(delivered.String(), delivered.Code) {
		t.Fatalf("protected delivery boundary leaked or resolved incorrectly: %s", delivered.String())
	}
	if _, err := fixture.access.VerifyOTP(context.Background(), issued[0].Selector, receipt.ChallengeID, "999999"); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("wrong OTP returned a distinguishable error: %v", err)
	}
	redeemed, err := fixture.access.VerifyOTP(context.Background(), issued[0].Selector, receipt.ChallengeID, delivered.Code)
	if err != nil || redeemed.Assurance != AssuranceEmailVerified {
		t.Fatalf("correct OTP failed: %+v %v", redeemed, err)
	}
	if _, err := fixture.access.VerifyOTP(context.Background(), issued[0].Selector, receipt.ChallengeID, delivered.Code); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("consumed OTP replay did not fail generically: %v", err)
	}
	if session, _, err := fixture.access.SessionRequest(context.Background(), redeemed.SessionToken); err != nil || session.Assurance != AssuranceEmailVerified {
		t.Fatalf("verified session could not recover its assurance: %+v %v", session, err)
	}

	// A consumed challenge permits a fresh challenge. That challenge may be
	// resent exactly three times and no more.
	for attempt := 0; attempt < 4; attempt++ {
		if _, err := fixture.access.SendOTP(context.Background(), issued[0].Selector, start.Recipients[0].SelectorID); err != nil {
			t.Fatalf("initial+three resend sequence failed at %d: %v", attempt, err)
		}
	}
	if _, err := fixture.access.SendOTP(context.Background(), issued[0].Selector, start.Recipients[0].SelectorID); !errors.Is(err, ErrAccessVerificationFailed) {
		t.Fatalf("fourth resend was not rejected generically: %v", err)
	}
}

func TestDistributionAccessDirectEmailOTPCreatesOneBoundRoutePerExternalTO(t *testing.T) {
	fixture := newMemoryAccessFixture(t, AccessDirectEmailOTP, []DistributionRecipientInput{
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "alpha@example.test", AudienceHint: "a***@example.test", ContactLabel: "Alpha"},
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "beta@example.test", AudienceHint: "b***@example.test", ContactLabel: "Beta"},
	})
	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 2 {
		t.Fatalf("expected one direct OTP route per external TO: %+v %v", issued, err)
	}
	for _, route := range issued {
		start, err := fixture.access.StartDistributionAccess(context.Background(), route.Selector)
		if err != nil || len(start.Recipients) != 1 {
			t.Fatalf("direct OTP route exposed more than its bound recipient: %+v %v", start, err)
		}
	}
	start, _ := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector)
	receipt, err := fixture.access.SendOTP(context.Background(), issued[0].Selector, start.Recipients[0].SelectorID)
	if err != nil {
		t.Fatal(err)
	}
	delivered := fixture.delivery.values[len(fixture.delivery.values)-1]
	redeemed, err := fixture.access.VerifyOTP(context.Background(), issued[0].Selector, receipt.ChallengeID, delivered.Code)
	if err != nil || redeemed.Assurance != AssuranceEmailVerified {
		t.Fatalf("direct OTP verification failed: %+v %v", redeemed, err)
	}
}

func TestDistributionAccessOTPFailureContractAndRecovery(t *testing.T) {
	for _, scenario := range []string{"wrong", "expired", "attempts exhausted"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := newMemoryAccessFixture(t, AccessDirectEmailOTP, []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "alpha@example.test", AudienceHint: "a***@example.test"}})
			ctx := context.Background()
			routes, err := fixture.access.IssueDistributionAccessRoutes(ctx, "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
			if err != nil || len(routes) != 1 {
				t.Fatalf("issue routes: %v %#v", err, routes)
			}
			start, err := fixture.access.StartDistributionAccess(ctx, routes[0].Selector)
			if err != nil || len(start.Recipients) != 1 {
				t.Fatalf("start: %v %#v", err, start)
			}
			selector := start.Recipients[0].SelectorID
			receipt, err := fixture.access.SendOTP(ctx, routes[0].Selector, selector)
			if err != nil {
				t.Fatal(err)
			}
			correct := fixture.delivery.values[len(fixture.delivery.values)-1].Code
			wrong := "000000"
			if wrong == correct {
				wrong = "999999"
			}
			if scenario == "expired" {
				*fixture.now = fixture.now.Add(OTPValidity + time.Second)
			}
			if scenario == "attempts exhausted" {
				for i := 0; i < OTPMaxAttempts; i++ {
					if _, err := fixture.access.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, wrong); !errors.Is(err, ErrAccessVerificationFailed) {
						t.Fatalf("wrong attempt %d: %v", i, err)
					}
				}
			}
			code := correct
			if scenario == "wrong" {
				code = wrong
			}
			if _, err := fixture.access.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, code); !errors.Is(err, ErrAccessVerificationFailed) {
				t.Fatalf("failure was distinguishable: %v", err)
			}
			if scenario == "attempts exhausted" {
				if _, err := fixture.access.SendOTP(ctx, routes[0].Selector, selector); !errors.Is(err, ErrAccessVerificationFailed) {
					t.Fatalf("exhausted code was replaced before expiration: %v", err)
				}
				*fixture.now = fixture.now.Add(OTPValidity + time.Second)
			}
			fresh, err := fixture.access.SendOTP(ctx, routes[0].Selector, selector)
			if err != nil {
				t.Fatalf("request another code: %v", err)
			}
			latest := fixture.delivery.values[len(fixture.delivery.values)-1].Code
			if _, err := fixture.access.VerifyOTP(ctx, routes[0].Selector, fresh.ChallengeID, latest); err != nil {
				t.Fatalf("verify replacement code: %v", err)
			}
		})
	}
}

type memoryAccessFixture struct {
	access        *DistributionAccessService
	delivery      *recordingOTPDelivery
	distribution  FormDistribution
	distributions *MemoryDistributionStore
	now           *time.Time
}

func newMemoryAccessFixture(t *testing.T, policy AccessPolicy, recipients []DistributionRecipientInput) memoryAccessFixture {
	return newMemoryAccessFixtureForSubject(t, policy, recipients, "VENDOR")
}

func newMemoryAccessFixtureForSubject(t *testing.T, policy AccessPolicy, recipients []DistributionRecipientInput, subjectType string) memoryAccessFixture {
	t.Helper()
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	clock := &now
	encryptionKey := testSecurityKey(0x31)
	hmacKey := testSecurityKey(0x42)
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": encryptionKey})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMemoryRepository(nil, nil)
	distributions := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: activeDistributionForm()}, keyring)
	distributions.now = func() time.Time { return *clock }
	bundle, err := distributions.CreateDistribution(context.Background(), CreateDistributionInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		SubjectType: subjectType, SubjectID: "subject-a", Title: "Access review", Purpose: "Collect protected evidence.",
		AccessPolicy: policy, EstimatedMinutes: 5, Deadline: now.Add(2 * time.Hour), RouteExpiresAt: now.Add(time.Hour),
		CreatedBy: "actor-a", Recipients: recipients,
	})
	if err != nil {
		t.Fatal(err)
	}
	distributions.mu.Lock()
	distribution := distributions.distributions[bundle.Distribution.ID]
	distribution.Status = DistributionOpen
	distributions.distributions[distribution.ID] = distribution
	distributions.mu.Unlock()

	accessStore := NewMemoryDistributionAccessStore(distributions)
	delivery := &recordingOTPDelivery{}
	access, err := NewDistributionAccessService(accessStore, keyring, delivery, hmacKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access.now = func() time.Time { return *clock }
	return memoryAccessFixture{access: access, delivery: delivery, distribution: distribution, distributions: distributions, now: clock}
}

func testSecurityKey(value byte) [32]byte {
	var key [32]byte
	for index := range key {
		key[index] = value
	}
	return key
}
