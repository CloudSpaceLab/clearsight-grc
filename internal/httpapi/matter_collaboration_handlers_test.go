package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestMatterActivityUsesBoundedPage(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3, Title: "Gap", Summary: "Resolve the gap.", Scope: json.RawMessage(`{}`), ActorID: "hakeem"})
	if err != nil {
		t.Fatal(err)
	}
	current := matter
	for _, body := range []string{"One", "Two", "Three"} {
		current, err = service.AddMatterComment(continuity.WithTrustedSystemScope(t.Context()), continuity.AddMatterCommentInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: current.Matter.Version, ActorID: "hakeem", Body: body})
		if err != nil {
			t.Fatal(err)
		}
	}
	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/matters/"+matter.Matter.ID+"/activity?tenant_id=bank&limit=2", nil)
	request.SetPathValue("id", matter.Matter.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "hakeem", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	recorder := httptest.NewRecorder()
	api.getMatterActivity(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activity status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var value continuity.MatterActivityPage
	if err := json.NewDecoder(recorder.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if len(value.Items) != 2 || value.Items[0].Comment == nil || value.Items[0].Comment.Body != "Three" || value.NextBeforeVersion == 0 {
		t.Fatalf("activity page = %#v", value)
	}
}

func TestMatterCommentBindsVerifiedActor(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	matter, err := service.CreateMatter(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3, Title: "Gap", Summary: "Resolve the gap.", Scope: json.RawMessage(`{}`), ActorID: "hakeem"})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Continuity: service}}
	body := bytes.NewBufferString(`{"tenant_id":"bank","expected_version":1,"actor_id":"forged","body":"Current status is pending."}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/comments", body)
	request.SetPathValue("id", matter.Matter.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "hakeem", LegalEntityID: "entity-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	recorder := httptest.NewRecorder()
	api.addMatterComment(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("comment status = %d: %s", recorder.Code, recorder.Body.String())
	}
	events, err := service.MatterActivity(continuity.WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if events.Items[0].ActorID != "hakeem" {
		t.Fatalf("comment actor = %q", events.Items[0].ActorID)
	}
}
