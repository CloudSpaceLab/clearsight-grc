from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label} anchor changed")
    return text.replace(old, new, 1)

p = Path("internal/continuity/model.go")
s = p.read_text()
s = replace_once(
    s,
    '''\tObservedAt  time.Time       `json:"observed_at"`
\tSource      string          `json:"source"`
}''',
    '''\tObservedAt  time.Time       `json:"observed_at"`
\tSource      string          `json:"source"`
\tActorID     string          `json:"actor_id,omitempty"`
}''',
    "Trigger actor field",
)
p.write_text(s)

p = Path("internal/continuity/compound_service.go")
s = p.read_text()
s = replace_once(
    s,
    '''programEvent, err := newEvent(trigger.TenantID, "PROGRAM", trigger.ProgramID, aggregate.Program.Version+1, EventProgramTriggerRecorded, trigger, ActorSystem, "", trigger.ObservedAt)''',
    '''programEvent, err := newEvent(trigger.TenantID, "PROGRAM", trigger.ProgramID, aggregate.Program.Version+1, EventProgramTriggerRecorded, trigger, actorFor(trigger.ActorID), trigger.ActorID, trigger.ObservedAt)''',
    "atomic trigger actor",
)
s = s.replace('newEvent(trigger.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, ActorSystem, "", now)', 'newEvent(trigger.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, actorFor(trigger.ActorID), trigger.ActorID, now)')
s = s.replace('newEvent(trigger.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, ActorSystem, "", now)', 'newEvent(trigger.TenantID, "MATTER", matter.ID, 2, EventMatterLinked, link, actorFor(trigger.ActorID), trigger.ActorID, now)')
p.write_text(s)

p = Path("internal/continuity/service.go")
s = p.read_text()
s = replace_once(
    s,
    '''s.applyProgramValue(ctx, trigger.TenantID, trigger.ProgramID, aggregate.Program.Version, EventProgramTriggerRecorded, trigger, "")''',
    '''s.applyProgramValue(ctx, trigger.TenantID, trigger.ProgramID, aggregate.Program.Version, EventProgramTriggerRecorded, trigger, trigger.ActorID)''',
    "legacy trigger actor",
)
s = replace_once(
    s,
    '''Contradictions: json.RawMessage(`[]`), ProgramID: trigger.ProgramID})''',
    '''Contradictions: json.RawMessage(`[]`), ProgramID: trigger.ProgramID, ActorID: trigger.ActorID})''',
    "legacy trigger Matter actor",
)
p.write_text(s)
