import type { ProgramOperations, ProgramOperation } from "../programOperationsApi";
import type { ProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperations;
  digest: ProgramReviewDigest;
};

const actionOrder = [
  "program.review.accept",
  "program.evidence.assess",
  "program.applicability.decide",
  "program.transition",
  "program.requirement.add",
  "program.safeguard.define",
  "program.evidence.define",
  "program.details.update",
  "program.assign",
  "program.approval-authority.assign",
];

function dominantAction(operations: ProgramOperation[], digest: ProgramReviewDigest) {
  const executable = operations.filter((operation) => operation.can_act);
  if (!digest.review_required) {
    const withoutReview = executable.filter((operation) => operation.command !== "program.review.accept");
    if (withoutReview.length > 0) return [...withoutReview].sort((left, right) => actionOrder.indexOf(left.command) - actionOrder.indexOf(right.command))[0];
  }
  return [...executable].sort((left, right) => actionOrder.indexOf(left.command) - actionOrder.indexOf(right.command))[0];
}

function actionTarget(command: string) {
  if (command === "program.review.accept") return "program-review-panel";
  if (command === "program.evidence.assess" || command === "program.evidence.define") return "program-evidence-panel";
  if (command === "program.safeguard.define") return "program-safeguards-panel";
  if (command === "program.requirement.add" || command === "program.applicability.decide") return "program-requirements-panel";
  if (command === "program.transition") return "program-status-panel";
  return "program-details-panel";
}

export function ProgramCurrentPosition({ aggregate, operations, digest }: Props) {
  const current = aggregate.current_state;
  const assessedVersion = current?.program_version ?? 0;
  const stale = !current || assessedVersion < aggregate.program.version;
  const ownerOperation = operations.operations.find((operation) => operation.command === "program.details.update" || operation.command === "program.assign");
  const owner = ownerOperation?.assigned_to;
  const storedOwner = operations.responsible_parties?.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  const action = dominantAction(operations.operations, digest);
  const reasons = current?.reasons ?? [];

  function goToAction() {
    if (!action) return;
	const target = document.getElementById(actionTarget(action.command));
	if (target && typeof target.scrollIntoView === "function") target.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  return <section className="program-current-position" aria-labelledby="program-current-position-heading">
    <div>
      <span className="eyebrow">Current position</span>
      <h2 id="program-current-position-heading">{stale ? "Updating status" : aggregate.state_label}</h2>
      <p>{stale ? `Assessed version ${assessedVersion} · current version ${aggregate.program.version}` : `Calculated from Program version ${aggregate.program.version} and projection ${current?.projection_version ?? 0}.`}</p>
      <div className="program-position-facts">
        <span><strong>Owner</strong> {owner?.display_name ?? storedOwner ?? (aggregate.program.owner_principal_id ? "Recorded Program owner unavailable" : "Program owner not assigned")}</span>
        <span><strong>Open issues</strong> {current?.open_matter_count ?? 0}</span>
        <span><strong>Requirements</strong> {aggregate.requirements.filter((requirement) => requirement.status === "APPROVED").length}</span>
      </div>
      {reasons.length > 0 ? <div className="program-position-reasons"><h3>Why this status</h3><ul>{reasons.map((reason) => <li key={`${reason.code}-${reason.object_id ?? ""}`}>{reason.summary}</li>)}</ul></div> : <p>No status exceptions are recorded for the latest calculation.</p>}
    </div>
    <div className="program-dominant-next">
      {action ? <><button data-testid="program-dominant-action" className="primary-button" type="button" onClick={goToAction}>{action.label}</button><small>{action.reason}</small></> : <div className="program-readonly-next"><strong>No change is assigned to you</strong><span>Current Program details and responsibilities remain visible.</span></div>}
    </div>
  </section>;
}
