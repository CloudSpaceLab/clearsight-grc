import { useEffect, useRef, useState, type FormEvent } from "react";
import { FocusedSheet } from "./FocusedSheet";
import type { GovernanceReadState } from "../governanceAdminApi";
import { GovernancePolicyDefinition, PeopleGovernanceView } from "./PeopleGovernanceView";
import { readGovernancePolicyDefinition } from "./governancePolicyPresentation";
import { Button, Notice, SelectField, TextArea, TextField } from "./ui";

export type GovernanceLoadState = "loading" | "ready" | "degraded" | "unavailable" | "restricted";
export type GovernanceView = "policies" | "delegations";

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
  definition?: unknown;
  checksum?: string;
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
  scope?: Record<string, unknown>;
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
  active?: boolean;
  view?: GovernanceView;
  selectedPolicyID?: string;
  onSelectPolicy?: (id: string | undefined) => void;
  selectedDelegationID?: string;
  onSelectDelegation?: (id: string | undefined) => void;
  policiesState?: GovernanceReadState;
  delegationsState?: GovernanceReadState;
  onReload?: () => void;
  onOpenRoutingChecks?: () => void;
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

export function GovernanceAdminWorkspace({ active = true, view, selectedPolicyID, onSelectPolicy, selectedDelegationID, onSelectDelegation, policiesState, delegationsState, onReload, onOpenRoutingChecks, policies, delegations, eligiblePeople, currentEntity, policyRoles = [], actorId, canConfigure, delegationCreationAvailable = true, loadState, degradedReason, createDelegation, loadDelegationCandidates, createPolicyDraft, policyAction, delegationAction }: GovernanceAdminWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [receipt, setReceipt] = useState("");
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
  const interactionGeneration = useRef(0);
  const [policyName, setPolicyName] = useState("");
  const [policyCode, setPolicyCode] = useState("");
  const [policyResponsibility, setPolicyResponsibility] = useState<GovernanceResponsibility | "">("");
  const [policyRoleCode, setPolicyRoleCode] = useState("");
  const [policyObjectType, setPolicyObjectType] = useState<"*" | "PROGRAM" | "MATTER">("*");
  const [policyMinMateriality, setPolicyMinMateriality] = useState(0);
  const [policyEffectiveFrom, setPolicyEffectiveFrom] = useState("");
  const entityOptions = uniqueEntities([...(currentEntity ? [currentEntity] : []), ...candidatePeople.map((person) => person.legalEntity)]);
  const peopleForEntity = candidatePeople.filter((person) => person.legalEntity.id === legalEntityId);
  const mutationsEnabled = active && canConfigure && loadState === "ready";
  const delegationDraftEnabled = mutationsEnabled && delegationCreationAvailable;
  const policyDraftEnabled = mutationsEnabled && Boolean(createPolicyDraft) && policyRoles.length > 0;
  const visiblePolicies = loadState === "restricted" || policiesState === "restricted" ? [] : policies.slice(0, INVENTORY_LIMIT);
  const visibleDelegations = loadState === "restricted" || delegationsState === "restricted" ? [] : delegations.slice(0, INVENTORY_LIMIT);

  useEffect(() => {
    if (currentEntity?.id) setLegalEntityId(currentEntity.id);
  }, [currentEntity?.id]);

  useEffect(() => () => { candidateGeneration.current += 1; interactionGeneration.current += 1; }, []);

  useEffect(() => {
    if (active) return;
    interactionGeneration.current += 1;
    setComposer(null);
    setReceipt("");
    setError("");
    candidateGeneration.current += 1;
    setCandidatePeople([]);
    setCandidateState("idle");
    setFromPrincipalId("");
    setToPrincipalId("");
  }, [active]);

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
    if (!mutationsEnabled || busy) return;
    const generation = interactionGeneration.current;
    setBusy(`policy:${input.policyId}`);
    setError("");
    try {
      await policyAction(input);
    } catch {
      if (generation === interactionGeneration.current) setError("The policy action could not be completed. Refresh the governance inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function runDelegationAction(input: GovernanceDelegationActionInput) {
    if (!mutationsEnabled || busy) return;
    const generation = interactionGeneration.current;
    setBusy(`delegation:${input.delegationId}`);
    setError("");
    try {
      await delegationAction(input);
    } catch {
      if (generation === interactionGeneration.current) setError("The delegation action could not be completed. Refresh the governance inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function create(event: FormEvent) {
    event.preventDefault();
    if (!delegationDraftEnabled || busy) return;
    const generation = interactionGeneration.current;
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
      if (generation !== interactionGeneration.current) return;
      setReceipt("Delegation draft created.");
      setFromPrincipalId("");
      setToPrincipalId("");
      setResponsibility("");
      setStartsAt("");
      setEndsAt("");
      setReason("");
      setComposer(null);
    } catch {
      if (generation === interactionGeneration.current) setError("The delegation draft could not be created. Your entries remain on this screen; refresh the inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  async function createPolicy(event: FormEvent) {
    event.preventDefault();
    if (!policyDraftEnabled || busy) return;
    const generation = interactionGeneration.current;
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
      if (generation !== interactionGeneration.current) return;
      setReceipt("Routing policy draft created.");
      setPolicyName("");
      setPolicyCode("");
      setPolicyResponsibility("");
      setPolicyRoleCode("");
      setPolicyObjectType("*");
      setPolicyMinMateriality(0);
      setPolicyEffectiveFrom("");
      setComposer(null);
    } catch {
      if (generation === interactionGeneration.current) setError("The routing policy draft could not be created. Your entries remain on this screen; refresh the inventory and try again.");
    } finally {
      setBusy("");
    }
  }

  return <section className={`governance-admin-workspace${view ? " governance-admin-workspace--embedded" : ""}`} aria-labelledby={view ? undefined : "governance-admin-title"} aria-label={view ? `${view === "policies" ? "Approval routes" : "Delegations"} workspace` : undefined}>
    {view ? <>
      <div className="people-governance-notices">
        {receipt && <Notice tone="success">{receipt}</Notice>}
        {loadState === "loading" && <Notice>Refreshing routing policies and delegations. Changes are paused.</Notice>}
        {(loadState === "degraded" || loadState === "unavailable") && <Notice tone="warning">{degradedReason || "Governance records could not be refreshed."} Previously loaded records may be out of date. Changes are paused until the records can be refreshed.</Notice>}
        {loadState === "restricted" && <Notice tone="warning">{degradedReason || "Your current access does not allow governance records."}</Notice>}
        {loadState === "ready" && !canConfigure && <Notice>Your current access allows review of policies and delegations. Changes are unavailable.</Notice>}
        {loadState === "ready" && canConfigure && createPolicyDraft && policyRoles.length === 0 && view === "policies" && <Notice>New policies are unavailable until current role labels can be confirmed.</Notice>}
        {loadState === "ready" && canConfigure && !delegationCreationAvailable && view === "delegations" && <Notice>New delegations are unavailable until eligible people can be confirmed.</Notice>}
        {error && !composer && <Notice tone="error">{error}</Notice>}
        {onReload && loadState !== "loading" && loadState !== "ready" && <Button onPress={onReload}>Reload governance records</Button>}
      </div>
      <PeopleGovernanceView view={view} policies={policies} delegations={delegations} currentEntity={currentEntity} loadState={loadState} policiesState={policiesState} delegationsState={delegationsState}
        onOpenRoutingChecks={onOpenRoutingChecks}
        selectedPolicyID={selectedPolicyID} onSelectPolicy={onSelectPolicy} selectedDelegationID={selectedDelegationID} onSelectDelegation={onSelectDelegation}
        creationAction={canConfigure && (view === "policies" ? createPolicyDraft && <Button isDisabled={!policyDraftEnabled || busy !== ""} onPress={() => { setError(""); setComposer("policy"); }}>New routing policy</Button> : <Button isDisabled={!delegationDraftEnabled || busy !== ""} onPress={() => { setError(""); setComposer("delegation"); }}>Propose delegation</Button>)}
        renderPolicyDetail={(policy) => <>
          <GovernancePolicyDefinition key={`${policy.id}:${policy.version}`} policy={policy}/>
          <section><h3>Policy details</h3><dl className="people-access-facts"><dt>Legal entity</dt><dd>{policy.legalEntity.label}</dd><dt>Policy version</dt><dd>{policy.currentVersion}</dd><dt>Record version</dt><dd>{policy.version}</dd><dt>Effective dates</dt><dd>{dateWindow(policy.effectiveFrom, policy.effectiveUntil)}</dd><dt>Made by</dt><dd>{policy.maker.label}</dd><dt>Checked by</dt><dd>{policy.checker?.label ?? "Not checked yet"}</dd></dl></section>
          {policy.latestDecision && <section><h3>Latest decision</h3><PolicyDecisionSummary decision={policy.latestDecision}/></section>}
          <div className="people-governance-actions"><PolicyNextAction key={`${policy.id}:${policy.version}`} policy={policy} actorId={actorId} canConfigure={canConfigure} mutationDisabled={!mutationsEnabled || busy !== ""} busy={busy === `policy:${policy.id}`} onAction={runPolicyAction}/></div>
          <Notice>Approval authority depends on the current policy, record, responsibility and conflict checks.</Notice>
        </>}
        renderDelegationDetail={(delegation) => <>
          <section><h3>Delegation details</h3><dl className="people-access-facts"><dt>Legal entity</dt><dd>{delegation.legalEntity.label}</dd><dt>Giving responsibility</dt><dd>{delegation.from.label}</dd><dt>Receiving responsibility</dt><dd>{delegation.to.label}</dd><dt>Responsibility</dt><dd>{humanize(delegation.responsibility)}</dd><dt>Effective dates</dt><dd>{dateWindow(delegation.startsAt, delegation.endsAt)}</dd><dt>Record version</dt><dd>{delegation.version}</dd><dt>Made by</dt><dd>{delegation.maker.label}</dd><dt>Checked by</dt><dd>{delegation.checker?.label ?? "Not checked yet"}</dd></dl></section>
          <section><h3>Delegation scope</h3><DelegationScopeFacts scope={delegation.scope}/></section>
          <section><h3>Reason</h3><p>{delegation.reason}</p></section>
          {delegation.latestDecision && <section><h3>Latest decision</h3><p>{delegation.latestDecision}</p></section>}
          <div className="people-governance-actions"><DelegationNextAction key={`${delegation.id}:${delegation.version}`} delegation={delegation} actorId={actorId} canConfigure={canConfigure} mutationDisabled={!mutationsEnabled || busy !== ""} busy={busy === `delegation:${delegation.id}`} onAction={runDelegationAction}/></div>
          <Notice>Draft and pending delegations grant no authority. Approval, effective dates and conflict checks still apply.</Notice>
        </>}/>
    </> : <>
    <header className="governance-admin-header">
      <div>
        <span className="eyebrow">Configure · governance</span>
        <h2 id="governance-admin-title">Governance policies and delegations</h2>
        <p>Review approval routes and active delegations.</p>
      </div>
      {canConfigure && <div className="governance-admin-actions" aria-label="Governance creation actions">
        <Button isDisabled={!delegationDraftEnabled || busy !== ""} onPress={() => { setError(""); setComposer("delegation"); }}>New delegation</Button>
        {createPolicyDraft && <Button variant="secondary" isDisabled={!policyDraftEnabled || busy !== ""} onPress={() => { setError(""); setComposer("policy"); }}>New routing policy</Button>}
      </div>}
    </header>
    {loadState === "loading" && <p role="status">Loading the current legal-entity governance inventory. Changes remain unavailable until the current approval route is confirmed.</p>}
    {loadState === "degraded" && <p role="alert">{degradedReason || "Current approval authority could not be confirmed."} Existing records remain available, but changes are disabled until authority can be confirmed.</p>}
    {loadState === "unavailable" && (policies.length > 0 || delegations.length > 0) && <p role="alert">{degradedReason || "The latest governance inventory could not be loaded."} Previously loaded records remain available, but changes are disabled.</p>}
    {loadState === "unavailable" && policies.length === 0 && delegations.length === 0 && <p role="alert">Governance data unavailable. Refresh before making changes.</p>}
    {loadState === "restricted" && <p role="alert">Governance access restricted. Your current access does not allow these records.</p>}
    {loadState === "ready" && !canConfigure && <p>You can review this governance inventory, but your current access does not allow policy or delegation changes.</p>}
    {loadState === "ready" && canConfigure && !delegationCreationAvailable && <p role="status">New delegations are unavailable until the organization directory can confirm who holds the selected responsibility in this legal entity. Existing delegation actions remain available.</p>}
    {loadState === "ready" && canConfigure && createPolicyDraft && policyRoles.length === 0 && <p role="status">New routing policies are unavailable until current role labels can be confirmed. Existing policy actions remain available.</p>}
    {error && <p role="alert">{error}</p>}
    {receipt && <p role="status">{receipt}</p>}

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
        <GovernancePolicyDefinition policy={policy}/>
        {policy.latestDecision && <PolicyDecisionSummary decision={policy.latestDecision}/>}
        <PolicyNextAction policy={policy} actorId={actorId} canConfigure={canConfigure} mutationDisabled={!mutationsEnabled || busy !== ""} busy={busy === `policy:${policy.id}`} onAction={runPolicyAction}/>
      </article>)}
    </section>

    <section aria-labelledby="governance-delegation-inventory">
      <h3 id="governance-delegation-inventory">Delegations</h3>
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

    </>}

    {active && composer === "delegation" && <FocusedSheet label="New delegation" closeLabel="Close delegation form" panelClassName="governance-composer-sheet" onClose={() => setComposer(null)}>
      <section className="governance-composer" aria-labelledby="create-governance-delegation">
        <span className="eyebrow">Authority cover</span>
        <h3 id="create-governance-delegation">Create whole-entity delegation</h3>
        <p>Create a time-bound draft for one legal entity. A different authorized person must approve it before the delegated responsibility becomes active.</p>
        {error && <Notice tone="error">{error}</Notice>}
        <form className="governance-composer-form" onSubmit={(event) => void create(event)}>
          <SelectField label="Legal entity" isRequired allowsEmpty={false} value={legalEntityId || undefined} placeholder="Choose legal entity" options={entityOptions.map((entity) => ({ id: entity.id, label: entity.label }))} isDisabled={!delegationDraftEnabled || busy !== ""} onChange={(value) => { setLegalEntityId(value ?? ""); setFromPrincipalId(""); setToPrincipalId(""); }}/>
          <SelectField label="Responsibility" isRequired allowsEmpty={false} value={responsibility || undefined} placeholder="Choose responsibility" options={RESPONSIBILITIES.map((value) => ({ id: value, label: humanize(value) }))} isDisabled={!delegationDraftEnabled || busy !== ""} onChange={(next) => { setResponsibility(next ?? ""); setCandidateQuery(""); if (next) void refreshCandidates(next, ""); }}/>
          {loadDelegationCandidates && responsibility && <div className="governance-composer__search governance-composer__wide">
            <TextField label="Find eligible people" maxLength={100} value={candidateQuery} isDisabled={!delegationDraftEnabled || candidateState === "loading"} onChange={setCandidateQuery}/>
            <Button variant="secondary" isDisabled={!delegationDraftEnabled || candidateState === "loading"} onPress={() => void refreshCandidates(responsibility, candidateQuery)}>{candidateState === "loading" ? "Loading people…" : "Search people"}</Button>
          </div>}
          <SelectField label="Person giving authority" isRequired allowsEmpty={false} value={fromPrincipalId || undefined} placeholder="Choose person" options={peopleForEntity.filter((person) => person.canGive !== false).map((person) => ({ id: person.id, label: personOptionLabel(person) }))} isDisabled={!delegationDraftEnabled || !legalEntityId || (!responsibility && Boolean(loadDelegationCandidates)) || candidateState === "loading" || busy !== ""} onChange={(value) => setFromPrincipalId(value ?? "")}/>
          <SelectField label="Person receiving authority" isRequired allowsEmpty={false} value={toPrincipalId || undefined} placeholder="Choose person" options={peopleForEntity.filter((person) => person.id !== fromPrincipalId && person.canReceive !== false).map((person) => ({ id: person.id, label: personOptionLabel(person) }))} isDisabled={!delegationDraftEnabled || !legalEntityId || (!responsibility && Boolean(loadDelegationCandidates)) || candidateState === "loading" || busy !== ""} onChange={(value) => setToPrincipalId(value ?? "")}/>
          {candidateState === "unavailable" && <div className="governance-composer__wide"><Notice tone="warning">Eligible people could not be confirmed for this responsibility. Search again before creating a delegation.</Notice></div>}
          {candidateState === "ready" && peopleForEntity.length === 0 && <div className="governance-composer__wide"><Notice>No eligible people were found for this responsibility in the current legal entity. Ask the access administrator to check current assignments.</Notice></div>}
          <TextField label="Starts at" isRequired type="datetime-local" value={startsAt} isDisabled={!delegationDraftEnabled || busy !== ""} onChange={setStartsAt} description="Your device's local time."/>
          <TextField label="Ends at" isRequired type="datetime-local" value={endsAt} isDisabled={!delegationDraftEnabled || busy !== ""} onChange={setEndsAt} description="Your device's local time."/>
          <div className="governance-composer__wide"><TextArea label="Reason" isRequired value={reason} isDisabled={!delegationDraftEnabled || busy !== ""} onChange={setReason}/></div>
          <Button type="submit" isDisabled={!delegationDraftEnabled || busy !== "" || !legalEntityId || !fromPrincipalId || !toPrincipalId || !responsibility || !startsAt || !endsAt || !reason.trim()}>{busy === "create-delegation" ? "Creating draft…" : "Create delegation draft"}</Button>
        </form>
      </section>
    </FocusedSheet>}

    {active && composer === "policy" && <FocusedSheet label="New routing policy" closeLabel="Close routing policy form" panelClassName="governance-composer-sheet" onClose={() => setComposer(null)}>
      <section className="governance-composer" aria-labelledby="create-governance-policy">
        <span className="eyebrow">Approval routing</span>
        <h3 id="create-governance-policy">Create routing policy draft</h3>
        <p>Define one responsibility route for {currentEntity?.label ?? "the current legal entity"}. A different authorized person must review the draft before it can become active.</p>
        {error && <Notice tone="error">{error}</Notice>}
        <form className="governance-composer-form" onSubmit={(event) => void createPolicy(event)}>
          <TextField label="Policy name" isRequired value={policyName} isDisabled={!policyDraftEnabled || busy !== ""} onChange={setPolicyName}/>
          <TextField label="Policy code" isRequired value={policyCode} isDisabled={!policyDraftEnabled || busy !== ""} onChange={setPolicyCode}/>
          <SelectField label="Policy responsibility" isRequired allowsEmpty={false} value={policyResponsibility || undefined} placeholder="Choose responsibility" options={RESPONSIBILITIES.map((value) => ({ id: value, label: humanize(value) }))} isDisabled={!policyDraftEnabled || busy !== ""} onChange={(value) => setPolicyResponsibility(value ?? "")}/>
          <SelectField label="Responsible role" isRequired allowsEmpty={false} value={policyRoleCode || undefined} placeholder="Choose role" options={policyRoles.map((role) => ({ id: role.code, label: role.label }))} isDisabled={!policyDraftEnabled || busy !== ""} onChange={(value) => setPolicyRoleCode(value ?? "")}/>
          <SelectField label="Applies to" value={policyObjectType} placeholder="Choose records" allowsEmpty={false} options={[{ id: "*", label: "Programs and issues" }, { id: "PROGRAM", label: "Programs" }, { id: "MATTER", label: "Issues and changes" }]} isDisabled={!policyDraftEnabled || busy !== ""} onChange={(value) => setPolicyObjectType(value ?? "*")}/>
          <TextField label="Minimum materiality" type="number" min={0} max={5} value={String(policyMinMateriality)} isDisabled={!policyDraftEnabled || busy !== ""} onChange={(value) => setPolicyMinMateriality(Number(value))}/>
          <TextField label="Effective from" type="datetime-local" value={policyEffectiveFrom} isDisabled={!policyDraftEnabled || busy !== ""} onChange={setPolicyEffectiveFrom} description="Your device's local time."/>
          <Button type="submit" isDisabled={!policyDraftEnabled || busy !== "" || !policyName.trim() || !policyCode.trim() || !policyResponsibility || !policyRoleCode}>{busy === "create-policy" ? "Creating draft…" : "Create policy draft"}</Button>
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

function DelegationScopeFacts({ scope }: { scope?: Record<string, unknown> }) {
  if (!scope) return <p>Scope details were not returned.</p>;
  const recordType = typeof scope.object_type === "string" ? scope.object_type : "";
  const record = typeof scope.object_id === "string" ? scope.object_id : "";
  const decision = typeof scope.decision_type === "string" ? scope.decision_type : "";
  const minimum = typeof scope.min_materiality === "number" ? scope.min_materiality : undefined;
  const maximum = typeof scope.max_materiality === "number" ? scope.max_materiality : undefined;
  const otherRestrictions = Object.keys(scope).some((key) => !["legal_entity_id", "object_type", "object_id", "decision_type", "min_materiality", "max_materiality"].includes(key));
  if (!recordType && !record && !decision && minimum === undefined && maximum === undefined && !otherRestrictions) return <p>Whole legal entity</p>;
  const typeLabel = ({ PROGRAM: "Programs", MATTER: "Issues and changes", VENDOR_RELATIONSHIP: "Vendor relationships" } as Record<string, string>)[recordType] || humanize(recordType);
  return <dl className="people-access-facts">
    <dt>Records</dt><dd>{record ? `One record${typeLabel ? ` · ${typeLabel}` : ""}` : typeLabel || "All record types"}</dd>
    {record && <><dt>Record reference</dt><dd>{record}</dd></>}
    {decision && <><dt>Decision</dt><dd>{decision}</dd></>}
    {minimum !== undefined && <><dt>Minimum materiality</dt><dd>{minimum}</dd></>}
    {maximum !== undefined && <><dt>Maximum materiality</dt><dd>{maximum}</dd></>}
    {otherRestrictions && <><dt>Other restrictions</dt><dd>Additional scope restrictions were returned.</dd></>}
  </dl>;
}

function DelegationNextAction({ delegation, actorId, canConfigure, mutationDisabled, busy, onAction }: { delegation: GovernanceDelegationItem; actorId: string; canConfigure: boolean; mutationDisabled: boolean; busy: boolean; onAction: (input: GovernanceDelegationActionInput) => Promise<void> }) {
  if (!canConfigure) return null;
  const label = `${delegation.from.label} to ${delegation.to.label}`;
  if (delegation.status === "DRAFT") {
    if (delegation.maker.id !== actorId) return <p>{delegation.maker.label} must submit this delegation for independent approval.</p>;
    return <Button isDisabled={busy || mutationDisabled} onPress={() => void onAction({ delegationId: delegation.id, action: "submit", expectedVersion: delegation.version })}>{busy ? "Submitting…" : `Submit ${label} for approval`}</Button>;
  }
  if (["PENDING_APPROVAL", "IN_REVIEW"].includes(delegation.status)) {
    if ([delegation.maker.id, delegation.from.id, delegation.to.id].includes(actorId)) return <p>A person who did not make, give, or receive this delegation must approve it. The delegation remains inactive.</p>;
    return <Button isDisabled={busy || mutationDisabled} onPress={() => void onAction({ delegationId: delegation.id, action: "approve", expectedVersion: delegation.version })}>{busy ? "Approving…" : `Approve ${label}`}</Button>;
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
    return <Button isDisabled={busy || mutationDisabled} onPress={() => void onAction({ policyId: policy.id, action: "submit", expectedVersion: policy.version })}>{busy ? "Submitting…" : `Submit ${policy.name} for approval`}</Button>;
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
  const definitionAvailable = readGovernancePolicyDefinition(policy.definition).reviewable;
  return <div>
    <Button variant="primary" isDisabled={busy || disabled || !definitionAvailable} onPress={() => { if (definitionAvailable) void onAction({ policyId: policy.id, action: "approve", expectedVersion: policy.version }); }}>{busy ? "Recording review…" : `Approve ${policy.name}`}</Button>
    <form onSubmit={(event) => { event.preventDefault(); if (rationale.trim()) void onAction({ policyId: policy.id, action: "reject", expectedVersion: policy.version, rationale: rationale.trim() }); }}>
      <TextField label={`Changes needed for ${policy.name}`} value={rationale} isDisabled={busy || disabled} onChange={setRationale}/>
      <Button type="submit" variant="secondary" isDisabled={busy || disabled || !rationale.trim()}>{busy ? "Recording review…" : `Return ${policy.name} for changes`}</Button>
    </form>
  </div>;
}

function ReasonedAction({ label, verb, reasonLabel, busyLabel, busy, disabled, onAction }: { label: string; verb: string; reasonLabel: string; busyLabel: string; busy: boolean; disabled: boolean; onAction: (rationale: string) => Promise<void> }) {
  const [rationale, setRationale] = useState("");
  return <form onSubmit={(event) => { event.preventDefault(); if (rationale.trim()) void onAction(rationale.trim()); }}>
    <TextField label={reasonLabel} value={rationale} isDisabled={disabled || busy} onChange={setRationale}/>
    <Button type="submit" isDisabled={disabled || busy || !rationale.trim()}>{busy ? busyLabel : `${verb} ${label}`}</Button>
  </form>;
}

function dateWindow(from?: string, until?: string) {
  return `${from ? formatDate(from) : "Not scheduled"} to ${until ? formatDate(until) : "No end date"}`;
}

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false, timeZone: "UTC", timeZoneName: "short" }).format(date);
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
