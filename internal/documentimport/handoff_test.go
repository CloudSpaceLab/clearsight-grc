package documentimport

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcceptedProposalCreatesStableAwaitingReviewHandoff(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	createdAt := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	_, err := repo.Create(ctx, Document{
		ID: "11111111-1111-4111-8111-111111111111", TenantID: "tenant-a", Version: 1,
		CreatedAt: createdAt, UpdatedAt: createdAt,
		Proposals: []Proposal{{
			ID: "22222222-2222-4222-8222-222222222222", Kind: "REQUIREMENT_CANDIDATE",
			Title: "Possible requirement", Statement: "The bank shall review access quarterly.", Status: ProposalPending,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	acceptedAt := createdAt.Add(time.Minute)
	input := ReviewInput{
		TenantID: "tenant-a", DocumentID: "11111111-1111-4111-8111-111111111111",
		ProposalID: "22222222-2222-4222-8222-222222222222", ReviewerID: "intake-a",
		Status: ProposalAccepted, ExpectedVersion: 1,
	}
	accepted, err := repo.ReviewProposal(ctx, input, acceptedAt)
	if err != nil {
		t.Fatal(err)
	}
	handoff := accepted.Proposals[0].Handoff
	if handoff == nil {
		t.Fatal("accepted proposal did not create a handoff")
	}
	if handoff.Status != HandoffAwaitingReview || handoff.Version != 1 {
		t.Fatalf("unexpected handoff state: %#v", handoff)
	}
	if handoff.IntakePrincipalID != "intake-a" || handoff.DraftStatement != "The bank shall review access quarterly." {
		t.Fatalf("unexpected handoff content: %#v", handoff)
	}
	if handoff.ID != proposalHandoffID(input.DocumentID, input.ProposalID) {
		t.Fatalf("handoff identity is not deterministic: %q", handoff.ID)
	}

	replayed, err := repo.ReviewProposal(ctx, input, acceptedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("exact acceptance replay should be idempotent: %v", err)
	}
	if replayed.Version != accepted.Version || replayed.Proposals[0].Handoff.Version != 1 {
		t.Fatalf("replay mutated accepted state: document=%d handoff=%d", replayed.Version, replayed.Proposals[0].Handoff.Version)
	}
}

func TestRejectedProposalDoesNotCreateHandoff(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	_, _ = repo.Create(ctx, Document{
		ID: "11111111-1111-4111-8111-111111111111", TenantID: "tenant-a", Version: 1, CreatedAt: now, UpdatedAt: now,
		Proposals: []Proposal{{ID: "22222222-2222-4222-8222-222222222222", Status: ProposalPending}},
	})
	updated, err := repo.ReviewProposal(ctx, ReviewInput{
		TenantID: "tenant-a", DocumentID: "11111111-1111-4111-8111-111111111111",
		ProposalID: "22222222-2222-4222-8222-222222222222", ReviewerID: "intake-a",
		Status: ProposalRejected, ExpectedVersion: 1,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Proposals[0].Handoff != nil {
		t.Fatal("rejected extraction proposal must not create governed work")
	}
}

func TestProposalHandoffEnforcesReviewerAndAuthorizerSegregation(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	_, _ = repo.Create(ctx, Document{
		ID: "11111111-1111-4111-8111-111111111111", TenantID: "tenant-a", Version: 1, CreatedAt: now, UpdatedAt: now,
		Proposals: []Proposal{{
			ID: "22222222-2222-4222-8222-222222222222", Title: "Access review", Statement: "Review access quarterly.", Status: ProposalPending,
		}},
	})
	accepted, err := service.ReviewProposal(ctx, ReviewInput{
		TenantID: "tenant-a", DocumentID: "11111111-1111-4111-8111-111111111111",
		ProposalID: "22222222-2222-4222-8222-222222222222", ReviewerID: "intake-a",
		Status: ProposalAccepted, ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ReviewProposalHandoff(ctx, HandoffReviewInput{
		TenantID: "tenant-a", DocumentID: accepted.ID, ProposalID: accepted.Proposals[0].ID,
		ActorID: "intake-a", Action: HandoffReviewReject,
		ExpectedDocumentVersion: accepted.Version, ExpectedHandoffVersion: 1, Note: "cannot self-review",
	})
	if !errors.Is(err, ErrHandoffSegregation) {
		t.Fatalf("intake self-review should fail segregation, got %v", err)
	}

	now = now.Add(time.Minute)
	reviewed, err := service.ReviewProposalHandoff(ctx, HandoffReviewInput{
		TenantID: "tenant-a", DocumentID: accepted.ID, ProposalID: accepted.Proposals[0].ID,
		ActorID: "reviewer-b", Action: HandoffReviewSubmit,
		ExpectedDocumentVersion: accepted.Version, ExpectedHandoffVersion: 1,
		Title: "Quarterly access review", Statement: "The bank shall review privileged access quarterly.",
		TargetType: ConversionRequirement, TargetProgramID: "33333333-3333-4333-8333-333333333333", TargetProgramVersion: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	handoff := reviewed.Proposals[0].Handoff
	if handoff.Status != HandoffAwaitingAuthorization || handoff.ReviewerPrincipalID != "reviewer-b" || handoff.Version != 2 {
		t.Fatalf("unexpected reviewed handoff: %#v", handoff)
	}

	_, err = service.AuthorizeProposalHandoff(ctx, HandoffAuthorizationInput{
		TenantID: "tenant-a", DocumentID: reviewed.ID, ProposalID: reviewed.Proposals[0].ID,
		ActorID: "reviewer-b", Action: HandoffAuthorizeApprove,
		ExpectedDocumentVersion: reviewed.Version, ExpectedHandoffVersion: 2,
		Note: "approve", ResultObjectType: "REQUIREMENT", ResultObjectID: "44444444-4444-4444-8444-444444444444",
	})
	if !errors.Is(err, ErrHandoffSegregation) {
		t.Fatalf("reviewer self-authorization should fail segregation, got %v", err)
	}

	now = now.Add(time.Minute)
	approved, err := service.AuthorizeProposalHandoff(ctx, HandoffAuthorizationInput{
		TenantID: "tenant-a", DocumentID: reviewed.ID, ProposalID: reviewed.Proposals[0].ID,
		ActorID: "authorizer-c", Action: HandoffAuthorizeApprove,
		ExpectedDocumentVersion: reviewed.Version, ExpectedHandoffVersion: 2,
		Note: "approved after independent review", ResultObjectType: "REQUIREMENT", ResultObjectID: "44444444-4444-4444-8444-444444444444",
	})
	if err != nil {
		t.Fatal(err)
	}
	handoff = approved.Proposals[0].Handoff
	if handoff.Status != HandoffApproved || handoff.AuthorizerPrincipalID != "authorizer-c" || handoff.ResultObjectID == "" || handoff.Version != 3 {
		t.Fatalf("unexpected approved handoff: %#v", handoff)
	}
}

func TestProposalHandoffReturnAndRejectRemainInactive(t *testing.T) {
	for _, action := range []HandoffReviewAction{HandoffReviewReturn, HandoffReviewReject} {
		t.Run(string(action), func(t *testing.T) {
			ctx := context.Background()
			repo := NewMemoryRepository()
			now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
			_, _ = repo.Create(ctx, Document{
				ID: "11111111-1111-4111-8111-111111111111", TenantID: "tenant-a", Version: 1, CreatedAt: now, UpdatedAt: now,
				Proposals: []Proposal{{ID: "22222222-2222-4222-8222-222222222222", Title: "Candidate", Statement: "Candidate statement", Status: ProposalPending}},
			})
			accepted, err := repo.ReviewProposal(ctx, ReviewInput{
				TenantID: "tenant-a", DocumentID: "11111111-1111-4111-8111-111111111111",
				ProposalID: "22222222-2222-4222-8222-222222222222", ReviewerID: "intake-a", Status: ProposalAccepted, ExpectedVersion: 1,
			}, now)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := repo.ReviewProposalHandoff(ctx, HandoffReviewInput{
				TenantID: "tenant-a", DocumentID: accepted.ID, ProposalID: accepted.Proposals[0].ID, ActorID: "reviewer-b", Action: action,
				ExpectedDocumentVersion: accepted.Version, ExpectedHandoffVersion: 1, Note: "review disposition",
			}, now.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			if updated.Proposals[0].Handoff.ResultObjectID != "" || updated.Proposals[0].Handoff.Status == HandoffApproved {
				t.Fatalf("%s produced an active object: %#v", action, updated.Proposals[0].Handoff)
			}
		})
	}
}
