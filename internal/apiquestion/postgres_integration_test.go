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
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type allowGuard struct{}

func (allowGuard) Authorize(context.Context, commandauth.Request) (commandauth.Decision, error) {
	return commandauth.Decision{Allowed: true, Enforced: true, Resolution: &authority.Resolution{RuleID: "rule", PolicyVersion: "1"}}, nil
}

type auditOnlyGuard struct{}

func (auditOnlyGuard) Authorize(context.Context, commandauth.Request) (commandauth.Decision, error) {
	return commandauth.Decision{Allowed: true, Enforced: false, Resolution: &authority.Resolution{RuleID: "rule", PolicyVersion: "1"}}, nil
}

type blockingGuard struct {
	entered chan struct{}
	release chan struct{}
}
type allowQuestionBinding struct{}

func (allowQuestionBinding) AuthorizeQuestionBinding(ctx context.Context, _ string, _ string, _ string, _ int64, _ string, _ string, _ int64, _ string, _ string) error {
	if _, ok := sourceaccess.CatalogProofTransaction(ctx); !ok {
		return errors.New("missing catalog transaction")
	}
	return nil
}

type denyQuestionBinding struct{}

func (denyQuestionBinding) AuthorizeQuestionBinding(context.Context, string, string, string, int64, string, string, int64, string, string) error {
	return errors.New("denied")
}

func (g blockingGuard) Authorize(ctx context.Context, _ commandauth.Request) (commandauth.Decision, error) {
	close(g.entered)
	select {
	case <-g.release:
		return allowGuard{}.Authorize(ctx, commandauth.Request{})
	case <-ctx.Done():
		return commandauth.Decision{}, ctx.Err()
	}
}

