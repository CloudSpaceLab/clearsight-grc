import { useState, type FormEvent } from "react";

export type GovernanceLoadState = "loading" | "ready" | "degraded" | "unavailable";

export type GovernanceParty = { id: string; label: string };
export type GovernanceEntity = GovernanceParty;
export type GovernancePerson = GovernanceParty & { legalEntity: GovernanceEntity; status?: string };

export type GovernancePolicyItem = {
  id: string;
  code: string;
  name: string;
  status: string;
  legalEntity: GovernanceEntity;
  currentVersion: number;
  version: number;
  effectiveFrom?: string;
  effectiveUntil?: string;
  maker: GovernanceParty;
  checker?: GovernanceParty;
  latestDecision?: string;
};

export type GovernanceDelegationItem = {
  id: string;
  status: string;
  legalEntity: GovernanceEntity;
  from: GovernanceParty;
  to: GovernanceParty;
  responsibility: string;
  startsAt: string;
  endsAt: string;
  reason: string;
  version: number;
  maker: GovernanceParty;
  checker?: GovernanceParty;
  latestDecision?: string;
};

export type GovernanceResponsibility = "PERFORMER" | "ACCOUNTABLE_OWNER" | "PROPOSER" | "REVIEWER" | "INDEPENDENT_CHALLENGER" | "AUTHORIZER" | "SIGNATORY" | "TRANSMITTER" | "ACKNOWLEDGEMENT_RECORDER" | "ESCALATION_OWNER";

export type CreateGovernanceDelegationInput = {
  legalEntityId: string;
  fromPrincipalId: string;
  toPrincipalId: string;
  responsibility: GovernanceResponsibility;
  scope: { kind: "LEGAL_ENTITY" };
  startsAt: string;
  endsAt: string;
  reason: string;
};

export type GovernancePolicyActionInput = { policyId: string; action: "submit" | "approve" | "retire"; expectedVersion: number; rationale?: string };
export type GovernanceDelegationActionInput = { delegationId: string; action: "submit" | "approve" | "revoke"; expectedVersion: number; rationale?: string };

export type GovernanceAdminWorkspaceProps = {
  policies: GovernancePolicyItem[];
  delegations: GovernanceDelegationItem[];
  eligiblePeople: GovernancePerson[];
  actorId: string;
  canConfigure: boolean;
  delegationCreationAvailable?: boolean;
  loadState: GovernanceLoadState;
  degradedReason?: string;
  createDelegation: (input: CreateGovernanceDelegationInput) => Promise<void>;
  policyAction: (input: GovernancePolicyActionInput) => Promise<void>;
  delegationAction: (input: GovernanceDelegationActionInput) => Promise<void>;
};

