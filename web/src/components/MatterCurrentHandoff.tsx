import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import type { RecordResponsibleParty } from "../types";
import { matterOperationControlID, responsibilityLabel, selectMatterHandoff } from "./matterHandoff";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  responsibleParties?: RecordResponsibleParty[];
  onInvokeOperation?: (operation: MatterOperation) => boolean;
};

function formatDate(value?: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return "No due date recorded";
  return `Due ${new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(value))}`;
}

export function MatterCurrentHandoff({ aggregate, operations, responsibleParties = [], onInvokeOperation }: Props) {
  const operation = selectMatterHandoff(aggregate, operations);
  const currentAction = aggregate.actions.find((candidate) => aggregate.next_action.toLowerCase().includes(candidate.title.toLowerCase()));
  const storedActionOwner = responsibleParties.find((party) => party.scope === "ACTION" && party.subresource_id === currentAction?.id)?.display_name;
  const storedRecordOwner = responsibleParties.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  const owner = operation?.assigned_to?.display_name ?? storedActionOwner ?? storedRecordOwner ?? "Owner not resolved";
  const ownerLabel = responsibilityLabel(operation?.responsibility ?? (currentAction ? "PERFORMER" : "ACCOUNTABLE_OWNER"));
  const missing = aggregate.matter.missing_facts.length;
  const contradictions = aggregate.matter.contradictions.length;

  function moveToOperation() {
    if (!operation) return;
    if (onInvokeOperation?.(operation)) return;
    const target = document.getElementById(matterOperationControlID(operation));
    target?.scrollIntoView({ behavior: "smooth", block: "center" });
    target?.focus();
    if (target instanceof HTMLButtonElement) target.click();
  }

  return <section className="matter-current-handoff" aria-labelledby="matter-current-handoff-title">
    <div className="matter-handoff-copy">
      <span className="eyebrow">What needs to happen next</span>
      <h2 id="matter-current-handoff-title">Current handoff</h2>
      <h3>{aggregate.next_action}</h3>
      <p>{operation?.reason ?? "No current responsibility route was returned for this issue."}</p>
      <div className="matter-handoff-facts" aria-label="Current responsibility and timing">
        <span>{ownerLabel} <strong>{owner}</strong></span>
        <span>{formatDate(aggregate.matter.due_at)}</span>
        <span>{missing} missing information item{missing === 1 ? "" : "s"}</span>
        {contradictions > 0 && <span>{contradictions} contradiction{contradictions === 1 ? "" : "s"}</span>}
      </div>
    </div>
    <div className="matter-dominant-next" data-testid="dominant-next-action">
      {operation?.can_act
        ? <button className="primary-button" type="button" onClick={moveToOperation}>{operation.label}</button>
        : <div className="matter-readonly-next"><strong>{operation?.label ?? "No action available"}</strong><span>{operation?.reason ?? "No operation is available for your current role."}</span></div>}
    </div>
  </section>;
}
