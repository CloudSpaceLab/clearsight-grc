import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { canCurrentActorTransitionProgram, transitionProgram } from "../continuityCommands";
import { apiErrorKind } from "../http";
import type { ProgramAggregate } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  onUpdated: (value: ProgramAggregate) => void;
};

type AccessState = "loading" | "allowed" | "read-only";
type SubmitState = "idle" | "saving" | "saved" | "error";

function humanize(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

// UI affordance only. The command service remains authoritative and rechecks
// lifecycle legality, actor authority, optimistic version and activation gates.
export function programTransitionTargets(status: string): string[] {
  switch (status) {
    case "DRAFT": return ["ACTIVE", "RETIRED"];
    case "ACTIVE": return ["PAUSED", "RETIRED"];
    case "PAUSED": return ["ACTIVE", "RETIRED"];
    default: return [];
  }
}

function errorMessage(error: unknown) {
  const kind = apiErrorKind(error);
  if (kind === "forbidden" || kind === "unauthorized") return "Your current authority no longer permits this status change.";
  if (kind === "conflict") return "This Program changed since it was loaded. Reload it before changing status.";
  return error instanceof Error && error.message ? error.message : "The Program status change could not be recorded.";
}

export function ProgramLifecycleControls({ aggregate, onUpdated }: Props) {
  const program = aggregate.program;
  const [access, setAccess] = useState<AccessState>("loading");
  const choices = useMemo(() => programTransitionTargets(program.status), [program.status]);
  const [target, setTarget] = useState(() => programTransitionTargets(program.status)[0] ?? "");
  const [rationale, setRationale] = useState("");
  const [submitState, setSubmitState] = useState<SubmitState>("idle");
  const [error, setError] = useState("");
  const selectedTarget = choices.includes(target) ? target : choices[0] ?? "";

  useEffect(() => {
    setTarget(programTransitionTargets(program.status)[0] ?? "");
    setSubmitState("idle");
    setError("");
  }, [program.status]);

  useEffect(() => {
    let active = true;
    if (!choices.length) {
      setAccess("read-only");
      return () => { active = false; };
    }
    setAccess("loading");
    void canCurrentActorTransitionProgram(program.id).then((allowed) => {
      if (active) setAccess(allowed ? "allowed" : "read-only");
    }).catch(() => {
      if (active) setAccess("read-only");
    });
    return () => { active = false; };
  }, [program.id, choices.length]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (submitState === "saving" || !selectedTarget) return;
    setSubmitState("saving");
    setError("");
    try {
      const updated = await transitionProgram(program.id, program.version, selectedTarget, rationale.trim());
      onUpdated(updated);
      setRationale("");
      setSubmitState("saved");
    } catch (value) {
      setError(errorMessage(value));
      setSubmitState("error");
    }
  }

  if (access !== "allowed" || !choices.length) return null;

  return <section className="handoff-summary program-operation" aria-label="Program status action">
    <span className="eyebrow">Authorized Program action</span>
    <h3>Change operating status</h3>
    <p>The server rechecks authority, current version, lifecycle validity and activation prerequisites when this is submitted.</p>
    <form className="governed-command-form" onSubmit={submit}>
      <label><span>Requested status</span><select value={selectedTarget} onChange={(event) => setTarget(event.target.value)}>{choices.map((value) => <option value={value} key={value}>{humanize(value)}</option>)}</select></label>
      <label><span>Rationale</span><textarea rows={3} value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Why should the Program status change now?" required/></label>
      {error && <p className="inline-error" role="alert">{error}</p>}
      {submitState === "saved" && <div className="inline-notice" role="status">Program status recorded from the authoritative server response.</div>}
      <button className="primary-button" type="submit" disabled={submitState === "saving" || !selectedTarget}>{submitState === "saving" ? "Recording…" : `Request ${humanize(selectedTarget)}`}</button>
    </form>
  </section>;
}
