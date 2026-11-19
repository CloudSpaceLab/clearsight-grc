package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"testing"
	"time"
)

func TestActivityFiltersAndScopedCursor(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person"}
	q, c, err := activityQuery(actor, "program", ProgramActivityQuery{Category: "DOCUMENT", From: "2026-11-01", To: "2026-11-30"}, now)
	if err != nil || q.Limit != 25 {
		t.Fatalf("default: %v %#v", err, q)
	}
	c.ID = "event"
	c.At = now.Add(-time.Hour)
	raw, _ := json.Marshal(c)
	q.Cursor = base64.RawURLEncoding.EncodeToString(raw)
	if _, _, err = activityQuery(actor, "program", q, now); err != nil {
		t.Fatal(err)
	}
	other := actor
	other.PrincipalID = "someone-else"
	if _, _, err = activityQuery(other, "program", q, now); err == nil {
		t.Fatal("cross-actor cursor accepted")
	}
	if _, _, err = activityQuery(actor, "other-program", q, now); err == nil {
		t.Fatal("cross-program cursor accepted")
	}
	q.Search = "changed filter"
	if _, _, err = activityQuery(actor, "program", q, now); err == nil {
		t.Fatal("cross-query cursor accepted")
	}
	for _, bad := range []ProgramActivityQuery{{From: "2026-02-30"}, {From: "2026-11-20", To: "2026-11-01"}, {Category: "SECRET"}, {Limit: 101}, {Cursor: "corrupt"}} {
		if _, _, err = activityQuery(actor, "program", bad, now); err == nil {
			t.Fatalf("invalid accepted: %#v", bad)
		}
	}
}

func TestActivityRequiresVerifiedIdentity(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), nil)
	if _, err := service.ProgramActivity(context.Background(), "program", ProgramActivityQuery{}); err != ErrRecipientMismatch {
		t.Fatalf("anonymous read: %v", err)
	}
}
