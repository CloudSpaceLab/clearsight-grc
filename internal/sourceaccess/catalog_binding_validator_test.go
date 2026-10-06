package sourceaccess

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBindingDraftValidatorRejectsBeforePersistence(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	repository := NewMemoryCatalogRepository([]SourceScope{{TenantID: catalogTenantID, SourceID: catalogSourceID}})
	if _, err := repository.CreateConnectionRevision(ctx, catalogConnectionRevision(now)); err != nil {
		t.Fatal(err)
	}
	view, err := repository.CreateViewRevision(ctx, catalogViewRevision(now))
	if err != nil {
		t.Fatal(err)
	}
	service := NewCatalogService(repository, nil, nil)
	ids := []string{
		"7c111111-1111-7111-8111-111111111111",
		"7c222222-2222-7222-8222-222222222222",
	}
	service.newID = func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	}
	validationErr := errors.New("reviewed source mapping is invalid")
	service.ConfigureBindingDraftValidator(func(binding BindingRevision, parent ViewRevision) error {
		if binding.ViewID != parent.ViewID || binding.ViewVersion != parent.Version {
			t.Fatalf("validator received mismatched parent: binding=%#v view=%#v", binding, parent)
		}
		return validationErr
	})

	_, err = service.CreateBindingDraft(ctx, CatalogActor{TenantID: catalogTenantID, PrincipalID: catalogActorID}, view.ViewID, CreateBindingDraftInput{
		ViewVersion:    view.Version,
		Code:           "CHANNEL-METRICS",
		Name:           "Channel metrics",
		Purpose:        "IT_GOVERNANCE_CHANNEL_PERFORMANCE",
		Operations:     []Operation{OperationPage},
		SelectedFields: []string{"account_id"},
		KeyFields:      []string{"account_id"},
		Limits:         DefaultResourceLimits(),
	})
	if !errors.Is(err, ErrCatalogInvalid) || !errors.Is(err, validationErr) {
		t.Fatalf("binding validator error=%v", err)
	}
	if values, listErr := service.Bindings(ctx, catalogTenantID, view.ViewID, 20); listErr != nil || len(values) != 0 {
		t.Fatalf("invalid binding was persisted: values=%#v err=%v", values, listErr)
	}
	if !strings.Contains(err.Error(), "reviewed source mapping is invalid") {
		t.Fatalf("validation context was lost: %v", err)
	}
}
