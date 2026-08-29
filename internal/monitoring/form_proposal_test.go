package monitoring

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type proposalDocumentStub struct {
	document documentimport.Document
}

func (s *proposalDocumentStub) Get(_ context.Context, tenantID, documentID string) (documentimport.Document, error) {
	if s.document.TenantID != tenantID || s.document.ID != documentID {
		return documentimport.Document{}, documentimport.ErrNotFound
	}
	return s.document, nil
}

func proposalSourceDocument() documentimport.Document {
	return documentimport.Document{
		ID: "document-a", TenantID: "bank-a", LegalEntityID: "entity-a",
		FileName: "vendor-questionnaire.docx", Purpose: "Collect vendor assurance evidence.",
		SHA256: strings.Repeat("a", 64), ExtractionStatus: documentimport.ExtractionExtracted,
		ExtractionMethod: "DOCX_STREAM_V3", ParserVersion: "DOCX_STREAM_V3", AdapterVersion: "DOCX_FORM_CONTROLS_V1",
		Elements: []documentimport.ExtractedElement{
			{Ref: "control-1", Kind: documentimport.ElementFormControl, Control: &documentimport.FormControl{Kind: "TEXT", Label: "Legal name"}, Anchor: documentimport.SourceAnchor{Paragraph: "p1"}},
			{Ref: "control-2", Kind: documentimport.ElementFormControl, Control: &documentimport.FormControl{Kind: "DROPDOWN", Label: "Country", Options: []string{"Nigeria", "Ghana"}}, Anchor: documentimport.SourceAnchor{Paragraph: "p2"}},
		},
		Version: 3,
	}
}

func TestFormTemplateProposalRequestGeneratesReviewableDeterministicProposal(t *testing.T) {
	forms := libraryService(t, NewMemoryRepository(), "maker-a")
	store := NewMemoryFormProposalStore()
	docs := &proposalDocumentStub{document: proposalSourceDocument()}
	service := NewFormProposalService(store, docs, forms)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000001", nil }

	created, err := service.RequestFromDocument(formActorContext("bank-a", "entity-a", "maker-a"), docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != FormProposalReviewRequired || created.Version != 2 {
		t.Fatalf("proposal lifecycle = %s v%d", created.Status, created.Version)
	}
	if created.SourceDocumentID != docs.document.ID || created.SourceDocumentVersion != docs.document.Version || created.SourceSHA256 != docs.document.SHA256 {
		t.Fatalf("source snapshot = %#v", created)
	}
	if created.ProposedContract.ScoringMode != formcontract.ScoringNone || len(created.FieldChanges) != 2 {
		t.Fatalf("generated contract = %#v, changes = %#v", created.ProposedContract, created.FieldChanges)
	}
	for _, change := range created.FieldChanges {
		if change.Field.Required || change.Field.Scoring != nil {
			t.Fatalf("proposal guessed governed semantics: %#v", change.Field)
		}
	}

	again, err := service.RequestFromDocument(formActorContext("bank-a", "entity-a", "maker-a"), docs.document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != created.ID || again.Version != created.Version {
		t.Fatalf("same exact source created a duplicate proposal: first=%#v second=%#v", created, again)
	}
}

func TestFormTemplateProposalRejectsPartialExtraction(t *testing.T) {
	forms := libraryService(t, NewMemoryRepository(), "maker-a")
	document := proposalSourceDocument()
	document.ExtractionStatus = documentimport.ExtractionPartial
	service := NewFormProposalService(NewMemoryFormProposalStore(), &proposalDocumentStub{document: document}, forms)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000002", nil }

	_, err := service.RequestFromDocument(formActorContext("bank-a", "entity-a", "maker-a"), document.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: document.Version})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("partial extraction error = %v, want invalid", err)
	}
}

