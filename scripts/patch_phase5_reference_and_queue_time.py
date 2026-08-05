from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

p = Path("internal/continuity/service.go")
s = p.read_text()
s = replace_once(
    s,
    '''func matterReference(value string) string {
\tclean := strings.ToUpper(strings.ReplaceAll(value, "-", ""))
\tif len(clean) > 8 {
\t\tclean = clean[:8]
\t}
\treturn "MAT-" + clean
}''',
    '''func matterReference(value string) string {
\tclean := strings.ToUpper(strings.ReplaceAll(value, "-", ""))
\t// UUIDv7 prefixes are time-dominant and collide for records created in the
\t// same millisecond. Use the entropy-bearing suffix for human references.
\tif len(clean) > 16 {
\t\tclean = clean[len(clean)-16:]
\t}
\treturn "MAT-" + clean
}''',
    "Matter reference generator",
)
p.write_text(s)

p = Path("internal/continuity/projection_postgres.go")
s = p.read_text()
s = replace_once(
    s,
    '''if _, err = queueProgramStateTx(ctx, tx, trigger.TenantID, trigger.ProgramID, bundle.ProgramEvent.AggregateVersion, trigger.Type, trigger.ID, bundle.ProgramEvent.ActorID, bundle.ProgramEvent.OccurredAt); err != nil {''',
    '''if _, err = queueProgramStateTx(ctx, tx, trigger.TenantID, trigger.ProgramID, bundle.ProgramEvent.AggregateVersion, trigger.Type, trigger.ID, bundle.ProgramEvent.ActorID, time.Now().UTC()); err != nil {''',
    "trigger projection availability",
)
p.write_text(s)
