import { useId, useRef, useState, type ReactNode } from "react";
import type { GovernanceReadState } from "../governanceAdminApi";
import type { GovernanceDelegationItem, GovernanceEntity, GovernanceLoadState, GovernancePolicyItem, GovernanceView } from "./GovernanceAdminWorkspace";
import { Button, MetricCard, Notice, SelectableRecord, StatusBadge, Surface } from "./ui";
import { humanAccessLabel } from "./access/peopleAccessPresentation";
import { readGovernancePolicyDefinition, type GovernanceRuleFacts } from "./governancePolicyPresentation";
import "./access/people-access.css";
import "../people-governance.css";

type Props = {
  view: GovernanceView;
  policies: GovernancePolicyItem[];
  delegations: GovernanceDelegationItem[];
  currentEntity?: GovernanceEntity;
  loadState: GovernanceLoadState;
  policiesState?: GovernanceReadState;
  delegationsState?: GovernanceReadState;
  selectedPolicyID?: string;
  onSelectPolicy?: (id: string | undefined) => void;
  selectedDelegationID?: string;
  onSelectDelegation?: (id: string | undefined) => void;
  creationAction?: ReactNode;
  onOpenRoutingChecks?: () => void;
  renderPolicyDetail: (policy: GovernancePolicyItem) => ReactNode;
  renderDelegationDetail: (delegation: GovernanceDelegationItem) => ReactNode;
};

