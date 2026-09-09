package monitoring

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// Interleave a real competing store mutation immediately before completion.
type competingGenerationStore struct {
	*MemoryFormProposalStore
	beforeComplete func(context.Context, FormTemplateProposal, int64)
	readError      error
	completed      bool
}

func (s *competingGenerationStore) CompleteGeneration(ctx context.Context, value FormTemplateProposal, version int64) (FormTemplateProposal, error) {
	s.beforeComplete(ctx, value, version)
	s.completed = true
	return FormTemplateProposal{}, ErrConflict
}

func (s *competingGenerationStore) Get(ctx context.Context, tenant, entity, id string) (FormTemplateProposal, error) {
	if s.completed && s.readError != nil {
		return FormTemplateProposal{}, s.readError
	}
	return s.MemoryFormProposalStore.Get(ctx, tenant, entity, id)
}

func TestFormProposalGenerationCompletionConflictRecovery(t *testing.T) {
	readFailure := errors.New("proposal receipt read unavailable")
	for _, scenario := range []struct {
		name      string
		status    FormProposalStatus
		readError error
		wantError error
	}{
		{"competitor completed", FormProposalReviewRequired, nil, nil},
		{"still generating", FormProposalGenerating, nil, ErrConflict},
		{"receipt unavailable", FormProposalReviewRequired, readFailure, readFailure},
		{"competitor failed", FormProposalFailed, nil, nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx := context.Background()
			document := proposalSourceDocument()
			store := &competingGenerationStore{MemoryFormProposalStore: NewMemoryFormProposalStore(), readError: scenario.readError}
			proposal := FormTemplateProposal{ID: "018f0000-0000-7000-8000-000000000003", TenantID: document.TenantID, LegalEntityID: document.LegalEntityID, SourceKind: FormProposalSourceDocument, SourceDocumentID: document.ID, SourceDocumentVersion: document.Version, SourceSHA256: document.SHA256, Status: FormProposalGenerating, CreatedBy: "maker-a", CreatedAt: testProposalTime(), UpdatedAt: testProposalTime(), Version: 1}
			if _, err := store.Create(ctx, proposal); err != nil {
				t.Fatal(err)
			}
			store.beforeComplete = func(ctx context.Context, generated FormTemplateProposal, version int64) {
				var err error
				switch scenario.status {
				case FormProposalReviewRequired:
					generated.FieldChanges = generated.FieldChanges[:1]
					_, err = store.MemoryFormProposalStore.CompleteGeneration(ctx, generated, version)
				case FormProposalFailed:
					_, err = store.FailGeneration(ctx, proposal.TenantID, proposal.LegalEntityID, proposal.ID, version, "COMPETING_FAILURE", "The source could not be read. Retry the import.", testProposalTime())
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			service := NewFormProposalService(store, &proposalDocumentStub{document: document}, nil)
			got, err := service.Generate(ctx, " bank-a ", " entity-a ", " "+proposal.ID+" ")
			if !errors.Is(err, scenario.wantError) {
				t.Fatalf("generation error = %v, want %v", err, scenario.wantError)
			}
			if scenario.wantError != nil {
				if got.ID != "" {
					t.Fatalf("unconfirmed receipt returned: %#v", got)
				}
				return
			}
			stored, err := store.MemoryFormProposalStore.Get(ctx, proposal.TenantID, proposal.LegalEntityID, proposal.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, stored) || got.Status != scenario.status {
				t.Fatalf("returned receipt differs from stored outcome: got=%#v stored=%#v", got, stored)
			}
		})
	}
}
