package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type stubDistributionFormReader struct {
	form DistributionFormRevision
	err  error
}

func (s stubDistributionFormReader) GetDistributionFormRevision(context.Context, string, string, string, int64) (DistributionFormRevision, error) {
	return s.form, s.err
}

type stubRecipientProtector struct {
	err error
}

func (s stubRecipientProtector) ProtectRecipientAddress(_ context.Context, _, _, _, _ string) (protectedRecipientAddress, error) {
	if s.err != nil {
		return protectedRecipientAddress{}, s.err
	}
	return protectedRecipientAddress{Hash: []byte("hash"), Ciphertext: []byte("ciphertext"), KeyID: "key-v1"}, nil
}

func TestMemoryDistributionCreatesTORequestsCCWithoutRequestAndOneWorkspace(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: activeDistributionForm()}, stubRecipientProtector{})
	store.now = func() time.Time { return now }

	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		SubjectType: "VENDOR", SubjectID: "00000000-0000-0000-0000-000000000101",
		Title: "Quarterly resilience review", Purpose: "Confirm the current resilience controls.",
		AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 8,
		Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: "actor-a",
		Recipients: []DistributionRecipientInput{
			{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "principal-a", ContactLabel: "Control owner"},
			{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "v***@example.test", ContactLabel: "Vendor assurance"},
			{Role: RecipientCC, Type: RecipientExternalAudience, Address: "observer@example.test", AudienceHint: "o***@example.test", ContactLabel: "Observer"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Distribution.FormTemplateID != "form-a" || bundle.Distribution.FormTemplateVersion != 3 {
		t.Fatalf("distribution did not pin exact form revision: %+v", bundle.Distribution)
	}
	if bundle.Workspace.DistributionID != bundle.Distribution.ID || bundle.Workspace.Version != 1 {
		t.Fatalf("expected one version-1 shared workspace, got %+v", bundle.Workspace)
	}
	if len(bundle.Recipients) != 3 {
		t.Fatalf("expected 3 safe recipients, got %d", len(bundle.Recipients))
	}
	toRequests := 0
	for _, recipient := range bundle.Recipients {
		if recipient.Role == RecipientTo {
			if recipient.RequestID == "" {
				t.Fatalf("TO recipient missing canonical request: %+v", recipient)
			}
			toRequests++
		} else if recipient.RequestID != "" {
			t.Fatalf("CC recipient must not get edit/request capability: %+v", recipient)
		}
	}
	requests, err := repo.ListRequests(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != toRequests || toRequests != 2 {
		t.Fatalf("expected exactly 2 TO-backed requests, got %d", len(requests))
	}
	for _, request := range requests {
		if request.FormTemplateID != "form-a" || request.FormTemplateVersion != 3 || request.LegalEntityID != "entity-a" {
			t.Fatalf("request lost exact distribution scope: %+v", request)
		}
		if len(request.Fields) != 1 {
			t.Fatalf("request lost governed form fields: %+v", request.Fields)
		}
		field := request.Fields[0]
		if field.CollectionIntent != formcontract.IntentConfirmOrCorrect || field.BrowserCachePolicy != formcontract.BrowserCacheDenied || field.RecordTarget == nil || field.RecordTarget.Key != "registered_address" || field.RecordTarget.RequiredSubjectType != "VENDOR" {
			t.Fatalf("request lost governed collection semantics: %+v", field)
		}
	}

	stored := store.recipients[bundle.Distribution.ID]
	if string(stored[1].protected.Ciphertext) == "vendor@example.test" || len(stored[1].protected.Hash) == 0 || stored[1].protected.KeyID == "" {
		t.Fatal("external address was not stored as protected material")
	}
}

func TestMemoryDistributionRollsBackWhenRecipientProtectionFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: activeDistributionForm()}, stubRecipientProtector{err: errors.New("kms unavailable")})
	store.now = func() time.Time { return now }

	_, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		SubjectType: "VENDOR", SubjectID: "00000000-0000-0000-0000-000000000101", Title: "Review", Purpose: "Collect evidence.",
		AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour), CreatedBy: "actor-a",
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "v***@example.test"}},
	})
	if err == nil {
		t.Fatal("expected protection failure")
	}
	requests, listErr := repo.ListRequests(ctx, "tenant-a", 10)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(requests) != 0 || len(store.distributions) != 0 || len(store.workspaces) != 0 || len(store.events) != 0 || len(store.outbox) != 0 {
		t.Fatal("failed distribution creation left partial state")
	}
}

