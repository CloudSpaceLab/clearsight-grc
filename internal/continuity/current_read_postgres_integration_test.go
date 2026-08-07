//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryCounter struct{ count atomic.Int64 }

func (q *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	q.count.Add(1)
	return ctx
}

func (*queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (q *queryCounter) Reset() { q.count.Store(0) }
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
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'bounded-current-test','Bounded Current Test')`, tenantID); err != nil {
		t.Fatal(err)
	}

	raw := NewPostgresRepository(pool)
	current := NewCurrentPostgresRepository(pool)
	service := NewServiceWithClock(current, func() time.Time { return time.Date(2026, 8, 7, 18, 30, 0, 0, time.UTC) })

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

	events, err := raw.ProgramEvents(ctx, "bounded-current-test", program.Program.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	replayedProgram, err := reconstructProgram(events)
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
	if !reflect.DeepEqual(currentProgram.Program, replayedProgram.Program) || !reflect.DeepEqual(currentProgram.Requirements, replayedProgram.Requirements) {
		t.Fatalf("normalized Program current state diverged from replay\ncurrent=%#v\nreplayed=%#v", currentProgram, replayedProgram)
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
			Title: "Action " + twoDigits(index), Description: "Accountable remediation work.",
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
	counter.Reset()
	currentMatter, err := current.GetMatter(ctx, "bounded-current-test", matter.Matter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := counter.Count(); got != 1 {
		t.Fatalf("current Matter detail used %d SQL calls after %d events; want exactly 1", got, len(matterEvents))
	}
	if !reflect.DeepEqual(currentMatter.Matter, replayedMatter.Matter) || !reflect.DeepEqual(currentMatter.Actions, replayedMatter.Actions) {
		t.Fatalf("normalized Matter current state diverged from replay\ncurrent=%#v\nreplayed=%#v", currentMatter, replayedMatter)
	}
}

func twoDigits(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
