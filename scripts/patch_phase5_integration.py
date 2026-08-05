from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

p = Path("internal/integration/continuity_test.go")
s = p.read_text()
s = replace_once(
    s,
    '''\tprogram, err = service.TransitionProgram(ctx, continuity.ProgramTransitionInput{TenantID: "continuity-bank", ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: continuity.ProgramActive, ActorID: continuityReviewerID, Rationale: "Initial setup approved."})
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif program.CurrentState == nil || program.CurrentState.Overall != continuity.StateCurrent || program.StateLabel != "Up to date" {
\t\tt.Fatalf("unexpected current state %#v", program.CurrentState)
\t}

\tcheckpoint := time.Now().UTC()''',
    '''\tprogram, err = service.TransitionProgram(ctx, continuity.ProgramTransitionInput{TenantID: "continuity-bank", ID: program.Program.ID, ExpectedVersion: program.Program.Version, To: continuity.ProgramActive, ActorID: continuityReviewerID, Rationale: "Initial setup approved."})
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tmaintainer := &continuity.ProjectionMaintainer{Service: service, Repo: continuity.NewPostgresRepository(pool), WorkerID: "continuity-test-worker"}
\tif completed, err := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Second), 20); err != nil || completed != 1 {
\t\tt.Fatalf("Program status update failed completed=%d err=%v", completed, err)
\t}
\tprogram, err = service.GetProgram(ctx, "continuity-bank", program.Program.ID)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif program.CurrentState == nil || program.CurrentState.Overall != continuity.StateCurrent || program.StateLabel != "Up to date" || program.CurrentState.ProgramVersion != program.Program.Version {
\t\tt.Fatalf("unexpected current state %#v", program.CurrentState)
\t}

\tcheckpoint := time.Now().UTC()''',
    "active Program maintenance",
)
s = replace_once(
    s,
    '''\tprogram, created, inserted, err := service.ApplyTrigger(ctx, trigger)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif !inserted || created == nil || created.Type != continuity.MatterControlGap || program.CurrentState.OpenMatterCount != 1 {
\t\tt.Fatalf("unexpected trigger result inserted=%v matter=%#v state=%#v", inserted, created, program.CurrentState)
\t}''',
    '''\tprogram, created, inserted, err := service.ApplyTrigger(ctx, trigger)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif !inserted || created == nil || created.Type != continuity.MatterControlGap {
\t\tt.Fatalf("unexpected trigger result inserted=%v matter=%#v", inserted, created)
\t}
\tif completed, err := maintainer.Maintain(ctx, time.Now().UTC().Add(2*time.Second), 20); err != nil || completed != 1 {
\t\tt.Fatalf("trigger status update failed completed=%d err=%v", completed, err)
\t}
\tprogram, err = service.GetProgram(ctx, "continuity-bank", program.Program.ID)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif program.CurrentState == nil || program.CurrentState.OpenMatterCount != 1 || program.CurrentState.ProgramVersion != program.Program.Version {
\t\tt.Fatalf("unexpected trigger status %#v", program.CurrentState)
\t}''',
    "trigger maintenance",
)
p.write_text(s)

p = Path("internal/integration/summary_read_models_test.go")
s = p.read_text()
s = replace_once(
    s,
    '''INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,generated_at,program_version)
\t\tSELECT uuidv7(),tenant_id,id,CASE WHEN code='PRG-0019' THEN 'EVIDENCE_INSUFFICIENT' ELSE 'CURRENT' END,
\t\t\t'{}'::jsonb,CASE WHEN code='PRG-0019' THEN '[{"code":"EVIDENCE_NOT_ASSESSED","summary":"Evidence has not been assessed."}]'::jsonb ELSE '[]'::jsonb END,
\t\t\t0,updated_at,version''',
    '''INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,generated_at,program_version,projection_version)
\t\tSELECT uuidv7(),tenant_id,id,CASE WHEN code='PRG-0019' THEN 'EVIDENCE_INSUFFICIENT' ELSE 'CURRENT' END,
\t\t\t'{}'::jsonb,CASE WHEN code='PRG-0019' THEN '[{"code":"EVIDENCE_NOT_ASSESSED","summary":"Evidence has not been assessed."}]'::jsonb ELSE '[]'::jsonb END,
\t\t\t0,updated_at,version,1''',
    "summary projection fixture",
)
p.write_text(s)
