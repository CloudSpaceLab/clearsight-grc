//go:build postgres && postgresintegration

package continuity

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryCounter struct{ count atomic.Int64 }

func (q *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	q.count.Add(1)
	return ctx
}

func (*queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (q *queryCounter) Reset()       { q.count.Store(0) }
func (q *queryCounter) Count() int64 { return q.count.Load() }

func TestCurrentPostgresReadsMatchReplayAndStayFixedQuery(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	counter := &queryCounter{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const tenantID = "97777777-7777-7777-8777-777777777771"
	const legalEntityID = "97777777-7777-7777-8777-777777777774"
	const actionOwnerID = "97777777-7777-7777-8777-777777777773"
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'bounded-current-test','Bounded Current Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'BOUNDED-BANK','Bounded Bank','NG',clock_timestamp()-interval '1 day')`, legalEntityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Bounded action owner')`, actionOwnerID, tenantID); err != nil {
		t.Fatal(err)
	}
	ctx = identity.WithActor(ctx, identity.Actor{TenantID: "bounded-current-test", LegalEntityID: legalEntityID, PrincipalID: actionOwnerID})

	raw := NewPostgresRepository(pool)
	current := NewCurrentPostgresRepository(pool)
	now := time.Date(2026, 8, 7, 18, 30, 0, 0, time.UTC)
	service := NewServiceWithClock(current, func() time.Time { return now })

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bounded-current-test", Code: "BOUND", Name: "Bounded current reads", Type: "COMPLIANCE",
		OwningFunction: "Compliance", Scope: json.RawMessage(`{"scope":"bank"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 32; index++ {
		program, err = service.AddRequirement(ctx, AddRequirementInput{
			TenantID: "bounded-current-test", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version,
			Code: "REQ-" + twoDigits(index), Title: "Requirement", Statement: "Maintain a bounded current-state read.",
			Modality: "MUST", Status: RequirementApproved, EffectiveFrom: program.Program.EffectiveFrom,
		})
		if err != nil {
			t.Fatalf("add requirement %d: %v", index, err)
		}
	}
	state := ProgramStateSnapshot{
		ID:              "97777777-7777-7777-8777-777777777772",
		TenantID:        "bounded-current-test",
		ProgramID:       program.Program.ID,
		Overall:         StateAtRisk,
		Dimensions:      ComplianceDimensions{Interpretation: StateCurrent, Applicability: StateCurrent, ControlDesign: StateAtRisk},
		Reasons:         []StateReason{{Code: "BOUNDED_READ_TEST", Summary: "Current-state projection identity is preserved."}},
		OpenMatterCount: 1,
		TriggerType:     "TEST",
		TriggerID:       "projection-parity",
		GeneratedAt:     now.Add(time.Minute),
	}
	projectionVersion, err := current.SaveProgramState(ctx, "bounded-current-test", program.Program.ID, program.Program.Version, state)
	if err != nil {
		t.Fatal(err)
	}

	events, err := raw.ProgramEvents(ctx, "bounded-current-test", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	replayedProgram, err := raw.GetProgram(ctx, "bounded-current-test", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	counter.Reset()
	currentProgram, err := current.GetProgram(ctx, "bounded-current-test", program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := counter.Count(); got != 1 {
		t.Fatalf("current Program detail used %d SQL calls after %d events; want exactly 1", got, len(events))
	}
	canonicalizeProgramAggregateTenant(&replayedProgram, tenantID)
	if !sameJSON(currentProgram.Program, replayedProgram.Program) || !sameJSON(currentProgram.Requirements, replayedProgram.Requirements) || !sameJSON(currentProgram.CurrentState, replayedProgram.CurrentState) {
		t.Fatalf("normalized Program current state diverged from reconstruction\ncurrent=%#v\nreconstructed=%#v", currentProgram, replayedProgram)
	}
	if currentProgram.CurrentState == nil || currentProgram.CurrentState.Overall != StateAtRisk || currentProgram.CurrentState.ProjectionVersion != projectionVersion {
		t.Fatalf("normalized Program detail lost state/projection identity: %#v", currentProgram.CurrentState)
	}
	for name, repo := range map[string]Repository{"current": current, "raw": raw} {
		counter.Reset()
		listed, err := repo.ListPrograms(ctx, "bounded-current-test", 20)
		if err != nil {
			t.Fatalf("%s Program list: %v", name, err)
		}
		if counter.Count() != 1 || len(listed) != 1 || !sameJSON(listed[0].Requirements, currentProgram.Requirements) || !sameJSON(listed[0].CurrentState, currentProgram.CurrentState) {
			t.Fatalf("%s Program list lost full event-backed state or used multiple queries: count=%d value=%#v", name, counter.Count(), listed)
		}
	}

	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bounded-current-test", Type: MatterControlGap, Priority: 3,
		Title: "Bounded Matter", Summary: "Prove current Matter reads do not replay history.", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 32; index++ {
		matter, err = service.AddAction(ctx, AddActionInput{
			TenantID: "bounded-current-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
			Title: "Action " + twoDigits(index), Description: "Accountable remediation work.", OwnerPrincipalID: actionOwnerID,
		})
		if err != nil {
			t.Fatalf("add action %d: %v", index, err)
		}
	}
	matterEvents, err := raw.MatterEvents(ctx, "bounded-current-test", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	replayedMatter, err := reconstructMatter(matterEvents)
	if err != nil {
		t.Fatal(err)
	}
	canonicalizeMatterAggregateTenant(&replayedMatter, tenantID)
	counter.Reset()
	currentMatter, err := current.GetMatter(ctx, "bounded-current-test", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := counter.Count(); got != 1 {
		t.Fatalf("current Matter detail used %d SQL calls after %d events; want exactly 1", got, len(matterEvents))
	}
	if !sameJSON(currentMatter.Matter, replayedMatter.Matter) || !sameJSON(currentMatter.Actions, replayedMatter.Actions) {
		t.Fatalf("normalized Matter current state diverged from replay\ncurrent=%#v\nreplayed=%#v", currentMatter, replayedMatter)
	}
	for name, repo := range map[string]Repository{"current": current, "raw": raw} {
		counter.Reset()
		listed, err := repo.ListMatters(ctx, "bounded-current-test", "OPEN", 20)
		if err != nil {
			t.Fatalf("%s Matter list: %v", name, err)
		}
		if counter.Count() != 1 || len(listed) != 1 || !sameJSON(listed[0].Actions, currentMatter.Actions) {
			t.Fatalf("%s Matter list lost full event-backed state or used multiple queries: count=%d value=%#v", name, counter.Count(), listed)
		}
	}
}

func TestPostgresPortfolioListsStayOneQueryAtMaximumPageSize(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	counter := &queryCounter{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const tenantID = "97777777-7777-7777-8777-777777777781"
	const entityID = "97777777-7777-7777-8777-777777777782"
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'portfolio-list-load','Portfolio List Load')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'LOAD-BANK','Load Bank','NG',clock_timestamp()-interval '1 day')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(tenant_id,legal_entity_id,code,name,program_type,status,owning_function,effective_from,created_at,updated_at)
		SELECT $1::uuid,$2::uuid,'LOAD-P-'||lpad(value::text,3,'0'),'Load Program '||value,'COMPLIANCE','ACTIVE','Risk',clock_timestamp()-interval '1 day',clock_timestamp()-interval '1 day',clock_timestamp()-(value||' seconds')::interval
		FROM generate_series(1,200) value`, tenantID, entityID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,due_at,created_at,updated_at)
		SELECT $1::uuid,$2::uuid,'LOAD-M-'||lpad(value::text,3,'0'),'CONTROL_GAP','ASSESSMENT',1+(value%5),'Load Matter '||value,'Bounded portfolio load','{"access":"INTERNAL"}'::jsonb,clock_timestamp()+(value||' hours')::interval,clock_timestamp()-interval '1 day',clock_timestamp()-(value||' seconds')::interval
		FROM generate_series(1,200) value`, tenantID, entityID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO program_requirements(tenant_id,program_id,code,title,statement,modality,status,effective_from)
		SELECT p.tenant_id,p.id,'LOAD-REQ','Loaded requirement','The full portfolio aggregate retains requirements.','MUST','APPROVED',clock_timestamp()-interval '1 day'
		FROM programs p WHERE p.tenant_id=$1::uuid AND p.code='LOAD-P-001'`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matter_actions(tenant_id,matter_id,title,description,status)
		SELECT m.tenant_id,m.id,'Loaded action','The full portfolio aggregate retains actions.','PLANNED'
		FROM matters m WHERE m.tenant_id=$1::uuid AND m.reference='LOAD-M-001'`, tenantID); err != nil {
		t.Fatal(err)
	}

	actor := identity.WithActor(ctx, identity.Actor{TenantID: tenantID, LegalEntityID: entityID, PrincipalID: "portfolio-reviewer"})
	current := NewCurrentPostgresRepository(pool)
	raw := NewPostgresRepository(pool)
	for name, repo := range map[string]Repository{"current": current, "raw": raw} {
		counter.Reset()
		programs, err := repo.ListPrograms(actor, tenantID, 200)
		if err != nil {
			t.Fatalf("%s Program portfolio: %v", name, err)
		}
		if len(programs) != 200 || counter.Count() != 1 {
			t.Fatalf("%s Program portfolio returned %d rows with %d SQL calls; want 200 rows in one call", name, len(programs), counter.Count())
		}
		fullProgramFound := false
		for _, program := range programs {
			if program.Program.Code == "LOAD-P-001" {
				fullProgramFound = len(program.Requirements) == 1 && program.Requirements[0].Code == "LOAD-REQ"
			}
		}
		if !fullProgramFound {
			t.Fatalf("%s Program portfolio lost nested requirement data", name)
		}
		for index := 1; index < len(programs); index++ {
			if programs[index-1].Program.UpdatedAt.Before(programs[index].Program.UpdatedAt) {
				t.Fatalf("%s Program order is unstable at %d", name, index)
			}
		}

		counter.Reset()
		matters, err := repo.ListMatters(actor, tenantID, "OPEN", 200)
		if err != nil {
			t.Fatalf("%s Matter portfolio: %v", name, err)
		}
		if len(matters) != 200 || counter.Count() != 1 {
			t.Fatalf("%s Matter portfolio returned %d rows with %d SQL calls; want 200 rows in one call", name, len(matters), counter.Count())
		}
		fullMatterFound := false
		for _, matter := range matters {
			if matter.Matter.Reference == "LOAD-M-001" {
				fullMatterFound = len(matter.Actions) == 1 && matter.Actions[0].Title == "Loaded action"
			}
		}
		if !fullMatterFound {
			t.Fatalf("%s Matter portfolio lost nested action data", name)
		}
		for index := 1; index < len(matters); index++ {
			prior, next := matters[index-1].Matter, matters[index].Matter
			outOfOrder := prior.Priority < next.Priority
			if prior.Priority == next.Priority && prior.DueAt != nil && next.DueAt != nil {
				outOfOrder = prior.DueAt.After(*next.DueAt)
			}
			if prior.Priority == next.Priority && ((prior.DueAt == nil && next.DueAt == nil) || (prior.DueAt != nil && next.DueAt != nil && prior.DueAt.Equal(*next.DueAt))) {
				outOfOrder = prior.UpdatedAt.Before(next.UpdatedAt)
			}
			if outOfOrder {
				t.Fatalf("%s Matter order is unstable at %d: %#v before %#v", name, index, prior, next)
			}
		}
	}
}

func canonicalizeProgramAggregateTenant(value *ProgramAggregate, tenantID string) {
	value.Program.TenantID = tenantID
	for index := range value.Requirements {
		value.Requirements[index].TenantID = tenantID
	}
	if value.CurrentState != nil {
		value.CurrentState.TenantID = tenantID
	}
}

func canonicalizeMatterAggregateTenant(value *MatterAggregate, tenantID string) {
	value.Matter.TenantID = tenantID
	for index := range value.Actions {
		value.Actions[index].TenantID = tenantID
	}
}

func sameJSON(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func twoDigits(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
