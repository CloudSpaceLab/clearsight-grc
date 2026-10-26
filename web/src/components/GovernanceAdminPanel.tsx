import { useEffect, useRef, useState } from "react";
import { loadContext, type RuntimeContext } from "../api";
import { ApiError } from "../http";
import {
  createGovernanceDelegation,
  createGovernancePolicyDraft as createGovernancePolicyDraftAPI,
  loadGovernanceInventory,
  searchGovernanceDelegationCandidates,
  transitionGovernanceDelegation,
  transitionGovernancePolicy,
  type GovernanceDelegationRecord,
  type GovernanceInventory,
  type GovernancePolicyRecord,
  type GovernanceReadState,
} from "../governanceAdminApi";
import { loadIdentityAccessOverview, type IdentityAccessOverview } from "../identityAccessApi";
import {
  GovernanceAdminWorkspace,
  type CreateGovernanceDelegationInput,
  type CreateGovernancePolicyDraftInput,
  type GovernanceDelegationActionInput,
  type GovernanceDelegationItem,
  type GovernanceLoadState,
  type GovernancePolicyActionInput,
  type GovernancePolicyItem,
  type GovernanceView,
} from "./GovernanceAdminWorkspace";

type Props = {
  view?: GovernanceView;
  selectedPolicyID?: string;
  onSelectPolicy?: (id: string | undefined) => void;
  selectedDelegationID?: string;
  onSelectDelegation?: (id: string | undefined) => void;
  runtimeContext?: RuntimeContext;
  scopeKey?: string;
  active?: boolean;
  onOpenRoutingChecks?: () => void;
};

export function GovernanceAdminPanel(props: Props = {}) {
  const context = props.runtimeContext;
  const scope = props.scopeKey ?? (context ? `${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}` : "governance");
  return <ScopedGovernanceAdminPanel key={scope} {...props}/>;
}

function ScopedGovernanceAdminPanel({ view, selectedPolicyID, onSelectPolicy, selectedDelegationID, onSelectDelegation, runtimeContext, active = true, onOpenRoutingChecks }: Props) {
  const [inventory, setInventory] = useState<GovernanceInventory>({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: false });
  const [context, setContext] = useState<RuntimeContext | null>(runtimeContext ?? null);
  const [overview, setOverview] = useState<IdentityAccessOverview | null>(null);
  const [loadState, setLoadState] = useState<GovernanceLoadState>("loading");
  const [reason, setReason] = useState("");
  const mounted = useRef(true);
  const lifecycle = useRef(0);
  const [reload, setReload] = useState(0);

  useEffect(() => {
    lifecycle.current += 1;
    mounted.current = active;
    if (!active) return;
    const controller = new AbortController();
    setLoadState("loading");
    async function load() {
      let verified: RuntimeContext;
      try {
        verified = runtimeContext ?? await loadContext();
      } catch (error) {
        if (controller.signal.aborted) return;
        const state = readFailure(error);
        setContext(null);
        setOverview(null);
        setInventory({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: false, policiesState: state, delegationsState: state });
        setReason(state === "restricted" ? "Your sign-in does not allow governance records." : "Your current legal entity could not be confirmed. Reload to try again.");
        setLoadState(state === "restricted" ? "restricted" : "unavailable");
        return;
      }
      if (controller.signal.aborted) return;
      setContext(verified);
      if (verified.capabilities?.config_read === false) {
        setOverview(null);
        setInventory({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: false, policiesState: "restricted", delegationsState: "restricted" });
        setReason("Your current access does not allow routing policies or delegations.");
        setLoadState("restricted");
        return;
      }
      const [inventoryResult, overviewResult] = await Promise.allSettled([
        loadGovernanceInventory(controller.signal),
        verified.capabilities?.identity_read === false ? Promise.reject(new ApiError(403, "Directory access restricted.")) : loadIdentityAccessOverview(controller.signal),
      ]);
      if (controller.signal.aborted) return;
      setOverview(overviewResult.status === "fulfilled" ? overviewResult.value : null);
      if (inventoryResult.status === "fulfilled") {
        const loaded = inventoryResult.value;
        const policiesState = loaded.policiesState ?? (loaded.policiesAvailable ? "ready" : "unavailable");
        const delegationsState = loaded.delegationsState ?? (loaded.delegationsAvailable ? "ready" : "unavailable");
        setInventory((current) => ({
          policies: policiesState === "restricted" ? [] : policiesState === "ready" ? loaded.policies : current.policies,
          delegations: delegationsState === "restricted" ? [] : delegationsState === "ready" ? loaded.delegations : current.delegations,
          policiesAvailable: policiesState === "ready", delegationsAvailable: delegationsState === "ready",
          policiesState, delegationsState,
        }));
        const unavailable = [
          policiesState === "restricted" ? "Routing policy access is restricted." : policiesState === "unavailable" ? "Routing policies are unavailable." : "",
          delegationsState === "restricted" ? "Delegation access is restricted." : delegationsState === "unavailable" ? "Delegations are unavailable." : "",
          overviewResult.status !== "fulfilled" ? "Current people and legal-entity labels could not be confirmed." : "",
        ].filter(Boolean);
        setReason(unavailable.join(" "));
        setLoadState(unavailable.length ? "degraded" : "ready");
      } else {
        const state = readFailure(inventoryResult.reason);
        setInventory((current) => ({ ...current, ...(state === "restricted" ? { policies: [], delegations: [] } : {}), policiesAvailable: false, delegationsAvailable: false, policiesState: state, delegationsState: state }));
        setReason(state === "restricted" ? "Governance access is restricted." : "Routing policies and delegations could not be refreshed.");
        setLoadState(state === "restricted" ? "restricted" : "unavailable");
      }
    }
    void load();
    return () => { mounted.current = false; controller.abort(); };
  }, [runtimeContext, reload, active]);

  function refresh() {
    if (!mounted.current) return;
    setLoadState("loading");
    setReload((value) => value + 1);
  }

  const directoryEntity = overview?.legal_entities?.find((value) => value.id === context?.legal_entity.id);
  const contextEntityLabel = context && context.legal_entity.name !== context.legal_entity.id ? context.legal_entity.name : "Current legal entity";
  const entity = context ? { id: context.legal_entity.id, label: directoryEntity?.name ?? contextEntityLabel } : { id: "", label: "Current legal entity" };
  const parties = new Map((overview?.people ?? []).map((person) => [person.id, person.display_name]));
  if (context?.actor.id) parties.set(context.actor.id, context.actor.name);
  const party = (id: string) => ({ id, label: parties.get(id) ?? (id ? `Name unavailable · ${id}` : "Person not recorded") });
  const policies = inventory.policies.map((record) => mapPolicy(record, entity, party));
  const delegations = inventory.delegations.map((record) => mapDelegation(record, entity, party));
  async function create(input: CreateGovernanceDelegationInput) {
    const generation = lifecycle.current;
    const created = await createGovernanceDelegation(input);
    if (!mounted.current) return;
    if (generation !== lifecycle.current) { refresh(); return; }
    setInventory((current) => ({ ...current, delegations: upsert(current.delegations, created) }));
    onSelectDelegation?.(created.id);
    refresh();
  }

  async function createPolicyDraft(input: CreateGovernancePolicyDraftInput) {
    const generation = lifecycle.current;
    const created = await createGovernancePolicyDraftAPI(input);
    if (!mounted.current) return;
    if (generation !== lifecycle.current) { refresh(); return; }
    setInventory((current) => ({ ...current, policies: upsert(current.policies, created) }));
    onSelectPolicy?.(created.id);
    refresh();
  }

  async function policyAction(input: GovernancePolicyActionInput) {
    const generation = lifecycle.current;
    const updated = await transitionGovernancePolicy(input.policyId, input.action, input.expectedVersion, input.rationale);
    if (!mounted.current) return;
    if (generation !== lifecycle.current) { refresh(); return; }
    setInventory((current) => ({ ...current, policies: upsert(current.policies, updated) }));
    refresh();
  }

  async function delegationAction(input: GovernanceDelegationActionInput) {
    const generation = lifecycle.current;
    const updated = await transitionGovernanceDelegation(input.delegationId, input.action, input.expectedVersion, input.rationale);
    if (!mounted.current) return;
    if (generation !== lifecycle.current) { refresh(); return; }
    setInventory((current) => ({ ...current, delegations: upsert(current.delegations, updated) }));
    refresh();
  }

  return <GovernanceAdminWorkspace
    active={active}
    onOpenRoutingChecks={onOpenRoutingChecks}
    view={view}
    selectedPolicyID={selectedPolicyID}
    onSelectPolicy={onSelectPolicy}
    selectedDelegationID={selectedDelegationID}
    onSelectDelegation={onSelectDelegation}
    policies={policies}
    delegations={delegations}
    eligiblePeople={[]}
    currentEntity={context ? entity : undefined}
    policyRoles={(overview?.roles ?? []).map((role) => ({ code: role.code, label: role.name }))}
    actorId={context?.actor.id ?? ""}
    canConfigure={Boolean(context?.capabilities?.config_write)}
    delegationCreationAvailable={Boolean(context)}
    loadState={active ? loadState : "loading"}
    degradedReason={reason}
    policiesState={inventory.policiesState}
    delegationsState={inventory.delegationsState}
    onReload={refresh}
    createDelegation={create}
    loadDelegationCandidates={searchGovernanceDelegationCandidates}
    createPolicyDraft={createPolicyDraft}
    policyAction={policyAction}
    delegationAction={delegationAction}
  />;
}

