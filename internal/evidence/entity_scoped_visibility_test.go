package evidence

import (
	"context"
	"testing"
	"time"
)

func TestActorRequestQueuesFilterLegalEntityBeforeLimit(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, []Request{
		{ID: "other-first", TenantID: "bank", LegalEntityID: "entity-2", SubjectType: "PROGRAM", SubjectID: "p2", Deadline: now.Add(time.Minute), Status: RequestReady, CreatedBy: "creator", Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "actor", State: RecipientStateAssigned}},
		{ID: "expected", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "p1", Deadline: now.Add(2 * time.Minute), Status: RequestReady, CreatedBy: "creator", Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "actor", State: RecipientStateAssigned}},
	})
	service := NewService(repo, nil)
	visible, err := service.ListVisibleRequestsForEntity(context.Background(), ActorRequestScope{TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "actor"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != "expected" {
		t.Fatalf("entity filtering happened after limit: %#v", visible)
	}
}

func TestReviewerQueueIncludesOnlySubmittedAssignedReviews(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, []Request{
		{ID: "submitted", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "p1", Deadline: now.Add(time.Hour), Status: RequestSubmitted, CreatedBy: "owner", KnownFacts: map[string]string{"reviewer": "auditor"}, Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "respondent", State: RecipientStateAssigned}},
		{ID: "open", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "p1", Deadline: now.Add(2 * time.Hour), Status: RequestReady, CreatedBy: "owner", KnownFacts: map[string]string{"reviewer": "auditor"}, Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "respondent", State: RecipientStateAssigned}},
		{ID: "other-reviewer", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "p1", Deadline: now.Add(3 * time.Hour), Status: RequestSubmitted, CreatedBy: "owner", KnownFacts: map[string]string{"reviewer": "auditor-other"}, Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "respondent", State: RecipientStateAssigned}},
	})
	service := NewService(repo, nil)
	values, err := service.ListManageableRequestsForEntity(context.Background(), ActorRequestScope{TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "auditor"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "submitted" {
		t.Fatalf("reviewer queue = %#v", values)
	}
}
