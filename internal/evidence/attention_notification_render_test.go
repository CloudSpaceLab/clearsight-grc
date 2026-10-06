package evidence

import (
	"strings"
	"testing"
)

func TestAttentionNotificationIsBoundedAndGeneric(t *testing.T) {
	request, err := BuildAttentionNotificationRequest("risk@example.test", AttentionNotificationContext{
		BrandName:       "Meridian Bank",
		RecipientName:   "Risk Officer",
		ConditionLabel:  "Risk indicator",
		TransitionLabel: "Worsened",
		RecordURL:       "https://clearsight.example.test/#risks/11111111-1111-4111-8111-111111111111",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Subject != "Critical change: Risk indicator" {
		t.Fatalf("subject=%q", request.Subject)
	}
	for _, body := range []string{request.Subject, request.PlainText, request.HTML} {
		for _, forbidden := range []string{"customer name", "threshold value", "evidence payload", "risk title"} {
			if strings.Contains(strings.ToLower(body), forbidden) {
				t.Fatalf("email leaked %q", forbidden)
			}
		}
	}
	if !strings.Contains(request.PlainText, "does not grant access") {
		t.Fatalf("missing authority boundary: %s", request.PlainText)
	}
}

func TestAttentionNotificationRejectsInsecureRecordURL(t *testing.T) {
	_, err := BuildAttentionNotificationRequest("risk@example.test", AttentionNotificationContext{
		BrandName:       "Meridian Bank",
		RecipientName:   "Risk Officer",
		ConditionLabel:  "Risk indicator",
		TransitionLabel: "Opened",
		RecordURL:       "http://clearsight.example.test/#risks/1",
	})
	if err == nil {
		t.Fatal("expected insecure URL rejection")
	}
}