function readFailure(error: unknown): GovernanceReadState {
  return error instanceof ApiError && (error.status === 401 || error.status === 403) ? "restricted" : "unavailable";
}

function mapPolicy(record: GovernancePolicyRecord, entity: { id: string; label: string }, party: (id: string) => { id: string; label: string }): GovernancePolicyItem {
  return {
    id: record.id,
    code: record.code,
    name: record.name,
    status: record.status,
    legalEntity: entity,
    currentVersion: record.current_version,
    version: record.version,
    effectiveFrom: record.effective_from,
    effectiveUntil: record.effective_until,
    maker: party(record.maker_id),
    checker: record.checker_id ? party(record.checker_id) : undefined,
    latestDecision: record.latest_decision ? {
      fromState: record.latest_decision.from_state,
      toState: record.latest_decision.to_state,
      actor: party(record.latest_decision.actor_id ?? ""),
      rationale: record.latest_decision.rationale,
      decidedAt: record.latest_decision.decided_at,
      recordVersion: record.latest_decision.record_version,
    } : undefined,
  };
}

function mapDelegation(record: GovernanceDelegationRecord, entity: { id: string; label: string }, party: (id: string) => { id: string; label: string }): GovernanceDelegationItem {
  return {
    id: record.id,
    status: record.status,
    legalEntity: entity,
    from: party(record.from_principal_id),
    to: party(record.to_principal_id),
    responsibility: record.responsibility,
    startsAt: record.starts_at,
    endsAt: record.ends_at,
    reason: record.reason,
    scope: record.scope,
    version: record.version,
    maker: party(record.maker_id),
    checker: record.approver_id ? party(record.approver_id) : undefined,
  };
}

function upsert<T extends { id: string }>(values: T[], value: T) {
  const existing = values.findIndex((item) => item.id === value.id);
  if (existing < 0) return [value, ...values];
  return values.map((item, index) => index === existing ? value : item);
}