func TestMemoryDistributionRejectsNonActiveOrCrossScopeRevision(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	form := activeDistributionForm()
	form.LegalEntityID = "entity-b"
	form.Active = false
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryDistributionStore(repo, stubDistributionFormReader{form: form}, stubRecipientProtector{})
	store.now = func() time.Time { return now }

	_, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		SubjectType: "VENDOR", SubjectID: "00000000-0000-0000-0000-000000000101", Title: "Review", Purpose: "Collect evidence.",
		AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour), RouteExpiresAt: now.Add(12 * time.Hour), CreatedBy: "actor-a",
		Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: "principal-a"}},
	})
	if err == nil {
		t.Fatal("expected exact-active form scope rejection")
	}
	requests, listErr := repo.ListRequests(ctx, "tenant-a", 10)
	if listErr != nil || len(requests) != 0 {
		t.Fatalf("rejected distribution mutated requests: %v %+v", listErr, requests)
	}
}

func TestMemoryDistributionUsesSharedKeysetCursor(t *testing.T) {
	store := NewMemoryDistributionStore(nil, nil, nil)
	base := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	const (
		firstID  = "00000000-0000-7000-8000-000000000103"
		secondID = "00000000-0000-7000-8000-000000000102"
		thirdID  = "00000000-0000-7000-8000-000000000101"
	)
	store.distributions[firstID] = FormDistribution{ID: firstID, TenantID: "tenant-a", LegalEntityID: "entity-a", Status: DistributionOpen, UpdatedAt: base.Add(2 * time.Hour)}
	store.distributions[secondID] = FormDistribution{ID: secondID, TenantID: "tenant-a", LegalEntityID: "entity-a", Status: DistributionOpen, UpdatedAt: base.Add(time.Hour)}
	store.distributions[thirdID] = FormDistribution{ID: thirdID, TenantID: "tenant-a", LegalEntityID: "entity-a", Status: DistributionOpen, UpdatedAt: base.Add(time.Hour)}
	store.distributions["00000000-0000-7000-8000-000000000999"] = FormDistribution{ID: "00000000-0000-7000-8000-000000000999", TenantID: "tenant-a", LegalEntityID: "entity-b", Status: DistributionOpen, UpdatedAt: base.Add(3 * time.Hour)}

	query := DistributionListQuery{TenantID: "tenant-a", LegalEntityID: "entity-a", Status: DistributionOpen, Limit: 2}
	firstPage, err := store.ListDistributions(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage) != 2 || firstPage[0].ID != firstID || firstPage[1].ID != secondID {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}

	query.Cursor = encodeDistributionCursor(firstPage[len(firstPage)-1])
	secondPage, err := store.ListDistributions(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage) != 1 || secondPage[0].ID != thirdID {
		t.Fatalf("unexpected second page: %+v", secondPage)
	}

	query.Cursor = "not-a-valid-cursor"
	if _, err := store.ListDistributions(context.Background(), query); err == nil {
		t.Fatal("expected invalid shared cursor rejection")
	}
}

func activeDistributionForm() DistributionFormRevision {
	return DistributionFormRevision{
		ID: "form-a", TenantID: "tenant-a", LegalEntityID: "entity-a", Version: 3, Sensitivity: "CONFIDENTIAL", Active: true,
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields: []formcontract.Field{{
			ID:                 "q1",
			SectionID:          "general",
			Label:              "Control operating?",
			Type:               formcontract.TypeYesNo,
			Required:           true,
			CollectionIntent:   formcontract.IntentConfirmOrCorrect,
			RecordTarget:       &formcontract.RecordTarget{Key: "registered_address", RequiredSubjectType: "VENDOR"},
			BrowserCachePolicy: formcontract.BrowserCacheDenied,
		}},
	}
}
