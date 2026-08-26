package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

type scopedTestRepository struct {
	*MemoryRepository
	scope      SubjectScope
	resolveErr error
	readers    map[string]bool
	readErr    error
}

func (r *scopedTestRepository) ResolveSubjectScope(_ context.Context, tenant, subjectType, subjectID string) (SubjectScope, error) {
	if r.resolveErr != nil {
		return SubjectScope{}, r.resolveErr
	}
	if r.scope.TenantID != tenant || r.scope.SubjectType != subjectType || r.scope.SubjectID != subjectID {
		return SubjectScope{}, ErrSubjectUnsupported
	}
	return r.scope, nil
}

func (r *scopedTestRepository) CanReadSubject(_ context.Context, _, principalID, _, _ string) (bool, error) {
	if r.readErr != nil {
		return false, r.readErr
	}
	return r.readers[principalID], nil
}

func TestCreateRequestRequiresExactCreatorSubjectAccessAndCanonicalEntity(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	base := func() CreateRequestInput {
		return CreateRequestInput{
			TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
			Title: "Provide evidence", Purpose: "Complete the current review.", WhyYou: "You own this response.",
			Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
			Recipient:        RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "recipient"},
			EstimatedMinutes: 5, Deadline: now.Add(time.Hour),
			Fields: []Field{{ID: "answer", Label: "Answer", Type: "text", Required: true}}, CreatedBy: "creator",
		}
	}

	t.Run("creator denied", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		if _, err := service.CreateRequest(ctx, base()); !errors.Is(err, ErrSubjectAccessDenied) {
			t.Fatalf("creator without subject access created request: %v", err)
		}
	})

	t.Run("unsupported subject fails closed", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), resolveErr: ErrSubjectUnsupported, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		input := base()
		input.SubjectType = "VENDOR"
		if _, err := service.CreateRequest(ctx, input); !errors.Is(err, ErrSubjectUnsupported) {
			t.Fatalf("unsupported subject did not fail closed: %v", err)
		}
	})

	t.Run("entity mismatch fails closed", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-2", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		if _, err := service.CreateRequest(ctx, base()); !errors.Is(err, ErrSubjectScopeMismatch) {
			t.Fatalf("cross-entity subject request was accepted: %v", err)
		}
	})

	t.Run("missing verified entity fails closed", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		input := base()
		input.LegalEntityID = ""
		if _, err := service.CreateRequest(ctx, input); !errors.Is(err, ErrSubjectScopeMismatch) {
			t.Fatalf("request without verified legal entity was accepted: %v", err)
		}
	})

	t.Run("canonical entity is persisted", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		created, err := service.CreateRequest(ctx, base())
		if err != nil {
			t.Fatal(err)
		}
		stored, err := repo.GetRequest(ctx, "bank", created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.LegalEntityID != "entity-1" {
			t.Fatalf("stored legal entity = %q", stored.LegalEntityID)
		}
	})

	t.Run("wrong recipient mutation rejects entity mismatch", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		created, err := service.CreateRequest(ctx, base())
		if err != nil {
			t.Fatal(err)
		}
		_, err = service.DeclareWrongRecipient(ctx, DeclareWrongRecipientInput{
			TenantID: "bank", LegalEntityID: "entity-2", RequestID: created.ID, ActorPrincipalID: "recipient",
			Reason: "This request belongs to another owner.", ExpectedVersion: created.Version,
		})
		if !errors.Is(err, ErrSubjectScopeMismatch) {
			t.Fatalf("cross-entity wrong-recipient mutation was accepted: %v", err)
		}
	})

	t.Run("internal submission rejects entity mismatch", func(t *testing.T) {
		repo := &scopedTestRepository{MemoryRepository: NewMemoryRepository(nil, nil), scope: SubjectScope{TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1"}, readers: map[string]bool{"creator": true, "recipient": true}}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		created, err := service.CreateRequest(ctx, base())
		if err != nil {
			t.Fatal(err)
		}
		_, err = service.Submit(ctx, Submission{
			TenantID: "bank", LegalEntityID: "entity-2", RequestID: created.ID, SubmittedBy: "recipient", Channel: "INTERNAL",
			Answers: map[string]string{"answer": "Current evidence"}, ExpectedVersion: created.Version,
		})
		if !errors.Is(err, ErrSubjectScopeMismatch) {
			t.Fatalf("cross-entity internal submission was accepted: %v", err)
		}
	})
}
