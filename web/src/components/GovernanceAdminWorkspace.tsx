import { useEffect, useRef, useState, type FormEvent } from "react";
import { FocusedSheet } from "./FocusedSheet";

export type GovernanceLoadState = "loading" | "ready" | "degraded" | "unavailable";

export type GovernanceParty = { id: string; label: string };
export type GovernanceEntity = GovernanceParty;
export type GovernancePerson = GovernanceParty & { legalEntity: GovernanceEntity; status?: string; contextLabel?: string; canGive?: boolean; canReceive?: boolean };
export type GovernanceRole = { code: string; label: string };
export type GovernanceDelegationCandidatePage = { items: Array<{ principal_id: string; display_name: string; context_label?: string; can_give: boolean; can_receive: boolean }>; has_more: boolean };

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
  latestDecision?: { fromState: string; toState: string; actor: GovernanceParty; rationale: string; decidedAt: string; recordVersion: number };
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

export type CreateGovernancePolicyDraftInput = { legalEntityId: string; code: string; name: string; responsibility: GovernanceResponsibility; roleCode: string; objectType: "*" | "PROGRAM" | "MATTER"; decisionType: string; minMateriality: number; priority: number; effectiveFrom?: string };
export type GovernancePolicyActionInput = { policyId: string; action: "submit" | "approve" | "reject" | "retire"; expectedVersion: number; rationale?: string };
export type GovernanceDelegationActionInput = { delegationId: string; action: "submit" | "approve" | "revoke"; expectedVersion: number; rationale?: string };

export type GovernanceAdminWorkspaceProps = {
  policies: GovernancePolicyItem[];
  delegations: GovernanceDelegationItem[];
  eligiblePeople: GovernancePerson[];
  currentEntity?: GovernanceEntity;
  policyRoles?: GovernanceRole[];
  actorId: string;
  canConfigure: boolean;
  delegationCreationAvailable?: boolean;
  loadState: GovernanceLoadState;
  degradedReason?: string;
  createDelegation: (input: CreateGovernanceDelegationInput) => Promise<void>;
  loadDelegationCandidates?: (responsibility: GovernanceResponsibility, query: string) => Promise<GovernanceDelegationCandidatePage>;
  createPolicyDraft?: (input: CreateGovernancePolicyDraftInput) => Promise<void>;
  policyAction: (input: GovernancePolicyActionInput) => Promise<void>;
  delegationAction: (input: GovernanceDelegationActionInput) => Promise<void>;
};

type Composer = "delegation" | "policy" | null;

