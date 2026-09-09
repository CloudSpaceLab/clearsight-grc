package monitoring

import (
	"context"
	"errors"
	"testing"
)

func TestExactFormCodeLookupScopesAndRejectsDuplicateIdentities(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	actor := Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "maker"}
	base := FormTemplate{ID: "first", TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProgramID: "program", Code: "EXACT", Lifecycle: Lifecycle{Version: 1, Status: LifecycleDraft}}
	if _, err := repo.CreateFormRevision(ctx, base); err != nil {
		t.Fatal(err)
	}
	later := base
	later.Version = 2
	if _, err := repo.CreateFormRevision(ctx, later); err != nil {
		t.Fatal(err)
	}
	got, err := svc.LatestFormByCode(ctx, actor, "program", "EXACT")
	if err != nil || got.ID != "first" || got.Version != 2 {
		t.Fatalf("latest exact form: %+v %v", got, err)
	}
	for _, probe := range []struct {
		actor         Actor
		program, code string
	}{
		{Actor{TenantID: "other", LegalEntityID: "entity", PrincipalID: "maker"}, "program", "EXACT"},
		{Actor{TenantID: "tenant", LegalEntityID: "other", PrincipalID: "maker"}, "program", "EXACT"},
		{actor, "other", "EXACT"}, {actor, "program", "EXACT-OTHER"},
	} {
		if _, err := svc.LatestFormByCode(ctx, probe.actor, probe.program, probe.code); !errors.Is(err, ErrNotFound) {
			t.Fatalf("scope or code escaped: %v", err)
		}
	}
	if _, err := svc.LatestFormByCode(ctx, Actor{}, "program", "EXACT"); err == nil {
		t.Fatal("missing actor accepted")
	}
	duplicate := base
	duplicate.ID = "second"
	if _, err := repo.CreateFormRevision(ctx, duplicate); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.LatestFormByCode(ctx, actor, "program", "EXACT"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate identities were accepted: %v", err)
	}
}
