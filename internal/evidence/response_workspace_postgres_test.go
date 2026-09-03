//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresResponseWorkspaceMergesAndPersistsImmutableAmendments(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenantID := mustResponseWorkspaceID(t)
	entityID := mustResponseWorkspaceID(t)
	actorID := mustResponseWorkspaceID(t)
	formID := mustResponseWorkspaceID(t)
	subjectID := mustResponseWorkspaceID(t)
	tenantSlug := "response-workspace-" + tenantID[len(tenantID)-12:]
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	setupResponseWorkspaceFixture(t, ctx, pool, tenantID, tenantSlug, entityID, actorID, formID, now)
	t.Cleanup(func() { cleanupResponseWorkspaceTenant(context.Background(), pool, tenantID) })

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
		TenantID: tenantSlug, LegalEntityID: "WORKSPACE", FormTemplateID: formID, FormTemplateVersion: 1,
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
	var baselineEvents, baselineOutbox int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_%'),
		(SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_%')`,
		bundle.Distribution.ID).Scan(&baselineEvents, &baselineOutbox); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION response_score_outbox_failure_test() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.event_type='FORM_RESPONSE_SCORED' THEN RAISE EXCEPTION 'forced response score outbox failure'; END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER response_score_outbox_failure_test BEFORE INSERT ON outbox_events
		FOR EACH ROW EXECUTE FUNCTION response_score_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS response_score_outbox_failure_test ON outbox_events; DROP FUNCTION IF EXISTS response_score_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol)
	})
	if _, err := access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: merged.Workspace.Version}); err == nil {
		t.Fatal("expected forced scored-response outbox failure")
	}
	var failedSubmissions, failedRevisions, failedEvents, failedOutbox int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM capture_submissions WHERE distribution_id=$1::uuid),
		(SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid),
		(SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_%'),
		(SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_%')`,
		bundle.Distribution.ID).Scan(&failedSubmissions, &failedRevisions, &failedEvents, &failedOutbox); err != nil {
		t.Fatal(err)
	}
	if failedSubmissions != 0 || failedRevisions != 0 || failedEvents != baselineEvents || failedOutbox != baselineOutbox {
		t.Fatalf("failed score outbox leaked material state: submissions=%d revisions=%d events=%d/%d outbox=%d/%d", failedSubmissions, failedRevisions, failedEvents, baselineEvents, failedOutbox, baselineOutbox)
	}
	if _, err := pool.Exec(ctx, `DROP TRIGGER response_score_outbox_failure_test ON outbox_events; DROP FUNCTION response_score_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
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

	var editCount, revisionCount, currentCount, submissionCount, legacySessionCount, scoredEventCount, scoredOutboxCount int
	var workspaceStatus ResponseWorkspaceStatus
	var distributionStatus DistributionStatus
	var firstAddress, secondAddress, currentScoreState string
	var currentRawScore float64
	var currentScoreCalculatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM capture_response_workspace_edits WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current),
		  (SELECT count(*) FROM capture_submissions WHERE distribution_id=$1::uuid),
		  (SELECT count(*) FROM capture_submissions WHERE distribution_id=$1::uuid AND session_id IS NOT NULL),
		  (SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_SCORED_%'),
		  (SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='FORM_RESPONSE_SCORED'),
		  (SELECT status FROM capture_response_workspaces WHERE distribution_id=$1::uuid),
		  (SELECT status FROM capture_form_distributions WHERE id=$1::uuid),
		  (SELECT answers->'registered_address'->>'text' FROM capture_submissions WHERE id=$2::uuid),
		  (SELECT answers->'registered_address'->>'text' FROM capture_submissions WHERE id=$3::uuid),
		  (SELECT score_state FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current),
		  (SELECT raw_score FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current),
		  (SELECT score_calculated_at FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current)`,
		bundle.Distribution.ID, firstSubmission.Submission.SubmissionID, secondSubmission.Submission.SubmissionID,
	).Scan(&editCount, &revisionCount, &currentCount, &submissionCount, &legacySessionCount, &scoredEventCount, &scoredOutboxCount, &workspaceStatus, &distributionStatus, &firstAddress, &secondAddress, &currentScoreState, &currentRawScore, &currentScoreCalculatedAt); err != nil {
		t.Fatal(err)
	}
	if editCount != 3 || revisionCount != 2 || currentCount != 1 || submissionCount != 2 || legacySessionCount != 0 {
		t.Fatalf("unexpected durable workspace counts: edits=%d revisions=%d current=%d submissions=%d legacy_sessions=%d", editCount, revisionCount, currentCount, submissionCount, legacySessionCount)
	}
	if scoredEventCount != 2 || scoredOutboxCount != 2 || currentScoreState != string(ResponseScoreFinal) || currentRawScore != 0 || currentScoreCalculatedAt.IsZero() {
		t.Fatalf("generalized scoring was not persisted and emitted atomically: events=%d outbox=%d state=%s raw=%v calculated_at=%v", scoredEventCount, scoredOutboxCount, currentScoreState, currentRawScore, currentScoreCalculatedAt)
	}
	if workspaceStatus != ResponseWorkspaceOpen || distributionStatus != DistributionOpen {
		t.Fatalf("submission prematurely closed runtime: workspace=%s distribution=%s", workspaceStatus, distributionStatus)
	}
	if firstAddress != "Lagos" || secondAddress != "Abuja" {
		t.Fatalf("immutable submission snapshots changed: first=%q second=%q", firstAddress, secondAddress)
	}
}

