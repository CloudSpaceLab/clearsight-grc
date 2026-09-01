package evidence

import (
	"strings"
	"testing"
	"time"
)

func TestRenderOperationalNotificationSeparatesAssignmentFromAuthorization(t *testing.T) {
	message, err := RenderOperationalNotification(OperationalNotificationContext{
		BankName: "Clear Bank Nigeria", RecipientName: "Address Verification Officer",
		MatterTitle: "Verify Cloudspace registered address", WorkTitle: "Confirm the registered address",
		Responsibility: "PERFORMER", DueAt: time.Date(2026, 9, 5, 16, 0, 0, 0, time.UTC),
		IssueURL: "https://clearsight.example.test/#work/matters/matter-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	preview := revealPreview(message)
	for _, expected := range []string{"assigned issue work", "Verify Cloudspace registered address", "Confirm the registered address", "Open assigned issue"} {
		if !strings.Contains(preview.PlainText, expected) && !strings.Contains(preview.Subject, expected) {
			t.Fatalf("rendered message does not contain %q: %#v", expected, preview)
		}
	}
	for _, forbidden := range []string{"authorized", "approved", "signed off"} {
		if strings.Contains(strings.ToLower(preview.PlainText), forbidden) {
			t.Fatalf("assignment message claims %q: %s", forbidden, preview.PlainText)
		}
	}
}

func TestRenderOperationalNotificationRejectsNonHTTPSLinkAndControlCharacters(t *testing.T) {
	base := OperationalNotificationContext{
		BankName: "Clear Bank", RecipientName: "Program Owner", MatterTitle: "Vendor review",
		WorkTitle: "Confirm owner", Responsibility: "ACCOUNTABLE_OWNER", IssueURL: "http://example.test/matter",
	}
	if _, err := RenderOperationalNotification(base); err == nil {
		t.Fatal("non-HTTPS issue link accepted")
	}
	base.IssueURL = "https://example.test/matter"
	base.MatterTitle = "Vendor review\r\nBcc: attacker@example.test"
	if _, err := RenderOperationalNotification(base); err == nil {
		t.Fatal("header control characters accepted")
	}
}
