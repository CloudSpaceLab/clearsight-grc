import type { ProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  digest: ProgramReviewDigest | null;
  onReload: () => void;
};

function dateTime(value?: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return "Unavailable";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" })
    .format(new Date(value));
}

/**
 * Approved Program Activity view: the existing review digest is a bounded
 * since-checkpoint change summary, not a complete transaction/event timeline.
 * Never turn a missing/old digest into a "no changes" assertion.
 */
export function ProgramActivityPanel({ aggregate, digest, onReload }: Props) {
  const program = aggregate.program;
  const projection = aggregate.current_state?.projection_version ?? 0;
  const current = Boolean(digest &&
    digest.program_id === program.id &&
    digest.current_program_version === program.version &&
    digest.current_projection_version === projection);
  const noBaseline = current && digest?.state === "NO_BASELINE";
  const changed = current && digest?.state === "CHANGED";
  const unchanged = current && digest?.state === "CURRENT";
  const reliable = noBaseline || changed || unchanged;
  const changes = changed ? digest!.changes : [];

  return <article className="program-record-panel program-wide-panel program-activity" aria-labelledby="program-activity-heading">
    <div className="program-panel-heading">
      <div>
        <span className="eyebrow">Activity</span>
        <h2 id="program-activity-heading">Program changes and review history</h2>
        <p>Recorded changes since the last review</p>
      </div>
    </div>
    <dl className="program-history-facts">
      <div><dt>Program version</dt><dd>{program.version}</dd></div>
      <div><dt>Created</dt><dd><time dateTime={program.created_at}>{dateTime(program.created_at)}</time></dd></div>
      <div><dt>Last changed</dt><dd><time dateTime={program.updated_at}>{dateTime(program.updated_at)}</time></dd></div>
      <div><dt>Status calculated</dt><dd>{aggregate.current_state?.generated_at
        ? <time dateTime={aggregate.current_state.generated_at}>{dateTime(aggregate.current_state.generated_at)}</time>
        : "Unavailable"}</dd></div>
      <div><dt>Last accepted review</dt><dd>{reliable && digest?.checkpoint?.accepted_at
        ? <time dateTime={digest.checkpoint.accepted_at}>{dateTime(digest.checkpoint.accepted_at)}</time>
        : "Not recorded"}</dd></div>
    </dl>
    {!reliable && <div className="program-activity__notice" role="status">
      <strong>Review changes unavailable</strong>
      <p>The latest review status must match this Program revision before changes can be shown.</p>
      <button className="secondary-button" type="button" onClick={onReload}>Reload review status</button>
    </div>}
    {noBaseline && <div className="program-activity__notice" role="status">
      <strong>No accepted review yet</strong>
      <p>The current record is available; a change comparison requires a review checkpoint.</p>
    </div>}
    {unchanged && <div className="program-activity__notice" role="status">
      <strong>No recorded changes since the last review</strong>
    </div>}
    {changed && <section className="program-activity__changes" aria-label="Changes since last review">
      <div className="program-activity__summary">
        <strong>{digest!.changes_total} change{digest!.changes_total === 1 ? "" : "s"} since the last review</strong>
        {digest!.review_required && <span className="program-activity__review-status">Review required</span>}
      </div>
      {changes.length ? <ol className="program-activity__list">
        {changes.map((change, index) =>
          <li key={`${change.kind}-${change.object_type ?? ""}-${change.object_id ?? ""}-${index}`}>
            <strong>{change.kind.replaceAll("_", " ").toLowerCase().replace(/(^|\\s)\\S/g, (part) => part.toUpperCase())}</strong>
            <span>{change.summary}</span>
          </li>)}
      </ol> : <p>Individual changes were not supplied by the review service.</p>}
      {(digest!.changes_omitted > 0 || digest!.history_truncated) && <p role="status">
        Only part of the change history was returned. Earlier events are not shown here.
      </p>}
      {digest!.new_exceptions_total > 0 && <p>{digest!.new_exceptions_total} new exception{digest!.new_exceptions_total === 1 ? "" : "s"} recorded.</p>}
      {digest!.resolved_exceptions_total > 0 && <p>{digest!.resolved_exceptions_total} exception{digest!.resolved_exceptions_total === 1 ? "" : "s"} resolved.</p>}
    </section>}
  </article>;
}
