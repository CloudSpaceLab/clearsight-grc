package evidence

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"io"
	"sort"
	"strings"
	"time"
)

func eligibleAccessRecipients(route AccessRoute, recipients []DistributionRecipient) []DistributionRecipient {
	eligible := make([]DistributionRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient.TenantID != route.TenantID || recipient.LegalEntityID != route.LegalEntityID || recipient.DistributionID != route.DistributionID ||
			recipient.Role != RecipientTo || recipient.Type != RecipientExternalAudience || recipient.RequestID == "" ||
			recipient.State == DistributionRecipientRevoked || recipient.State == DistributionRecipientCompleted {
			continue
		}
		if route.Policy != AccessSharedEmailOTP && recipient.ID != route.RecipientID {
			continue
		}
		eligible = append(eligible, recipient)
	}
	sort.SliceStable(eligible, func(left, right int) bool {
		leftLabel := strings.ToLower(eligible[left].ContactLabel + "\x00" + eligible[left].AudienceHint)
		rightLabel := strings.ToLower(eligible[right].ContactLabel + "\x00" + eligible[right].AudienceHint)
		if leftLabel == rightLabel {
			return eligible[left].ID < eligible[right].ID
		}
		return leftLabel < rightLabel
	})
	return eligible
}

func accessRouteUsable(route AccessRoute, selector string, now time.Time) bool {
	if !accessRouteActive(route, now) || len(route.SelectorHash) != sha256.Size {
		return false
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(selector)))
	return subtle.ConstantTimeCompare(route.SelectorHash, digest[:]) == 1
}

func randomAccessSelector(source io.Reader) (string, []byte, error) {
	if source == nil {
		source = rand.Reader
	}
	raw := make([]byte, accessRouteSelectorBytes)
	if _, err := io.ReadFull(source, raw); err != nil {
		return "", nil, err
	}
	selector := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(selector))
	return selector, digest[:], nil
}

func (engine *AccessPolicyEngine) recipientSelector(routeID, recipientID string) string {
	mac := hmac.New(sha256.New, engine.hmacKey[:])
	_, _ = mac.Write([]byte("distribution-recipient-selector|" + routeID + "|" + recipientID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (engine *AccessPolicyEngine) consumeDummySelector(routeID, selector string) {
	if engine == nil {
		return
	}
	expected := engine.recipientSelector(routeID, "unknown-recipient")
	_ = constantTimeStringEqual(expected, selector)
}

func constantTimeStringEqual(expected, supplied string) bool {
	left := sha256.Sum256([]byte(expected))
	right := sha256.Sum256([]byte(strings.TrimSpace(supplied)))
	return subtle.ConstantTimeCompare(left[:], right[:]) == 1
}