export function GovernanceAdminWorkspace({ policies, delegations, eligiblePeople, currentEntity, policyRoles = [], actorId, canConfigure, delegationCreationAvailable = true, loadState, degradedReason, createDelegation, loadDelegationCandidates, createPolicyDraft, policyAction, delegationAction }: GovernanceAdminWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [composer, setComposer] = useState<Composer>(null);
  const [legalEntityId, setLegalEntityId] = useState(currentEntity?.id ?? "");
  const [fromPrincipalId, setFromPrincipalId] = useState("");
  const [toPrincipalId, setToPrincipalId] = useState("");
  const [responsibility, setResponsibility] = useState<GovernanceResponsibility | "">("");
  const [startsAt, setStartsAt] = useState("");
  const [endsAt, setEndsAt] = useState("");
  const [reason, setReason] = useState("");
  const [candidatePeople, setCandidatePeople] = useState(eligiblePeople);
  const [candidateState, setCandidateState] = useState<"idle" | "loading" | "ready" | "unavailable">("idle");
  const [candidateQuery, setCandidateQuery] = useState("");
  const candidateGeneration = useRef(0);
  const [policyName, setPolicyName] = useState("");
  const [policyCode, setPolicyCode] = useState("");
  const [policyResponsibility, setPolicyResponsibility] = useState<GovernanceResponsibility | "">("");
  const [policyRoleCode, setPolicyRoleCode] = useState("");
  const [policyObjectType, setPolicyObjectType] = useState<"*" | "PROGRAM" | "MATTER">("*");
  const [policyMinMateriality, setPolicyMinMateriality] = useState(0);
  const [policyEffectiveFrom, setPolicyEffectiveFrom] = useState("");
  const entityOptions = uniqueEntities([...(currentEntity ? [currentEntity] : []), ...candidatePeople.map((person) => person.legalEntity)]);
  const peopleForEntity = candidatePeople.filter((person) => person.legalEntity.id === legalEntityId);
  const mutationsEnabled = canConfigure && loadState === "ready";
  const delegationDraftEnabled = mutationsEnabled && delegationCreationAvailable;
  const policyDraftEnabled = mutationsEnabled && Boolean(createPolicyDraft) && policyRoles.length > 0;
  const visiblePolicies = policies.slice(0, INVENTORY_LIMIT);
  const visibleDelegations = delegations.slice(0, INVENTORY_LIMIT);

  useEffect(() => {
    if (currentEntity?.id) setLegalEntityId(currentEntity.id);
  }, [currentEntity?.id]);

  async function refreshCandidates(nextResponsibility: GovernanceResponsibility, query: string) {
    if (!loadDelegationCandidates || !currentEntity) return;
    const generation = ++candidateGeneration.current;
    setCandidateState("loading");
    setFromPrincipalId("");
    setToPrincipalId("");
    try {
      const page = await loadDelegationCandidates(nextResponsibility, query);
      if (generation !== candidateGeneration.current) return;
      setCandidatePeople(page.items.map((item) => ({
        id: item.principal_id, label: item.display_name, contextLabel: item.context_label,
        legalEntity: currentEntity, canGive: item.can_give, canReceive: item.can_receive,
      })));
      setCandidateState("ready");
    } catch {
      if (generation !== candidateGeneration.current) return;
      setCandidatePeople([]);
      setCandidateState("unavailable");
    }
  }

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
      setComposer(null);
    } catch {
      setError("The delegation draft could not be created. Your entries remain on this screen; refresh the inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function createPolicy(event: FormEvent) {
    event.preventDefault();
    if (!createPolicyDraft || !currentEntity || !policyName.trim() || !policyCode.trim() || !policyResponsibility || !policyRoleCode) return;
    const effectiveFrom = policyEffectiveFrom ? new Date(policyEffectiveFrom) : undefined;
    if (effectiveFrom && !Number.isFinite(effectiveFrom.valueOf())) {
      setError("The policy effective date is invalid. Check the date and time.");
      return;
    }
    setBusy("create-policy");
    setError("");
    try {
      await createPolicyDraft({
        legalEntityId: currentEntity.id, code: policyCode.trim(), name: policyName.trim(), responsibility: policyResponsibility,
        roleCode: policyRoleCode, objectType: policyObjectType, decisionType: "",
        minMateriality: policyMinMateriality, priority: 100, effectiveFrom: effectiveFrom?.toISOString(),
      });
      setPolicyName("");
      setPolicyCode("");
      setPolicyResponsibility("");
      setPolicyRoleCode("");
      setPolicyObjectType("*");
      setPolicyMinMateriality(0);
      setPolicyEffectiveFrom("");
      setComposer(null);
    } catch {
      setError("The routing policy draft could not be created. Your entries remain on this screen; refresh the inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  return <section className="governance-admin-workspace" aria-labelledby="governance-admin-title">
    <header className="governance-admin-header">
      <div>
        <span className="eyebrow">Configure · governance</span>
        <h2 id="governance-admin-title">Governance policies and delegations</h2>
        <p>Review the current legal-entity routes and time-bound authority cover before taking the next governed action.</p>
      </div>
      {canConfigure && <div className="governance-admin-actions" aria-label="Governance creation actions">
        <button type="button" disabled={!delegationDraftEnabled || busy !== ""} onClick={() => { setError(""); setComposer("delegation"); }}>New delegation</button>
        {createPolicyDraft && <button type="button" className="secondary-button" disabled={!policyDraftEnabled || busy !== ""} onClick={() => { setError(""); setComposer("policy"); }}>New routing policy</button>}
      </div>}
    </header>
    {loadState === "loading" && <p role="status">Loading the current legal-entity governance inventory. Changes remain unavailable until the current authority route is confirmed.</p>}
    {loadState === "degraded" && <p role="alert">{degradedReason || "Current approval authority could not be confirmed."} Existing records remain available, but changes are disabled until authority can be confirmed.</p>}
    {loadState === "unavailable" && (policies.length > 0 || delegations.length > 0) && <p role="alert">{degradedReason || "The latest governance inventory could not be loaded."} Previously loaded records remain available, but changes are disabled.</p>}
    {loadState === "unavailable" && policies.length === 0 && delegations.length === 0 && <p role="alert">The current legal-entity governance population could not be loaded. Refresh this page to try again; no changes are available.</p>}
    {loadState === "ready" && !canConfigure && <p>You can review this governance inventory, but your current access does not allow policy or delegation changes.</p>}
    {loadState === "ready" && canConfigure && !delegationCreationAvailable && <p role="status">New delegations are unavailable until the bank directory can confirm who holds the selected responsibility in this legal entity. Existing delegation actions remain available.</p>}
    {loadState === "ready" && canConfigure && createPolicyDraft && policyRoles.length === 0 && <p role="status">New routing policies are unavailable until current role labels can be confirmed. Existing policy actions remain available.</p>}
    {error && <p role="alert">{error}</p>}

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
        {policy.latestDecision && <PolicyDecisionSummary decision={policy.latestDecision}/>}
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

    {composer === "delegation" && <FocusedSheet label="New delegation" closeLabel="Close delegation form" panelClassName="governance-composer-sheet" onClose={() => setComposer(null)}>
      <section className="governance-composer" aria-labelledby="create-governance-delegation">
        <span className="eyebrow">Authority cover</span>
        <h3 id="create-governance-delegation">Create whole-entity delegation</h3>
        <p>Create a time-bound draft for one legal entity. A different authorized person must approve it before the delegated responsibility becomes active.</p>
        <form onSubmit={(event) => void create(event)}>
          <label>Legal entity<select required value={legalEntityId} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => { setLegalEntityId(event.target.value); setFromPrincipalId(""); setToPrincipalId(""); }}><option value="">Choose legal entity</option>{entityOptions.map((entity) => <option key={entity.id} value={entity.id}>{entity.label}</option>)}</select></label>
          <label>Responsibility<select required value={responsibility} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => { const next = event.target.value as GovernanceResponsibility; setResponsibility(next); setCandidateQuery(""); if (next) void refreshCandidates(next, ""); }}><option value="">Choose responsibility</option>{RESPONSIBILITIES.map((value) => <option key={value} value={value}>{humanize(value)}</option>)}</select></label>
          {loadDelegationCandidates && responsibility && <div>
            <label>Find eligible people<input maxLength={100} value={candidateQuery} disabled={!delegationDraftEnabled || candidateState === "loading"} onChange={(event) => setCandidateQuery(event.target.value)}/></label>
            <button type="button" className="secondary-button" disabled={!delegationDraftEnabled || candidateState === "loading"} onClick={() => void refreshCandidates(responsibility, candidateQuery)}>{candidateState === "loading" ? "Loading people…" : "Search people"}</button>
          </div>}
          <label>Person giving authority<select required value={fromPrincipalId} disabled={!delegationDraftEnabled || !legalEntityId || (!responsibility && Boolean(loadDelegationCandidates)) || candidateState === "loading" || busy !== ""} onChange={(event) => setFromPrincipalId(event.target.value)}><option value="">Choose person</option>{peopleForEntity.filter((person) => person.canGive !== false).map((person) => <option key={person.id} value={person.id}>{personOptionLabel(person)}</option>)}</select></label>
          <label>Person receiving authority<select required value={toPrincipalId} disabled={!delegationDraftEnabled || !legalEntityId || (!responsibility && Boolean(loadDelegationCandidates)) || candidateState === "loading" || busy !== ""} onChange={(event) => setToPrincipalId(event.target.value)}><option value="">Choose person</option>{peopleForEntity.filter((person) => person.id !== fromPrincipalId && person.canReceive !== false).map((person) => <option key={person.id} value={person.id}>{personOptionLabel(person)}</option>)}</select></label>
          {candidateState === "unavailable" && <p role="status">Eligible people could not be confirmed for this responsibility. Reload the governance workspace before creating a delegation.</p>}
          {candidateState === "ready" && peopleForEntity.length === 0 && <p role="status">No eligible people were found for this responsibility in the current legal entity. Ask the access administrator to check current assignments.</p>}
          <label>Starts at<input required type="datetime-local" value={startsAt} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setStartsAt(event.target.value)}/></label>
          <label>Ends at<input required type="datetime-local" value={endsAt} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setEndsAt(event.target.value)}/></label>
          <label>Reason<textarea required value={reason} disabled={!delegationDraftEnabled || busy !== ""} onChange={(event) => setReason(event.target.value)}/></label>
          <button type="submit" disabled={!delegationDraftEnabled || busy !== "" || !legalEntityId || !fromPrincipalId || !toPrincipalId || !responsibility || !startsAt || !endsAt || !reason.trim()}>{busy === "create-delegation" ? "Creating draft…" : "Create delegation draft"}</button>
        </form>
      </section>
    </FocusedSheet>}

    {composer === "policy" && <FocusedSheet label="New routing policy" closeLabel="Close routing policy form" panelClassName="governance-composer-sheet" onClose={() => setComposer(null)}>
      <section className="governance-composer" aria-labelledby="create-governance-policy">
        <span className="eyebrow">Approval routing</span>
        <h3 id="create-governance-policy">Create routing policy draft</h3>
        <p>Define one responsibility route for {currentEntity?.label ?? "the current legal entity"}. A different authorized person must review the draft before it can become active.</p>
        <form onSubmit={(event) => void createPolicy(event)}>
          <label>Policy name<input required value={policyName} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyName(event.target.value)}/></label>
          <label>Policy code<input required value={policyCode} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyCode(event.target.value)}/></label>
          <label>Policy responsibility<select required value={policyResponsibility} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyResponsibility(event.target.value as GovernanceResponsibility)}><option value="">Choose responsibility</option>{RESPONSIBILITIES.map((value) => <option key={value} value={value}>{humanize(value)}</option>)}</select></label>
          <label>Responsible role<select required value={policyRoleCode} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyRoleCode(event.target.value)}><option value="">Choose role</option>{policyRoles.map((role) => <option key={role.code} value={role.code}>{role.label}</option>)}</select></label>
          <label>Applies to<select value={policyObjectType} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyObjectType(event.target.value as "*" | "PROGRAM" | "MATTER")}><option value="*">Programs and issues</option><option value="PROGRAM">Programs</option><option value="MATTER">Issues and changes</option></select></label>
          <label>Minimum materiality<input type="number" min={0} max={5} value={policyMinMateriality} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyMinMateriality(Number(event.target.value))}/></label>
          <label>Effective from<input type="datetime-local" value={policyEffectiveFrom} disabled={!policyDraftEnabled || busy !== ""} onChange={(event) => setPolicyEffectiveFrom(event.target.value)}/></label>
          <button type="submit" disabled={!policyDraftEnabled || busy !== "" || !policyName.trim() || !policyCode.trim() || !policyResponsibility || !policyRoleCode}>{busy === "create-policy" ? "Creating draft…" : "Create policy draft"}</button>
        </form>
      </section>
    </FocusedSheet>}
  </section>;
}

