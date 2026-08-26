import { useCallback, useEffect, useRef, useState } from "react";
import { loadContext, type RuntimeContext } from "../api";
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
} from "./GovernanceAdminWorkspace";

export function GovernanceAdminPanel() {
  const [inventory, setInventory] = useState<GovernanceInventory>({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: false });
  const [context, setContext] = useState<RuntimeContext | null>(null);
  const [overview, setOverview] = useState<IdentityAccessOverview | null>(null);
  const [loadState, setLoadState] = useState<GovernanceLoadState>("loading");
  const [reason, setReason] = useState("");
  const mounted = useRef(true);
  const loadGeneration = useRef(0);

  const load = useCallback(async (initial: boolean) => {
    const generation = ++loadGeneration.current;
    setLoadState("loading");
    const [inventoryResult, contextResult, overviewResult] = await Promise.allSettled([
      loadGovernanceInventory(),
      initial || !context ? loadContext() : Promise.resolve(context),
      initial || !overview ? loadIdentityAccessOverview() : Promise.resolve(overview),
    ]);
    if (!mounted.current || generation !== loadGeneration.current) return;
    if (contextResult.status === "fulfilled") setContext(contextResult.value);
    if (overviewResult.status === "fulfilled") setOverview(overviewResult.value);
    else if (initial) setOverview(null);
    if (inventoryResult.status === "fulfilled" && contextResult.status === "fulfilled") {
      const loaded = inventoryResult.value;
      setInventory((current) => ({
        policies: loaded.policiesAvailable ? loaded.policies : current.policies,
        delegations: loaded.delegationsAvailable ? loaded.delegations : current.delegations,
        policiesAvailable: loaded.policiesAvailable,
        delegationsAvailable: loaded.delegationsAvailable,
      }));
      const directoryUnavailable = overviewResult.status !== "fulfilled";
      const sectionUnavailable = !loaded.policiesAvailable || !loaded.delegationsAvailable;
      if (directoryUnavailable || sectionUnavailable) {
        setReason(directoryUnavailable
          ? "Current people and legal-entity labels could not be confirmed."
          : "One governance inventory section could not be refreshed.");
        setLoadState("degraded");
      } else {
        setReason("");
        setLoadState("ready");
      }
      return;
    }
    setReason("The latest legal-entity governance inventory could not be confirmed.");
    setLoadState("unavailable");
  }, [context, inventory.delegations.length, inventory.policies.length, overview]);

  useEffect(() => {
    mounted.current = true;
    void load(true);
    return () => { mounted.current = false; };
  }, []); // the first load owns its request lifecycle; later refreshes are command-driven

  const directoryEntity = overview?.legal_entities?.find((value) => value.id === context?.legal_entity.id);
  const contextEntityLabel = context && context.legal_entity.name !== context.legal_entity.id ? context.legal_entity.name : "Current legal entity";
  const entity = context ? { id: context.legal_entity.id, label: directoryEntity?.name ?? contextEntityLabel } : { id: "", label: "Current legal entity" };
  const parties = new Map((overview?.people ?? []).map((person) => [person.id, person.display_name]));
  if (context?.actor.id) parties.set(context.actor.id, context.actor.name);
  const party = (id: string) => ({ id, label: parties.get(id) ?? "Recorded person unavailable" });
  const policies = inventory.policies.map((record) => mapPolicy(record, entity, party));
  const delegations = inventory.delegations.map((record) => mapDelegation(record, entity, party));
  async function create(input: CreateGovernanceDelegationInput) {
    const created = await createGovernanceDelegation(input);
    setInventory((current) => ({ ...current, delegations: upsert(current.delegations, created) }));
    void load(false);
  }

  async function createPolicyDraft(input: CreateGovernancePolicyDraftInput) {
    const created = await createGovernancePolicyDraftAPI(input);
    setInventory((current) => ({ ...current, policies: upsert(current.policies, created) }));
    void load(false);
  }

  async function policyAction(input: GovernancePolicyActionInput) {
    const updated = await transitionGovernancePolicy(input.policyId, input.action, input.expectedVersion, input.rationale);
    setInventory((current) => ({ ...current, policies: upsert(current.policies, updated) }));
    void load(false);
  }

  async function delegationAction(input: GovernanceDelegationActionInput) {
    const updated = await transitionGovernanceDelegation(input.delegationId, input.action, input.expectedVersion, input.rationale);
    setInventory((current) => ({ ...current, delegations: upsert(current.delegations, updated) }));
    void load(false);
  }

  return <GovernanceAdminWorkspace
    policies={policies}
    delegations={delegations}
    eligiblePeople={[]}
    currentEntity={context ? entity : undefined}
    policyRoles={(overview?.roles ?? []).map((role) => ({ code: role.code, label: role.name }))}
    actorId={context?.actor.id ?? ""}
    canConfigure={Boolean(context?.capabilities?.config_write)}
    delegationCreationAvailable={Boolean(context)}
    loadState={loadState}
    degradedReason={reason}
    createDelegation={create}
    loadDelegationCandidates={searchGovernanceDelegationCandidates}
    createPolicyDraft={createPolicyDraft}
    policyAction={policyAction}
    delegationAction={delegationAction}
  />;
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
