from pathlib import Path
import re


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)


# Service: commands queue derived status work without a second post-commit write,
# use atomic compound repositories where available, and keep status versions
# separate from command versions.
p = Path("internal/continuity/service.go")
s = p.read_text()
s = replace_once(
    s,
    '''\tif _, err = s.repo.CreateProgram(ctx, program, event); err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\tif err = s.refreshProgram(ctx, input.TenantID, program.ID, "PROGRAM_CREATED", program.ID); err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\treturn s.repo.GetProgram(ctx, input.TenantID, program.ID)''',
    '''\tif _, err = s.repo.CreateProgram(ctx, program, event); err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\treturn s.refreshAndGetProgram(ctx, input.TenantID, program.ID, "PROGRAM_CREATED", program.ID)''',
    "CreateProgram refresh",
)
s = replace_once(
    s,
    '''\tinserted, err := s.repo.RecordProgramTrigger(ctx, trigger)''',
    '''\tif bundleRepo, ok := s.repo.(TriggerBundleRepository); ok {
\t\taggregate, getErr := s.repo.GetProgram(ctx, trigger.TenantID, trigger.ProgramID)
\t\tif getErr != nil {
\t\t\treturn ProgramAggregate{}, nil, false, getErr
\t\t}
\t\treturn s.applyTriggerBundle(ctx, trigger, aggregate, bundleRepo)
\t}
\tinserted, err := s.repo.RecordProgramTrigger(ctx, trigger)''',
    "ApplyTrigger compound branch",
)
s = replace_once(
    s,
    '''\tif _, err = s.repo.CreateMatter(ctx, matter, event); err != nil {
\t\treturn MatterAggregate{}, err
\t}
\tif input.ProgramID != "" {
\t\tlinked, linkErr := s.AddMatterLink(ctx, AddMatterLinkInput{TenantID: input.TenantID, MatterID: matter.ID, ExpectedVersion: matter.Version, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlID: input.ControlID, Relationship: "AFFECTS", ActorID: input.ActorID})
\t\tif linkErr != nil {
\t\t\treturn MatterAggregate{}, linkErr
\t\t}
\t\treturn linked, nil
\t}
\treturn s.GetMatter(ctx, input.TenantID, matter.ID)''',
    '''\tif input.ProgramID != "" {
\t\tif compoundRepo, ok := s.repo.(CompoundRepository); ok {
\t\t\treturn s.createMatterWithInitialLink(ctx, input, matter, event, compoundRepo)
\t\t}
\t}
\tif _, err = s.repo.CreateMatter(ctx, matter, event); err != nil {
\t\treturn MatterAggregate{}, err
\t}
\tif input.ProgramID != "" {
\t\tlinked, linkErr := s.AddMatterLink(ctx, AddMatterLinkInput{TenantID: input.TenantID, MatterID: matter.ID, ExpectedVersion: matter.Version, ProgramID: input.ProgramID, RequirementID: input.RequirementID, ControlID: input.ControlID, Relationship: "AFFECTS", ActorID: input.ActorID})
\t\tif linkErr != nil {
\t\t\treturn MatterAggregate{}, linkErr
\t\t}
\t\treturn linked, nil
\t}
\treturn s.GetMatter(ctx, input.TenantID, matter.ID)''',
    "CreateMatter compound branch",
)
s = replace_once(
    s,
    '''\t_ = s.refreshProgram(ctx, input.TenantID, input.ProgramID, EventMatterLinked, input.MatterID)''',
    '''\t_ = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventMatterLinked, input.MatterID, input.ActorID)''',
    "link refresh",
)
s = replace_once(
    s,
    '''func (s *Service) ProgramAt(ctx context.Context, tenant, id string, at time.Time) (ProgramAggregate, error) {
\tevents, err := s.repo.ProgramEvents(ctx, tenant, id, &at)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\treturn reconstructProgram(events)
}''',
    '''func (s *Service) ProgramAt(ctx context.Context, tenant, id string, at time.Time) (ProgramAggregate, error) {
\tevents, err := s.repo.ProgramEvents(ctx, tenant, id, &at)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\taggregate, err := reconstructProgram(events)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\tstate, err := s.programStateAt(ctx, tenant, id, &at)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\tif state != nil {
\t\taggregate.CurrentState = state
\t}
\treturn decorateProgram(aggregate), nil
}''',
    "ProgramAt state projection",
)
refresh_pattern = re.compile(r'''func \(s \*Service\) refreshAndGetProgram\(ctx context\.Context, tenant, programID, triggerType, triggerID string\) \(ProgramAggregate, error\) \{.*?\n\}\n\nfunc \(s \*Service\) refreshProgram\(ctx context\.Context, tenant, programID, triggerType, triggerID string\) error \{.*?\n\}\n\nfunc \(s \*Service\) refreshLinkedPrograms\(ctx context\.Context, tenant, matterID, triggerType string\) \{.*?\n\}''', re.S)
replacement = '''func (s *Service) refreshAndGetProgram(ctx context.Context, tenant, programID, triggerType, triggerID string) (ProgramAggregate, error) {
\tif err := s.requestProgramRefresh(ctx, tenant, programID, triggerType, triggerID, "system"); err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\treturn s.repo.GetProgram(ctx, tenant, programID)
}

func (s *Service) refreshProgram(ctx context.Context, tenant, programID, triggerType, triggerID string) error {
\taggregate, err := s.repo.GetProgram(ctx, tenant, programID)
\tif err != nil {
\t\treturn err
\t}
\topenMatters, err := s.repo.OpenMatterCount(ctx, tenant, programID)
\tif err != nil {
\t\treturn err
\t}
\tstate := deriveProgramState(aggregate, openMatters, s.now().UTC())
\tstate.ID, err = id.NewUUIDv7()
\tif err != nil {
\t\treturn err
\t}
\tstate.TriggerType = triggerType
\tstate.TriggerID = triggerID
\tstate.ProgramVersion = aggregate.Program.Version
\tif aggregate.CurrentState != nil && stateEquivalent(*aggregate.CurrentState, state) && aggregate.CurrentState.ProgramVersion == aggregate.Program.Version {
\t\treturn nil
\t}
\tif projectionRepo, ok := s.repo.(ProgramStateRepository); ok {
\t\t_, err = projectionRepo.SaveProgramState(ctx, tenant, programID, aggregate.Program.Version, state)
\t\treturn err
\t}
\tevent, err := newEvent(tenant, "PROGRAM", programID, aggregate.Program.Version+1, EventProgramStateUpdated, state, ActorSystem, "", s.now().UTC())
\tif err != nil {
\t\treturn err
\t}
\t_, err = s.repo.ApplyProgramEvent(ctx, tenant, programID, aggregate.Program.Version, event)
\treturn err
}

func (s *Service) refreshLinkedPrograms(ctx context.Context, tenant, matterID, triggerType string) {
\tprogramIDs, err := s.repo.LinkedProgramIDs(ctx, tenant, matterID)
\tif err != nil {
\t\treturn
\t}
\tfor _, programID := range programIDs {
\t\t_ = s.requestProgramRefresh(ctx, tenant, programID, triggerType, matterID, "system")
\t}
}'''
s, count = refresh_pattern.subn(replacement, s, count=1)
if count != 1:
    raise SystemExit("refresh functions anchor changed")
