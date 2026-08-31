package monitoring

import (
	"errors"
	"testing"
	"time"
)

func TestAdvancedFormLibraryEvaluatesBoundedGroupsAndAuthoritativeStatusFacets(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 31, 5, 0, 0, 0, time.UTC)
	forms := []FormTemplate{
		libraryForm("form-a", "entity-a", "program-a", "ACCESS", LifecycleDraft, 1, false, now),
		libraryForm("form-b", "entity-a", "program-b", "VENDOR", LifecycleActive, 1, true, now.Add(-time.Minute)),
		libraryForm("form-c", "entity-a", "program-c", "RESILIENCE", LifecyclePaused, 1, true, now.Add(-2*time.Minute)),
	}
	forms[1].Tags = []string{"third-party"}
	for _, form := range forms {
		if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
			t.Fatal(err)
		}
	}

	expression := &FormFilterExpression{Kind: "group", Operator: "and", Children: []FormFilterExpression{
		{Kind: "group", Operator: "or", Children: []FormFilterExpression{
			{Kind: "condition", Field: FormFilterStatus, Operator: "is", Value: "draft"},
			{Kind: "condition", Field: FormFilterStatus, Operator: "is", Value: "active"},
		}},
		{Kind: "group", Operator: "or", Children: []FormFilterExpression{
			{Kind: "condition", Field: FormFilterProgram, Operator: "is", Value: "program-a"},
			{Kind: "condition", Field: FormFilterTag, Operator: "is", Value: "Third-Party"},
		}},
	}}
	page, err := repo.ListAdvancedFormLibrary(t.Context(), FormLibraryFilter{
		TenantID: "bank-a", LegalEntityID: "entity-a", Expression: expression, Limit: 25,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total == nil || *page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("advanced page = %#v", page)
	}
	if page.Items[0].Template.ID != "form-a" || page.Items[1].Template.ID != "form-b" {
		t.Fatalf("advanced ordering = %#v", page.Items)
	}
	if page.Facets == nil || page.Facets.Status[LifecycleDraft] != 1 || page.Facets.Status[LifecycleActive] != 1 || page.Facets.Status[LifecyclePaused] != 0 {
		t.Fatalf("status facets = %#v", page.Facets)
	}
}

func TestAdvancedFormFilterRejectsUnsupportedAndUnboundedTrees(t *testing.T) {
	for name, expression := range map[string]*FormFilterExpression{
		"field": {Kind: "condition", Field: "reviewer", Operator: "is", Value: "person-a"},
		"operator": {Kind: "condition", Field: FormFilterTag, Operator: "contains", Value: "third-party"},
		"depth": {Kind: "group", Operator: "and", Children: []FormFilterExpression{{Kind: "group", Operator: "and", Children: []FormFilterExpression{{Kind: "group", Operator: "and", Children: []FormFilterExpression{{Kind: "condition", Field: FormFilterTag, Operator: "is", Value: "third-party"}}}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeFormFilterExpression(expression); !errors.Is(err, ErrInvalid) {
				t.Fatalf("error = %v, want ErrInvalid", err)
			}
		})
	}

	children := make([]FormFilterExpression, 12)
	for index := range children {
		children[index] = FormFilterExpression{Kind: "condition", Field: FormFilterTag, Operator: "is", Value: "tag"}
	}
	if _, err := NormalizeFormFilterExpression(&FormFilterExpression{Kind: "group", Operator: "or", Children: children}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("node-limit error = %v, want ErrInvalid", err)
	}
}

func TestSavedFormViewClonesAdvancedExpression(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 31, 5, 30, 0, 0, time.UTC)
	view := SavedFormView{
		ID: "view-a", TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "person-a", Name: "Priority forms",
		Filter: FormLibraryFilter{Expression: &FormFilterExpression{Kind: "condition", Field: FormFilterStatus, Operator: "is", Value: "ACTIVE"}},
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := repo.SaveFormView(t.Context(), view); err != nil {
		t.Fatal(err)
	}
	views, err := repo.ListSavedFormViews(t.Context(), "bank-a", "entity-a", "person-a")
	if err != nil || len(views) != 1 {
		t.Fatalf("views = %#v, err = %v", views, err)
	}
	views[0].Filter.Expression.Value = "DRAFT"
	again, err := repo.ListSavedFormViews(t.Context(), "bank-a", "entity-a", "person-a")
	if err != nil || again[0].Filter.Expression.Value != "ACTIVE" {
		t.Fatalf("stored expression mutated: %#v, err = %v", again, err)
	}
}
