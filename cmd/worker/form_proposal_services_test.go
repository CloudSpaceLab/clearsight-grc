//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type proposalWorkerDocuments struct {
	document documentimport.Document
}

func (r proposalWorkerDocuments) Get(_ context.Context, tenantID, documentID string) (documentimport.Document, error) {
	if r.document.TenantID != tenantID || r.document.ID != documentID {
		return documentimport.Document{}, documentimport.ErrNotFound
	}
	return r.document, nil
}

func TestFormProposalGenerationPublisherConsumesOnlyExactEvent(t *testing.T) {
	document := documentimport.Document{
		ID: "doc-1", TenantID: "bank-a", LegalEntityID: "entity-a", Version: 4,
		SHA256: strings.Repeat("b", 64), ExtractionStatus: documentimport.ExtractionExtracted,
		Elements: []documentimport.ExtractedElement{{
			Kind: documentimport.ElementFormControl,
			Anchor: documentimport.SourceAnchor{Paragraph: "paragraph-1"},
			Control: &documentimport.FormControl{Kind: "TEXT", Label: "Legal name"},
		}},
	}
	store := monitoring.NewMemoryFormProposalStore()
	now := time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)
	created, err := store.Create(context.Background(), monitoring.FormTemplateProposal{
		ID: "proposal-1", TenantID: "bank-a", LegalEntityID: "entity-a",
		SourceKind: monitoring.FormProposalSourceDocument,
		SourceDocumentID: document.ID, SourceDocumentVersion: document.Version, SourceSHA256: document.SHA256,
		Status: monitoring.FormProposalGenerating, CreatedBy: "maker-a", CreatedAt: now, UpdatedAt: now, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	publisher := &formProposalGenerationPublisher{
		service: monitoring.NewFormProposalService(store, proposalWorkerDocuments{document: document}, nil),
	}

	if err := publisher.Publish(context.Background(), workflowruntime.OutboxEvent{EventType: "SOMETHING_ELSE"}); err != nil {
		t.Fatalf("unrelated event should be ignored: %v", err)
	}
	unchanged, err := store.Get(context.Background(), created.TenantID, created.LegalEntityID, created.ID)
	if err != nil || unchanged.Status != monitoring.FormProposalGenerating {
		t.Fatalf("unrelated event changed proposal: %#v %v", unchanged, err)
	}

	payload, err := json.Marshal(formProposalGenerationPayload{ProposalID: created.ID, LegalEntityID: created.LegalEntityID, SourceDocument: document.ID, SourceVersion: document.Version})
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), workflowruntime.OutboxEvent{
		TenantID: created.TenantID, AggregateType: formProposalAggregateType, AggregateID: created.ID,
		EventType: monitoring.EventFormProposalGenerationRequested, Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := store.Get(context.Background(), created.TenantID, created.LegalEntityID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Status != monitoring.FormProposalReviewRequired || generated.Version != 2 || len(generated.FieldChanges) != 1 {
		t.Fatalf("proposal was not generated: %#v", generated)
	}
}

func TestFormProposalGenerationPublisherRejectsMismatchedAggregate(t *testing.T) {
	publisher := &formProposalGenerationPublisher{}
	payload, err := json.Marshal(formProposalGenerationPayload{ProposalID: "proposal-1", LegalEntityID: "entity-a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), workflowruntime.OutboxEvent{
		TenantID: "bank-a", AggregateType: formProposalAggregateType, AggregateID: "proposal-2",
		EventType: monitoring.EventFormProposalGenerationRequested, Payload: payload,
	}); err == nil {
		t.Fatal("expected mismatched aggregate to fail")
	}
}