func TestFormTemplateProposalGenerationFailsClosedWhenSourceChanges(t *testing.T) {
	document := proposalSourceDocument()
	docs := &proposalDocumentStub{document: document}
	store := NewMemoryFormProposalStore()
	now := formActorContext("bank-a", "entity-a", "maker-a")
	_ = now
	proposal := FormTemplateProposal{
		ID: "018f0000-0000-7000-8000-000000000003", TenantID: "bank-a", LegalEntityID: "entity-a",
		SourceKind: FormProposalSourceDocument, SourceDocumentID: document.ID, SourceDocumentVersion: document.Version, SourceSHA256: document.SHA256,
		Status: FormProposalGenerating, CreatedBy: "maker-a", CreatedAt: document.CreatedAt, UpdatedAt: document.CreatedAt, Version: 1,
	}
	if proposal.CreatedAt.IsZero() {
		proposal.CreatedAt = testProposalTime()
		proposal.UpdatedAt = proposal.CreatedAt
	}
	if _, err := store.Create(context.Background(), proposal); err != nil {
		t.Fatal(err)
	}
	docs.document.Version++
	service := NewFormProposalService(store, docs, nil)
	failed, err := service.Generate(context.Background(), "bank-a", "entity-a", proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != FormProposalFailed || failed.FailureCode != "SOURCE_CHANGED" {
		t.Fatalf("changed source proposal = %#v", failed)
	}
}

func TestFormTemplateProposalAcceptanceAppliesOnlySelectedChangesToNormalDraftRevision(t *testing.T) {
	repo := NewMemoryRepository()
	forms := libraryService(t, repo, "maker-a")
	forms.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000010", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	base, err := forms.CreateLibraryForm(ctx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}

	docs := &proposalDocumentStub{document: proposalSourceDocument()}
	store := NewMemoryFormProposalStore()
	service := NewFormProposalService(store, docs, forms)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000011", nil }
	proposal, err := service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{
		ExpectedDocumentVersion: docs.document.Version, BaseTemplateID: base.ID, BaseTemplateVersion: base.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := proposal.FieldChanges[0]
	accepted, err := service.Accept(ctx, proposal.ID, AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: []string{selected.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != FormProposalAccepted || accepted.ResultTemplateID != base.ID || accepted.ResultTemplateVersion != 2 {
		t.Fatalf("accepted proposal = %#v", accepted)
	}
	draft, err := forms.GetLibraryForm(ctx, base.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != LifecycleDraft || draft.IsCurrent || len(draft.Fields) != len(base.Fields)+1 {
		t.Fatalf("resulting draft = %#v", draft)
	}
	if !containsTemplateField(draft.Fields, selected.Field.ID) || containsTemplateField(draft.Fields, proposal.FieldChanges[1].Field.ID) {
		t.Fatalf("selective acceptance failed: %#v", draft.Fields)
	}
}

func TestFormTemplateProposalAcceptanceRejectsChangedSourceBeforeDraftMutation(t *testing.T) {
	repo := NewMemoryRepository()
	forms := libraryService(t, repo, "maker-a")
	forms.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000020", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	base, err := forms.CreateLibraryForm(ctx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	docs := &proposalDocumentStub{document: proposalSourceDocument()}
	service := NewFormProposalService(NewMemoryFormProposalStore(), docs, forms)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000021", nil }
	proposal, err := service.RequestFromDocument(ctx, docs.document.ID, RequestDocumentFormProposalInput{
		ExpectedDocumentVersion: docs.document.Version, BaseTemplateID: base.ID, BaseTemplateVersion: base.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	docs.document.Version++
	_, err = service.Accept(ctx, proposal.ID, AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: []string{proposal.FieldChanges[0].ID}})
	if !errors.Is(err, ErrFormProposalSourceChanged) {
		t.Fatalf("changed source acceptance error = %v", err)
	}
	if _, err := forms.GetLibraryForm(ctx, base.ID, 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("changed source created a draft: %v", err)
	}
}

func containsTemplateField(fields []TemplateField, id string) bool {
	for _, field := range fields {
		if field.ID == id {
			return true
		}
	}
	return false
}

func testProposalTime() time.Time {
	return time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
}
