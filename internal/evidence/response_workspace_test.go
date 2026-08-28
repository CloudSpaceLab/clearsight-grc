package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestResponseWorkspaceMergesDifferentFieldsAndConflictsOnSameField(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()

	initialA, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	initialB, err := fixture.access.GetResponseWorkspace(ctx, tokens[1])
	if err != nil {
		t.Fatal(err)
	}
	if initialA.Workspace.ID != initialB.Workspace.ID || initialA.Workspace.Version != 1 {
		t.Fatalf("TO recipients did not share the same initial workspace: %+v %+v", initialA.Workspace, initialB.Workspace)
	}

	first, err := fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: initialA.Workspace.Version,
		Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes"), BaseSequence: initialA.FieldSequences["q1"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Workspace.Version != 2 || first.FieldSequences["q1"] != 2 {
		t.Fatalf("first field edit did not advance the shared sequence: %+v", first)
	}

	merged, err := fixture.access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: initialB.Workspace.Version,
		Edits: []FieldEdit{{FieldID: "q2", Value: formcontract.TextAnswer("No"), BaseSequence: initialB.FieldSequences["q2"]}},
	})
	if err != nil {
		t.Fatalf("stale different-field edit should merge: %v", err)
	}
	if merged.Workspace.Version != 3 || merged.FieldSequences["q1"] != 2 || merged.FieldSequences["q2"] != 3 {
		t.Fatalf("unexpected merged sequences: %+v", merged.FieldSequences)
	}
	if text, _ := merged.Answers["q1"].ScalarText(); text != "Yes" {
		t.Fatalf("q1 was lost during merge: %+v", merged.Answers)
	}
	if text, _ := merged.Answers["q2"].ScalarText(); text != "No" {
		t.Fatalf("q2 was not merged: %+v", merged.Answers)
	}
	if merged.FieldProvenance["q1"].RecipientID == merged.FieldProvenance["q2"].RecipientID {
		t.Fatalf("field provenance did not retain distinct TO contributors: %+v", merged.FieldProvenance)
	}
	if merged.FieldProvenance["q1"].Assurance != AssuranceEmailVerified || merged.FieldProvenance["q2"].Assurance != AssuranceEmailVerified {
		t.Fatalf("field provenance lost achieved assurance: %+v", merged.FieldProvenance)
	}

	_, err = fixture.access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: initialB.Workspace.Version,
		Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("No"), BaseSequence: initialB.FieldSequences["q1"]}},
	})
	var conflict WorkspaceConflict
	if !errors.As(err, &conflict) || conflict.CurrentVersion != 3 || len(conflict.Changed) != 1 || conflict.Changed[0].FieldID != "q1" || conflict.Changed[0].Sequence != 2 {
		t.Fatalf("same-field stale edit did not return the exact conflict: %#v %v", conflict, err)
	}
	if text, _ := conflict.Changed[0].ServerValue.ScalarText(); text != "Yes" {
		t.Fatalf("conflict did not return the current server value: %+v", conflict.Changed[0])
	}
}

