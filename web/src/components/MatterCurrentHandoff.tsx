import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import type { RecordResponsibleParty } from "../types";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  responsibleParties?: RecordResponsibleParty[];
};

function formatDate(value?: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return "No due date recorded";
  return `Due ${new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(value))}`;
}

function selectCurrentOperation(aggregate: MatterAggregate, operations: MatterOperation[]) {
  const action = aggregate.actions.find((candidate) => aggregate.next_action.toLowerCase().includes(candidate.title.toLowerCase()));
  return operations.find((operation) => operation.command === "matter.action.transition" && operation.subresource_id === action?.id)
    ?? operations.find((operation) => operation.command === "matter.action.transition")
    ?? operations.find((operation) => operation.can_act)
    ?? operations[0];
}

export function MatterCurrentHandoff({ aggregate, operations, responsibleParties = [] }: Props) {
  const operation = selectCurrentOperation(aggregate, operations);
  const currentAction = aggregate.actions.find((candidate) => aggregate.next_action.toLowerCase().includes(candidate.title.toLowerCase()));
  const storedActionOwner = responsibleParties.find((party) => party.scope === "ACTION" && party.subresource_id === currentAction?.id)?.display_name;
  const storedRecordOwner = responsibleParties.find((party) => party.scope === "RECORD" && party.responsibility === "ACCOUNTABLE_OWNER")?.display_name;
  const owner = operation?.assigned_to?.display_name ?? storedActionOwner ?? storedRecordOwner ?? "Owner not resolved";
  const missing = aggregate.matter.missing_facts.length;
  const contradictions = aggregate.matter.contradictions.length;

  function moveToOperation() {
    const target = operation?.subresource_id
      ? document.getElementById(`matter-operation-${operation.command}-${operation.subresource_id}`)
      : document.getElementById(`matter-operation-${operation?.command}`);
    target?.scrollIntoView({ behavior: "smooth", block: "center" });
    target?.querySelector<HTMLElement>("button, input, select, textarea")?.focus();
  }

  return <section className="matter-current-handoff" aria-labelledby="matter-current-handoff-title">
    <div className="matter-handoff-copy">
      <span className="eyebrow">What needs to happen next</span>
      <h2 id="matter-current-handoff-title">Current handoff</h2>
      <h3>{aggregate.next_action}</h3>
      <p>{operation?.reason ?? "No current responsibility route was returned for this issue."}</p>
      <div className="matter-handoff-facts" aria-label="Current responsibility and timing">
        <span>Assigned to <strong>{owner}</strong></span>
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