export function PeopleGovernanceView({ view, policies, delegations, currentEntity, loadState, policiesState, delegationsState, selectedPolicyID, onSelectPolicy, selectedDelegationID, onSelectDelegation, creationAction, onOpenRoutingChecks, renderPolicyDetail, renderDelegationDetail }: Props) {
  const [localPolicyID, setLocalPolicyID] = useState<string>();
  const [localDelegationID, setLocalDelegationID] = useState<string>();
  const detailRef = useRef<HTMLElement>(null);
  const detailHeading = useId();
  const isPolicies = view === "policies";
  const state = loadState === "restricted" ? "restricted" : (isPolicies ? policiesState : delegationsState) ?? (loadState === "unavailable" ? "unavailable" : "ready");
  const loaded = isPolicies ? policies : delegations;
  const records = state === "restricted" ? [] : loaded.slice(0, 50);
  const selectedID = isPolicies ? selectedPolicyID : selectedDelegationID;
  const localID = isPolicies ? localPolicyID : localDelegationID;
  const selectionControlled = isPolicies ? onSelectPolicy !== undefined : onSelectDelegation !== undefined;
  const effectiveID = selectedID ?? (!selectionControlled && records.some((record) => record.id === localID) ? localID : records[0]?.id);
  const selected = records.find((record) => record.id === effectiveID);
  const title = isPolicies ? "Approval routes" : "Delegations";
  const noun = isPolicies ? "routing policies" : "delegations";
  const recordNoun = isPolicies ? "policy" : "delegation";
  const hasKnownCount = state === "ready" || (state === "unavailable" && records.length > 0);
  const quality = state === "ready" && loadState !== "loading" ? "partial" : records.length ? "stale" : "unknown";
  const count = (predicate?: (record: GovernancePolicyItem | GovernanceDelegationItem) => boolean) => hasKnownCount && (loadState !== "loading" || records.length > 0) ? (predicate ? records.filter(predicate).length : records.length) : "—";
  const qualityLabel = state === "ready" && loadState !== "loading" ? "Loaded" : records.length ? "Previous list" : "Unavailable";

  function select(id: string | undefined) {
    if (isPolicies) { setLocalPolicyID(id); onSelectPolicy?.(id); }
    else { setLocalDelegationID(id); onSelectDelegation?.(id); }
    if (id && window.matchMedia("(max-width: 1024px)").matches) requestAnimationFrame(() => {
      detailRef.current?.focus({ preventScroll: true });
      detailRef.current?.scrollIntoView({ block: "start", behavior: "instant" });
    });
  }

  return <>
    <section className="people-access-summary people-governance-summary" aria-label={`${title} summary`}>
      <MetricCard label={isPolicies ? "Policies" : "Delegations"} value={count()} quality={quality} qualityLabel={qualityLabel}/>
      <MetricCard label="Recorded active" value={count((record) => record.status === "ACTIVE")} quality={quality} qualityLabel={qualityLabel}/>
      <MetricCard label="Awaiting approval" value={count((record) => ["PENDING_APPROVAL", "IN_REVIEW"].includes(record.status))} quality={quality} qualityLabel={qualityLabel}/>
      <MetricCard label="Drafts" value={count((record) => record.status === "DRAFT")} quality={quality} qualityLabel={qualityLabel}/>
    </section>
    <div className="people-access-layout people-governance-layout">
      <section className="people-access-list-panel" aria-label={`${title} list`}><Surface>
        <header className="people-access-panel-heading"><div><h2>{title} <span className="people-access-count">{count()}</span></h2><p>{currentEntity?.label || "Current legal entity"} · Up to 50 {noun}.</p></div><div className="people-governance-list-actions">{isPolicies && onOpenRoutingChecks && <Button variant="quiet" onPress={onOpenRoutingChecks}>Routing checks</Button>}{creationAction}</div></header>
        <div className="people-governance-layout__records">
          {records.map((record) => {
            const policy = isPolicies ? record as GovernancePolicyItem : undefined;
            const delegation = isPolicies ? undefined : record as GovernanceDelegationItem;
            const name = policy ? policy.name : `${delegation!.from.label} to ${delegation!.to.label}`;
            return <SelectableRecord key={record.id} variant="list" title={name} metadata={policy ? `Policy version ${policy.currentVersion} · Record version ${policy.version}` : `${humanAccessLabel(delegation!.responsibility)} · ${governanceDateWindow(delegation!.startsAt, delegation!.endsAt)}`}
              aria-label={`Inspect ${recordNoun} ${name}`} isSelected={effectiveID === record.id} onPress={() => select(record.id)}
              startContent={<span className="people-access-avatar" aria-hidden="true"><GovernanceGlyph delegation={!isPolicies}/></span>}
              endContent={<StatusBadge>{humanAccessLabel(record.status)}</StatusBadge>}/>;
          })}
          {!records.length && <div className="people-access-empty"><h3>{state === "restricted" ? `${title} access restricted` : loadState === "loading" ? `Loading ${noun}…` : state === "unavailable" ? `${title} unavailable` : `No ${noun} returned`}</h3><p>{state === "restricted" ? `Your current access does not allow these ${noun}.` : loadState === "loading" ? `Checking ${noun} for the current legal entity.` : state === "unavailable" ? `Reload to check ${noun} again.` : `No ${noun} were returned for the current legal entity.`}</p></div>}
        </div>
        <footer className="people-access-list-footer"><span>{count() === "—" ? "Count unavailable" : `${records.length} ${noun} in this list`}</span>{loaded.length > 50 && state !== "restricted" && <span>More records are available.</span>}</footer>
      </Surface></section>
      <aside ref={detailRef} tabIndex={-1} className="people-access-detail people-governance-layout__detail" aria-labelledby={detailHeading}><Surface>
        {selected ? <>
          <header className="people-access-detail__heading"><span className="people-access-kicker">{isPolicies ? "Routing policy" : "Delegation record"}</span><h2 id={detailHeading}>{isPolicies ? (selected as GovernancePolicyItem).name : `${(selected as GovernanceDelegationItem).from.label} to ${(selected as GovernanceDelegationItem).to.label}`}</h2><p><StatusBadge>{humanAccessLabel(selected.status)}</StatusBadge></p></header>
          <div className="people-access-detail__body">{isPolicies ? renderPolicyDetail(selected as GovernancePolicyItem) : renderDelegationDetail(selected as GovernanceDelegationItem)}</div>
        </> : <div className="people-access-empty"><h2 id={detailHeading}>{state === "restricted" ? "Record access restricted" : state === "unavailable" ? "Record unavailable" : selectedID ? `${isPolicies ? "Policy" : "Delegation"} not in this list` : `Select a ${recordNoun}`}</h2><p>{state === "restricted" ? `Your current access does not allow ${recordNoun} details.` : state === "unavailable" ? "Reload the list to check the record." : selectedID ? "This record was not returned in the current list." : `Choose a ${recordNoun} to review its dates and approval history.`}</p>{selectedID && state === "ready" && <Button onPress={() => select(undefined)}>Return to listed {noun}</Button>}</div>}
      </Surface></aside>
    </div>
  </>;
}

