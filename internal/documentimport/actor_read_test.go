package documentimport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type actorScopedReadFake struct {
	Repository
	principal, tenant, entity string
	limit                     int
	expiresAt                 time.Time
}

func (r *actorScopedReadFake) ListForActor(_ context.Context, tenant, entity, principal string, limit int, _ time.Time, expiresAt time.Time) ([]DocumentSummary, error) {
	r.tenant, r.entity, r.principal, r.limit, r.expiresAt = tenant, entity, principal, limit, expiresAt
	return []DocumentSummary{{ID: "one"}}, nil
}
func (r *actorScopedReadFake) GetForActor(_ context.Context, tenant, entity, principal, id string, _ time.Time, expiresAt time.Time) (Document, error) {
	r.tenant, r.entity, r.principal, r.expiresAt = tenant, entity, principal, expiresAt
	return Document{ID: id}, nil
}
func TestActorVisibleReadsRequireVerifiedActorAndActorScopedStore(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	repo := &actorScopedReadFake{}
	service := NewService(repo, nil)
	ctx := identity.WithActor(context.Background(), actor)
	values, err := service.ListVisibleForActor(ctx, 25)
	if err != nil || len(values) != 1 || repo.principal != "person" || repo.tenant != "bank" || repo.entity != "entity" || repo.limit != 25 || !repo.expiresAt.Equal(actor.ExpiresAt) {
		t.Fatalf("list=%+v err=%v repo=%+v", values, err, repo)
	}
	if _, err = service.GetVisibleForActor(ctx, "doc"); err != nil || repo.principal != "person" || !repo.expiresAt.Equal(actor.ExpiresAt) {
		t.Fatalf("get: %v", err)
	}
	if _, err = service.ListVisibleForActor(context.Background(), 25); err == nil {
		t.Fatal("anonymous read accepted")
	}
	actor.LegalEntityID = "*"
	if _, err = service.GetVisibleForActor(identity.WithActor(context.Background(), actor), "doc"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wildcard entity: %v", err)
	}
	actor.LegalEntityID = "entity"
	actor.ExpiresAt = now.Add(-time.Second)
	if _, err = service.GetVisibleForActor(identity.WithActor(context.Background(), actor), "doc"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired: %v", err)
	}
	fallback := NewService(NewMemoryRepository(), nil)
	actor.ExpiresAt = now.Add(time.Hour)
	if _, err = fallback.GetVisibleForActor(identity.WithActor(context.Background(), actor), "doc"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing actor adapter: %v", err)
	}
}
