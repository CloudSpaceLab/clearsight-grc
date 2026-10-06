package evidence

import (
	"strings"
	"testing"
)

func TestDailyDigestContainsCountsOnly(t *testing.T) {
	request, err := BuildDailyDigestRequest("risk@example.test", DailyDigestContext{
		BrandName:       "ClearSight",
		RecipientName:   "Risk Officer",
		HomeURL:         "https://clearsight.example.test/#home",
		MaterialChanges: 12,
		AssignedWork:    4,
		DueSoon:         2,
		Worsened:        1,
		Cleared:         3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Subject != "ClearSight daily summary" {
		t.Fatalf("subject=%q", request.Subject)
	}
	for _, value := range []string{request.PlainText, request.HTML} {
		for _, forbidden := range []string{"customer name", "record title", "evidence payload"} {
			if strings.Contains(strings.ToLower(value), forbidden) {
				t.Fatalf("digest leaked %q", forbidden)
			}
		}
	}
	if !strings.Contains(request.PlainText, "Assigned work") || !strings.Contains(request.PlainText, "Due in 7 days") {
		t.Fatalf("missing summary counts: %s", request.PlainText)
	}
}