func TestPostgresResponseWorkspaceSubmitAcceptsOnlyOwnInterveningAutosaves(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenantID := mustResponseWorkspaceID(t)
	entityID := mustResponseWorkspaceID(t)
	actorID := mustResponseWorkspaceID(t)
	formID := mustResponseWorkspaceID(t)
	subjectID := mustResponseWorkspaceID(t)
	tenantSlug := "response-workspace-" + tenantID[len(tenantID)-12:]
	now := time.Date(2026, 8, 28, 4, 0, 0, 0, time.UTC)
	setupResponseWorkspaceFixture(t, ctx, pool, tenantID, tenantSlug, entityID, actorID, formID, now)
	t.Cleanup(func() { cleanupResponseWorkspaceTenant(context.Background(), pool, tenantID) })

	var recipientKey, accessKey [32]byte
	for index := range recipientKey {
		recipientKey[index] = 0x73
		accessKey[index] = 0x74
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: tenantSlug, LegalEntityID: "WORKSPACE", FormTemplateID: formID, FormTemplateVersion: 1,
		SubjectType: "VENDOR", SubjectID: subjectID, Title: "Submission concurrency review", Purpose: "Verify submission after autosave.",
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
		receipt, sendErr := access.SendOTP(ctx, routes[0].Selector, start.Recipients[index].SelectorID)
		if sendErr != nil {
			t.Fatal(sendErr)
		}
		redeemed, verifyErr := access.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, delivery.code)
		if verifyErr != nil {
			t.Fatal(verifyErr)
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
	if _, err := access.SaveResponseWorkspace(ctx, tokens[0], SaveWorkspaceInput{
		ExpectedVersion: initialA.Workspace.Version,
		Edits: []FieldEdit{{
			FieldID: "registered_address", Value: formcontract.TextAnswer("Lagos"), BaseSequence: initialA.FieldSequences["registered_address"],
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := access.SubmitResponseWorkspace(ctx, tokens[0], SubmitWorkspaceInput{ExpectedVersion: initialA.Workspace.Version}); err != nil {
		t.Fatalf("the verified session could not submit after its own PostgreSQL autosave: %v", err)
	}
	_, err = access.SubmitResponseWorkspace(ctx, tokens[1], SubmitWorkspaceInput{ExpectedVersion: initialB.Workspace.Version})
	var conflict WorkspaceConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("another recipient's PostgreSQL edit did not block stale submission: %v", err)
	}
}

func mustResponseWorkspaceID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func setupResponseWorkspaceFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, tenantSlug, entityID, actorID, formID string, now time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Response Workspace Integration');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($3::uuid,$1::uuid,'WORKSPACE','Response Workspace Entity','NG',$6);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($4::uuid,$1::uuid,'PERSON','Response Workspace Actor','ACTIVE',$6);
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,code,name,purpose,presentation,scoring_mode,score_profile,sections,fields,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES(
			$5::uuid,$1::uuid,$3::uuid,'WORKSPACE-FORM','Workspace form','Shared response workspace integration.',
			'{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
			'RISK',
			'{"version":"risk-v2","mode":"RISK","direction":"HIGH_IS_POOR","contributions":[{"id":"control-score","label":"Control concern","weight":100,"predicate":{"field_id":"control_confirmed","operator":"EQUALS","values":["No"]},"match_points":100,"non_match_points":0,"missing":"INDETERMINATE"}],"bands":[{"band":"LOW","from":0,"through":24},{"band":"MODERATE","from":25,"through":49},{"band":"HIGH","from":50,"through":74},{"band":"CRITICAL","from":75,"through":100}]}'::jsonb,
			'[{"id":"general","title":"General"}]'::jsonb,
			'[
			  {"id":"registered_address","section_id":"general","label":"Registered address","type":"short_text","required":true},
			  {"id":"control_confirmed","section_id":"general","label":"Control confirmed","type":"yes_no","required":false,"options":["Yes","No"]}
			]'::jsonb,
			'ACTIVE',true,$6,1,$4::uuid,$6,$6
		)`, pgx.QueryExecModeSimpleProtocol, tenantID, tenantSlug, entityID, actorID, formID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func cleanupResponseWorkspaceTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	for _, statement := range []string{
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_response_revisions WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_response_workspace_edits WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_otp_challenges WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_distribution_sessions WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_access_routes WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_submissions WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_artifacts WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_distribution_recipients WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_response_workspaces WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_requests WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_form_distributions WHERE tenant_id=$1::uuid`,
		`DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = pool.Exec(ctx, statement, tenantID)
	}
}
