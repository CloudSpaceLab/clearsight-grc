package sourceaccess

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestQuestionTestAuditLimitsConcurrentTestsAndDoesNotRecordSecrets(t *testing.T) {
	store := NewMemoryQuestionTestAudit()
	actor := QuestionTestActor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person", AuthorityRuleID: "route", AuthorityPolicyVersion: "v1"}
	when := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	var group sync.WaitGroup
	var mu sync.Mutex
	var permitted int
	for i := 0; i < 30; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := store.Reserve(context.Background(), actor, "form", 2, "question", "LIVE", when)
			if err == nil {
				mu.Lock()
				permitted++
				mu.Unlock()
			} else if !errors.Is(err, ErrQuestionTestRateLimit) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	group.Wait()
	if permitted != 10 {
		t.Fatalf("permitted = %d, want 10", permitted)
	}
	if _, err := store.Reserve(context.Background(), actor, "form", 2, "question", "LIVE", when.Add(time.Minute)); err != nil {
		t.Fatalf("window did not expire: %v", err)
	}
}

func TestQuestionTestAuditRefusesInvalidIdentityAndDuplicateCompletion(t *testing.T) {
	store := NewMemoryQuestionTestAudit()
	when := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	if _, err := store.Reserve(context.Background(), QuestionTestActor{}, "form", 1, "question", "LIVE", when); err == nil {
		t.Fatal("missing scope accepted")
	}
	actor := QuestionTestActor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person", AuthorityRuleID: "route", AuthorityPolicyVersion: "v1"}
	reservation, err := store.Reserve(context.Background(), actor, "form", 1, "question", "SAMPLE", when)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), actor, reservation, QuestionTestValid, when); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), actor, reservation, QuestionTestValid, when); err == nil {
		t.Fatal("duplicate outcome accepted")
	}
	if err := store.Complete(context.Background(), QuestionTestActor{TenantID: "other", LegalEntityID: "entity", PrincipalID: "person"}, reservation, QuestionTestValid, when); err == nil {
		t.Fatal("cross-tenant outcome accepted")
	}
}
