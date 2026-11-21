//go:build postgres && postgresintegration

package apiquestion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type captureAdoptionGuard struct{ request commandauth.Request }
type allowAdoptionForm struct{}

func (allowAdoptionForm) AuthorizeAdoptionForm(context.Context, string, string, string, string, int64) error {
	return nil
}

type denyAdoptionForm struct{}

func (denyAdoptionForm) AuthorizeAdoptionForm(context.Context, string, string, string, string, int64) error {
	return errors.New("denied")
}

type allowCapture struct{ request string }
type denyCaptureProof struct{}

func (denyCaptureProof) AuthorizeCaptureRequest(context.Context, string) error { return nil }
func (denyCaptureProof) VerifyApprovedQuestionBinding(context.Context, string, string, string) error {
	return errors.New("binding proof unavailable")
}

func (a allowCapture) AuthorizeCaptureRequest(_ context.Context, request string) error {
	if request != a.request {
		return errors.New("denied")
	}
	return nil
}
func (a allowCapture) VerifyApprovedQuestionBinding(_ context.Context, request, field, approval string) error {
	if request != a.request || field != "q" || approval == "" {
		return errors.New("denied")
	}
	return nil
}

type allowPin struct{ distribution, approval string }
type denyPinProof struct{}

func (denyPinProof) AuthorizeDistributionPin(context.Context, string, string) error { return nil }
func (denyPinProof) VerifyApprovedQuestionBinding(context.Context, string, string, string) error {
	return errors.New("binding proof unavailable")
}

func (a allowPin) AuthorizeDistributionPin(_ context.Context, distribution, approval string) error {
	if distribution != a.distribution || approval != a.approval {
		return errors.New("denied")
	}
	return nil
}
func (a allowPin) VerifyApprovedQuestionBinding(_ context.Context, distribution, field, approval string) error {
	if distribution != a.distribution || field != "q" || approval != a.approval {
		return errors.New("denied")
	}
	return nil
}

func (g *captureAdoptionGuard) Authorize(_ context.Context, request commandauth.Request) (commandauth.Decision, error) {
	g.request = request
	return commandauth.Decision{Allowed: true, Enforced: true, Resolution: &authority.Resolution{RuleID: "program-owner", PolicyVersion: "1"}}, nil
}