p.write_text(s)

# PostgreSQL: attach the latest calculated state to detail reads and write a
# deduplicated maintenance job in every material command transaction.
p = Path("internal/continuity/postgres.go")
s = p.read_text()
s = replace_once(
    s,
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn Program{}, err
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn Program{}, err
\t}
\tif _, err = queueProgramStateTx(ctx, tx, program.TenantID, program.ID, program.Version, "PROGRAM_CREATED", program.ID, event.ActorID, event.OccurredAt); err != nil {
\t\treturn Program{}, err
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    "CreateProgram transactional projection queue",
)
s = replace_once(
    s,
    '''func (r *PostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
\tevents, err := r.ProgramEvents(ctx, tenant, id, nil)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\treturn reconstructProgram(events)
}''',
    '''func (r *PostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
\tevents, err := r.ProgramEvents(ctx, tenant, id, nil)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\taggregate, err := reconstructProgram(events)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\tstate, err := r.ProgramStateAt(ctx, tenant, id, nil)
\tif err != nil {
\t\treturn ProgramAggregate{}, err
\t}
\tif state != nil {
\t\taggregate.CurrentState = state
\t}
\treturn decorateProgram(aggregate), nil
}''',
    "GetProgram current state",
)
# Narrow replacements inside ApplyProgramEvent and ApplyMatterEvent by slicing
# their function bodies.
program_start = s.index("func (r *PostgresRepository) ApplyProgramEvent")
program_end = s.index("\nfunc (r *PostgresRepository) RecordProgramTrigger", program_start)
program_body = s[program_start:program_end]
program_body = replace_once(
    program_body,
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn 0, err
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn 0, err
\t}
\tif event.Type != EventProgramStateUpdated {
\t\tif _, err = queueProgramStateTx(ctx, tx, tenant, id, event.AggregateVersion, event.Type, event.ID, event.ActorID, event.OccurredAt); err != nil {
\t\t\treturn 0, err
\t\t}
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    "ApplyProgramEvent transactional projection queue",
)
s = s[:program_start] + program_body + s[program_end:]
matter_start = s.index("func (r *PostgresRepository) ApplyMatterEvent")
matter_end = s.index("\nfunc (r *PostgresRepository) MatterByTriggerKey", matter_start)
matter_body = s[matter_start:matter_end]
matter_body = replace_once(
    matter_body,
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn 0, err
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    '''\tif err = insertOutbox(ctx, tx, event); err != nil {
\t\treturn 0, err
\t}
\tprogramRows, err := tx.Query(ctx, `SELECT DISTINCT program_id::text FROM matter_links WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND program_id IS NOT NULL`, tenant, id)
\tif err != nil {
\t\treturn 0, err
\t}
\tprogramIDs := []string{}
\tfor programRows.Next() {
\t\tvar programID string
\t\tif err := programRows.Scan(&programID); err != nil {
\t\t\tprogramRows.Close()
\t\t\treturn 0, err
\t\t}
\t\tprogramIDs = append(programIDs, programID)
\t}
\tif err := programRows.Err(); err != nil {
\t\tprogramRows.Close()
\t\treturn 0, err
\t}
\tprogramRows.Close()
\tfor _, programID := range programIDs {
\t\tif _, err = queueProgramStateTx(ctx, tx, tenant, programID, 0, event.Type, id, event.ActorID, event.OccurredAt); err != nil {
\t\t\treturn 0, err
\t\t}
\t}
\tif err = tx.Commit(ctx); err != nil {''',
    "ApplyMatterEvent transactional projection queue",
)
s = s[:matter_start] + matter_body + s[matter_end:]
# Keep the legacy state event path valid after migration 000010.
s = replace_once(
    s,
    '''INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11)''',
    '''INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version,projection_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,(SELECT COALESCE(max(projection_version),0)+1 FROM program_state_snapshots WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND program_id=$3::uuid))''',
    "legacy Program state insert",
)
p.write_text(s)
