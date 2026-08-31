package monitoring

import (
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestFormLibraryReturnsLatestRevisionAndActiveVersionWithKeysetPaging(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	forms := []FormTemplate{
		libraryForm("form-a", "entity-a", "program-a", "ACCESS", LifecycleActive, 3, true, now.Add(-4*time.Hour)),
		libraryForm("form-a", "entity-a", "program-a", "ACCESS", LifecycleDraft, 4, false, now),
		libraryForm("form-b", "entity-a", "", "VENDOR", LifecycleActive, 2, true, now.Add(-time.Hour)),
		libraryForm("form-c", "entity-b", "", "OTHER", LifecycleActive, 1, true, now.Add(time.Hour)),
	}
	for _, form := range forms {
		if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
			t.Fatalf("create %s/%d: %v", form.ID, form.Version, err)
		}
	}
	// Historical unscoped rows remain stored but never enter the canonical library.
	repo.forms[revisionKey("bank-a", "legacy", 1)] = libraryForm("legacy", "", "", "LEGACY", LifecycleActive, 1, true, now.Add(2*time.Hour))

	first, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{TenantID: "bank-a", LegalEntityID: "entity-a", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].Template.ID != "form-a" || first.Items[0].Template.Version != 4 || first.Items[0].ActiveVersion != 3 || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{TenantID: "bank-a", LegalEntityID: "entity-a", Cursor: first.NextCursor, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Template.ID != "form-b" || second.Items[0].ActiveVersion != 2 || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestFormLibraryFiltersBeforeLimit(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	forms := []FormTemplate{
		libraryForm("form-a", "entity-a", "program-a", "ACCESS", LifecycleActive, 1, true, now),
		libraryForm("form-b", "entity-a", "program-b", "VENDOR", LifecycleActive, 1, true, now.Add(-time.Minute)),
		libraryForm("form-c", "entity-a", "program-b", "RESILIENCE", LifecyclePaused, 1, true, now.Add(-2*time.Minute)),
	}
	forms[1].Name = "Vendor due diligence"
	forms[1].OwnerPrincipalID = "owner-a"
	forms[1].ApprovedUses = []string{"VENDOR_DUE_DILIGENCE"}
	forms[1].Tags = []string{"third-party"}
	for _, form := range forms {
		if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
			t.Fatal(err)
		}
	}

	page, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{
		TenantID: "bank-a", LegalEntityID: "entity-a", Search: "vendor", ProgramID: "program-b",
		OwnerPrincipalID: "owner-a", Use: "VENDOR_DUE_DILIGENCE", Tag: "third-party", Status: LifecycleActive, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Template.ID != "form-b" || page.NextCursor != "" {
		t.Fatalf("filters were not applied before limit: %#v", page)
	}
}

func TestFormLibraryOldestUpdatedSortUsesMatchingKeysetDirection(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	for _, form := range []FormTemplate{
		libraryForm("newest", "entity-a", "", "NEW", LifecycleDraft, 1, false, now),
		libraryForm("oldest", "entity-a", "", "OLD", LifecycleDraft, 1, false, now.Add(-time.Hour)),
	} {
		if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{TenantID: "bank-a", LegalEntityID: "entity-a", Sort: FormLibraryUpdatedAsc, Limit: 1})
	if err != nil || len(first.Items) != 1 || first.Items[0].Template.ID != "oldest" || first.NextCursor == "" {
		t.Fatalf("oldest first page = %#v, err = %v", first, err)
	}
	second, err := repo.ListFormLibrary(t.Context(), FormLibraryFilter{TenantID: "bank-a", LegalEntityID: "entity-a", Sort: FormLibraryUpdatedAsc, Cursor: first.NextCursor, Limit: 1})
	if err != nil || len(second.Items) != 1 || second.Items[0].Template.ID != "newest" {
		t.Fatalf("oldest second page = %#v, err = %v", second, err)
	}
}

func TestFormLibrarySavedViewsArePrincipalScoped(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 27, 11, 0, 0, 0, time.UTC)
	view := SavedFormView{
		ID: "view-a", TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "person-a", Name: "Vendor forms",
		Filter: FormLibraryFilter{Search: "vendor", Status: LifecycleActive, Limit: 25}, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := repo.SaveFormView(t.Context(), view); err != nil {
		t.Fatal(err)
	}
	views, err := repo.ListSavedFormViews(t.Context(), "bank-a", "entity-a", "person-a")
	if err != nil || len(views) != 1 || views[0].Filter.Search != "vendor" {
		t.Fatalf("saved views = %#v, err = %v", views, err)
	}
	other, err := repo.ListSavedFormViews(t.Context(), "bank-a", "entity-a", "person-b")
	if err != nil || len(other) != 0 {
		t.Fatalf("cross-principal views = %#v, err = %v", other, err)
	}
	if err := repo.DeleteSavedFormView(t.Context(), "bank-a", "entity-a", "person-b", "view-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-principal delete error = %v", err)
	}
}

func libraryForm(id, entityID, programID, code string, status LifecycleStatus, version int64, current bool, updated time.Time) FormTemplate {
	return FormTemplate{
		ID: id, TenantID: "bank-a", LegalEntityID: entityID, ProgramID: programID, Code: code, Name: code + " review", Purpose: "Collect evidence for " + code + ".",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields:       []TemplateField{{ID: "answer", SectionID: "general", Label: "Answer", Type: formcontract.TypeShortText}},
		Lifecycle:    Lifecycle{Status: status, IsCurrent: current, Version: version, CreatedBy: "maker-a", CreatedAt: updated, UpdatedAt: updated},
	}
}