func TestAdoptionManualDraftIsScopedVersionedAndIdempotent(t *testing.T) {
	db := os.Getenv("TEST_DATABASE_URL")
	if db == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	unique, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "adoption_question_" + strings.ReplaceAll(unique, "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY,slug text); CREATE TABLE legal_entities(id uuid PRIMARY KEY); CREATE TABLE principals(id uuid PRIMARY KEY); CREATE TABLE programs(id uuid PRIMARY KEY,tenant_id uuid,legal_entity_id uuid,version bigint,status text); CREATE TABLE program_framework_adoptions(id uuid PRIMARY KEY,tenant_id uuid,legal_entity_id uuid,program_id uuid,edition_id uuid,status text,version bigint,proposed_by uuid,UNIQUE(id,tenant_id,legal_entity_id,program_id,edition_id)); CREATE TABLE monitoring_form_templates(revision_id uuid PRIMARY KEY,id uuid,tenant_id uuid,legal_entity_id uuid,program_id uuid,version bigint,status text,is_current boolean,fields jsonb); CREATE TABLE capture_form_distributions(id uuid PRIMARY KEY,tenant_id uuid,legal_entity_id uuid,form_template_id uuid,form_template_version bigint,subject_type text,subject_id uuid,status text,deadline timestamptz,UNIQUE(id,tenant_id,legal_entity_id)); CREATE TABLE capture_requests(id uuid PRIMARY KEY,tenant_id uuid,legal_entity_id uuid,distribution_id uuid,form_template_id uuid,form_template_version bigint,subject_type text,subject_id text,status text,deadline timestamptz,fields jsonb); CREATE TABLE outbox_events(tenant_id uuid,aggregate_type text,aggregate_id uuid,event_type text,payload jsonb,occurred_at timestamptz,available_at timestamptz,next_attempt_at timestamptz);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000144_program_adoption_question_drafts.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	approvalMigration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000146_adoption_question_approvals.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(approvalMigration)); err != nil {
		t.Fatal(err)
	}
	pinMigration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000147_distribution_manual_question_pins.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(pinMigration)); err != nil {
		t.Fatal(err)
	}
	const tenant = "00000000-0000-4000-8000-000000000001"
	const entity = "00000000-0000-4000-8000-000000000002"
	const actorID = "00000000-0000-4000-8000-000000000003"
	const form = "00000000-0000-4000-8000-000000000004"
	const revision = "00000000-0000-4000-8000-000000000005"
	const program = "00000000-0000-4000-8000-000000000006"
	const adoption = "00000000-0000-4000-8000-000000000007"
	const edition = "00000000-0000-4000-8000-000000000008"
	const reviewerID = "00000000-0000-4000-8000-000000000010"
	_, err = pool.Exec(ctx, `INSERT INTO tenants VALUES($1::uuid,'bank'); INSERT INTO legal_entities VALUES($2::uuid); INSERT INTO principals VALUES($3::uuid),($9::uuid); INSERT INTO programs VALUES($6::uuid,$1::uuid,$2::uuid,2,'ACTIVE'); INSERT INTO program_framework_adoptions VALUES($7::uuid,$1::uuid,$2::uuid,$6::uuid,$8::uuid,'PENDING',1,$3::uuid); INSERT INTO monitoring_form_templates VALUES($5::uuid,$4::uuid,$1::uuid,$2::uuid,NULL,1,'ACTIVE',true,'[{"id":"q","type":"yes_no"}]'::jsonb)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actorID, form, revision, program, adoption, edition, reviewerID)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: entity, PrincipalID: actorID, Kind: "PERSON", PermissionCodes: []string{identity.PermissionConfigWrite}, ExpiresAt: time.Now().Add(time.Hour)}
	actorCtx := identity.WithActor(ctx, actor)
	guard := &captureAdoptionGuard{}
	store := &PostgresStore{Pool: pool, Guard: guard, FormAccess: allowAdoptionForm{}}
	input := AdoptionManualDraft{ProgramID: program, ProgramVersion: 2, AdoptionID: adoption, AdoptionVersion: 1, FormID: form, FormVersion: 1, FieldID: "q", Rationale: "Manual review required", Key: "11111111-1111-4111-8111-111111111111"}
	t.Run("session expires while authorization is pending", func(t *testing.T) {
		blocking := blockingGuard{entered: make(chan struct{}), release: make(chan struct{})}
		blockedStore := &PostgresStore{Pool: pool, Guard: blocking, FormAccess: allowAdoptionForm{}}
		expiring := actor
		expiring.ExpiresAt = time.Now().Add(500 * time.Millisecond)
		result := make(chan error, 1)
		go func() {
			_, err := blockedStore.WriteAdoptionManualDraft(identity.WithActor(ctx, expiring), input)
			result <- err
		}()
		<-blocking.entered
		time.Sleep(time.Until(expiring.ExpiresAt) + 50*time.Millisecond)
		close(blocking.release)
		if err := <-result; !errors.Is(err, ErrUnavailable) {
			t.Fatalf("expired session wrote adoption draft: %v", err)
		}
		var drafts, outbox int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM program_adoption_question_draft_revisions), (SELECT count(*) FROM outbox_events)`).Scan(&drafts, &outbox); err != nil || drafts != 0 || outbox != 0 {
			t.Fatalf("expired session persisted draft=%d outbox=%d err=%v", drafts, outbox, err)
		}
	})
	receipt, err := store.WriteAdoptionManualDraft(actorCtx, input)
	if err != nil || receipt.Version != 1 {
		t.Fatalf("first association: %+v %v", receipt, err)
	}
	if guard.request.ObjectType != "PROGRAM" || guard.request.ObjectID != program || guard.request.Responsibility != authority.ResponsibilityOwner || guard.request.DecisionType != "program.framework.propose" || guard.request.Materiality != 3 {
		t.Fatalf("unexpected authority request: %+v", guard.request)
	}
	draft, err := store.ReadAdoptionManualDraft(actorCtx, input)
	if err != nil || draft.Version != 1 || draft.Mode != "MANUAL" || draft.Rationale != input.Rationale {
		t.Fatalf("draft read %+v %v", draft, err)
	}
	store.Guard = allowGuard{}
	retry, err := store.WriteAdoptionManualDraft(actorCtx, input)
	if err != nil || retry != receipt {
		t.Fatalf("idempotent retry: %+v %v", retry, err)
	}
	changed := input
	changed.Rationale = "Changed"
	if _, err = store.WriteAdoptionManualDraft(actorCtx, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("key reuse: %v", err)
	}
	changed.Key = "22222222-2222-4222-8222-222222222222"
	if _, err = store.WriteAdoptionManualDraft(actorCtx, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale association: %v", err)
	}
	other := actor
	other.LegalEntityID = "00000000-0000-4000-8000-000000000009"
	changed.ExpectedVersion = 1
	if _, err = store.ReadAdoptionManualDraft(identity.WithActor(ctx, other), input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-entity read: %v", err)
	}
	if _, err = store.WriteAdoptionManualDraft(identity.WithActor(ctx, other), changed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-entity write: %v", err)
	}
	unauthorized := &PostgresStore{Pool: pool, Guard: auditOnlyGuard{}, FormAccess: allowAdoptionForm{}}
	if _, err = unauthorized.WriteAdoptionManualDraft(actorCtx, changed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unenforced authority: %v", err)
	}
	if _, err = unauthorized.ReadAdoptionManualDraft(actorCtx, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unenforced read authority: %v", err)
	}
	inaccessible := &PostgresStore{Pool: pool, Guard: allowGuard{}, FormAccess: denyAdoptionForm{}}
	if _, err = inaccessible.WriteAdoptionManualDraft(actorCtx, changed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("form access denied: %v", err)
	}
	if _, err = inaccessible.ReadAdoptionManualDraft(actorCtx, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("form read denied: %v", err)
	}
	wrongField := changed
	wrongField.FieldID = "not-in-form"
	if _, err = store.WriteAdoptionManualDraft(actorCtx, wrongField); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unknown field: %v", err)
	}
	wrongVersion := changed
	wrongVersion.ProgramVersion++
	if _, err = store.WriteAdoptionManualDraft(actorCtx, wrongVersion); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale program: %v", err)
	}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		next := changed
		if i == 1 {
			next.Key = "33333333-3333-4333-8333-333333333333"
		}
		wait.Add(1)
		go func() { defer wait.Done(); _, err := store.WriteAdoptionManualDraft(actorCtx, next); results <- err }()
	}
	wait.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("concurrent success=%d conflicts=%d", success, conflicts)
	}
	// Approval must serialize with a pending association command in both orders.
	entered, release := make(chan struct{}), make(chan struct{})
	blocked := &PostgresStore{Pool: pool, Guard: blockingGuard{entered: entered, release: release}, FormAccess: allowAdoptionForm{}}
	third := changed
	third.ExpectedVersion = 2
	third.Key = "55555555-5555-4555-8555-555555555555"
	written := make(chan error, 1)
	go func() { _, err := blocked.WriteAdoptionManualDraft(actorCtx, third); written <- err }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("adoption command did not acquire locks")
	}
	updated := make(chan error, 1)
	go func() {
		_, err := pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='ACTIVE',version=2 WHERE id=$1::uuid`, adoption)
		updated <- err
	}()
	select {
	case err := <-updated:
		t.Fatalf("adoption approval did not wait: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	var writeNotBefore time.Time
	if err = pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&writeNotBefore); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err = <-written; err != nil {
		t.Fatalf("pending association: %v", err)
	}
	var draftAt, outboxAt time.Time
	if err = pool.QueryRow(ctx, `SELECT d.created_at,o.occurred_at FROM program_adoption_question_draft_revisions d JOIN outbox_events o ON o.aggregate_id=d.id WHERE d.idempotency_key=$1`, third.Key).Scan(&draftAt, &outboxAt); err != nil || draftAt.Before(writeNotBefore) || !outboxAt.Equal(draftAt) {
		t.Fatalf("write time not shared or precedes authorization: draft=%v outbox=%v boundary=%v err=%v", draftAt, outboxAt, writeNotBefore, err)
	}
	if err = <-updated; err != nil {
		t.Fatal(err)
	}
	if _, err = store.WriteAdoptionManualDraft(actorCtx, third); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("approved adoption: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='PENDING',version=1 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	approval, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer approval.Rollback(ctx)
	if _, err = approval.Exec(ctx, `UPDATE program_framework_adoptions SET status='ACTIVE',version=2 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	fourth := changed
	fourth.ExpectedVersion = 3
	fourth.Key = "66666666-6666-4666-8666-666666666666"
	written = make(chan error, 1)
	go func() { _, err := store.WriteAdoptionManualDraft(actorCtx, fourth); written <- err }()
	select {
	case err := <-written:
		t.Fatalf("association did not wait for approval: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err = approval.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-written; !errors.Is(err, ErrUnavailable) {
		t.Fatalf("approval-first command: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='PENDING',version=1 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	var latestID string
	if err = pool.QueryRow(ctx, `SELECT id::text FROM program_adoption_question_draft_revisions ORDER BY version DESC LIMIT 1`).Scan(&latestID); err != nil {
		t.Fatal(err)
	}
	approvalInput := AdoptionQuestionApproval{ProgramID: program, ProgramVersion: 2, AdoptionID: adoption, AdoptionVersion: 1, DraftID: latestID, DraftVersion: 3, Rationale: "Reviewed manual collection", Key: "77777777-7777-4777-8777-777777777777"}
	reviewer := actor
	reviewer.PrincipalID = reviewerID
	reviewerCtx := identity.WithActor(ctx, reviewer)
	if _, err = store.ApproveAdoptionQuestion(actorCtx, approvalInput); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("maker approved own choice: %v", err)
	}
	checkerGuard := &captureAdoptionGuard{}
	store.Guard = checkerGuard
	candidate, err := store.ReadAdoptionQuestionCandidate(reviewerCtx, approvalInput)
	if err != nil || candidate.DraftID != latestID || candidate.DraftVersion != 3 || candidate.Mode != "MANUAL" || candidate.FormID != form || candidate.FieldID != "q" {
		t.Fatalf("approval candidate %+v %v", candidate, err)
	}
	if checkerGuard.request.ObjectType != "PROGRAM" || checkerGuard.request.ObjectID != program || checkerGuard.request.Responsibility != authority.ResponsibilityAuthorizer || checkerGuard.request.DecisionType != "program.framework.approve" || checkerGuard.request.Materiality != 4 {
		t.Fatalf("checker authority request: %+v", checkerGuard.request)
	}
	staleApproval := approvalInput
	staleApproval.DraftVersion = 2
	if _, err = store.ApproveAdoptionQuestion(reviewerCtx, staleApproval); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale draft approval: %v", err)
	}
	otherReviewer := reviewer
	otherReviewer.LegalEntityID = "00000000-0000-4000-8000-000000000009"
	if _, err = store.ApproveAdoptionQuestion(identity.WithActor(ctx, otherReviewer), approvalInput); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-entity approval: %v", err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_approval_outbox CHECK (aggregate_type <> 'ADOPTION_QUESTION_APPROVAL') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ApproveAdoptionQuestion(reviewerCtx, approvalInput); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("approval outbox failure: %v", err)
	}
	var approvalCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM program_adoption_question_approvals`).Scan(&approvalCount); err != nil || approvalCount != 0 {
		t.Fatalf("approval rollback count=%d error=%v", approvalCount, err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT reject_approval_outbox`); err != nil {
		t.Fatal(err)
	}
	approved, err := store.ApproveAdoptionQuestion(reviewerCtx, approvalInput)
	if err != nil || approved.Version != 3 {
		t.Fatalf("approval %+v %v", approved, err)
	}
	approvedRetry, err := store.ApproveAdoptionQuestion(reviewerCtx, approvalInput)
	if err != nil || approvedRetry != approved {
		t.Fatalf("approval retry %+v %v", approvedRetry, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_adoption_question_approvals SET rationale='tampered'`); err == nil {
		t.Fatal("approval was mutable")
	}
	const distribution = "00000000-0000-4000-8000-000000000011"
	const request = "00000000-0000-4000-8000-000000000012"
	_, err = pool.Exec(ctx, `INSERT INTO capture_form_distributions VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'PROGRAM',$5::uuid,'DRAFT',$7); INSERT INTO capture_requests VALUES($6::uuid,$2::uuid,$3::uuid,$1::uuid,$4::uuid,1,'PROGRAM',$5,'READY',$7,'[{"id":"q","type":"yes_no"}]'::jsonb)`, pgx.QueryExecModeSimpleProtocol, distribution, tenant, entity, form, program, request, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	resolve := func(access CaptureRequestAccess, requestID, field string) (bool, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return false, err
		}
		defer tx.Rollback(ctx)
		return store.ResolvePinnedManual(ctx, tx, access, requestID, field)
	}
	if _, err = resolve(allowCapture{request: request}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("pending adoption runtime: %v", err)
	}
	pendingPin, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pendingPin.Rollback(ctx)
	if _, err = store.PinApprovedManual(ctx, pendingPin, allowPin{distribution: distribution, approval: approved.ID}, distribution, "q", approved.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("pin pending adoption: %v", err)
	}
	if err = pendingPin.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='ACTIVE',version=2 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	deniedPin, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer deniedPin.Rollback(ctx)
	if _, err = store.PinApprovedManual(ctx, deniedPin, allowPin{distribution: "other", approval: approved.ID}, distribution, "q", approved.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("pin without authority: %v", err)
	}
	if err = deniedPin.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	unprovedPin, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer unprovedPin.Rollback(ctx)
	if _, err = store.PinApprovedManual(ctx, unprovedPin, denyPinProof{}, distribution, "q", approved.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unproved question binding: %v", err)
	}
	if err = unprovedPin.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_pin_outbox CHECK (aggregate_type <> 'DISTRIBUTION_MANUAL_QUESTION_PIN') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	failedPin, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer failedPin.Rollback(ctx)
	if _, err = store.PinApprovedManual(ctx, failedPin, allowPin{distribution: distribution, approval: approved.ID}, distribution, "q", approved.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("pin outbox failure: %v", err)
	}
	if err = failedPin.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var pinCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_manual_question_pins`).Scan(&pinCount); err != nil || pinCount != 0 {
		t.Fatalf("pin rollback count=%d error=%v", pinCount, err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT reject_pin_outbox`); err != nil {
		t.Fatal(err)
	}
	pinning, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pinning.Rollback(ctx)
	pin, err := store.PinApprovedManual(ctx, pinning, allowPin{distribution: distribution, approval: approved.ID}, distribution, "q", approved.ID)
	if err != nil || pin == "" {
		t.Fatalf("pin approved manual choice: %q %v", pin, err)
	}
	if err = pinning.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`, distribution); err != nil {
		t.Fatal(err)
	}
	manual, err := resolve(allowCapture{request: request}, request, "q")
	if err != nil || !manual {
		t.Fatalf("active approved manual runtime: %v %v", manual, err)
	}
	if _, err = resolve(allowCapture{request: "other"}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unauthorized request: %v", err)
	}
	if _, err = resolve(allowCapture{request: request}, request, "other"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unpinned field: %v", err)
	}
	if _, err = resolve(denyCaptureProof{}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unproved runtime binding: %v", err)
	}
	if _, err = store.ResolvePinnedManual(ctx, nil, allowCapture{request: request}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing transaction: %v", err)
	}
	if _, err = resolve(nil, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing capture authorization: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET form_template_version=2 WHERE id=$1::uuid`, distribution); err != nil {
		t.Fatal(err)
	}
	if _, err = resolve(allowCapture{request: request}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("different form revision: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET form_template_version=1 WHERE id=$1::uuid`, distribution); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_id='00000000-0000-4000-8000-000000000014'::uuid WHERE id=$1::uuid`, distribution); err != nil {
		t.Fatal(err)
	}
	if _, err = resolve(allowCapture{request: request}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("different Program: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_id=$2::uuid WHERE id=$1::uuid`, distribution, program); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_distribution_manual_question_pins SET field_id='other' WHERE id=$1::uuid`, pin); err == nil {
		t.Fatal("distribution pin was mutable")
	}
	resolution, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if manual, err = store.ResolvePinnedManual(ctx, resolution, allowCapture{request: request}, request, "q"); err != nil || !manual {
		t.Fatalf("transactional resolution %v %v", manual, err)
	}
	retired := make(chan error, 1)
	go func() {
		_, err := pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='RETIRED',version=3 WHERE id=$1::uuid`, adoption)
		retired <- err
	}()
	select {
	case err := <-retired:
		t.Fatalf("adoption retirement did not wait: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err = resolution.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-retired; err != nil {
		t.Fatal(err)
	}
	if _, err = resolve(allowCapture{request: request}, request, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("retired adoption: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='PENDING',version=1 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	store.Guard = allowGuard{}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&count); err != nil || count != 5 {
		t.Fatalf("outbox count=%d error=%v", count, err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_manual_outbox CHECK (aggregate_type <> 'ADOPTION_QUESTION_DRAFT') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	failed := changed
	failed.ExpectedVersion = 3
	failed.Key = "44444444-4444-4444-8444-444444444444"
	if _, err = store.WriteAdoptionManualDraft(actorCtx, failed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("outbox failure: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM program_adoption_question_draft_revisions`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("draft rollback count=%d error=%v", count, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE program_adoption_question_draft_revisions SET rationale='changed'`); err == nil {
		t.Fatal("draft revision was mutable")
	}
	if _, err = pool.Exec(ctx, `UPDATE program_framework_adoptions SET status='ACTIVE',version=2 WHERE id=$1::uuid`, adoption); err != nil {
		t.Fatal(err)
	}
	if _, err = store.WriteAdoptionManualDraft(actorCtx, changed); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("closed adoption: %v", err)
	}
}
