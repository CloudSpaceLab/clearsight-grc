package evidence

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
)

func TestResponseWorkspaceSameRespondentCanAmendSubmission(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()
	view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	saved, err := fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: view.Workspace.Version,
		Edits:           []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes"), BaseSequence: view.FieldSequences["q1"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	view, err = fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	saved, err = fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: view.Workspace.Version,
		Edits:           []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("No"), BaseSequence: view.FieldSequences["q1"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version})
	if err != nil {
		t.Fatalf("same respondent cannot submit amendment while workspace remains open: %v", err)
	}
	if result.Revision.Revision != 2 {
		t.Fatal("amendment did not create revision 2")
	}
	if result.Revision.SubmittedRecipientID == "" {
		t.Fatal("submission did not retain the access recipient")
	}
	stored := fixture.access.store.(*MemoryDistributionAccessStore).distributions.responseRevisions[fixture.distribution.ID]
	if len(stored) != 2 || stored[0].SubmittedRecipientID != result.Revision.SubmittedRecipientID || stored[1].SubmittedRecipientID != result.Revision.SubmittedRecipientID {
		t.Fatal("revision history lost its submission recipient")
	}
}