function PolicyDecisionSummary({ decision }: { decision: NonNullable<GovernancePolicyItem["latestDecision"]> }) {
  const action = decision.fromState === "PENDING_APPROVAL" && decision.toState === "DRAFT" ? "Returned for changes" : `${humanize(decision.fromState)} to ${humanize(decision.toState)}`;
  return <div>
    <p>Latest decision: {action} by {decision.actor.label} on {formatDate(decision.decidedAt)}.</p>
    {decision.rationale && <p>Reason: {decision.rationale} · Record version {decision.recordVersion}</p>}
  </div>;
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
    return <PolicyReviewAction policy={policy} busy={busy} disabled={mutationDisabled} onAction={onAction}/>;
  }
  if (policy.status === "ACTIVE") return <ReasonedAction label={policy.name} verb="Retire" reasonLabel={`Reason to retire ${policy.name}`} busyLabel="Retiring…" busy={busy} disabled={mutationDisabled} onAction={(rationale) => onAction({ policyId: policy.id, action: "retire", expectedVersion: policy.version, rationale })}/>;
  return null;
}

function PolicyReviewAction({ policy, busy, disabled, onAction }: { policy: GovernancePolicyItem; busy: boolean; disabled: boolean; onAction: (input: GovernancePolicyActionInput) => Promise<void> }) {
  const [rationale, setRationale] = useState("");
  return <div>
    <button type="button" disabled={busy || disabled} onClick={() => void onAction({ policyId: policy.id, action: "approve", expectedVersion: policy.version })}>{busy ? "Recording review…" : `Approve ${policy.name}`}</button>
    <form onSubmit={(event) => { event.preventDefault(); if (rationale.trim()) void onAction({ policyId: policy.id, action: "reject", expectedVersion: policy.version, rationale: rationale.trim() }); }}>
      <label>Changes needed for {policy.name}<input value={rationale} disabled={busy || disabled} onChange={(event) => setRationale(event.target.value)}/></label>
      <button type="submit" className="secondary-button" disabled={busy || disabled || !rationale.trim()}>{busy ? "Recording review…" : `Return ${policy.name} for changes`}</button>
    </form>
  </div>;
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

function personOptionLabel(person: GovernancePerson) {
  return person.contextLabel ? `${person.label} · ${person.contextLabel}` : person.label;
}

const RESPONSIBILITIES: GovernanceResponsibility[] = ["PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER"];
const INVENTORY_LIMIT = 50;

function uniqueEntities(values: GovernanceEntity[]) {
  return Array.from(new Map(values.map((value) => [value.id, value])).values());
}