import type { ProgramAggregate } from "../types";
import "./program-coverage.css";

export function ProgramCoverage({ aggregate }: { aggregate: ProgramAggregate }) {
  const snapshot = aggregate.current_state; const policy = aggregate.program.calculation_policy?.active; const calculation = snapshot?.calculation;
  const requirements = calculation?.categories?.filter((category) => category.code === "REQUIREMENTS") ?? [];
  const category = requirements.length === 1 ? requirements[0] : undefined;
  const validCount = (value: unknown): value is number => typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
  const usable = Boolean(policy?.approver_id && policy.checksum && validCount(policy.number) && policy.number > 0 && policy.approved_at && Number.isFinite(Date.parse(policy.approved_at)) && policy.definition?.revision && calculation?.policy_revision && ["NOT_ASSESSED", "PROVISIONAL", "FINAL"].includes(calculation.state) && snapshot && validCount(snapshot.projection_version) && snapshot.projection_version > 0 && validCount(snapshot.program_version) && snapshot.program_version > 0 && Number.isFinite(Date.parse(snapshot.generated_at)) && validCount(aggregate.program.version) && aggregate.program.version > 0 && category && validCount(category.eligible_count) && category.eligible_count > 0 && validCount(category.assessed_count) && category.assessed_count <= category.eligible_count && (calculation.state !== "NOT_ASSESSED" || category.assessed_count === 0) && validCount(category.unknown_count) && category.unknown_count === category.eligible_count - category.assessed_count && validCount(category.excluded_count));
  if (!usable || !snapshot || !policy || !calculation || !category) return <section className="v-panel v-program-coverage"><h2>Requirement coverage</h2><p>Coverage unavailable.</p></section>;
  const stale = snapshot.program_version !== aggregate.program.version || calculation.policy_revision !== policy.definition.revision;
  const percent = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(category.assessed_count / category.eligible_count * 100);
  const asOf = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(snapshot.generated_at));
  return <section className="v-panel v-program-coverage"><h2>Requirement coverage</h2><p>{stale && <><span>Out of date</span> · </>}As of {asOf} UTC</p><div className="v-program-coverage-value"><strong>{percent}%</strong><span>{category.assessed_count} of {category.eligible_count} requirements assessed</span></div><progress aria-label="Assessed requirements" value={category.assessed_count} max={category.eligible_count}/><p>{category.unknown_count} unknown{category.excluded_count > 0 && ` · ${category.excluded_count} not applicable`}</p><p>Control area results unavailable.</p></section>;
}
