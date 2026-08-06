package documentimport

import (
	"context"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestUnscannedAnalysisCanBeDisabled(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	service.Configure(1<<20, false)
	value, err := service.Import(context.Background(), ImportInput{
		TenantID: "bank-demo", FileName: "notice.txt", MediaType: "text/plain", CreatedBy: "reviewer-1",
	}, strings.NewReader("The bank must retain records for five years."))
	if err != nil {
		t.Fatal(err)
	}
	if value.ExtractionStatus != ExtractionExtracted || len(value.Sections) == 0 {
		t.Fatalf("expected extraction to remain available: %#v", value)
	}
	if value.AnalysisStatus != AnalysisUnavailable || len(value.Proposals) != 0 {
		t.Fatalf("unscanned analysis was not blocked: %#v", value)
	}
}

func TestReviewedProposalCannotBeReviewedAgain(t *testing.T) {
	service := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	value, err := service.Import(context.Background(), ImportInput{
		TenantID: "bank-demo", FileName: "notice.txt", MediaType: "text/plain", CreatedBy: "reviewer-1",
	}, strings.NewReader("The bank must retain records for five years."))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.ReviewProposal(context.Background(), ReviewInput{
		TenantID: value.TenantID, DocumentID: value.ID, ProposalID: value.Proposals[0].ID,
		ReviewerID: "reviewer-2", Status: ProposalAccepted, ExpectedVersion: value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ReviewProposal(context.Background(), ReviewInput{
		TenantID: updated.TenantID, DocumentID: updated.ID, ProposalID: updated.Proposals[0].ID,
		ReviewerID: "reviewer-3", Status: ProposalRejected, ExpectedVersion: updated.Version,
	})
	if err != ErrInvalidReview {
		t.Fatalf("expected reviewed proposal to be immutable, got %v", err)
	}
}