function GovernanceGlyph({ delegation }: { delegation: boolean }) {
  return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{delegation ? <><circle cx="8" cy="7" r="3"/><path d="M2 20v-2a6 6 0 0 1 12 0v2M15 7h7m-3-3 3 3-3 3M17 15h5m-3-3 3 3-3 3"/></> : <><rect x="3" y="3" width="6" height="5" rx="1"/><rect x="15" y="16" width="6" height="5" rx="1"/><path d="M6 8v10h9M9 5h9v11"/></>}</svg>;
}

export function governanceDateWindow(from?: string, until?: string) {
  return `${from ? governanceDate(from) : "Not scheduled"} to ${until ? governanceDate(until) : "No end date"}`;
}

export function governanceDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
}

export function GovernancePolicyDefinition({ policy }: { policy: GovernancePolicyItem }) {
  const definition = readGovernancePolicyDefinition(policy.definition);
  const [expanded, setExpanded] = useState(false);
  const definitionID = useId();
  return <section className="people-governance-definition">
    <h3>Rules · Policy version {policy.currentVersion}</h3>
    {definition.reviewable ? <>
      <p>Rules apply by scope and priority. Approval authority is checked for each decision.</p>
      <ul className="people-governance-rules">{definition.rules.map((rule) => <li key={rule.id}>
        <h4>{humanAccessLabel(rule.responsibility)}</h4>
        <PolicyRuleFacts rule={rule} entity={policy.legalEntity}/>
      </li>)}</ul>
    </> : <Notice tone="warning">{policy.definition == null ? "No policy definition was returned." : "The policy definition could not be fully read."} Approval is unavailable until the rules can be reviewed.</Notice>}
    {definition.hasAdditionalSettings && <p>Additional settings are included in the full policy definition.</p>}
    <dl className="people-access-facts"><dt>Definition checksum</dt><dd>{policy.checksum || "Not returned"}</dd></dl>
    {definition.fullDefinition && <>
      <Button variant="quiet" aria-expanded={expanded} aria-controls={definitionID} onPress={() => setExpanded((value) => !value)}>{expanded ? "Hide policy definition" : "View full policy definition"}</Button>
      {expanded && <pre id={definitionID} className="people-governance-definition__source" role="region" tabIndex={0} aria-label="Full policy definition">{definition.fullDefinition}</pre>}
    </>}
  </section>;
}

function PolicyRuleFacts({ rule, entity }: { rule: GovernanceRuleFacts; entity: GovernanceEntity }) {
  const noRestriction = (value: string) => !value || value === "*";
  const recordType = ({ PROGRAM: "Programs", MATTER: "Issues and changes", VENDOR_RELATIONSHIP: "Vendor relationships" } as Record<string, string>)[rule.objectType] || humanAccessLabel(rule.objectType);
  return <dl className="people-access-facts">
    {rule.selector && <><dt>{({ PRINCIPAL: "Person reference", POSITION: "Position reference", ROLE: "Role" } as Record<string, string>)[rule.selector.kind]}</dt><dd>{rule.selector.ref}</dd></>}
    {rule.lifecycle && <><dt>Lifecycle</dt><dd>{humanAccessLabel(rule.lifecycle.type)}</dd><dt>State</dt><dd>{humanAccessLabel(rule.lifecycle.state)}</dd>{rule.lifecycle.subtype && <><dt>Subtype</dt><dd>{rule.lifecycle.subtype}</dd></>}</>}
    <dt>Legal entity</dt><dd>{noRestriction(rule.legalEntityID) ? "No entity restriction in this rule" : rule.legalEntityID === entity.id ? entity.label : rule.legalEntityID}</dd>
    <dt>Records</dt><dd>{noRestriction(rule.objectType) ? "All record types" : recordType}</dd>
    {!noRestriction(rule.objectID) && <><dt>Record reference</dt><dd>{rule.objectID}</dd></>}
    <dt>Decision</dt><dd>{noRestriction(rule.decisionType) ? "Any decision type" : rule.decisionType}</dd>
    <dt>Minimum materiality</dt><dd>{rule.minimumMateriality}</dd>
    <dt>Priority</dt><dd>{rule.priority}</dd>
    <dt>Rule reference</dt><dd>{rule.id}</dd>
  </dl>;
}
