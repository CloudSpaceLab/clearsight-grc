import { useEffect, useState } from "react";
import { apiErrorKind } from "../http";
import { acceptProgramReview, loadProgramReviewDigest } from "../programReviewApi";
import type { ProgramReviewDigest as ReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";

type Props = { aggregate: ProgramAggregate };
type LoadState = "loading" | "live" | "unavailable";
type SaveState = "idle" | "saving" | "error";

function reviewedAt(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

function acceptError(error: unknown) {
  const kind = apiErrorKind(error);
  if (kind === "conflict") return "This Program changed while you were reviewing it. Reload the current status before marking it reviewed.";
  if (kind === "forbidden" || kind === "unauthorized") return "Your current sign-in can no longer record this review.";
  if (kind === "not_found") return "This Program is no longer available in your current scope.";
  return error instanceof Error && error.message ? error.message : "The review acknowledgement could not be recorded.";
}

export function ProgramReviewDigest({ aggregate }: Props) {
  const program = aggregate.program;
  const projectionVersion = aggregate.current_state?.projection_version ?? 0;
  const [digest, setDigest] = useState<ReviewDigest | null>(null);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoadState("loading");
    setSaveState("idle");
    setError("");
    void loadProgramReviewDigest(program.id).then((value) => {
      if (!active) return;
      setDigest(value);
      setLoadState("live");
    }).catch(() => {
      if (active) setLoadState("unavailable");
    });
    return () => { active = false; };
  }, [program.id, program.version, projectionVersion]);

  async function markReviewed() {
    if (!digest || saveState === "saving") return;
    setSaveState("saving");
    setError("");
    try {
      const value = await acceptProgramReview(program.id, digest.current_program_version, digest.current_projection_version);
      setDigest(value);
      setSaveState("idle");
    } catch (value) {
      setError(acceptError(value));
      setSaveState("error");
    }
  }

  if (loadState === "loading") return <div className="inline-notice" aria-live="polite">Loading review history…</div>;
  if (loadState === "unavailable" || !digest) return <div className="inline-notice">Review history could not be loaded. The latest Program status and reasons are still shown below.</div>;

  const noBaseline = digest.state === "NO_BASELINE";
  const changed = digest.state === "CHANGED";
  const acceptedLabel = reviewedAt(digest.checkpoint?.accepted_at);
  const heading = noBaseline
    ? "Start review-by-exception"
    : changed
      ? digest.changes_total > 0
        ? `${digest.changes_total} change${digest.changes_total === 1 ? "" : "s"} since your last review`
        : "Program changed since your last review"
      : "No changes since your review";

  return <section className="handoff-summary program-review-digest" aria-labelledby={`program-review-${program.id}`}>
    <span className="eyebrow">{changed ? "Since your last review" : "Review history"}</span>
    <h3 id={`program-review-${program.id}`}>{heading}</h3>
    {noBaseline
      ? <p>Review the current exceptions, then mark this status reviewed. Future visits will highlight changes made after this review.</p>
      : acceptedLabel && <p>Last reviewed {acceptedLabel}. {changed ? "Changes made after that review are highlighted here." : "The Program has not changed since that review."}</p>}

    {changed && digest.changes.length > 0 && <div className="status-reasons">
      <h4>What changed</h4>
      <ul>{digest.changes.map((change, index) => <li key={`${change.kind}-${change.object_type ?? ""}-${change.object_id ?? ""}-${index}`}>{change.summary}</li>)}</ul>
      {digest.changes_omitted > 0 && <p>{digest.changes_omitted} additional change{digest.changes_omitted === 1 ? " is" : "s are"} available in Program history.</p>}
    </div>}

    {changed && digest.history_truncated && <p className="inline-notice">This summary does not include older Program changes. Open Program history to review them.</p>}

    {(noBaseline || changed) && digest.current_exceptions.length > 0 && <div className="status-reasons">
      <h4>{noBaseline ? "Current exceptions" : "Exceptions still current"}</h4>
      <ul>{digest.current_exceptions.map((reason) => <li key={`${reason.code}-${reason.object_type ?? ""}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul>
      {digest.current_exceptions_total > digest.current_exceptions.length && <p>{digest.current_exceptions_total - digest.current_exceptions.length} additional current exception{digest.current_exceptions_total - digest.current_exceptions.length === 1 ? " is" : "s are"} available in the full status reasons.</p>}
    </div>}

    {changed && digest.resolved_exceptions_total > 0 && <p className="inline-notice">{digest.resolved_exceptions_total} previously recorded exception{digest.resolved_exceptions_total === 1 ? " has" : "s have"} been resolved.</p>}
    {error && <p className="inline-error" role="alert">{error}</p>}
    {digest.review_required && <button className="primary-button" type="button" onClick={() => void markReviewed()} disabled={saveState === "saving" || projectionVersion < 1}>{saveState === "saving" ? "Recording review…" : "Mark current state reviewed"}</button>}
  </section>;
}
