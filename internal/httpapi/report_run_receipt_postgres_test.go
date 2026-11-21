//go:build postgres && postgresintegration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresReportRunHTTPConcurrentReceipt(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	newID := func() string {
		t.Helper()
		value, err := id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	tenant, entity, maker, reviewer, authorizer, performer := newID(), newID(), newID(), newID(), newID(), newID()
	now := time.Now().UTC()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$7,'Report receipt test'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'BANK','Bank','NG',$8); INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Maker','ACTIVE',$8),($4::uuid,$1::uuid,'PERSON','Reviewer','ACTIVE',$8),($5::uuid,$1::uuid,'PERSON','Authorizer','ACTIVE',$8),($6::uuid,$1::uuid,'PERSON','Performer','ACTIVE',$8)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, maker, reviewer, authorizer, performer, "report-http-"+tenant, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	checker := &reportingHTTPAuthority{expected: map[authority.Responsibility]string{authority.ResponsibilityProposer: maker, authority.ResponsibilityReviewer: reviewer, authority.ResponsibilityAuthorizer: authorizer, authority.ResponsibilityPerformer: performer}}
	service := reporting.NewService(reporting.NewPostgresRepository(pool), evidence.NewMemoryObjectStore(), checker)
	scope := reporting.ReportScope{TenantID: tenant, LegalEntityID: entity}
	actor := func(principal string) context.Context {
		return identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: principal, Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)})
	}
	definition, err := service.Propose(actor(maker), reporting.ProposeInput{Code: "REPORT-RECEIPT", Name: "Report receipt test", Dataset: reporting.DatasetPrograms, ScopeKind: reporting.ScopeLegalEntity, Format: reporting.FormatCSV, Filter: &reporting.ReportFilterExpression{Kind: "group", Operator: "and"}})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Submit(actor(maker), reporting.DefinitionTransitionInput{Scope: scope, DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Review(actor(reviewer), reporting.DefinitionTransitionInput{Scope: scope, DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum})
	if err != nil {
		t.Fatal(err)
	}
	definition, err = service.Activate(actor(authorizer), reporting.DefinitionTransitionInput{Scope: scope, DefinitionID: definition.ID, ExpectedVersion: definition.Version, ChecksumSeen: definition.StoredChecksum})
	if err != nil {
		t.Fatal(err)
	}
	guard, err := commandauth.New(checker, commandauth.ModeEnforce, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator(tenant, maker, entity, "GRC_ADMIN"), CommandGuard: guard, Reporting: service})
	key := newID()
	body := fmt.Sprintf(`{"definition_id":%q,"expected_definition_version":%d,"idempotency_key":%q}`, definition.ID, definition.CurrentVersion, key)
	type outcome struct {
		status int
		body   []byte
	}
	outcomes := make(chan outcome, 2)
	for range 2 {
		go func() {
			response := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", performer, nil, body)
			outcomes <- outcome{response.Code, response.Body.Bytes()}
		}()
	}
	a, b := <-outcomes, <-outcomes
	if a.status != http.StatusCreated || b.status != http.StatusCreated {
		t.Fatalf("statuses %d/%d: %s / %s", a.status, b.status, a.body, b.body)
	}
	var first, second reporting.ReportRun
	if err := json.Unmarshal(a.body, &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.body, &second); err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.ID != second.ID || first.RequestedByRef != performer {
		t.Fatalf("receipts: %+v %+v", first, second)
	}
	var receipts, events int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM report_run_receipts WHERE tenant_id=$1::uuid AND idempotency_key=$2::uuid),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$3::uuid AND event_type='ReportRunQueued')`, tenant, key, first.ID).Scan(&receipts, &events); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || events != 1 {
		t.Fatalf("receipts=%d events=%d", receipts, events)
	}
	replayed := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", performer, nil, body)
	var recovered reporting.ReportRun
	if replayed.Code != http.StatusCreated || json.Unmarshal(replayed.Body.Bytes(), &recovered) != nil || recovered.ID != first.ID {
		t.Fatalf("recovered receipt: %d %s", replayed.Code, replayed.Body.String())
	}
	changed := fmt.Sprintf(`{"definition_id":%q,"expected_definition_version":%d,"idempotency_key":%q,"parameters":{"start_date":"2026-01-01","end_date":"2026-02-01"}}`, definition.ID, definition.CurrentVersion, key)
	conflict := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", performer, nil, changed)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("changed request: %d %s", conflict.Code, conflict.Body.String())
	}
	denied := reportingRequest(handler, http.MethodPost, "/api/v1/reports/runs", maker, nil, body)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("wrong actor: %d %s", denied.Code, denied.Body.String())
	}
}
