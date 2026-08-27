package evidence

import (
	"bytes"
	"time"
)

func testAccessPolicyEngine(now time.Time, random []byte) *AccessPolicyEngine {
	engine := NewAccessPolicyEngine(repeatedRecipientKey(0x96))
	engine.now = func() time.Time { return now }
	engine.random = bytes.NewReader(random)
	return engine
}

func accessRecipient(id string, role RecipientRole, recipientType RecipientType, state DistributionRecipientState, label string) DistributionRecipient {
	return DistributionRecipient{
		ID: id, DistributionID: "distribution-a", TenantID: "tenant-a", LegalEntityID: "entity-a",
		Role: role, Type: recipientType, RequestID: "request-" + id, AudienceHint: "r***@example.test",
		ContactLabel: label, State: state,
	}
}
