import type { ProgramSection } from "../appRouting";
import type { ProgramAggregate } from "../types";
import { programOverviewMetrics } from "../programOverviewMetrics";

/**
 * Approved Program HTML's Requirement coverage surface, using only the
 * canonical #433 versioned Program requirement population.
 *
 * Do not infer supported/gap counts from incomplete evidence, percentages,
 * a framework catalog or submitted (not reviewed) form responses.
 */
export function ProgramRequirementCoverageSummary({
  aggregate, onNavigate,
}: {
  aggregate: ProgramAggregate;
  onNavigate: (section: ProgramSection) => void;
}) {
  const assessed = programOverviewMetrics(aggregate).find((metric) => metric.id === "assessed");
  const policy = aggregate.program.calculation_policy?.active;
  const snapshot = aggregate.current_state;
  const category = snapshot?.calculation?.categories?.find((value) => value.code === "REQUIREMENTS");
  const verified = Boolean(
    assessed && typeof assessed.value === "number" &&
    category && Number.isSafeInteger(category.eligible_count) && category.eligible_count >= 0 &&
    category.assessed_count === assessed.value && category.assessed_count <= category.eligible_count &&
    policy?.approved_at && snapshot?.calculation?.policy_revision === policy.definition.revision,
  );
  const partial = verified && (assessed?.quality === "partial" || assessed?.quality === "stale");
  const stale = assessed?.quality === "stale";

  return <section className="program-overview-coverage" aria-labelledby="program-overview-coverage-heading">
    <header className="program-overview-coverage__header">
      <div>
        <span className="eyebrow">Obligations</span>
        <h2 id="program-overview-coverage-heading">Requirement coverage</h2>
        <p>Current approved calculation and assessment coverage</p>
      </div>
      <button type="button" className="text-button" onClick={() => onNavigate("requirements-controls")}>View requirements</button>
    </header>
    {verified && category ? <>
      <div className="program-overview-coverage__totals">
        <strong>{category.assessed_count} of {category.eligible_count}</strong>
        <span>{stale ? "Out-of-date assessment" : partial ? "Coverage incomplete" : "Assessed requirements"}</span>
      </div>
      {category.eligible_count > 0
        ? <progress aria-label="Assessed requirement coverage" max={category.eligible_count} value={category.assessed_count}/>
        : <p>No eligible requirements in this calculation.</p>}
      <div className="program-overview-coverage__facts">
        <span><strong>{category.unknown_count}</strong> unknown</span>
        <span><strong>{category.excluded_count}</strong> excluded</span>
        <span>{assessed?.meta ?? "Calculation time unavailable"}</span>
      </div>
      <p className="program-overview-coverage__basis">
        Support and gap breakdowns remain unavailable without separately verified assessments.
      </p>
    </> : <div className="program-overview-coverage__unavailable" role="status">
      <strong>Coverage not assessed</strong>
      <p>{snapshot?.program_version && snapshot.program_version !== aggregate.program.version
        ? "The previous calculation does not match this Program revision."
        : "A current approved calculation is required before coverage can be shown."}</p>
      <p>Requirements remain available for review without an inferred compliance result.</p>
    </div>}
    <section className="program-overview-coverage__areas" aria-labelledby="program-overview-areas-heading">
      <h3 id="program-overview-areas-heading">By control area</h3>
      <p>Reviewed control-area results are unavailable.</p>
      <button type="button" className="text-button" onClick={() => onNavigate("requirements-controls")}>
        Open requirements
      </button>
    </section>
  </section>;
}
