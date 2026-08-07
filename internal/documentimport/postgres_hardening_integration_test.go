//go:build postgres && postgresintegration

package documentimport

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresImportReceiptSurvivesRestartAndReviewIsAtomic(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}

	const (
		tenantID    = "98888888-8888-7888-8888-888888888881"
		principalID = "98888888-8888-7888-8888-888888888882"
	)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'document-hardening-test','Document Hardening Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Document Reviewer')`, principalID, tenantID); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	storeA, err := evidence.NewLocalObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	serviceA := NewService(repo, storeA)
	created, err := serviceA.Import(ctx, ImportInput{
		TenantID: "document-hardening-test", FileName: "requirements.txt", MediaType: "text/plain",
		Purpose: "Review requirements", SourceType: "REGULATORY", CreatedBy: principalID,
	}, strings.NewReader("Records\n\nThe bank must retain records for five years. The bank shall review privileged access annually."))
	if err != nil {
		t.Fatal(err)
	}
	if created.ExtractionStatus != ExtractionPending || created.AnalysisStatus != AnalysisPending || created.Version != 1 {
		t.Fatalf("upload did not return durable pending receipt: %#v", created)
	}
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='DOCUMENT_IMPORT' AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, created.ID, EventDocumentProcessingRequested).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("expected one transactional processing request, got %d", queued)
	}
	before, err := serviceA.List(ctx, "document-hardening-test", 20)
	if err != nil || len(before) != 1 || before[0].ExtractionStatus != ExtractionPending {
		t.Fatalf("pending summary unavailable: %#v err=%v", before, err)
	}
	summaryJSON, _ := json.Marshal(before[0])
	if bytes.Contains(summaryJSON, []byte(`"sections":`)) || bytes.Contains(summaryJSON, []byte(`"proposals":`)) {
		t.Fatalf("summary leaked source/proposal bodies: %s", summaryJSON)
	}

	// A new service/store instance simulates a worker restart. The durable file,
	// row and outbox request are sufficient to continue processing.
	storeB, err := evidence.NewLocalObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	serviceB := NewService(NewPostgresRepository(pool), storeB)
	event := workflowruntime.OutboxEvent{TenantID: "document-hardening-test", AggregateType: "DOCUMENT_IMPORT", AggregateID: created.ID, EventType: EventDocumentProcessingRequested}
	if err := serviceB.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	processed, err := serviceB.Get(ctx, "document-hardening-test", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processed.ExtractionStatus != ExtractionExtracted || processed.AnalysisStatus != AnalysisReviewRequired || processed.Version != 2 || processed.ProcessedAt == nil {
		t.Fatalf("restart processing did not converge: %#v", processed)
	}
	if processed.SectionsTotal < 1 || processed.ProposalsTotal < 2 || len(processed.Proposals) < 2 {
		t.Fatalf("processing completeness metadata missing: %#v", processed)
	}

	if err := serviceB.Publish(ctx, event); err != nil {
		t.Fatalf("duplicate delivery must be idempotent: %v", err)
	}
	duplicate, err := serviceB.Get(ctx, "document-hardening-test", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Version != processed.Version {
		t.Fatalf("duplicate processing changed version: before=%d after=%d", processed.Version, duplicate.Version)
	}

	firstID := processed.Proposals[0].ID
	secondID := processed.Proposals[1].ID
	reviewed, err := serviceB.ReviewProposal(ctx, ReviewInput{
		TenantID: "document-hardening-test", DocumentID: processed.ID, ProposalID: firstID,
		ReviewerID: principalID, Status: ProposalAccepted, ExpectedVersion: processed.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Version != processed.Version+1 {
		t.Fatalf("review did not advance document version once: %d", reviewed.Version)
	}
	statuses := map[string]ProposalStatus{}
	for _, proposal := range reviewed.Proposals {
		statuses[proposal.ID] = proposal.Status
	}
	if statuses[firstID] != ProposalAccepted || statuses[secondID] != ProposalPending {
		t.Fatalf("review changed the wrong proposal: %#v", statuses)
	}
	if _, err := serviceB.ReviewProposal(ctx, ReviewInput{
		TenantID: "document-hardening-test", DocumentID: processed.ID, ProposalID: secondID,
		ReviewerID: principalID, Status: ProposalRejected, ExpectedVersion: processed.Version,
	}); err != ErrVersionConflict {
		t.Fatalf("stale review should conflict, got %v", err)
	}

	after, err := serviceB.List(ctx, "document-hardening-test", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].PendingProposalCount != len(reviewed.Proposals)-1 || after[0].ReviewedProposalCount != 1 {
		t.Fatalf("review summary counts are wrong: %#v", after)
	}
}
