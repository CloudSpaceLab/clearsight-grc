package monitoring

import (
	"context"
	"fmt"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

type competingGenerationStore struct{ *MemoryFormProposalStore }

func (s competingGenerationStore) CompleteGeneration(ctx context.Context, value FormTemplateProposal, version int64) (FormTemplateProposal, error) {
	if _, err := s.MemoryFormProposalStore.CompleteGeneration(ctx, value, version); err != nil {
		return FormTemplateProposal{}, err
	}
	return FormTemplateProposal{}, ErrConflict
}

func TestFormProposalGenerationUsesCompetingWorkerResult(t *testing.T) {
	d := proposalSourceDocument()
	forms := libraryService(t, NewMemoryRepository(), "maker-a")
	service := NewFormProposalService(competingGenerationStore{NewMemoryFormProposalStore()}, &proposalDocumentStub{document: d}, forms)
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	p, err := service.RequestFromDocument(ctx, d.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: d.Version})
	if err != nil || p.Status != FormProposalReviewRequired {
		t.Fatalf("worker completion was not recovered: %s %v", p.Status, err)
	}
}

func TestFindingFollowUpCreatesTwoIsolatedDraftsAndReusesRetries(t *testing.T) {
	d := proposalSourceDocument()
	d.FileName = "register.xlsx"
	d.Elements = nil
	d.Tabular = &documentimport.TabularMetadata{Format: documentimport.TabularXLSX, Resources: []documentimport.TabularResource{{Name: "Sheet 1", Fields: []documentimport.TabularField{{Name: "S/N"}, {Name: "SERVICE PROVIDER"}, {Name: "SERVICES OFFERED"}, {Name: "FINDINGS"}, {Name: "RECOMMENDATIONS"}, {Name: "DATE OF ASSESSMENT"}}}}}
	for i := 1; i <= 2; i++ {
		d.Sections = append(d.Sections, documentimport.Section{Sheet: "Sheet 1", RowStart: i + 1, RowEnd: i + 1, Text: fmt.Sprintf("Column 1: %d\nColumn 2: xxxxx\nColumn 3: Service %d\nColumn 4: Finding %d\nColumn 5: Recommendation %d\nColumn 6: 6th February 2026", i, i, i, i)})
	}
	forms := libraryService(t, NewMemoryRepository(), "maker-a")
	service := NewFormProposalService(NewMemoryFormProposalStore(), &proposalDocumentStub{document: d}, forms)
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	p, err := service.RequestFromDocument(ctx, d.ID, RequestDocumentFormProposalInput{ExpectedDocumentVersion: d.Version})
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string][]string{}
	for _, c := range p.FieldChanges {
		groups[c.GroupID] = append(groups[c.GroupID], c.ID)
	}
	seen := map[string]bool{}
	for _, ids := range groups {
		input := AcceptFormProposalInput{ExpectedVersion: p.Version, ChangeIDs: ids, AssessmentConfirmed: true}
		a, err := service.Accept(ctx, p.ID, input)
		if err != nil {
			t.Fatal(err)
		}
		if seen[a.ResultTemplateID] {
			t.Fatal("assessments share a draft")
		}
		seen[a.ResultTemplateID] = true
		draft, err := forms.GetLibraryForm(ctx, a.ResultTemplateID, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(draft.Fields) != 5 || len(draft.Sections) != 1 || draft.Status != LifecycleDraft {
			t.Fatalf("unexpected draft %+v", draft)
		}
		retry, err := service.Accept(ctx, p.ID, input)
		if err != nil || retry.ResultTemplateID != a.ResultTemplateID {
			t.Fatalf("retry %+v %v", retry, err)
		}
	}
	if len(seen) != 2 {
		t.Fatal("missing assessment")
	}
}

func TestFindingSelectionRequiresConfirmedCompleteSingleGroup(t *testing.T) {
	p := FormTemplateProposal{Provenance: FormProposalProvenance{FormProposalProvenance: documentimport.FormProposalProvenance{ProposalVersion: documentimport.FindingFollowUpVersion}}, FieldChanges: []documentimport.FormFieldChange{{ID: "a", GroupID: "one"}, {ID: "b", GroupID: "one"}, {ID: "c", GroupID: "two"}}}
	for _, ids := range [][]string{{"a"}, {"a", "b", "c"}, {"unknown"}} {
		if validateFindingFollowUpSelection(p, ids, true) == nil {
			t.Fatalf("accepted %v", ids)
		}
	}
	if validateFindingFollowUpSelection(p, []string{"a", "b"}, false) == nil {
		t.Fatal("accepted unconfirmed grouping")
	}
	if err := validateFindingFollowUpSelection(p, []string{"a", "b"}, true); err != nil {
		t.Fatal(err)
	}
}
