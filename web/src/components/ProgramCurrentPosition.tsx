import type { ProgramOperations, ProgramOperation } from "../programOperationsApi";
import type { ProgramReviewDigest } from "../programReviewApi";
import type { ProgramAggregate } from "../types";
import type { ProgramSection } from "../appRouting";
import { ProgramAttention } from "./ProgramAttention";

type Props = {
  aggregate: ProgramAggregate;
  operations: ProgramOperations;
  digest: ProgramReviewDigest;
  onOpenOwnerChange?: () => void;
  onNavigate?: (section: ProgramSection) => void;
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
  "program.monitoring.transition",
  "program.monitoring.evaluate",
  "program.monitoring.define",
];

export function dominantProgramAction(operations: ProgramOperation[], digest: ProgramReviewDigest) {
  const executable = operations.filter((operation) => operation.can_act);
  const priority = (command: string) => { const index = actionOrder.indexOf(command); return index < 0 ? actionOrder.length : index; };
  if (!digest.review_required) {
    const withoutReview = executable.filter((operation) => operation.command !== "program.review.accept");
    if (withoutReview.length > 0) return [...withoutReview].sort((left, right) => priority(left.command) - priority(right.command))[0];
  }
  return [...executable].sort((left, right) => priority(left.command) - priority(right.command))[0];
}

function actionTarget(command: string) {
  if (command === "program.review.accept") return "program-review-panel";
  if (command === "program.evidence.assess" || command === "program.evidence.define" || command.startsWith("program.monitoring.")) return "program-evidence-panel";
  if (command === "program.safeguard.define") return "program-safeguards-panel";
  if (command === "program.requirement.add" || command === "program.applicability.decide") return "program-requirements-panel";
  if (command === "program.transition") return "program-status-panel";
  return "program-details-panel";
}

function requirementComplianceSummary(aggregate: ProgramAggregate, stale: boolean): string | null {
  const policy = aggregate.program.calculation_policy?.active;
  const result = aggregate.current_state?.calculation;
  const approvedCategory = policy?.definition.categories[0];
  const measuredCategory = result?.categories?.[0];
  if (stale || !policy?.approved_at || !result ||
      policy.definition.categories.length !== 1 ||
      approvedCategory?.code !== "REQUIREMENTS" || approvedCategory.weight_bps !== 10000 ||
      result.categories?.length !== 1 ||
      measuredCategory?.code !== "REQUIREMENTS" || measuredCategory.weight_bps !== 10000 ||
      measuredCategory.coverage_bps !== result.coverage_bps ||
      result.policy_revision !== policy.definition.revision ||
      result.unpopulated_categories?.length ||
      !Number.isInteger(result.coverage_bps) ||
      result.coverage_bps < 0 || result.coverage_bps > 10000) return null;

  const formatted = (basisPoints: number) => `${(basisPoints / 100).toLocaleString(undefined, { maximumFractionDigits: 1 })}%`;
  const coverage = formatted(result.coverage_bps);
  if (result.state !== "FINAL") return `Not assessed · ${coverage} coverage`;
  if (!Number.isInteger(result.score_bps) || result.score_bps === undefined ||
      result.score_bps < 0 || result.score_bps > 10000 ||
      measuredCategory.score_bps !== result.score_bps) return null;
  return `${formatted(result.score_bps)} · ${coverage} coverage`;
}

export function ProgramCurrentPosition({ aggregate, operations, digest, onOpenOwnerChange, onNavigate }: Props) {
  const current = aggregate.current_state;
  const assessedVersion = current?.program_version ?? 0;
  const assessmentVersionKnown = Number.isInteger(assessedVersion) && assessedVersion > 0;
  const stale = Boolean(current && assessedVersion !== aggregate.program.version);
  const ownerOperation = operations.operations.find((operation) => operation.command === "program.details.update" || operation.command === "program.assign");
  const owner = ownerOperation?.assigned_to;
  const storedOwner = operations.responsible_parties?.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  const action = dominantProgramAction(operations.operations, digest);
  const reasons = current?.reasons ?? [];
  const validCalculationTime = Boolean(current?.generated_at && Number.isFinite(Date.parse(current.generated_at)) && new Date(current.generated_at).getUTCFullYear() >= 2000);
  const calculatedAt = validCalculationTime ? new Date(current!.generated_at).toLocaleString() : "Time unavailable";
  const openIssues = current?.open_matter_count;
  const knownOpenIssues = typeof openIssues === "number" && Number.isInteger(openIssues) && openIssues >= 0;
  const requirementsCompliance = requirementComplianceSummary(aggregate, stale || !assessmentVersionKnown);
  const navigate = onNavigate ?? ((section: ProgramSection) => { window.location.hash = `#programs/${encodeURIComponent(aggregate.program.id)}/${section}`; });

  function goToAction() {
    if (!action) return;
	if (action.command === "program.assign" && onOpenOwnerChange) {
	  navigate("overview");
	  onOpenOwnerChange();
	  return;
	}
	const command = action.command;
	navigate(command.startsWith("program.monitoring.") ? "monitoring" : command.startsWith("program.evidence.") ? "evidence-results" : command === "program.safeguard.define" || command === "program.requirement.add" || command === "program.applicability.decide" ? "requirements-controls" : "overview");
	window.requestAnimationFrame(() => {
      const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
      document.getElementById(actionTarget(command))?.scrollIntoView?.({ behavior: reducedMotion ? "auto" : "smooth", block: "start" });
    });
  }

  return <section className="program-current-position" aria-labelledby="program-current-position-heading">
    <div className="program-position-header"><div>
      <span className="eyebrow">Current position</span>
      <h2 id="program-current-position-heading">{!current || !assessmentVersionKnown ? "Unknown" : stale ? "Out of date" : aggregate.state_label === "Evidence incomplete" ? "Supporting information needs review" : aggregate.state_label}</h2>
      <p>{!current ? "No status calculation." : <>{!assessmentVersionKnown ? "Assessment version unavailable" : stale ? `Assessment v${assessedVersion} · Program v${aggregate.program.version}` : `Program v${aggregate.program.version}`}{validCalculationTime ? <> · <time dateTime={current.generated_at}>{calculatedAt}</time></> : null}</>}</p>
      </div><div className="program-dominant-next">
        {action ? <button data-testid="program-dominant-action" className="primary-button" type="button" onClick={goToAction}>{action.label}</button> : <small>No available action.</small>}
      </div></div><div>
      <div className="program-position-facts">
        <span><strong>Owner</strong> {owner?.display_name ?? storedOwner ?? (aggregate.program.owner_principal_id ? "Recorded Program owner unavailable" : "Program owner not assigned")}</span>
        <span><strong>Open issues</strong> {knownOpenIssues ? `${openIssues}${stale ? " (last calculation)" : ""}` : "Unknown"}</span>
        <span><strong>Requirements</strong> {aggregate.requirements.filter((requirement) => requirement.status === "APPROVED").length}</span>
        {requirementsCompliance && <span><strong>Requirement compliance</strong> {requirementsCompliance}</span>}
      </div>
      {reasons.length > 0 ? <div className="program-position-reasons"><h3>{stale ? "Follow-up from the last calculation" : "What needs attention"}</h3><ProgramAttention aggregate={aggregate} onNavigate={navigate}/></div> : <p>{!current || !Array.isArray(current.reasons) ? "Status reasons unavailable." : stale ? "No prior status reasons." : "No status reasons."}</p>}
    </div>
  </section>;
}