func TestResponseWorkspaceCreatesImmutableAmendmentWithoutClosingDistribution(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	ctx := context.Background()

	view, err := fixture.access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	view, err = fixture.access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: view.Workspace.Version,
		Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes"), BaseSequence: view.FieldSequences["q1"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := fixture.access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision.Revision != 1 || first.Revision.SupersedesRevisionID != "" || !first.Revision.Current || first.Revision.AchievedAssurance != AssuranceEmailVerified {
		t.Fatalf("unexpected first immutable revision: %+v", first.Revision)
	}
	if first.Workspace.Status != ResponseWorkspaceOpen {
		t.Fatalf("submission closed the workspace before sender lock: %+v", first.Workspace)
	}

	store := fixture.access.store.(*MemoryDistributionAccessStore)
	store.distributions.mu.RLock()
	distribution := store.distributions.distributions[fixture.distribution.ID]
	store.distributions.mu.RUnlock()
	if distribution.Status != DistributionOpen {
		t.Fatalf("submission closed the distribution before deadline: %+v", distribution)
	}

	afterFirst, err := fixture.access.GetResponseWorkspace(ctx, tokens[1])
	if err != nil {
		t.Fatal(err)
	}
	if afterFirst.CurrentRevision == nil || afterFirst.CurrentRevision.ID != first.Revision.ID {
		t.Fatalf("current revision projection was not shared: %+v", afterFirst.CurrentRevision)
	}
	amended, err := fixture.access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: afterFirst.Workspace.Version,
		Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("No"), BaseSequence: afterFirst.FieldSequences["q1"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fixture.access.SubmitResponseWorkspace(ctx, tokens[1], SubmitWorkspaceInput{ExpectedVersion: amended.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	if second.Revision.Revision != 2 || second.Revision.SupersedesRevisionID != first.Revision.ID || !second.Revision.Current {
		t.Fatalf("amendment did not supersede revision 1: %+v", second.Revision)
	}

	registry := memoryWorkspaceRegistryFor(store)
	registry.mu.Lock()
	state := registry.workspaces[fixture.distribution.ID]
	registry.mu.Unlock()
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.revisions) != 2 || state.revisions[0].Current || !state.revisions[1].Current {
		t.Fatalf("immutable revision current flags are wrong: %+v", state.revisions)
	}
	if len(store.distributions.repo.submissions) != 2 {
		t.Fatalf("expected two immutable submission snapshots, got %d", len(store.distributions.repo.submissions))
	}
	firstSubmission := store.distributions.repo.submissions[first.Submission.SubmissionID]
	secondSubmission := store.distributions.repo.submissions[second.Submission.SubmissionID]
	if text, _ := firstSubmission.Answers["q1"].ScalarText(); text != "Yes" {
		t.Fatalf("first submission snapshot mutated after amendment: %+v", firstSubmission.Answers)
	}
	if text, _ := secondSubmission.Answers["q1"].ScalarText(); text != "No" {
		t.Fatalf("amended submission did not snapshot current answers: %+v", secondSubmission.Answers)
	}
	if firstSubmission.SessionID != "" || secondSubmission.SessionID != "" {
		t.Fatal("distribution submission leaked the distribution session into the legacy invitation-session slot")
	}
}

func TestResponseWorkspaceBlocksLockRevocationAndDeadline(t *testing.T) {
	t.Run("distribution lock", func(t *testing.T) {
		fixture, tokens := newTwoRecipientWorkspaceFixture(t)
		store := fixture.access.store.(*MemoryDistributionAccessStore)
		store.distributions.mu.Lock()
		distribution := store.distributions.distributions[fixture.distribution.ID]
		distribution.Status = DistributionLocked
		store.distributions.distributions[fixture.distribution.ID] = distribution
		store.distributions.mu.Unlock()
		if _, err := fixture.access.GetResponseWorkspace(context.Background(), tokens[0]); !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("locked distribution left workspace usable: %v", err)
		}
	})

	t.Run("workspace lock", func(t *testing.T) {
		fixture, tokens := newTwoRecipientWorkspaceFixture(t)
		store := fixture.access.store.(*MemoryDistributionAccessStore)
		store.distributions.mu.Lock()
		workspace := store.distributions.workspaces[fixture.distribution.ID]
		workspace.Status = ResponseWorkspaceLocked
		store.distributions.workspaces[fixture.distribution.ID] = workspace
		store.distributions.mu.Unlock()
		if _, err := fixture.access.GetResponseWorkspace(context.Background(), tokens[0]); !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("locked workspace remained usable: %v", err)
		}
	})

	t.Run("route revocation", func(t *testing.T) {
		fixture, tokens := newTwoRecipientWorkspaceFixture(t)
		store := fixture.access.store.(*MemoryDistributionAccessStore)
		store.mu.RLock()
		var routeID string
		for _, session := range store.sessions {
			if session.DistributionID == fixture.distribution.ID {
				routeID = session.RouteID
				break
			}
		}
		store.mu.RUnlock()
		if routeID == "" {
			t.Fatal("missing access route")
		}
		if err := fixture.access.RevokeDistributionAccessRoute(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, routeID); err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.access.GetResponseWorkspace(context.Background(), tokens[0]); !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("revoked route left workspace usable: %v", err)
		}
	})

	t.Run("deadline", func(t *testing.T) {
		fixture, tokens := newTwoRecipientWorkspaceFixture(t)
		fixture.access.now = func() time.Time { return fixture.distribution.Deadline.Add(time.Second) }
		if _, err := fixture.access.GetResponseWorkspace(context.Background(), tokens[0]); !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("expired distribution left workspace usable: %v", err)
		}
	})
}

