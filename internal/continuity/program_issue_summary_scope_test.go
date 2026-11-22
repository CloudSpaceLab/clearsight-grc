package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramVisibleOpenIssueCountMatchesScopedListWithoutSnapshot(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	retired := now.Add(-time.Minute)
	program := Program{ID: "p1", TenantID: "bank", LegalEntityID: "entity", Name: "Assurance", Code: "P1", Status: ProgramActive, Scope: json.RawMessage(`{}`), Version: 1, UpdatedAt: now}
	repo.programs["bank"] = map[string]ProgramAggregate{"p1": {Program: program}}
	mk := func(id, entity string, scope string, retiredAt *time.Time, status MatterStatus) MatterAggregate {
		return MatterAggregate{Matter: Matter{ID: id, TenantID: "bank", LegalEntityID: entity, Reference: id, Title: id, Status: status, Scope: json.RawMessage(scope), Version: 1, UpdatedAt: now}, Links: []MatterLink{{MatterID: id, ProgramID: "p1", RetiredAt: retiredAt}}}
	}
	repo.matters["bank"] = map[string]MatterAggregate{"visible": mk("visible", "entity", `{"access":"INTERNAL"}`, nil, MatterAssessment), "retired": mk("retired", "entity", `{"access":"INTERNAL"}`, &retired, MatterAssessment), "hidden": mk("hidden", "entity", `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`, nil, MatterAssessment), "other-entity": mk("other-entity", "elsewhere", `{"access":"INTERNAL"}`, nil, MatterAssessment), "closed": mk("closed", "entity", `{"access":"INTERNAL"}`, nil, MatterClosed)}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reader"})
	service := NewService(repo)
	programs, err := service.ListProgramSummaries(ctx, "bank", SummaryQuery{Limit: 20})
	if err != nil || len(programs.Items) != 1 {
		t.Fatalf("programs: %#v %v", programs, err)
	}
	if programs.Items[0].OpenMatterCount != 1 {
		t.Fatalf("visible count = %d, want 1", programs.Items[0].OpenMatterCount)
	}
	matters, err := service.ListMatterSummaries(ctx, "bank", SummaryQuery{ProgramID: "p1", Status: "OPEN", Limit: 20})
	if err != nil || len(matters.Items) != 1 || matters.Items[0].Matter.ID != "visible" {
		t.Fatalf("visible linked matters = %#v, %v", matters, err)
	}
}

type commandOnlyMatterRepository struct {
	Repository
	listCalled bool
}

func (r *commandOnlyMatterRepository) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	r.listCalled = true
	return r.Repository.ListMatters(ctx, tenant, status, limit)
}
func TestProgramIssueSummaryFailsClosedWithoutScopedRepository(t *testing.T) {
	repo := &commandOnlyMatterRepository{Repository: NewMemoryRepository()}
	svc := NewService(repo)
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reader"})
	_, err := svc.ListMatterSummaries(ctx, "bank", SummaryQuery{ProgramID: "p1", Status: "OPEN", Limit: 20})
	if !errors.Is(err, ErrSummaryUnavailable) || repo.listCalled {
		t.Fatalf("err=%v broad read=%v", err, repo.listCalled)
	}
}
func TestProgramIssueSummariesNeverCrossProgramEntityForTenantWideActor(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	repo.programs["bank"] = map[string]ProgramAggregate{"p1": {Program: Program{ID: "p1", TenantID: "bank", LegalEntityID: "entity-a", Name: "Program", Status: ProgramActive, Scope: json.RawMessage(`{}`), Version: 1, UpdatedAt: now}}}
	repo.matters["bank"] = map[string]MatterAggregate{"same": {Matter: Matter{ID: "same", TenantID: "bank", LegalEntityID: "entity-a", Status: MatterAssessment, Title: "Same", Scope: json.RawMessage(`{}`), UpdatedAt: now}, Links: []MatterLink{{ProgramID: "p1"}}}, "other": {Matter: Matter{ID: "other", TenantID: "bank", LegalEntityID: "entity-b", Status: MatterAssessment, Title: "Other", Scope: json.RawMessage(`{}`), UpdatedAt: now}, Links: []MatterLink{{ProgramID: "p1"}}}}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: "*", PrincipalID: "reader"})
	svc := NewService(repo)
	programs, err := svc.ListProgramSummaries(ctx, "bank", SummaryQuery{Limit: 20})
	if err != nil || len(programs.Items) != 1 || programs.Items[0].OpenMatterCount != 1 {
		t.Fatalf("programs: %#v %v", programs, err)
	}
	matters, err := svc.ListMatterSummaries(ctx, "bank", SummaryQuery{ProgramID: "p1", Status: "OPEN", Limit: 20})
	if err != nil || len(matters.Items) != 1 || matters.Items[0].Matter.ID != "same" {
		t.Fatalf("issues: %#v %v", matters, err)
	}
}
