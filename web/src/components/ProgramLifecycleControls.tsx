import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { loadContext, resolveAuthority } from "../api";
import { transitionProgram } from "../continuityCommands";
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

function errorMessage(error: unknown) {
  const kind = apiErrorKind(error);
  if (kind === "forbidden" || kind === "unauthorized") return "Your current authority no longer permits this status change.";
  if (kind === "conflict") return "This Program changed since it was loaded. Reload it before changing status.";
  return error instanceof Error && error.message ? error.message : "The Program status change could not be recorded.";
}

export function ProgramLifecycleControls({ aggregate, onUpdated }: Props) {
  const program = aggregate.program;
  const [access, setAccess] = useState<AccessState>("loading");
  const [target, setTarget] = useState(program.status === "PAUSED" ? "ACTIVE" : program.status === "DRAFT" ? "ACTIVE" : "PAUSED");
  const [rationale, setRationale] = useState("");
  const [submitState, setSubmitState] = useState<SubmitState>("idle");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    if (program.status === "RETIRED") {
      setAccess("read-only");
      return () => { active = false; };
    }
    void Promise.all([
      loadContext(),
      resolveAuthority({ object_type: "PROGRAM", object_id: program.id, responsibility: "AUTHORIZER", decision_type: "program.transition", materiality: 3 }),
    ]).then(([context, resolution]) => {
      if (!active) return;
      const candidates = [resolution.principal, ...(resolution.candidate_principals ?? [])].filter(Boolean);
      setAccess(candidates.some((candidate) => candidate.id === context.actor.id) ? "allowed" : "read-only");
    }).catch(() => {
      if (active) setAccess("read-only");
    });
    return () => { active = false; };
  }, [program.id, program.status]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (submitState === "saving") return;
    setSubmitState("saving");
    setError("");
    try {
      const updated = await transitionProgram(program.id, program.version, target, rationale.trim());
      onUpdated(updated);
      setRationale("");
      setSubmitState("saved");
    } catch (value) {
      setError(errorMessage(value));
      setSubmitState("error");
    }
  }

  if (access !== "allowed" || program.status === "RETIRED") return null;

  const choices = ["ACTIVE", "PAUSED", "RETIRED"].filter((value) => value !== program.status);
  if (!choices.includes(target)) setTarget(choices[0] ?? "RETIRED");

  return <section className="handoff-summary program-operation" aria-label="Program status action">
    <span className="eyebrow">Authorized Program action</span>
    <h3>Change operating status</h3>
    <p>The server rechecks authority, current version, lifecycle validity and activation prerequisites when this is submitted.</p>
    <form className="governed-command-form" onSubmit={submit}>
      <label><span>Requested status</span><select value={target} onChange={(event) => setTarget(event.target.value)}>{choices.map((value) => <option value={value} key={value}>{humanize(value)}</option>)}</select></label>
      <label><span>Rationale</span><textarea rows={3} value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Why should the Program status change now?" required/></label>
      {error && <p className="inline-error" role="alert">{error}</p>}
      {submitState === "saved" && <div className="inline-notice" role="status">Program status recorded from the authoritative server response.</div>}
      <button className="primary-button" type="submit" disabled={submitState === "saving" || !target}>{submitState === "saving" ? "Recording…" : `Request ${humanize(target)}`}</button>
    </form>
  </section>;
}
