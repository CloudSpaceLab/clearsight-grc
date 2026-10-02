package controlcatalog

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPromoteCreatesReusableDefinitionAndExactImplementationLink(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC) }

	definition, link, err := service.Promote(context.Background(), PromoteInput{
		TenantID: "bank", LegalEntityID: "entity-a",
		Code: " ac-01 ", Name: " Access review ", Objective: "Privileged access remains approved.",
		Description: "Quarterly privileged access review.", Category: "Access",
		ProgramID: "program-1", ImplementationID: "implementation-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if definition.Code != "AC-01" || definition.Status != DefinitionActive || definition.Version != 1 {
		t.Fatalf("definition=%#v", definition)
	}
	if link.DefinitionID != definition.ID || link.LegalEntityID != "entity-a" || link.ImplementationID != "implementation-1" {
		t.Fatalf("link=%#v", link)
	}
}

func TestDefinitionCanBeReusedAcrossImplementationsWithoutDuplicatingState(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC) }
	definition, _, err := service.Promote(context.Background(), PromoteInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "BCP-01", Name: "Recovery exercise",
		Objective: "Critical recovery arrangements are exercised.", ProgramID: "program-1", ImplementationID: "implementation-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.LinkImplementation(context.Background(), LinkImplementationInput{
		TenantID: "bank", LegalEntityID: "entity-b", DefinitionID: definition.ID,
		ProgramID: "program-2", ImplementationID: "implementation-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	links, err := service.ListImplementationLinks(context.Background(), "bank", definition.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || second.DefinitionID != definition.ID {
		t.Fatalf("links=%#v", links)
	}
}

func TestImplementationCannotBeCataloguedTwiceInSameEntity(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC) }
	definition, _, err := service.Promote(context.Background(), PromoteInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "CTRL-1", Name: "Control",
		Objective: "Control objective.", ProgramID: "program-1", ImplementationID: "implementation-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.LinkImplementation(context.Background(), LinkImplementationInput{
		TenantID: "bank", LegalEntityID: "entity-a", DefinitionID: definition.ID,
		ProgramID: "program-1", ImplementationID: "implementation-1",
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate link error=%v", err)
	}
}

func TestCatalogRejectsWildcardOrIncompleteScope(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC) }
	_, _, err := service.Promote(context.Background(), PromoteInput{
		TenantID: "bank", LegalEntityID: "*", Code: "CTRL-1", Name: "Control",
		Objective: "Control objective.", ProgramID: "program-1", ImplementationID: "implementation-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("scope error=%v", err)
	}
}
