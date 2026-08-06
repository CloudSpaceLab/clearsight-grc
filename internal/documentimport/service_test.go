package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestImportExtractsAndAnchorsTextProposals(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	value, err := service.Import(context.Background(), ImportInput{
		TenantID: "bank-demo", LegalEntityID: "bank-ng", FileName: "circular.md", MediaType: "text/markdown",
		Purpose: "Assess a new circular", SourceType: "REGULATORY", CreatedBy: "reviewer-1",
	}, strings.NewReader("# Customer records\n\nThe bank must retain customer records for five years.\n\nThe control owner shall review access quarterly."))
	if err != nil {
		t.Fatal(err)
	}
	if value.ExtractionStatus != ExtractionExtracted || value.AnalysisStatus != AnalysisReviewRequired {
		t.Fatalf("unexpected states: extraction=%s analysis=%s", value.ExtractionStatus, value.AnalysisStatus)
	}
	if len(value.Sections) == 0 || len(value.Proposals) < 2 {
		t.Fatalf("expected extracted sections and proposals: %#v", value)
	}
	for _, proposal := range value.Proposals {
		if proposal.Anchor.SectionID == "" || proposal.Anchor.Quote == "" || proposal.Status != ProposalPending {
			t.Fatalf("proposal lacks a pending source anchor: %#v", proposal)
		}
	}
	if value.ArtifactStatus != "STORED_UNSCANNED" {
		t.Fatalf("expected explicit unscanned state, got %s", value.ArtifactStatus)
	}
}

func TestReviewProposalUsesOptimisticVersion(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	value, err := service.Import(context.Background(), ImportInput{TenantID: "bank-demo", FileName: "policy.txt", MediaType: "text/plain", CreatedBy: "reviewer-1"}, strings.NewReader("The institution must review access every quarter."))
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Proposals) != 1 {
		t.Fatalf("expected one proposal, got %d", len(value.Proposals))
	}
	updated, err := service.ReviewProposal(context.Background(), ReviewInput{TenantID: value.TenantID, DocumentID: value.ID, ProposalID: value.Proposals[0].ID, ReviewerID: "reviewer-2", Status: ProposalAccepted, ExpectedVersion: value.Version})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Proposals[0].Status != ProposalAccepted || updated.Version != value.Version+1 {
		t.Fatalf("review was not persisted: %#v", updated)
	}
	_, err = service.ReviewProposal(context.Background(), ReviewInput{TenantID: value.TenantID, DocumentID: value.ID, ProposalID: value.Proposals[0].ID, ReviewerID: "reviewer-3", Status: ProposalRejected, ExpectedVersion: value.Version})
	if err != ErrVersionConflict {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestDOCXExtractionReadsParagraphs(t *testing.T) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	file, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="urn:test"><w:body><w:p><w:r><w:t>Records policy</w:t></w:r></w:p><w:p><w:r><w:t>The bank shall retain records.</w:t></w:r></w:p></w:body></w:document>`))
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	result := Extract("policy.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", buffer.Bytes())
	if result.Status != ExtractionExtracted || len(result.Sections) == 0 {
		t.Fatalf("DOCX was not extracted: %#v", result)
	}
	if !strings.Contains(result.Sections[0].Text, "shall retain") {
		t.Fatalf("expected paragraph text, got %#v", result.Sections)
	}
}

func TestPDFIsStoredWithoutPretendingItWasAnalyzed(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	value, err := service.Import(context.Background(), ImportInput{TenantID: "bank-demo", FileName: "notice.pdf", MediaType: "application/pdf", CreatedBy: "reviewer-1"}, bytes.NewReader([]byte("%PDF-1.7 placeholder")))
	if err != nil {
		t.Fatal(err)
	}
	if value.ExtractionStatus != ExtractionUnsupported || value.AnalysisStatus != AnalysisUnavailable || len(value.Proposals) != 0 {
		t.Fatalf("unsupported PDF was overstated: %#v", value)
	}
}
