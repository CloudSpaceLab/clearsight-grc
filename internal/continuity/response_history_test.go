package continuity

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestResponsePackageHistoryIsNewestFirstBoundedAndEntityScoped(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	ctx := WithTrustedSystemScope(context.Background())
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: MatterAuthorityRequest, Priority: 3, Title: "Regulator response", Summary: "Prepare a response.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddResponsePackage(ctx, AddResponsePackageInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Purpose: "Provide records", Audience: "Regulator", Manifest: json.RawMessage(`[]`), ActorID: "preparer"})
	if err != nil {
		t.Fatal(err)
	}
	responseID := matter.ResponsePackages[0].ID
	for _, step := range []struct {
		status ResponseStatus
		actor  string
	}{{ResponseInReview, "reviewer"}, {ResponseApproved, "signatory"}} {
		matter, err = service.TransitionResponsePackage(ctx, TransitionResponseInput{TenantID: "bank", MatterID: matter.Matter.ID, ResponseID: responseID, ExpectedVersion: matter.Matter.Version, To: step.status, ActorID: step.actor, Rationale: "Progress response."})
		if err != nil {
			t.Fatal(err)
		}
	}

	page, err := service.ResponsePackageHistory(ctx, "bank", matter.Matter.ID, responseID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || !page.HasMore || page.Items[0].Status != ResponseApproved || page.Items[1].Status != ResponseInReview {
		t.Fatalf("unexpected bounded history: %#v", page)
	}
	raw, _ := json.Marshal(page)
	if page.Items[0].ActorLabel != "Recorded person unavailable" || strings.Contains(string(raw), "signatory") {
		t.Fatalf("history leaked an actor identifier: %#v", page.Items[0])
	}

	otherEntity := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "reader", LegalEntityID: "entity-b"})
	if _, err = service.ResponsePackageHistory(otherEntity, "bank", matter.Matter.ID, responseID, 20); err != ErrNotFound {
		t.Fatalf("expected entity-scoped not found, got %v", err)
	}
}
