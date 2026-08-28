package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type supersessionFormReader struct {
	forms map[int64]DistributionFormRevision
}

func (reader supersessionFormReader) GetDistributionFormRevision(_ context.Context, tenantID, legalEntityID, formID string, version int64) (DistributionFormRevision, error) {
	form, ok := reader.forms[version]
	if !ok || form.TenantID != tenantID || form.LegalEntityID != legalEntityID || form.ID != formID {
		return DistributionFormRevision{}, ErrNotFound
	}
	return form, nil
}

func TestMemoryDistributionSupersessionCarriesOnlyCompatibleAnswers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 28, 6, 0, 0, 0, time.UTC)
	source, target := supersessionForms()
	keyring := supersessionKeyring(t)
	repo := NewMemoryRepository(nil, nil)
	distributions := NewMemoryDistributionStore(repo, supersessionFormReader{forms: map[int64]DistributionFormRevision{3: source, 4: target}}, keyring)
	distributions.now = func() time.Time { return now }

	created, err := distributions.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		SubjectType: "VENDOR", SubjectID: "00000000-0000-0000-0000-000000000101",
		Title: "Resilience review", Purpose: "Confirm current controls.", AccessPolicy: AccessDirectMagicLink,
		EstimatedMinutes: 9, Deadline: now.Add(48 * time.Hour), RouteExpiresAt: now.Add(24 * time.Hour),
		CreatedBy: "actor-a", Recipients: []DistributionRecipientInput{
			{Role: RecipientTo, Type: RecipientExternalAudience, Address: "owner@example.test", AudienceHint: "o***@example.test", ContactLabel: "Owner"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := distributions.TransitionDistribution(ctx, "tenant-a", "entity-a", created.Distribution.ID, TransitionDistributionInput{
		ExpectedVersion: created.Distribution.Version, To: DistributionOpen, ActorID: "actor-a",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	accessStore := NewMemoryDistributionAccessStore(distributions)
	hmacKey := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	access, err := NewDistributionAccessService(accessStore, keyring, nil, hmacKey, 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	access.now = func() time.Time { return now }
	if _, err := access.EnsureDistributionAccessRoutes(ctx, "tenant-a", "entity-a", opened.Distribution.ID, "actor-a"); err != nil {
		t.Fatal(err)
	}

	request, err := repo.GetRequest(ctx, "tenant-a", opened.Recipients[0].RequestID)
	if err != nil {
		t.Fatal(err)
	}
	state := accessStore.memoryWorkspaceState(opened.Workspace, request)
	state.mu.Lock()
	state.workspace.Version = opened.Workspace.Version + 3
	state.answers = map[string]formcontract.AnswerValue{
		"stable":   formcontract.TextAnswer("alpha"),
		"choice":   formcontract.TextAnswer("retired"),
		"evidence": {ArtifactIDs: []string{"artifact-a"}},
	}
	state.mode = formcontract.PresentationClassic
	state.sequences = map[string]int64{"stable": opened.Workspace.Version + 1, "choice": opened.Workspace.Version + 2, "evidence": opened.Workspace.Version + 3}
	state.provenance = map[string]WorkspaceFieldProvenance{}
	for fieldID, sequence := range state.sequences {
		state.provenance[fieldID] = WorkspaceFieldProvenance{
			RecipientID: opened.Recipients[0].ID, RequestID: opened.Recipients[0].RequestID,
			Assurance: AssuranceLinkPossession, Sequence: sequence, UpdatedAt: now,
		}
	}
	workspaceVersion := state.workspace.Version
	state.mu.Unlock()
	distributions.mu.Lock()
	workspace := distributions.workspaces[opened.Distribution.ID]
	workspace.Version = workspaceVersion
	workspace.UpdatedAt = now
	distributions.workspaces[opened.Distribution.ID] = workspace
	distributions.mu.Unlock()

	preview, err := access.PreviewDistributionSupersession(ctx, "tenant-a", "entity-a", opened.Distribution.ID, SupersessionPreviewInput{
		ExpectedVersion: opened.Distribution.Version, TargetFormVersion: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := decisionFieldIDs(preview.CompatibleFields); !sameStrings(got, []string{"stable"}) {
		t.Fatalf("compatible fields = %v, want [stable]", got)
	}
	if len(preview.ExcludedFields) != 2 {
		t.Fatalf("excluded fields = %+v, want 2", preview.ExcludedFields)
	}

	if _, err := access.SupersedeDistribution(ctx, "tenant-a", "entity-a", opened.Distribution.ID, SupersedeDistributionInput{
		ExpectedVersion: opened.Distribution.Version, ExpectedWorkspaceVersion: workspaceVersion - 1,
		TargetFormVersion: 4, CarryForward: true, ConfirmedFieldIDs: []string{"stable"}, ActorID: "actor-a",
	}); !errors.Is(err, ErrSupersessionPreviewMismatch) {
		t.Fatalf("stale workspace error = %v, want preview mismatch", err)
	}

	result, err := access.SupersedeDistribution(ctx, "tenant-a", "entity-a", opened.Distribution.ID, SupersedeDistributionInput{
		ExpectedVersion: opened.Distribution.Version, ExpectedWorkspaceVersion: workspaceVersion,
		TargetFormVersion: 4, CarryForward: true, ConfirmedFieldIDs: []string{"stable"}, ActorID: "actor-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Previous.Distribution.Status != DistributionSuperseded || result.Replacement.Distribution.Status != DistributionOpen {
		t.Fatalf("unexpected supersession states: old=%s new=%s", result.Previous.Distribution.Status, result.Replacement.Distribution.Status)
	}
	if result.Replacement.Distribution.FormTemplateVersion != 4 || len(result.CarriedFieldIDs) != 1 || result.CarriedFieldIDs[0] != "stable" {
		t.Fatalf("replacement/carry result = %+v", result)
	}
	replacementRequest, err := repo.GetRequest(ctx, "tenant-a", result.Replacement.Recipients[0].RequestID)
	if err != nil {
		t.Fatal(err)
	}
	replacementState := accessStore.memoryWorkspaceState(result.Replacement.Workspace, replacementRequest)
	replacementState.mu.Lock()
	view := memoryWorkspaceView(replacementState)
	replacementState.mu.Unlock()
	if value, ok := view.Answers["stable"].ScalarText(); !ok || value != "alpha" || len(view.Answers) != 1 {
		t.Fatalf("replacement answers = %+v", view.Answers)
	}
	provenance := view.FieldProvenance["stable"]
	if provenance.RecipientID != result.Replacement.Recipients[0].ID || provenance.RequestID != result.Replacement.Recipients[0].RequestID || provenance.Assurance != AssuranceLinkPossession {
		t.Fatalf("replacement provenance = %+v", provenance)
	}
	oldRoutes, err := accessStore.ListActiveAccessRoutes(ctx, "tenant-a", "entity-a", opened.Distribution.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldRoutes) != 0 {
		t.Fatalf("superseded distribution retained %d active routes", len(oldRoutes))
	}
}

func supersessionForms() (DistributionFormRevision, DistributionFormRevision) {
	base := DistributionFormRevision{
		ID: "form-a", TenantID: "tenant-a", LegalEntityID: "entity-a", Version: 3, Active: true,
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationClassic},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields: []formcontract.Field{
			{ID: "stable", SectionID: "general", Label: "Stable", Type: formcontract.TypeShortText},
			{ID: "choice", SectionID: "general", Label: "Choice", Type: formcontract.TypeSingleSelect, Options: []string{"current", "retired"}},
			{ID: "evidence", SectionID: "general", Label: "Evidence", Type: formcontract.TypeFile},
		},
	}
	target := base
	target.Version = 4
	target.Fields = append([]formcontract.Field(nil), base.Fields...)
	target.Fields[1].Options = []string{"current"}
	return base, target
}

func supersessionKeyring(t *testing.T) RecipientKeyring {
	t.Helper()
	key := [32]byte{11, 21, 31, 41, 51, 61, 71, 81, 91, 101, 111, 121, 12, 22, 32, 42, 52, 62, 72, 82, 92, 102, 112, 122, 13, 23, 33, 43, 53, 63, 73, 83}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": key})
	if err != nil {
		t.Fatal(err)
	}
	return keyring
}
