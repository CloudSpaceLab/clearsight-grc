//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestPostgresResponseWorkspaceMergesAndPersistsImmutableAmendments(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const (
		tenantID  = "9e666666-6666-7666-8666-666666666661"
		entityID  = "9e666666-6666-7666-8666-666666666662"
		actorID   = "9e666666-6666-7666-8666-666666666663"
		formID    = "9e666666-6666-7666-8666-666666666664"
		subjectID = "9e666666-6666-7666-8666-666666666665"
	)
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	setupDistributionAccessFixture(t, ctx, pool, tenantID, entityID, actorID, formID, now)
	defer cleanupDistributionTenant(context.Background(), pool, tenantID)
	if _, err := pool.Exec(ctx, `
		UPDATE monitoring_form_templates
		SET fields='[
		  {"id":"registered_address","section_id":"general","label":"Registered address","type":"short_text","required":true},
		  {"id":"control_confirmed","section_id":"general","label":"Control confirmed","type":"yes_no","required":false,"options":["Yes","No"]}
		]'::jsonb
		WHERE id=$1::uuid`, formID); err != nil {
		t.Fatal(err)
	}

	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index] = 0x71
		accessKey[index] = 0x72
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "distribution-access-integration", LegalEntityID: "ACCESS", FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Shared workspace review", Purpose: "Exercise shared response amendments.",
		AccessPolicy: AccessSharedEmailOTP, EstimatedMinutes: 5,
		Deadline: now.Add(4 * time.Hour), RouteExpiresAt: now.Add(3 * time.Hour), CreatedBy: actorID,
		Recipients: []DistributionRecipientInput{
			{Role: RecipientTo, Type: RecipientExternalAudience, Address: "alpha@example.test", AudienceHint: "a***@example.test", ContactLabel: "Alpha"},
			{Role: RecipientTo, Type: RecipientExternalAudience, Address: "beta@example.test", AudienceHint: "b***@example.test", ContactLabel: "Beta"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN',updated_at=$2 WHERE id=$1::uuid`, bundle.Distribution.ID, now); err != nil {
		t.Fatal(err)
	}

	delivery := &postgresAccessOTPDelivery{}
	access, err := NewDistributionAccessService(store, keyring, delivery, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access.now = func() time.Time { return now }
	routes, err := access.IssueDistributionAccessRoutes(ctx, tenantID, entityID, bundle.Distribution.ID, actorID)
	if err != nil || len(routes) != 1 {
		t.Fatalf("issue shared route: %+v %v", routes, err)
	}
	start, err := access.StartDistributionAccess(ctx, routes[0].Selector)
	if err != nil || len(start.Recipients) != 2 {
		t.Fatalf("start shared route: %+v %v", start, err)
	}
	var tokens [2]string
	for index := range start.Recipients {
		receipt, err := access.SendOTP(ctx, routes[0].Selector, start.Recipients[index].SelectorID)
		if err != nil {
			t.Fatal(err)
		}
		redeemed, err := access.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, delivery.code)
		if err != nil {
			t.Fatal(err)
		}
		tokens[index] = redeemed.SessionToken
	}

	initialA, err := access.GetResponseWorkspace(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	initialB, err := access.GetResponseWorkspace(ctx, tokens[1])
	if err != nil {
		t.Fatal(err)
	}
	first, err := access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: initialA.Workspace.Version,
		Edits:           []FieldEdit{{FieldID: "registered_address", Value: formcontract.TextAnswer("Lagos"), BaseSequence: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: initialB.Workspace.Version,
		Edits:           []FieldEdit{{FieldID: "control_confirmed", Value: formcontract.TextAnswer("Yes"), BaseSequence: 0}},
	})
	if err != nil {
		t.Fatalf("stale different-field PostgreSQL edit should merge: %v", err)
	}
	if first.Workspace.Version != 2 || merged.Workspace.Version != 3 {
		t.Fatalf("unexpected workspace versions: %d %d", first.Workspace.Version, merged.Workspace.Version)
	}
	_, err = access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: initialB.Workspace.Version,
		Edits:           []FieldEdit{{FieldID: "registered_address", Value: formcontract.TextAnswer("Abuja"), BaseSequence: 0}},
	})
	var conflict WorkspaceConflict
	if !errors.As(err, &conflict) || len(conflict.Changed) != 1 || conflict.Changed[0].FieldID != "registered_address" {
		t.Fatalf("PostgreSQL same-field conflict = %#v, err=%v", conflict, err)
	}

	firstSubmission, err := access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: merged.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	afterSubmit, err := access.GetResponseWorkspace(ctx, tokens[1])
	if err != nil {
		t.Fatal(err)
	}
	amended, err := access.SaveResponseWorkspace(ctx, tokens[1], SaveWorkspaceInput{
		ExpectedVersion: afterSubmit.Workspace.Version,
		Edits: []FieldEdit{{
			FieldID: "registered_address", Value: formcontract.TextAnswer("Abuja"),
			BaseSequence: afterSubmit.FieldSequences["registered_address"],
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	secondSubmission, err := access.SubmitResponseWorkspace(ctx, tokens[1], SubmitWorkspaceInput{ExpectedVersion: amended.Workspace.Version})
	if err != nil {
		t.Fatal(err)
	}
	if secondSubmission.Revision.Revision != 2 || secondSubmission.Revision.SupersedesRevisionID != firstSubmission.Revision.ID {
		t.Fatalf("PostgreSQL amendment did not supersede revision 1: %+v", secondSubmission.Revision)
	}

	var editCount, revisionCount, currentCount, submissionCount, legacySessionCount int
	var workspaceStatus ResponseWorkspaceStatus
	var distributionStatus DistributionStatus
	var firstAddress, secondAddress string
	if err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM capture_response_workspace_edits WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current),
		  (SELECT count(*) FROM capture_submissions WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_submissions WHERE distribution_id=$1::uuid AND session_id IS NOT NULL),
		  (SELECT status FROM capture_response_workspaces WHERE distribution_id=$1::uuid),
		  (SELECT status FROM capture_form_distributions WHERE id=$1::uuid),
		  (SELECT answers->'registered_address'->>'text' FROM capture_submissions WHERE id=$2::uuid),
		  (SELECT answers->'registered_address'->>'text' FROM capture_submissions WHERE id=$3::uuid)`,
		bundle.Distribution.ID, firstSubmission.Submission.SubmissionID, secondSubmission.Submission.SubmissionID,
	).Scan(&editCount, &revisionCount, &currentCount, &submissionCount, &legacySessionCount, &workspaceStatus, &distributionStatus, &firstAddress, &secondAddress); err != nil {
		t.Fatal(err)
	}
	if editCount != 3 || revisionCount != 2 || currentCount != 1 || submissionCount != 2 || legacySessionCount != 0 {
		t.Fatalf("unexpected durable workspace counts: edits=%d revisions=%d current=%d submissions=%d legacy_sessions=%d", editCount, revisionCount, currentCount, submissionCount, legacySessionCount)
	}
	if workspaceStatus != ResponseWorkspaceOpen || distributionStatus != DistributionOpen {
		t.Fatalf("submission prematurely closed runtime: workspace=%s distribution=%s", workspaceStatus, distributionStatus)
	}
	if firstAddress != "Lagos" || secondAddress != "Abuja" {
		t.Fatalf("immutable submission snapshots changed: first=%q second=%q", firstAddress, secondAddress)
	}
}
