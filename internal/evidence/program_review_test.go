package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type programReviewTestStore struct {
	*MemoryDistributionStore
	called bool
}

func (s *programReviewTestStore) listPendingProgramReviews(_ context.Context, actor identity.Actor, query ProgramReviewQuery) (ProgramReviewPage, error) {
	s.called = true
	return ProgramReviewPage{Coverage: "PROGRAM_RESPONSE_REVIEWS", Items: []CompletedResponseSummary{{ID: "revision"}}}, nil
}

func TestProgramReviewListRequiresVerifiedPersonAndExactScope(t *testing.T) {
	store := &programReviewTestStore{MemoryDistributionStore: NewMemoryDistributionStore(NewMemoryRepository(nil, nil), nil, nil)}
	service := NewDistributionService(store).WithResponseReviewAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) (string, error) {
		return "policy:rule", nil
	})
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	for _, tc := range []struct {
		name  string
		actor identity.Actor
		query ProgramReviewQuery
	}{
		{"other tenant", actor, ProgramReviewQuery{TenantID: "other", LegalEntityID: "entity", Limit: 20}},
		{"other entity", actor, ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "other", Limit: 20}},
		{"wildcard entity", actor, ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "*", Limit: 20}},
		{"invalid limit", actor, ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 101}},
		{"service identity", identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "system", Kind: "SERVICE", ExpiresAt: time.Now().Add(time.Hour)}, ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 20}},
		{"expired identity", identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(-time.Second)}, ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store.called = false
			_, err := service.ListPendingProgramReviews(identity.WithActor(context.Background(), tc.actor), tc.query)
			if err == nil || store.called {
				t.Fatalf("err=%v store called=%v", err, store.called)
			}
		})
	}
	_, err := service.ListPendingProgramReviews(context.Background(), ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 20})
	if err == nil || store.called {
		t.Fatalf("missing identity err=%v called=%v", err, store.called)
	}
	page, err := service.ListPendingProgramReviews(identity.WithActor(context.Background(), actor), ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 20})
	if err != nil || !store.called || page.Coverage != "PROGRAM_RESPONSE_REVIEWS" {
		t.Fatalf("page=%+v called=%v err=%v", page, store.called, err)
	}
}

func TestProgramReviewListFailsClosedWhenStoreDoesNotSupportIt(t *testing.T) {
	service := NewDistributionService(NewMemoryDistributionStore(NewMemoryRepository(nil, nil), nil, nil))
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	_, err := service.ListPendingProgramReviews(identity.WithActor(context.Background(), actor), ProgramReviewQuery{TenantID: "tenant", LegalEntityID: "entity", Limit: 20})
	if !errors.Is(err, ErrProgramReviewUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestProgramReviewCursorIsBoundToReviewerAndEntity(t *testing.T) {
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	last := CompletedResponseSummary{ID: "revision", CompletedAt: time.Now().UTC()}
	cursor := makeProgramReviewCursor(actor, "entity", last)
	decoded, err := parseProgramReviewCursor(cursor, actor, "entity")
	if err != nil || decoded.ID != last.ID || !decoded.CompletedAt.Equal(last.CompletedAt) {
		t.Fatalf("cursor=%+v err=%v", decoded, err)
	}
	other := actor
	other.PrincipalID = "another"
	if _, err := parseProgramReviewCursor(cursor, other, "entity"); err == nil {
		t.Fatal("cursor accepted for another reviewer")
	}
	if _, err := parseProgramReviewCursor(cursor, actor, "other"); err == nil {
		t.Fatal("cursor accepted for another entity")
	}
	if _, err := parseProgramReviewCursor("not-a-cursor", actor, "entity"); err == nil {
		t.Fatal("malformed cursor accepted")
	}
}
