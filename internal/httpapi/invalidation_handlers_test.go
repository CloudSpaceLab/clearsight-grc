package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/invalidation"
)

func TestInvalidationStreamEmitsOnlyOpaqueRevisionForVerifiedActor(t *testing.T) {
	hub := invalidation.NewHub()
	api := &API{deps: Dependencies{Invalidations: hub}}
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/updates/stream", nil).WithContext(ctx)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-a",
		ExpiresAt: time.Now().Add(time.Minute),
	}))
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		api.invalidationStream(response, request)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	hub.Publish(invalidation.Event{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-a", Revision: "opaque-revision",
	})
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	body := response.Body.String()
	if !strings.Contains(body, "event: invalidate") || !strings.Contains(body, `{"revision":"opaque-revision"}`) {
		t.Fatalf("unexpected stream body: %q", body)
	}
	for _, secret := range []string{"tenant-a", "entity-a", "person-a"} {
		if strings.Contains(body, secret) {
			t.Fatalf("stream leaked actor scope %q: %q", secret, body)
		}
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type=%q", got)
	}
}
