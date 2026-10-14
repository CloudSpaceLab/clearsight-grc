package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"
)

const (
	linkTenant = "tenant-a"
	linkEntity = "entity-a"
	linkMatter = "00000000-0000-4000-8000-000000000001"
	linkForm   = "00000000-0000-4000-8000-000000000002"
)

func seededSubjectLinkRepository() *MemoryRepository {
	repo := NewMemoryRepository()
	repo.forms[revisionKey(linkTenant, linkForm, 3)] = FormTemplate{
		ID: linkForm, TenantID: linkTenant, LegalEntityID: linkEntity, Name: "Vendor evidence check",
		Lifecycle: Lifecycle{Version: 3, Status: LifecycleActive, IsCurrent: true},
	}
	return repo
}

func sampleSubjectLink(id string) FormSubjectLink {
	return FormSubjectLink{
		ID: id, TenantID: linkTenant, LegalEntityID: linkEntity,
		SubjectType: FormSubjectMatter, SubjectID: linkMatter,
		FormTemplateID: linkForm, FormTemplateVersion: 3,
		LinkedBy: "principal-a", LinkedAt: time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC),
		Version: 1,
	}
}

func TestFormSubjectLinkMemoryKeepsOneActiveAssociationAndHistory(t *testing.T) {
	ctx := context.Background()
	repo := seededSubjectLinkRepository()
	first, err := repo.CreateFormSubjectLink(ctx, sampleSubjectLink("00000000-0000-7000-8000-000000000001"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateFormSubjectLink(ctx, sampleSubjectLink("00000000-0000-7000-8000-000000000002"))
	if err != nil || second.ID != first.ID {
		t.Fatalf("retry created another link: %+v, %v", second, err)
	}
	if len(repo.subjectLinkEvents) != 1 {
		t.Fatal("idempotent retry appended an event")
	}
	page, err := repo.ListFormSubjectLinks(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, "", 10)
	if err != nil || len(page.Items) != 1 || page.Items[0].FormName != "Vendor evidence check" {
		t.Fatalf("current links mismatch: %+v, %v", page, err)
	}
	other, err := repo.ListFormSubjectLinks(ctx, linkTenant, "another-entity", FormSubjectMatter, linkMatter, "", 10)
	if err != nil || len(other.Items) != 0 {
		t.Fatalf("link leaked across entities: %+v, %v", other, err)
	}
	_, err = repo.EndFormSubjectLink(ctx, linkTenant, "another-entity", FormSubjectMatter, linkMatter, first.ID, 1, "principal-a", time.Now(), "Wrong entity")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity retirement: %v", err)
	}
	now := time.Now().UTC()
	ended, err := repo.EndFormSubjectLink(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, first.ID, 1, "principal-a", now, "No longer relevant")
	if err != nil || ended.EndedAt == nil || ended.Version != 2 {
		t.Fatalf("end failed: %+v, %v", ended, err)
	}
	_, err = repo.EndFormSubjectLink(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, first.ID, 1, "principal-a", now, "Replay")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale unlink unexpectedly accepted: %v", err)
	}
	page, _ = repo.ListFormSubjectLinks(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, "", 10)
	if len(page.Items) != 0 || len(repo.subjectLinkEvents) != 2 {
		t.Fatalf("link history/revocation invalid: page=%+v events=%d", page, len(repo.subjectLinkEvents))
	}
	third, err := repo.CreateFormSubjectLink(ctx, sampleSubjectLink("00000000-0000-7000-8000-000000000003"))
	if err != nil || third.ID == first.ID {
		t.Fatalf("re-link after retirement failed: %+v, %v", third, err)
	}
}

func TestFormSubjectLinkMemoryDeniesCrossEntityOrInactiveRevision(t *testing.T) {
	repo := seededSubjectLinkRepository()
	ctx := context.Background()
	value := sampleSubjectLink("00000000-0000-7000-8000-000000000004")
	value.LegalEntityID = "another-entity"
	if _, err := repo.CreateFormSubjectLink(ctx, value); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross entity accepted: %v", err)
	}
	value = sampleSubjectLink("00000000-0000-7000-8000-000000000005")
	value.FormTemplateVersion = 2
	if _, err := repo.CreateFormSubjectLink(ctx, value); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing revision accepted: %v", err)
	}
	repo.forms[revisionKey(linkTenant, linkForm, 3)] = FormTemplate{
		ID: linkForm, TenantID: linkTenant, LegalEntityID: linkEntity,
		Lifecycle: Lifecycle{Version: 3, Status: LifecycleRetired, IsCurrent: false},
	}
	if _, err := repo.CreateFormSubjectLink(ctx, sampleSubjectLink(value.ID)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired revision accepted: %v", err)
	}
}

func TestFormSubjectLinkMemoryPagesWithoutFalseTotals(t *testing.T) {
	repo := seededSubjectLinkRepository()
	ctx := context.Background()
	ids := []string{"00000000-0000-7000-8000-000000000003", "00000000-0000-7000-8000-000000000002", "00000000-0000-7000-8000-000000000001"}
	for index, id := range ids {
		value := sampleSubjectLink(id)
		value.FormTemplateID = []string{"00000000-0000-4000-8000-000000000012", "00000000-0000-4000-8000-000000000011", linkForm}[index]
		repo.forms[revisionKey(linkTenant, value.FormTemplateID, 3)] = FormTemplate{
			ID: value.FormTemplateID, TenantID: linkTenant, LegalEntityID: linkEntity, Name: value.FormTemplateID,
			Lifecycle: Lifecycle{Version: 3, Status: LifecycleActive, IsCurrent: true},
		}
		if _, err := repo.CreateFormSubjectLink(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListFormSubjectLinks(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, "", 2)
	if err != nil || len(first.Items) != 2 || first.NextCursor != ids[1] {
		t.Fatalf("first page: %+v, %v", first, err)
	}
	second, err := repo.ListFormSubjectLinks(ctx, linkTenant, linkEntity, FormSubjectMatter, linkMatter, first.NextCursor, 2)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("next page: %+v, %v", second, err)
	}
}