export function GovernanceAdminWorkspace({ policies, delegations, eligiblePeople, actorId, canConfigure, delegationCreationAvailable = true, loadState, degradedReason, createDelegation, policyAction, delegationAction }: GovernanceAdminWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [legalEntityId, setLegalEntityId] = useState("");
  const [fromPrincipalId, setFromPrincipalId] = useState("");
  const [toPrincipalId, setToPrincipalId] = useState("");
  const [responsibility, setResponsibility] = useState<GovernanceResponsibility | "">("");
  const [startsAt, setStartsAt] = useState("");
  const [endsAt, setEndsAt] = useState("");
  const [reason, setReason] = useState("");
  const entityOptions = uniqueEntities(eligiblePeople.map((person) => person.legalEntity));
  const peopleForEntity = eligiblePeople.filter((person) => person.legalEntity.id === legalEntityId);
  const mutationsEnabled = canConfigure && loadState === "ready";
  const delegationDraftEnabled = mutationsEnabled && delegationCreationAvailable;
  const visiblePolicies = policies.slice(0, INVENTORY_LIMIT);
  const visibleDelegations = delegations.slice(0, INVENTORY_LIMIT);

  async function runPolicyAction(input: GovernancePolicyActionInput) {
    setBusy(`policy:${input.policyId}`);
    setError("");
    try {
      await policyAction(input);
    } catch {
      setError("The policy action could not be completed. Refresh the governance inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function runDelegationAction(input: GovernanceDelegationActionInput) {
    setBusy(`delegation:${input.delegationId}`);
    setError("");
    try {
      await delegationAction(input);
    } catch {
      setError("The delegation action could not be completed. Refresh the governance inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function create(event: FormEvent) {
    event.preventDefault();
    if (!legalEntityId || !fromPrincipalId || !toPrincipalId || !responsibility || !startsAt || !endsAt || !reason.trim()) return;
    if (fromPrincipalId === toPrincipalId) {
      setError("Choose a different person to receive the delegated responsibility.");
      return;
    }
    const start = new Date(startsAt);
    const end = new Date(endsAt);
    if (!Number.isFinite(start.valueOf()) || !Number.isFinite(end.valueOf()) || start >= end) {
      setError("The delegation end must be after its start. Check both dates and times.");
      return;
    }
    setBusy("create-delegation");
    setError("");
    try {
      await createDelegation({ legalEntityId, fromPrincipalId, toPrincipalId, responsibility, scope: { kind: "LEGAL_ENTITY" }, startsAt: start.toISOString(), endsAt: end.toISOString(), reason: reason.trim() });
      setFromPrincipalId("");
      setToPrincipalId("");
      setResponsibility("");
      setStartsAt("");
      setEndsAt("");
      setReason("");
    } catch {
      setError("The delegation draft could not be created. Your entries remain on this screen; refresh the inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  return <section aria-labelledby="governance-admin-title">
    <header>
      <span className="eyebrow">Configure · governance</span>
      <h2 id="governance-admin-title">Governance policies and delegations</h2>
      <p>Review the current legal-entity routes and time-bound authority cover before taking the next governed action.</p>
    </header>
    {loadState === "loading" && <p role="status">Loading the current legal-entity governance inventory. Changes remain unavailable until the current authority route is confirmed.</p>}
    {loadState === "degraded" && <p role="alert">{degradedReason || "Current approval authority could not be confirmed."} Existing records remain available, but changes are disabled until authority can be confirmed.</p>}
    {loadState === "unavailable" && (policies.length > 0 || delegations.length > 0) && <p role="alert">{degradedReason || "The latest governance inventory could not be loaded."} Previously loaded records remain available, but changes are disabled.</p>}
    {loadState === "unavailable" && policies.length === 0 && delegations.length === 0 && <p role="alert">The current legal-entity governance population could not be loaded. Refresh this page to try again; no changes are available.</p>}
    {loadState === "ready" && !canConfigure && <p>You can review this governance inventory, but your current access does not allow policy or delegation changes.</p>}
    {error && <p role="alert">{error}</p>}

    <section aria-labelledby="create-governance-delegation">
      <h3 id="create-governance-delegation">Create whole-entity delegation</h3>
      <p>Create a time-bound draft for one legal entity. A different authorized person must approve it before the delegated responsibility becomes active.</p>
      {!delegationCreationAvailable && <p role="status">Delegation creation is unavailable until the bank directory can confirm who holds the selected responsibility in this legal entity. Existing delegation actions remain available.</p>}
      <form onSubmit={(event) => void create(event)}>
        <label>Legal entity<select required value={legalEntityId} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => { setLegalEntityId(event.target.value); setFromPrincipalId(""); setToPrincipalId(""); }}><option value="">Choose legal entity</option>{entityOptions.map((entity) => <option key={entity.id} value={entity.id}>{entity.label}</option>)}</select></label>
        <label>Person giving authority<select required value={fromPrincipalId} disabled={!delegationDraftEnabled || !legalEntityId || busy !== ""} onChange={(event) => setFromPrincipalId(event.target.value)}><option value="">Choose person</option>{peopleForEntity.map((person) => <option key={person.id} value={person.id}>{person.label}</option>)}</select></label>
        <label>Person receiving authority<select required value={toPrincipalId} disabled={!delegationDraftEnabled || !legalEntityId || busy !== ""} onChange={(event) => setToPrincipalId(event.target.value)}><option value="">Choose person</option>{peopleForEntity.filter((person) => person.id !== fromPrincipalId).map((person) => <option key={person.id} value={person.id}>{person.label}</option>)}</select></label>
        <label>Responsibility<select required value={responsibility} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setResponsibility(event.target.value as GovernanceResponsibility)}><option value="">Choose responsibility</option>{RESPONSIBILITIES.map((value) => <option key={value} value={value}>{humanize(value)}</option>)}</select></label>
        <label>Starts at<input required type="datetime-local" value={startsAt} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setStartsAt(event.target.value)}/></label>
        <label>Ends at<input required type="datetime-local" value={endsAt} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setEndsAt(event.target.value)}/></label>
        <label>Reason<textarea required value={reason} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setReason(event.target.value)}/></label>
        <button type="submit" disabled={!delegationDraftEnabled || busy !== "" || !legalEntityId || !fromPrincipalId || !toPrincipalId || !responsibility || !startsAt || !endsAt || !reason.trim()}>{busy === "create-delegation" ? "Creating draft…" : "Create delegation draft"}</button>
      </form>
    </section>

    <section aria-labelledby="governance-policy-inventory">
      <h3 id="governance-policy-inventory">Routing policies</h3>
      {policies.length > INVENTORY_LIMIT && <p>Showing the first {INVENTORY_LIMIT} routing policies. More records are available.</p>}
      {loadState === "ready" && policies.length === 0 && <p>No routing policies were returned for the current legal-entity population.</p>}
      {visiblePolicies.map((policy) => <article key={policy.id}>
        <header><div><span className="eyebrow">{policy.code}</span><h4>{policy.name}</h4></div><strong>{humanize(policy.status)}</strong></header>
        <dl>
          <div><dt>Legal entity</dt><dd>{policy.legalEntity.label}</dd></div>
          <div><dt>Versions</dt><dd>Policy version {policy.currentVersion} · record version {policy.version}</dd></div>
          <div><dt>Effective dates</dt><dd>{dateWindow(policy.effectiveFrom, policy.effectiveUntil)}</dd></div>
        </dl>
        <p>Made by {policy.maker.label} · checked by {policy.checker?.label ?? "Not checked yet"}</p>
        {policy.latestDecision && <p>Latest decision: {policy.latestDecision}</p>}
        <PolicyNextAction policy={policy} actorId={actorId} canConfigure={canConfigure} mutationDisabled={!mutationsEnabled || busy !== ""} busy={busy === `policy:${policy.id}`} onAction={runPolicyAction}/>
      </article>)}
    </section>

    <section aria-labelledby="governance-delegation-inventory">
      <h3 id="governance-delegation-inventory">Whole-entity delegations</h3>
      {delegations.length > INVENTORY_LIMIT && <p>Showing the first {INVENTORY_LIMIT} delegations. More records are available.</p>}
      {loadState === "ready" && delegations.length === 0 && <p>No delegations were returned for the current legal-entity population.</p>}
      {visibleDelegations.map((delegation) => <article key={delegation.id}>
        <header><div><span className="eyebrow">{humanize(delegation.responsibility)}</span><h4>{delegation.from.label} to {delegation.to.label}</h4></div><strong>{humanize(delegation.status)}</strong></header>
        <dl>
          <div><dt>Legal entity</dt><dd>{delegation.legalEntity.label}</dd></div>
          <div><dt>Effective dates</dt><dd>{dateWindow(delegation.startsAt, delegation.endsAt)}</dd></div>
          <div><dt>Record version</dt><dd>{delegation.version}</dd></div>
          <div><dt>Reason</dt><dd>{delegation.reason}</dd></div>
        </dl>
        <p>Made by {delegation.maker.label} · checked by {delegation.checker?.label ?? "Not checked yet"}</p>
        {delegation.latestDecision && <p>Latest decision: {delegation.latestDecision}</p>}
        <DelegationNextAction delegation={delegation} actorId={actorId} canConfigure={canConfigure} mutationDisabled={!mutationsEnabled || busy !== ""} busy={busy === `delegation:${delegation.id}`} onAction={runDelegationAction}/>
      </article>)}
    </section>
  </section>;
}

function DelegationNextAction({ delegation, actorId, canConfigure, mutationDisabled, busy, onAction }: { delegation: GovernanceDelegationItem; actorId: string; canConfigure: boolean; mutationDisabled: boolean; busy: boolean; onAction: (input: GovernanceDelegationActionInput) => Promise<void> }) {
  if (!canConfigure) return null;
  const label = `${delegation.from.label} to ${delegation.to.label}`;
  if (delegation.status === "DRAFT") {
    if (delegation.maker.id !== actorId) return <p>{delegation.maker.label} must submit this delegation for independent approval.</p>;
    return <button type="button" disabled={busy || mutationDisabled} onClick={() => void onAction({ delegationId: delegation.id, action: "submit", expectedVersion: delegation.version })}>{busy ? "Submitting…" : `Submit ${label} for approval`}</button>;
  }
  if (["PENDING_APPROVAL", "IN_REVIEW"].includes(delegation.status)) {
    if ([delegation.maker.id, delegation.from.id, delegation.to.id].includes(actorId)) return <p>A person who did not make, give, or receive this delegation must approve it. The delegation remains inactive.</p>;
    return <button type="button" disabled={busy || mutationDisabled} onClick={() => void onAction({ delegationId: delegation.id, action: "approve", expectedVersion: delegation.version })}>{busy ? "Approving…" : `Approve ${label}`}</button>;
  }
  if (["APPROVED", "ACTIVE"].includes(delegation.status)) {
    if (actorId !== delegation.from.id && actorId !== delegation.checker?.id) return <p>Only {delegation.from.label} or the recorded independent checker can revoke this delegation.</p>;
    return <ReasonedAction label={label} verb="Revoke" reasonLabel={`Reason to revoke ${label}`} busyLabel="Revoking…" busy={busy} disabled={mutationDisabled} onAction={(rationale) => onAction({ delegationId: delegation.id, action: "revoke", expectedVersion: delegation.version, rationale })}/>;
  }
  return null;
}

function PolicyNextAction({ policy, actorId, canConfigure, mutationDisabled, busy, onAction }: { policy: GovernancePolicyItem; actorId: string; canConfigure: boolean; mutationDisabled: boolean; busy: boolean; onAction: (input: GovernancePolicyActionInput) => Promise<void> }) {
  if (!canConfigure) return null;
  if (policy.status === "DRAFT") {
    if (policy.maker.id !== actorId) return <p>{policy.maker.label} must submit {policy.name} for independent approval.</p>;
    return <button type="button" disabled={busy || mutationDisabled} onClick={() => void onAction({ policyId: policy.id, action: "submit", expectedVersion: policy.version })}>{busy ? "Submitting…" : `Submit ${policy.name} for approval`}</button>;
  }
  if (["PENDING_APPROVAL", "IN_REVIEW"].includes(policy.status)) {
    if (policy.maker.id === actorId) return <p>Another authorized person must approve {policy.name}. Your proposed policy remains inactive.</p>;
    return <button type="button" disabled={busy || mutationDisabled} onClick={() => void onAction({ policyId: policy.id, action: "approve", expectedVersion: policy.version })}>{busy ? "Approving…" : `Approve ${policy.name}`}</button>;
  }
  if (policy.status === "ACTIVE") return <ReasonedAction label={policy.name} verb="Retire" reasonLabel={`Reason to retire ${policy.name}`} busyLabel="Retiring…" busy={busy} disabled={mutationDisabled} onAction={(rationale) => onAction({ policyId: policy.id, action: "retire", expectedVersion: policy.version, rationale })}/>;
  return null;
}

function ReasonedAction({ label, verb, reasonLabel, busyLabel, busy, disabled, onAction }: { label: string; verb: string; reasonLabel: string; busyLabel: string; busy: boolean; disabled: boolean; onAction: (rationale: string) => Promise<void> }) {
  const [rationale, setRationale] = useState("");
  return <form onSubmit={(event) => { event.preventDefault(); if (rationale.trim()) void onAction(rationale.trim()); }}>
    <label>{reasonLabel}<input value={rationale} disabled={disabled || busy} onChange={(event) => setRationale(event.target.value)}/></label>
    <button type="submit" disabled={disabled || busy || !rationale.trim()}>{busy ? busyLabel : `${verb} ${label}`}</button>
  </form>;
}

function dateWindow(from?: string, until?: string) {
  return `${from ? formatDate(from) : "Not scheduled"} to ${until ? formatDate(until) : "No end date"}`;
}

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Invalid date" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

const RESPONSIBILITIES: GovernanceResponsibility[] = ["PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER"];
const INVENTORY_LIMIT = 50;

function uniqueEntities(values: GovernanceEntity[]) {
  return Array.from(new Map(values.map((value) => [value.id, value])).values());
}
