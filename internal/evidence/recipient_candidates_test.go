package evidence

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRecipientCandidatesRequireExactRequesterScopeAndSubjectAccess(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	request := Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		AudienceType: "INTERNAL", Status: RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
		Recipient: Recipient{Type: RecipientInternalPrincipal, PrincipalID: "eligible", State: RecipientStateAssigned},
	}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{
		{PrincipalID: "requester", DisplayName: "Reni Requester", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "eligible", DisplayName: "Ada Eligible", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "blocked", DisplayName: "Ben Blocked", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{}},
		{PrincipalID: "inactive", DisplayName: "Cara Inactive", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: false, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "team", DisplayName: "Controls Team", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "TEAM", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "other-entity", DisplayName: "Dayo Other", TenantID: "bank", LegalEntityIDs: []string{"entity-2"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "other-tenant", DisplayName: "Efe Other", TenantID: "other-bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
	})
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	if allowed, err := repo.CanReadSubject(context.Background(), "bank", "blocked", "PROGRAM", "program-1"); err != nil || allowed {
		t.Fatalf("memory directory allowed a principal without exact subject access: allowed=%v err=%v", allowed, err)
	}

	values, err := service.ListRecipientCandidates(context.Background(), ActorRequestScope{
		TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "requester",
	}, request.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].PrincipalID != "eligible" || values[0].DisplayName != "Ada Eligible" || values[1].PrincipalID != "requester" {
		t.Fatalf("unexpected candidates: %#v", values)
	}

	loaded, err := service.GetRequestForEntity(context.Background(), "bank", "entity-1", request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Recipient.DisplayName != "Ada Eligible" {
		t.Fatalf("current recipient display name = %q", loaded.Recipient.DisplayName)
	}

	_, err = service.ListRecipientCandidates(context.Background(), ActorRequestScope{
		TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "eligible",
	}, request.ID, 50)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-requester candidate read error = %v, want not found", err)
	}

	_, err = service.ListRecipientCandidates(context.Background(), ActorRequestScope{
		TenantID: "bank", LegalEntityID: "entity-2", ActorPrincipalID: "requester",
	}, request.ID, 50)
	if !errors.Is(err, ErrSubjectScopeMismatch) {
		t.Fatalf("cross-entity candidate read error = %v", err)
	}

	_, err = repo.ListRecipientCandidates(context.Background(), "bank", "entity-1", request.ID, "eligible", 50)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("repository accepted a non-requester candidate read: %v", err)
	}
}

func TestRecipientCandidateLimitIsCappedAtFifty(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	request := Request{ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "MATTER", SubjectID: "matter-1", AudienceType: "INTERNAL", Status: RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour)}
	candidates := make([]RecipientCandidate, 0, 60)
	candidates = append(candidates, RecipientCandidate{PrincipalID: "requester", DisplayName: "Requester", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"MATTER:matter-1": true}})
	for index := 0; index < 60; index++ {
		candidates = append(candidates, RecipientCandidate{
			PrincipalID: fmt.Sprintf("person-%02d", index), DisplayName: fmt.Sprintf("Person %02d", index),
			TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true,
			ReadableSubjects: map[string]bool{"MATTER:matter-1": true},
		})
	}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, candidates)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	values, err := service.ListRecipientCandidates(context.Background(), ActorRequestScope{
		TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "requester",
	}, request.ID, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 50 {
		t.Fatalf("candidate count = %d, want 50", len(values))
	}
}

func TestRecipientCandidateSearchFiltersBeforeLimitAndReportsMore(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	request := Request{ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "MATTER", SubjectID: "matter-1", AudienceType: "INTERNAL", Status: RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour)}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{
		{PrincipalID: "requester", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"MATTER:matter-1": true}},
		{PrincipalID: "person-a", DisplayName: "Ada Okafor", ContextLabel: "Operations manager", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"MATTER:matter-1": true}},
		{PrincipalID: "person-b", DisplayName: "Ada Okafor", ContextLabel: "Risk manager", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"MATTER:matter-1": true}},
		{PrincipalID: "person-c", DisplayName: "Bola James", ContextLabel: "Risk analyst", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"MATTER:matter-1": true}},
	})
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	page, err := service.SearchRecipientCandidates(context.Background(), ActorRequestScope{
		TenantID: "bank", LegalEntityID: "entity-1", ActorPrincipalID: "requester",
	}, request.ID, RecipientCandidateSearch{Query: "risk", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].PrincipalID != "person-b" || page.Items[0].ContextLabel != "Risk manager" || !page.HasMore {
		t.Fatalf("search was not filtered before the bounded page: %#v", page)
	}
}

func TestMemoryRecipientCandidatesRecheckRequesterActiveStatus(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	request := Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		AudienceType: "INTERNAL", Status: RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
	}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{
		{PrincipalID: "requester", TenantID: "bank", Kind: "PERSON", Active: false, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "eligible", DisplayName: "Ada Eligible", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
	})

	_, err := repo.ListRecipientCandidates(context.Background(), "bank", "entity-1", request.ID, "requester", 50)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("inactive requester retained candidate access: %v", err)
	}
}

func TestMemoryRecipientCandidatesRejectRequesterWithoutCurrentEntityMembership(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	request := Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		AudienceType: "INTERNAL", Status: RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
	}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, []RecipientCandidate{
		{PrincipalID: "requester", TenantID: "bank", LegalEntityIDs: []string{"entity-2"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "eligible", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
	})

	_, err := repo.ListRecipientCandidates(context.Background(), "bank", "entity-1", request.ID, "requester", 50)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong-entity requester retained candidate access: %v", err)
	}
}

func TestDemoInternalRecipientHasSafeDisplayName(t *testing.T) {
	requests := DemoRequests()
	if len(requests) == 0 || requests[0].Recipient.DisplayName == "" {
		t.Fatalf("demo request exposes only an internal recipient ID: %#v", requests)
	}
}
