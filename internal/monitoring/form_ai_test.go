package monitoring

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type formAIClientStub struct {
	calls   int
	request FormAIClientRequest
	result  FormAIClientResult
	err     error
}

func (s *formAIClientStub) Propose(_ context.Context, request FormAIClientRequest) (FormAIClientResult, error) {
	s.calls++
	s.request = request
	if s.err != nil {
		return FormAIClientResult{}, s.err
	}
	result := s.result
	result.Provenance.SnapshotSHA256 = request.SnapshotSHA256
	if request.Source != nil {
		result.Provenance.SourceDocumentSHA256 = request.Source.SHA256
		result.Provenance.SourceElementRefs = make([]string, 0, len(request.Source.Elements))
		for _, element := range request.Source.Elements {
			result.Provenance.SourceElementRefs = append(result.Provenance.SourceElementRefs, element.Ref)
		}
	}
	return result, nil
}

func TestAIFormProposalUsesExactSelectedSourceAndIsIdempotent(t *testing.T) {
	forms := libraryService(t, NewMemoryRepository(), "maker-a")
	document := proposalSourceDocument()
	docs := &proposalDocumentStub{document: document}
	field := formcontract.Field{ID: "legal_name", SectionID: formcontract.DefaultSectionID, Label: "Legal name", Type: formcontract.TypeShortText}
	contract, err := formcontract.Normalize(formcontract.Contract{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringNone,
		Sections:     []formcontract.Section{{ID: formcontract.DefaultSectionID, Title: "General"}},
		Fields:       []formcontract.Field{field},
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &formAIClientStub{result: FormAIClientResult{
		Contract:     contract,
		FieldChanges: []documentimport.FormFieldChange{{ID: "ai-change-1", Kind: "ADD_FIELD", Field: contract.Fields[0], Confidence: 0.95}},
		Provenance:   FormAIProvenance{WorkloadID: "forms-authoring", ModelAlias: "authoring", PromptVersion: formAIPromptVersionDefault, ValidationResults: []string{"LOCAL_CONTRACT_NORMALIZATION"}},
	}}
	service := NewFormProposalService(NewMemoryFormProposalStore(), docs, forms)
	service.ConfigureAIClient(client)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000101", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	input := RequestAIFormProposalInput{
		Objective:        "Draft only the legal-name question from the selected source.",
		SourceDocumentID: document.ID, ExpectedSourceDocumentVersion: document.Version,
		SourceElementRefs: []string{"control-1"},
	}

	proposal, err := service.RequestAI(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != FormProposalReviewRequired || proposal.SourceKind != FormProposalSourceAI || !validProposalSHA256(proposal.SourceSHA256) {
		t.Fatalf("unexpected AI proposal: %#v", proposal)
	}
	if client.calls != 1 || client.request.TenantID != "bank-a" || client.request.LegalEntityID != "entity-a" || client.request.Source == nil || len(client.request.Source.Elements) != 1 || client.request.Source.Elements[0].Ref != "control-1" {
		t.Fatalf("AI request was not source bounded: %#v", client.request)
	}
	if proposal.Provenance.AI == nil || proposal.Provenance.AI.SourceDocumentSHA256 != document.SHA256 || len(proposal.Provenance.AI.SourceElementRefs) != 1 {
		t.Fatalf("AI provenance lost the exact source: %#v", proposal.Provenance)
	}

	again, err := service.RequestAI(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != proposal.ID || client.calls != 1 {
		t.Fatalf("identical AI snapshot was not idempotent: first=%s second=%s calls=%d", proposal.ID, again.ID, client.calls)
	}
}

func TestAIFormProposalProviderFailureLeavesBaseRevisionUnchanged(t *testing.T) {
	repo := NewMemoryRepository()
	forms := libraryService(t, repo, "maker-a")
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	base, err := forms.CreateLibraryForm(ctx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	client := &formAIClientStub{err: errors.Join(ErrFormAIUnavailable, errors.New("provider offline"))}
	service := NewFormProposalService(NewMemoryFormProposalStore(), nil, forms)
	service.ConfigureAIClient(client)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000102", nil }

	proposal, err := service.RequestAIRevision(ctx, base.ID, base.Version, RequestAIFormProposalInput{Objective: "Add a short control-owner question."})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != FormProposalFailed || proposal.FailureCode != "AI_GATEWAY_UNAVAILABLE" {
		t.Fatalf("provider failure was not retained as recoverable proposal state: %#v", proposal)
	}
	if _, err := forms.GetLibraryForm(ctx, base.ID, base.Version+1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("provider failure mutated a draft revision: %v", err)
	}
}

func TestAIFormProposalSelectiveAcceptanceAppliesOnlyChosenBaseDiff(t *testing.T) {
	repo := NewMemoryRepository()
	forms := libraryService(t, repo, "maker-a")
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	base, err := forms.CreateLibraryForm(ctx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	updated := cloneTemplateField(base.Fields[0])
	updated.Description = "Clarified by governed AI for author review."
	added := formcontract.Field{ID: "ai_optional_note", SectionID: base.Sections[0].ID, Label: "Optional note", Type: formcontract.TypeShortText}
	client := &formAIClientStub{result: FormAIClientResult{
		Contract: contractFromFormTemplate(base),
		FieldChanges: []documentimport.FormFieldChange{
			{ID: "ai-update", Kind: "UPDATE_FIELD", Field: updated, Confidence: 0.9},
			{ID: "ai-add", Kind: "ADD_FIELD", Field: added, Confidence: 0.7},
		},
		Provenance: FormAIProvenance{WorkloadID: "forms-authoring", ModelAlias: "authoring", PromptVersion: formAIPromptVersionDefault, ValidationResults: []string{"EXACT_BASE_REVISION"}},
	}}
	service := NewFormProposalService(NewMemoryFormProposalStore(), nil, forms)
	service.ConfigureAIClient(client)
	service.newID = func() (string, error) { return "018f0000-0000-7000-8000-000000000103", nil }

	proposal, err := service.RequestAIRevision(ctx, base.ID, base.Version, RequestAIFormProposalInput{Objective: "Clarify the first question and suggest one optional note."})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := service.Accept(ctx, proposal.ID, AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: []string{"ai-update"}})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != FormProposalAccepted || accepted.ResultTemplateVersion != base.Version+1 {
		t.Fatalf("unexpected accepted AI proposal: %#v", accepted)
	}
	draft, err := forms.GetLibraryForm(ctx, base.ID, base.Version+1)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Fields[0].Description != updated.Description {
		t.Fatalf("selected update was not applied: %#v", draft.Fields[0])
	}
	for _, field := range draft.Fields {
		if strings.EqualFold(field.ID, added.ID) {
			t.Fatalf("unselected AI addition leaked into draft: %#v", draft.Fields)
		}
	}
}
