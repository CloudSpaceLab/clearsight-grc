package continuity

import (
	"encoding/json"
	"testing"
)

func TestMatterActivityReturnsNewestBoundedPage(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterVendorDeficiency, Priority: 3,
		Title: "Vendor assurance gap", Summary: "Obtain current assurance evidence.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	current := matter
	for _, body := range []string{"First update", "Second update", "Third update"} {
		current, err = service.AddMatterComment(WithTrustedSystemScope(t.Context()), AddMatterCommentInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: current.Matter.Version, ActorID: "hakeem", Body: body})
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err := service.MatterActivity(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Comment == nil || page.Items[0].Comment.Body != "Third update" || page.NextBeforeVersion == 0 {
		t.Fatalf("first page = %#v", page)
	}
	older, err := service.MatterActivity(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, page.NextBeforeVersion, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(older.Items) != 2 || older.Items[0].Comment == nil || older.Items[0].Comment.Body != "First update" {
		t.Fatalf("older page = %#v", older)
	}
}

func TestAddMatterCommentRecordsAuthorAndMention(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterVendorDeficiency, Priority: 3,
		Title: "Vendor assurance gap", Summary: "Obtain current assurance evidence.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.AddMatterComment(WithTrustedSystemScope(t.Context()), AddMatterCommentInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ActorID: "hakeem", Body: "@Blessing Please confirm whether the vendor has sent the audit letter.", MentionedPrincipalIDs: []string{"blessing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Matter.Version != matter.Matter.Version+1 {
		t.Fatalf("version = %d, want %d", updated.Matter.Version, matter.Matter.Version+1)
	}
	events, err := repo.MatterEvents(WithTrustedSystemScope(t.Context()), "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := events[len(events)-1]; got.Type != EventMatterCommentAdded || got.ActorID != "hakeem" {
		t.Fatalf("comment event = %#v", got)
	}
	var comment MatterComment
	if err := json.Unmarshal(events[len(events)-1].Payload, &comment); err != nil {
		t.Fatal(err)
	}
	if comment.Body == "" || len(comment.MentionedPrincipalIDs) != 1 || comment.MentionedPrincipalIDs[0] != "blessing" {
		t.Fatalf("comment = %#v", comment)
	}
}

func TestRequestMatterActionUpdateKeepsActionOpen(t *testing.T) {
	service := NewService(NewMemoryRepository())
	matter, err := service.CreateMatter(WithTrustedSystemScope(t.Context()), CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterVendorDeficiency, Priority: 3,
		Title: "Vendor assurance gap", Summary: "Obtain current assurance evidence.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	withAction, err := service.AddAction(WithTrustedSystemScope(t.Context()), AddActionInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		Title: "Obtain audit letter", Description: "Request the current independent audit letter.", OwnerPrincipalID: "hakeem",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.RequestMatterActionUpdate(WithTrustedSystemScope(t.Context()), RequestMatterActionUpdateInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ActionID: withAction.Actions[0].ID, ExpectedVersion: withAction.Matter.Version,
		ActorID: "blessing", Message: "Confirm the expected delivery date.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Actions[0].Status; got != ActionPlanned {
		t.Fatalf("action status = %s, want %s", got, ActionPlanned)
	}
}