func TestProtectedDraftTransactionAndScope(t *testing.T) {
	db := os.Getenv("TEST_DATABASE_URL")
	if db == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	idValue, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "api_question_" + strings.ReplaceAll(idValue, "-", "")
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	config, err := pgxpool.ParseConfig(db)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY,slug text NOT NULL); CREATE TABLE legal_entities(id uuid PRIMARY KEY); CREATE TABLE principals(id uuid PRIMARY KEY); CREATE TABLE monitoring_form_templates(revision_id uuid PRIMARY KEY,id uuid,tenant_id uuid,legal_entity_id uuid,version bigint,status text,fields jsonb); CREATE TABLE source_bindings(binding_id uuid,tenant_id uuid,source_id uuid,version bigint,status text,is_current boolean,UNIQUE(binding_id,tenant_id,source_id,version)); CREATE TABLE outbox_events(tenant_id uuid,aggregate_type text,aggregate_id uuid,event_type text,payload jsonb,occurred_at timestamptz,available_at timestamptz,next_attempt_at timestamptz);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000143_api_question_drafts.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	bindingMigration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000148_form_question_binding_proposals.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(bindingMigration)); err != nil {
		t.Fatal(err)
	}
	approvalMigration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000149_question_binding_approvals.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(approvalMigration)); err != nil {
		t.Fatal(err)
	}
	const tenant = "00000000-0000-4000-8000-000000000001"
	const entity = "00000000-0000-4000-8000-000000000002"
	const actorID = "00000000-0000-4000-8000-000000000003"
	const form = "00000000-0000-4000-8000-000000000004"
	const revision = "00000000-0000-4000-8000-000000000005"
	const reviewerID = "00000000-0000-4000-8000-000000000017"
	_, err = pool.Exec(ctx, `INSERT INTO tenants VALUES($1::uuid,'bank'); INSERT INTO legal_entities VALUES($2::uuid); INSERT INTO principals VALUES($3::uuid),($6::uuid); INSERT INTO monitoring_form_templates VALUES($5::uuid,$4::uuid,$1::uuid,$2::uuid,1,'DRAFT','[{"id":"q","type":"yes_no"}]'::jsonb)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actorID, form, revision, reviewerID)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: entity, PrincipalID: actorID, Kind: "PERSON", PermissionCodes: []string{identity.PermissionConfigWrite}, ExpiresAt: time.Now().Add(time.Hour)}
	actorCtx := identity.WithActor(ctx, actor)
	store := &PostgresStore{Pool: pool, Guard: allowGuard{}}
	input := DraftInput{Mode: "API", Endpoint: "https://bank.example/check", ResultType: "BOOLEAN", Key: "11111111-1111-4111-8111-111111111111"}
	t.Run("session expires while authorization is pending", func(t *testing.T) {
		guard := blockingGuard{entered: make(chan struct{}), release: make(chan struct{})}
		blockedStore := &PostgresStore{Pool: pool, Guard: guard}
		expiring := actor
		expiring.ExpiresAt = time.Now().Add(500 * time.Millisecond)
		result := make(chan error, 1)
		go func() {
			_, err := blockedStore.WriteDraft(identity.WithActor(ctx, expiring), form, 1, "q", 0, input)
			result <- err
		}()
		<-guard.entered
		time.Sleep(time.Until(expiring.ExpiresAt) + 50*time.Millisecond)
		close(guard.release)
		if err := <-result; !errors.Is(err, ErrUnavailable) {
			t.Fatalf("expired session wrote draft: %v", err)
		}
		var drafts, outbox int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM api_question_draft_revisions), (SELECT count(*) FROM outbox_events)`).Scan(&drafts, &outbox); err != nil || drafts != 0 || outbox != 0 {
			t.Fatalf("expired session persisted draft=%d outbox=%d err=%v", drafts, outbox, err)
		}
	})
	unauthorized := &PostgresStore{Pool: pool, Guard: auditOnlyGuard{}}
	if _, err = unauthorized.WriteDraft(actorCtx, form, 1, "q", 0, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unenforced authority: %v", err)
	}
	var wait sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wait.Add(1)
		next := input
		if i == 1 {
			next.Key = "22222222-2222-4222-8222-222222222222"
		}
		go func() { defer wait.Done(); _, err := store.WriteDraft(actorCtx, form, 1, "q", 0, next); errs <- err }()
	}
	wait.Wait()
	close(errs)
	var success, conflicts int
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflicts)
	}
	idempotent := DraftInput{Mode: "API", Endpoint: "https://bank.example/check", ResultType: "BOOLEAN", Key: "33333333-3333-4333-8333-333333333333"}
	receipt, err := store.WriteDraft(actorCtx, form, 1, "q", 1, idempotent)
	if err != nil {
		t.Fatalf("idempotent initial write: %v", err)
	}
	retry, err := store.WriteDraft(actorCtx, form, 1, "q", 1, idempotent)
	if err != nil || retry != receipt {
		t.Fatalf("idempotent retry: %+v %v", retry, err)
	}
	idempotent.Endpoint = "https://bank.example/different"
	if _, err = store.WriteDraft(actorCtx, form, 1, "q", 1, idempotent); !errors.Is(err, ErrConflict) {
		t.Fatalf("idempotency payload mismatch: %v", err)
	}
	const bindingID = "00000000-0000-4000-8000-000000000015"
	const sourceID = "00000000-0000-4000-8000-000000000016"
	if _, err = pool.Exec(ctx, `INSERT INTO source_bindings VALUES($1::uuid,$2::uuid,$3::uuid,1,'ACTIVE',true)`, bindingID, tenant, sourceID); err != nil {
		t.Fatal(err)
	}
	store.BindingAccess = allowQuestionBinding{}
	proposal := BindingProposal{FormID: form, FormVersion: 1, FieldID: "q", DraftID: receipt.ID, DraftVersion: 2, BindingID: bindingID, BindingVersion: 1, Key: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}
	proposed, err := store.ProposeQuestionBinding(actorCtx, proposal)
	if err != nil || proposed.Version != 1 {
		t.Fatalf("binding proposal %+v %v", proposed, err)
	}
	proposedRetry, err := store.ProposeQuestionBinding(actorCtx, proposal)
	if err != nil || proposedRetry != proposed {
		t.Fatalf("binding proposal retry %+v %v", proposedRetry, err)
	}
	review := BindingReviewInput{FormID: form, FormVersion: 1, ProposalID: proposed.ID, ProposalVersion: 1, Rationale: "Reviewed source binding", Key: "dddddddd-dddd-4ddd-8ddd-dddddddddddd"}
	reviewer := actor
	reviewer.PrincipalID = reviewerID
	reviewerCtx := identity.WithActor(ctx, reviewer)
	if _, err = store.ApproveQuestionBinding(actorCtx, review); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("maker approved own binding: %v", err)
	}
	checkerGuard := &captureAdoptionGuard{}
	store.Guard = checkerGuard
	candidate, err := store.ReadBindingReviewCandidate(reviewerCtx, review)
	if err != nil || candidate.ProposalID != proposed.ID || candidate.BindingID != bindingID || candidate.FieldID != "q" || candidate.ResultType != "BOOLEAN" {
		t.Fatalf("binding candidate %+v %v", candidate, err)
	}
	if checkerGuard.request.ObjectType != "FORM_TEMPLATE" || checkerGuard.request.ObjectID != form || checkerGuard.request.Responsibility != authority.ResponsibilityReviewer || checkerGuard.request.DecisionType != "forms.template.transition" || checkerGuard.request.Materiality != 3 {
		t.Fatalf("binding checker authority: %+v", checkerGuard.request)
	}
	staleReview := review
	staleReview.ProposalVersion = 2
	if _, err = store.ApproveQuestionBinding(reviewerCtx, staleReview); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale binding proposal: %v", err)
	}
	deniedReview := &PostgresStore{Pool: pool, Guard: allowGuard{}, BindingAccess: denyQuestionBinding{}}
	if _, err = deniedReview.ReadBindingReviewCandidate(reviewerCtx, review); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("binding proof denied: %v", err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_binding_approval CHECK (aggregate_type <> 'FORM_QUESTION_BINDING_APPROVAL') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ApproveQuestionBinding(reviewerCtx, review); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("binding approval outbox failure: %v", err)
	}
	var approvalCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM form_question_binding_approvals`).Scan(&approvalCount); err != nil || approvalCount != 0 {
		t.Fatalf("binding approval rollback=%d %v", approvalCount, err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT reject_binding_approval`); err != nil {
		t.Fatal(err)
	}
	approved, err := store.ApproveQuestionBinding(reviewerCtx, review)
	if err != nil || approved.Version != 1 {
		t.Fatalf("binding approval %+v %v", approved, err)
	}
	retryApproval, err := store.ApproveQuestionBinding(reviewerCtx, review)
	if err != nil || retryApproval != approved {
		t.Fatalf("binding approval retry %+v %v", retryApproval, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE form_question_binding_approvals SET rationale='changed'`); err == nil {
		t.Fatal("binding approval mutable")
	}
	store.Guard = allowGuard{}
	changedProposal := proposal
	changedProposal.Key = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	changedProposal.ExpectedVersion = 1
	deniedBinding := &PostgresStore{Pool: pool, Guard: allowGuard{}, BindingAccess: denyQuestionBinding{}}
	if _, err = deniedBinding.ProposeQuestionBinding(actorCtx, changedProposal); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("binding access denied: %v", err)
	}
	wrongField := changedProposal
	wrongField.FieldID = "different"
	if _, err = store.ProposeQuestionBinding(actorCtx, wrongField); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("wrong binding field: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE source_bindings SET status='PAUSED' WHERE binding_id=$1::uuid`, bindingID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ProposeQuestionBinding(actorCtx, changedProposal); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("inactive binding: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE source_bindings SET status='ACTIVE' WHERE binding_id=$1::uuid`, bindingID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_binding_outbox CHECK (aggregate_type <> 'FORM_QUESTION_BINDING_PROPOSAL') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ProposeQuestionBinding(actorCtx, changedProposal); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("binding outbox failure: %v", err)
	}
	var proposalCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM form_question_binding_proposals`).Scan(&proposalCount); err != nil || proposalCount != 1 {
		t.Fatalf("binding rollback count=%d error=%v", proposalCount, err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events DROP CONSTRAINT reject_binding_outbox`); err != nil {
		t.Fatal(err)
	}
	draft, err := store.ReadDraft(actorCtx, form, 1, "q")
	if err != nil || draft.Version != 2 || draft.Mode != "API" || draft.Endpoint != input.Endpoint {
		t.Fatalf("authorized draft read: %+v %v", draft, err)
	}
	if _, err = unauthorized.ReadDraft(actorCtx, form, 1, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unenforced read authority: %v", err)
	}
	var count int
	var leaked bool
	if err = pool.QueryRow(ctx, `SELECT count(*),bool_or(payload::text LIKE '%bank.example%') FROM outbox_events`).Scan(&count, &leaked); err != nil || count != 4 || leaked {
		t.Fatalf("outbox count=%d leaked=%v error=%v", count, leaked, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE api_question_draft_revisions SET mode='MANUAL'`); err == nil {
		t.Fatal("draft revision was mutable")
	}
	other := actor
	other.LegalEntityID = "00000000-0000-4000-8000-000000000006"
	if _, err = store.ReadDraft(identity.WithActor(ctx, other), form, 1, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-entity read: %v", err)
	}
	if _, err = store.WriteDraft(identity.WithActor(ctx, other), form, 1, "q", 1, DraftInput{Mode: "MANUAL", Key: "66666666-6666-4666-8666-666666666666"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("cross-entity write: %v", err)
	}
	if _, err = store.WriteDraft(actorCtx, form, 1, "q", 2, DraftInput{Mode: "API", Endpoint: "https://bank.example/other", ResultType: "BOOLEAN", Key: "44444444-4444-4444-8444-444444444444"}); err != nil {
		t.Fatalf("API draft revision: %v", err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	blockedStore := &PostgresStore{Pool: pool, Guard: blockingGuard{entered: entered, release: release}}
	blockedInput := input
	blockedInput.Key = "88888888-8888-4888-8888-888888888888"
	written := make(chan error, 1)
	go func() { _, err := blockedStore.WriteDraft(actorCtx, form, 1, "q", 3, blockedInput); written <- err }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("draft command did not acquire form lock")
	}
	updated := make(chan error, 1)
	go func() {
		_, err := pool.Exec(ctx, `UPDATE monitoring_form_templates SET status='PENDING_APPROVAL' WHERE revision_id=$1::uuid`, revision)
		updated <- err
	}()
	select {
	case err := <-updated:
		t.Fatalf("form transition did not wait: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	var writeNotBefore time.Time
	if err = pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&writeNotBefore); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err = <-written; err != nil {
		t.Fatalf("draft command: %v", err)
	}
	var draftAt, outboxAt time.Time
	if err = pool.QueryRow(ctx, `SELECT d.created_at,o.occurred_at FROM api_question_draft_revisions d JOIN outbox_events o ON o.aggregate_id=d.id WHERE d.idempotency_key=$1`, blockedInput.Key).Scan(&draftAt, &outboxAt); err != nil || draftAt.Before(writeNotBefore) || !outboxAt.Equal(draftAt) {
		t.Fatalf("write time not shared or precedes authorization: draft=%v outbox=%v boundary=%v err=%v", draftAt, outboxAt, writeNotBefore, err)
	}
	if err = <-updated; err != nil {
		t.Fatalf("form transition: %v", err)
	}
	if _, err = store.WriteDraft(actorCtx, form, 1, "q", 4, DraftInput{Mode: "MANUAL", Key: "99999999-9999-4999-8999-999999999999"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("transitioned form write: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE monitoring_form_templates SET status='DRAFT' WHERE revision_id=$1::uuid`, revision); err != nil {
		t.Fatal(err)
	}
	transition, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer transition.Rollback(ctx)
	if _, err = transition.Exec(ctx, `UPDATE monitoring_form_templates SET status='PENDING_APPROVAL' WHERE revision_id=$1::uuid`, revision); err != nil {
		t.Fatal(err)
	}
	written = make(chan error, 1)
	go func() {
		_, err := store.WriteDraft(actorCtx, form, 1, "q", 4, DraftInput{Mode: "MANUAL", Key: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"})
		written <- err
	}()
	select {
	case err := <-written:
		t.Fatalf("draft did not wait for transition: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err = transition.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-written; !errors.Is(err, ErrUnavailable) {
		t.Fatalf("transition-first command: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE monitoring_form_templates SET status='DRAFT' WHERE revision_id=$1::uuid`, revision); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `ALTER TABLE outbox_events ADD CONSTRAINT reject_draft_outbox CHECK (aggregate_type <> 'API_QUESTION_DRAFT') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	input.Key = "55555555-5555-4555-8555-555555555555"
	if _, err = store.WriteDraft(actorCtx, form, 1, "q", 4, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("outbox failure: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM api_question_draft_revisions`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("draft rollback count=%d error=%v", count, err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO monitoring_form_templates VALUES('00000000-0000-4000-8000-000000000007'::uuid,$1::uuid,$2::uuid,$3::uuid,2,'DRAFT','[{"id":"q","type":"yes_no"}]'::jsonb)`, form, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadDraft(actorCtx, form, 1, "q"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale form revision read: %v", err)
	}
	input.Key = "77777777-7777-4777-8777-777777777777"
	if _, err = store.WriteDraft(actorCtx, form, 1, "q", 4, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale form revision write: %v", err)
	}
}
