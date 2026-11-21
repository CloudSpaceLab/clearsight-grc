//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresQuestionAuditSerializesAcrossInstances(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	unique, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "question_test_" + strings.ReplaceAll(unique, "-", "")
	if !regexp.MustCompile(`^question_test_[a-f0-9]{32}$`).MatchString(schema) {
		t.Fatal("invalid test schema")
	}
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	}()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY,slug text NOT NULL UNIQUE); CREATE TABLE legal_entities(id uuid PRIMARY KEY,tenant_id uuid NOT NULL REFERENCES tenants(id)); CREATE TABLE principals(id uuid PRIMARY KEY,tenant_id uuid NOT NULL REFERENCES tenants(id));`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/000142_api_question_test_audit.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	const tenant = "11111111-1111-7111-8111-111111111111"
	const entity = "22222222-2222-7222-8222-222222222222"
	const person = "33333333-3333-7333-8333-333333333333"
	const form = "44444444-4444-7444-8444-444444444444"
	if _, err = pool.Exec(ctx, `INSERT INTO tenants VALUES($1,'bank')`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO legal_entities VALUES($1,$2)`, entity, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals VALUES($1,$2)`, person, tenant); err != nil {
		t.Fatal(err)
	}
	actor := QuestionTestActor{TenantID: "bank", LegalEntityID: entity, PrincipalID: person, AuthorityRuleID: "route", AuthorityPolicyVersion: "v1"}
	stores := []*PostgresQuestionTestAudit{NewPostgresQuestionTestAudit(pool), NewPostgresQuestionTestAudit(pool)}
	var group sync.WaitGroup
	var mu sync.Mutex
	var allowed int
	var first string
	for index := 0; index < 30; index++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			alias := actor
			if i%2 == 1 {
				alias.TenantID = tenant
			}
			value, err := stores[i%2].Reserve(ctx, alias, form, 1, "question", "LIVE", time.Now())
			if err == nil {
				mu.Lock()
				allowed++
				if first == "" {
					first = value
				}
				mu.Unlock()
			} else if !errors.Is(err, ErrQuestionTestRateLimit) {
				t.Errorf("reservation: %v", err)
			}
		}(index)
	}
	group.Wait()
	if allowed != 10 {
		t.Fatalf("allowed %d, want 10", allowed)
	}
	for i := 1; i <= 5; i++ {
		principal := fmt.Sprintf("%08x-3333-7333-8333-333333333333", i+50)
		if _, err := pool.Exec(ctx, `INSERT INTO principals VALUES($1,$2)`, principal, tenant); err != nil {
			t.Fatal(err)
		}
		other := actor
		other.PrincipalID = principal
		for j := 0; j < 10; j++ {
			if _, err := stores[j%2].Reserve(ctx, other, form, 1, "question", "SAMPLE", time.Now()); err != nil {
				t.Fatalf("entity allowance %d/%d: %v", i, j, err)
			}
		}
	}
	seventh := "77777777-3333-7333-8333-333333333333"
	if _, err := pool.Exec(ctx, `INSERT INTO principals VALUES($1,$2)`, seventh, tenant); err != nil {
		t.Fatal(err)
	}
	other := actor
	other.PrincipalID = seventh
	if _, err := stores[1].Reserve(ctx, other, form, 1, "question", "LIVE", time.Now()); !errors.Is(err, ErrQuestionTestRateLimit) {
		t.Fatalf("entity limit not enforced: %v", err)
	}
	actor.TenantID = tenant
	if err := stores[0].Complete(ctx, actor, first, QuestionTestValid, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := stores[1].Complete(ctx, actor, first, QuestionTestValid, time.Now()); err == nil {
		t.Fatal("duplicate outcome accepted")
	}
	if _, err := pool.Exec(ctx, `UPDATE api_question_test_attempts SET mode='SAMPLE' WHERE id=$1::uuid`, first); err == nil {
		t.Fatal("attempt was mutable")
	}
	var mode, outcome, rule, policy string
	if err := pool.QueryRow(ctx, `SELECT a.mode,o.outcome,a.authority_rule_id,a.authority_policy_version FROM api_question_test_attempts a JOIN api_question_test_outcomes o ON o.attempt_id=a.id WHERE a.id=$1::uuid`, first).Scan(&mode, &outcome, &rule, &policy); err != nil {
		t.Fatal(err)
	}
	if mode != "LIVE" || outcome != "VALID" || rule != "route" || policy != "v1" {
		t.Fatal(fmt.Sprintf("invalid audit: %s %s %s %s", mode, outcome, rule, policy))
	}
}
