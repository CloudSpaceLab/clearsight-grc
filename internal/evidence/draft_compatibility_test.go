package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestDraftCompatibilityTranslatesDistributionSnapshotsToFieldEdits(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	compatibility := NewDraftCompatibilityService(nil, fixture.access)
	ctx := context.Background()

	first, err := compatibility.GetDraft(ctx, tokens[0])
	if err != nil || first.Version != 1 || len(first.Answers) != 0 {
		t.Fatalf("initial compatibility draft = %+v, err = %v", first, err)
	}
	first, err = compatibility.SaveDraft(ctx, tokens[0], SaveDraftInput{
		ExpectedVersion: first.Version,
		Answers:         map[string]formcontract.AnswerValue{"q1": formcontract.TextAnswer("Yes")},
		PresentationMode: formcontract.PresentationWizard,
	})
	if err != nil || first.Version != 2 {
		t.Fatalf("first translated save = %+v, err = %v", first, err)
	}

	second, err := compatibility.GetDraft(ctx, tokens[1])
	if err != nil || second.Version != 2 {
		t.Fatalf("shared compatibility read = %+v, err = %v", second, err)
	}
	second, err = compatibility.SaveDraft(ctx, tokens[1], SaveDraftInput{
		ExpectedVersion: second.Version,
		Answers: map[string]formcontract.AnswerValue{
			"q1": formcontract.TextAnswer("Yes"),
			"q2": formcontract.TextAnswer("No"),
		},
		PresentationMode: formcontract.PresentationClassic,
	})
	if err != nil || second.Version != 3 {
		t.Fatalf("second translated save = %+v, err = %v", second, err)
	}

	if err := compatibility.DeleteDraft(ctx, tokens[0]); err != nil {
		t.Fatalf("delete first recipient contribution: %v", err)
	}
	remaining, err := compatibility.GetDraft(ctx, tokens[1])
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := remaining.Answers["q1"]; exists {
		t.Fatalf("recipient-owned delete left q1 behind: %+v", remaining.Answers)
	}
	if text, _ := remaining.Answers["q2"].ScalarText(); text != "No" {
		t.Fatalf("recipient-owned delete erased another TO contribution: %+v", remaining.Answers)
	}

	if _, err := compatibility.SaveDraft(ctx, tokens[1], SaveDraftInput{
		ExpectedVersion: second.Version,
		Answers:         map[string]formcontract.AnswerValue{"q2": formcontract.TextAnswer("Yes")},
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale compatibility snapshot did not fail closed: %v", err)
	}
}

func TestDraftCompatibilityPreservesLegacySessionOwnedDrafts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	legacy := NewService(repo, nil)
	legacy.now = func() time.Time { return now }
	token := createDraftSession(t, ctx, legacy, now, []Field{{
		ID: "answer", Label: "Answer", Type: string(formcontract.TypeShortText),
	}})
	compatibility := NewDraftCompatibilityService(legacy, nil)

	saved, err := compatibility.SaveDraft(ctx, token, SaveDraftInput{
		ExpectedVersion: 0,
		Answers:         map[string]formcontract.AnswerValue{"answer": formcontract.TextAnswer("legacy")},
		PresentationMode: formcontract.PresentationClassic,
	})
	if err != nil || saved.Version != 1 {
		t.Fatalf("legacy save changed under compatibility facade: %+v %v", saved, err)
	}
	loaded, err := compatibility.GetDraft(ctx, token)
	if err != nil || loaded.Version != 1 {
		t.Fatalf("legacy get changed under compatibility facade: %+v %v", loaded, err)
	}
	if err := compatibility.DeleteDraft(ctx, token); err != nil {
		t.Fatal(err)
	}
	loaded, err = compatibility.GetDraft(ctx, token)
	if err != nil || loaded.Version != 0 || len(loaded.Answers) != 0 {
		t.Fatalf("legacy delete changed under compatibility facade: %+v %v", loaded, err)
	}
}