func TestResponseWorkspaceRejectsDuplicateFieldReplacement(t *testing.T) {
	fixture, tokens := newTwoRecipientWorkspaceFixture(t)
	view, err := fixture.access.GetResponseWorkspace(context.Background(), tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.access.SaveResponseWorkspace(context.Background(), tokens[0], SaveWorkspaceInput{
		ExpectedVersion: view.Workspace.Version,
		Edits: []FieldEdit{
			{FieldID: "q1", Value: formcontract.TextAnswer("Yes"), BaseSequence: 0},
			{FieldID: "q1", Value: formcontract.TextAnswer("No"), BaseSequence: 0},
		},
	})
	if err == nil {
		t.Fatal("duplicate field replacement was accepted")
	}
}

func newTwoRecipientWorkspaceFixture(t *testing.T) (memoryAccessFixture, [2]string) {
	t.Helper()
	fixture := newMemoryAccessFixture(t, AccessSharedEmailOTP, []DistributionRecipientInput{
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "alpha@example.test", AudienceHint: "a***@example.test", ContactLabel: "Alpha"},
		{Role: RecipientTo, Type: RecipientExternalAudience, Address: "beta@example.test", AudienceHint: "b***@example.test", ContactLabel: "Beta"},
	})
	store := fixture.access.store.(*MemoryDistributionAccessStore)
	store.distributions.repo.mu.Lock()
	for requestID, request := range store.distributions.repo.requests {
		if request.TenantID != "tenant-a" {
			continue
		}
		request.Fields = append(request.Fields, Field{
			ID: "q2", SectionID: "general", Label: "Secondary control operating?", Type: string(formcontract.TypeYesNo),
			Options: []string{"Yes", "No"},
		})
		store.distributions.repo.requests[requestID] = request
	}
	store.distributions.repo.mu.Unlock()

	issued, err := fixture.access.IssueDistributionAccessRoutes(context.Background(), "tenant-a", "entity-a", fixture.distribution.ID, "actor-a")
	if err != nil || len(issued) != 1 {
		t.Fatalf("issue shared route: %+v %v", issued, err)
	}
	start, err := fixture.access.StartDistributionAccess(context.Background(), issued[0].Selector)
	if err != nil || len(start.Recipients) != 2 {
		t.Fatalf("start shared route: %+v %v", start, err)
	}
	var tokens [2]string
	for index := range start.Recipients {
		receipt, err := fixture.access.SendOTP(context.Background(), issued[0].Selector, start.Recipients[index].SelectorID)
		if err != nil {
			t.Fatal(err)
		}
		code := fixture.delivery.values[len(fixture.delivery.values)-1].Code
		redeemed, err := fixture.access.VerifyOTP(context.Background(), issued[0].Selector, receipt.ChallengeID, code)
		if err != nil {
			t.Fatal(err)
		}
		tokens[index] = redeemed.SessionToken
	}
	return fixture, tokens
}
