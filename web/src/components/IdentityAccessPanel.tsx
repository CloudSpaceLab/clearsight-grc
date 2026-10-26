import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "../http";
import {
  approveOrganizationPosition,
  approveOrganizationPositionRole,
  approveLegalEntityDataBoundary,
  approveOrganizationScope,
  createGroupRoleBinding,
  createIdentitySource,
  loadIdentityAccessOverview,
  proposeOrganizationPosition,
  proposeOrganizationPositionRole,
  proposeLegalEntityDataBoundary,
  proposeOrganizationScope,
  rejectOrganizationPosition,
  rejectOrganizationPositionRole,
  rejectLegalEntityDataBoundary,
  rejectOrganizationScope,
  restoreOrganizationPosition,
  simulateOrganizationPosition,
  retireGroupRoleBinding,
  revokeIdentitySource,
  rotateIdentitySourceToken,
  type GroupRoleBinding,
  type IdentityAccessOverview,
  type IdentitySource,
  type OrganizationPositionRevision,
  type OrganizationPositionRoleRevision,
  type OrganizationPositionRouteSimulation,
  type LegalEntityDataBoundaryRevision,
  type OrganizationScopeRevision,
  type ProposeOrganizationPositionInput,
  type ProposeOrganizationPositionRoleInput,
  type ProposeLegalEntityDataBoundaryInput,
  type ProposeOrganizationScopeInput,
} from "../identityAccessApi";
import "../identity-access.css";
import { GroupRoleBindingComposer, ProvisioningSourceComposer } from "./access/IdentityAccessComposers";
import { IdentityAccessInventory } from "./access/IdentityAccessInventory";
import { EscalationRoutesWorkspace } from "./access/EscalationRoutesWorkspace";
import { OrganizationInventory } from "./access/OrganizationInventory";
import { PeopleDirectory } from "./access/PeopleDirectory";
import { PeopleRoleCatalog } from "./access/PeopleRoleCatalog";
import { Button, MetricCard, Notice } from "./ui";
import "./access/people-access.css";

type Composer = "source" | "binding" | null;
export type IdentityAccessArea = "people" | "roles" | "positions" | "reporting" | "directory" | "escalation";
type WorkspaceArea = IdentityAccessArea;
type Props = {
  area?: IdentityAccessArea;
  onAreaChange?: (area: IdentityAccessArea) => void;
  organizationName?: string;
  legalEntityName?: string;
  scopeKey?: string;
  canRead?: boolean;
  active?: boolean;
  selectedPersonID?: string;
  onSelectPerson?: (id: string | undefined) => void;
  selectedRoleID?: string;
  onSelectRole?: (id: string | undefined) => void;
};

export function IdentityAccessPanel(props: Props) {
  return <IdentityAccessWorkspace key={props.scopeKey || "access"} {...props}/>;
}

function IdentityAccessWorkspace({ area: controlledArea, onAreaChange, organizationName, legalEntityName, canRead = true, active = true, selectedPersonID, onSelectPerson, selectedRoleID, onSelectRole }: Props) {
  const [overview, setOverview] = useState<IdentityAccessOverview | null>(null);
  const [state, setState] = useState<"loading" | "live" | "restricted" | "unavailable">("loading");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [token, setToken] = useState("");
  const [composer, setComposer] = useState<Composer>(null);
  const [localArea, setLocalArea] = useState<WorkspaceArea>("positions");
  const [reload, setReload] = useState(0);
  const [refreshState, setRefreshState] = useState<"idle" | "loading" | "error">("idle");
  const readController = useRef<AbortController | null>(null);
  const readGeneration = useRef(0);
  const currentAccess = useRef({ active, canRead, mounted: true, generation: 0 });
  if (currentAccess.current.active !== active || currentAccess.current.canRead !== canRead) currentAccess.current.generation += 1;
  currentAccess.current.active = active;
  currentAccess.current.canRead = canRead;
  const area = controlledArea ?? localArea;
  const [visitedAreas, setVisitedAreas] = useState<WorkspaceArea[]>([area]);
  const embedded = controlledArea !== undefined;
  const panelClass = `identity-access-panel${embedded ? " identity-access-panel--embedded" : ""}`;

  useEffect(() => {
    currentAccess.current.mounted = true;
    return () => { currentAccess.current.mounted = false; };
  }, []);

  useEffect(() => {
    setVisitedAreas((current) => current.includes(area) ? current : [...current, area]);
  }, [area]);

  function setArea(next: WorkspaceArea) {
    setLocalArea(next);
    onAreaChange?.(next);
  }

  async function readOverview(retainRecords: boolean) {
    if (!currentAccess.current.mounted || !currentAccess.current.canRead || !currentAccess.current.active) return;
    readController.current?.abort();
    const controller = new AbortController();
    readController.current = controller;
    const generation = ++readGeneration.current;
    if (retainRecords) setRefreshState("loading");
    else setState("loading");
    try {
      const value = await loadIdentityAccessOverview(controller.signal);
      if (controller.signal.aborted || generation !== readGeneration.current) return;
      setOverview(value);
      setState("live");
      setRefreshState("idle");
    } catch (error) {
      if (controller.signal.aborted || generation !== readGeneration.current) return;
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        setOverview(null);
        setState("restricted");
        setRefreshState("idle");
      } else if (retainRecords) setRefreshState("error");
      else {
        setOverview(null);
        setState("unavailable");
      }
    }
  }

  useEffect(() => {
    if (!active || !canRead) {
      setComposer(null);
      setToken("");
    }
    if (!canRead) {
      setOverview(null);
      setState("restricted");
    } else if (active) {
      void readOverview(overview !== null);
    }
    return () => readController.current?.abort();
  }, [canRead, active, reload]); // Reactivation rechecks the snapshot while retained view state stays mounted.

  async function refresh() { await readOverview(overview !== null); }

  function acceptsResult(generation: number) {
    const current = currentAccess.current;
    return current.mounted && current.active && current.canRead && generation === current.generation;
  }

  const activeGroups = useMemo(() => overview?.groups.filter((group) => group.source_state === "ACTIVE") ?? [], [overview]);
  const activeRoles = overview?.roles ?? [];

  async function run(label: string, action: () => Promise<void>): Promise<boolean> {
    if (!active || !canRead || state !== "live" || refreshState !== "idle") return false;
    setBusy(label);
    setNotice("");
    try {
      await action();
      return true;
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "The change could not be completed.");
      return false;
    } finally {
      setBusy("");
    }
  }

  async function createSource(input: { code: string; identity_issuer?: string; subject_attribute: "externalId" | "userName" }) {
    const generation = currentAccess.current.generation;
    return run("source-create", async () => {
      const result = await createIdentitySource(input);
      if (!acceptsResult(generation)) return;
      setToken(result.token);
      setNotice("Provisioning source created. Copy the token now; it will not be shown again.");
      setOverview((current) => current ? {
        ...current,
        sources: [...current.sources, result.source].sort((left, right) => left.code.localeCompare(right.code)),
      } : current);
    });
  }

  async function createBinding(input: { group_id: string; role_template_id: string; department_path: string[] }) {
    const generation = currentAccess.current.generation;
    return run("binding-create", async () => {
      const binding = await createGroupRoleBinding(input);
      if (!acceptsResult(generation)) return;
      setNotice("Directory group role mapping added. Material decision authority is unchanged.");
      setOverview((current) => current ? { ...current, bindings: [...current.bindings, binding] } : current);
    });
  }

  async function rotateSource(source: IdentitySource) {
    const generation = currentAccess.current.generation;
    await run(`rotate-${source.id}`, async () => {
      const next = await rotateIdentitySourceToken(source.id);
      if (!acceptsResult(generation)) return;
      setToken(next.token);
      setNotice("Token rotated. The previous token is no longer valid.");
    });
  }

  async function revokeSource(source: IdentitySource) {
    if (!window.confirm(`Revoke ${source.code}? Source-derived access will stop on the next request.`)) return;
    await run(`revoke-${source.id}`, async () => {
      await revokeIdentitySource(source.id);
      setOverview((current) => current ? { ...current, sources: current.sources.map((item) => item.id === source.id ? { ...item, status: "REVOKED" } : item) } : current);
      setNotice("Provisioning source revoked. Historical principals remain recorded, but source-derived eligibility is disabled.");
    });
  }

  async function retireBinding(binding: GroupRoleBinding) {
    if (!window.confirm("Retire this group role mapping?")) return;
    await run(`retire-${binding.id}`, async () => {
      await retireGroupRoleBinding(binding.id);
      setOverview((current) => current ? { ...current, bindings: current.bindings.filter((item) => item.id !== binding.id) } : current);
      setNotice("Group role mapping retired.");
    });
  }


  async function proposeBoundary(input: ProposeLegalEntityDataBoundaryInput) {
    return run("data-boundary-propose", async () => {
      await proposeLegalEntityDataBoundary(input);
      setNotice("Data boundary change proposed.");
      await refresh();
    });
  }

  async function approveBoundary(revision: LegalEntityDataBoundaryRevision, rationale: string) {
    return run("data-boundary-approve-" + revision.id, async () => {
      await approveLegalEntityDataBoundary(revision.id, rationale);
      setNotice("Data boundary approved.");
      await refresh();
    });
  }

  async function rejectBoundary(revision: LegalEntityDataBoundaryRevision, rationale: string) {
    return run("data-boundary-reject-" + revision.id, async () => {
      await rejectLegalEntityDataBoundary(revision.id, rationale);
      setNotice("Data boundary rejected.");
      await refresh();
    });
  }

  async function proposeScope(input: ProposeOrganizationScopeInput) {
    return run("scope-propose", async () => {
      await proposeOrganizationScope(input);
      setNotice("Change proposed.");
      await refresh();
    });
  }

  async function approveScope(revision: OrganizationScopeRevision, rationale: string) {
    return run("scope-approve-" + revision.id, async () => {
      await approveOrganizationScope(revision.id, rationale);
      setNotice("Change approved.");
      await refresh();
    });
  }

  async function rejectScope(revision: OrganizationScopeRevision, rationale: string) {
    return run("scope-reject-" + revision.id, async () => {
      await rejectOrganizationScope(revision.id, rationale);
      setNotice("Change rejected.");
      await refresh();
    });
  }

  async function proposePosition(input: ProposeOrganizationPositionInput) {
    return run("position-propose", async () => {
      await proposeOrganizationPosition(input);
      setNotice("Position change proposed.");
      await refresh();
    });
  }

  async function simulatePosition(revision: OrganizationPositionRevision): Promise<OrganizationPositionRouteSimulation | null> {
    if (!active || !canRead || state !== "live" || refreshState !== "idle") return null;
    setBusy("position-simulate-" + revision.id);
    setNotice("");
    try {
      return await simulateOrganizationPosition(revision.id);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Route impact could not be checked.");
      return null;
    } finally {
      setBusy("");
    }
  }

  async function restorePosition(revision: OrganizationPositionRevision) {
    return run("position-restore-" + revision.id, async () => {
      await restoreOrganizationPosition(revision.id);
      setNotice("Restore proposed.");
      await refresh();
    });
  }

  async function approvePosition(revision: OrganizationPositionRevision, rationale: string) {
    return run("position-approve-" + revision.id, async () => {
      await approveOrganizationPosition(revision.id, rationale);
      setNotice("Position change approved.");
      await refresh();
    });
  }

  async function rejectPosition(revision: OrganizationPositionRevision, rationale: string) {
    return run("position-reject-" + revision.id, async () => {
      await rejectOrganizationPosition(revision.id, rationale);
      setNotice("Position change rejected.");
      await refresh();
    });
  }

  async function proposePositionRole(input: ProposeOrganizationPositionRoleInput) {
    return run("position-role-propose", async () => {
      await proposeOrganizationPositionRole(input);
      setNotice("Workspace role change proposed.");
      await refresh();
    });
  }

  async function approvePositionRole(revision: OrganizationPositionRoleRevision, rationale: string) {
    return run("position-role-approve-" + revision.id, async () => {
      await approveOrganizationPositionRole(revision.id, rationale);
      setNotice("Workspace role change approved.");
      await refresh();
    });
  }

  async function rejectPositionRole(revision: OrganizationPositionRoleRevision, rationale: string) {
    return run("position-role-reject-" + revision.id, async () => {
      await rejectOrganizationPositionRole(revision.id, rationale);
      setNotice("Workspace role change rejected.");
      await refresh();
    });
  }

  if (!canRead || state === "restricted") return <section className={panelClass}><Notice><strong>People & access restricted.</strong> Ask your administrator for access to identity settings.</Notice></section>;
  if (state === "loading") return <section className={panelClass} aria-busy="true"><Notice>Loading people and access settings…</Notice></section>;
  if (state === "unavailable" || !overview) return <section className={panelClass}><Notice tone="error">People and access settings could not be loaded.</Notice><Button onPress={() => setReload((value) => value + 1)}>Retry</Button></section>;

  const isBusy = busy !== "" || refreshState !== "idle" || !active;
  const canChange = active && refreshState === "idle";

  return <section className={panelClass} aria-label="Identity and access configuration">
    {!embedded && <div className="section-header identity-access-heading"><div><span className="eyebrow">Organization & access</span><h2>Organization & access</h2></div><span className="identity-health">{overview.sources.filter((source) => source.status === "ACTIVE").length} active source{overview.sources.filter((source) => source.status === "ACTIVE").length === 1 ? "" : "s"}</span></div>}
    {notice && <div className="inline-notice" role="status">{notice}</div>}
    {refreshState !== "idle" && <Notice tone={refreshState === "error" ? "warning" : "info"}>{refreshState === "loading" ? "Checking the latest people and access settings…" : <>The latest access settings could not be loaded. The previous list remains visible; reload before making a change. <Button variant="quiet" onPress={() => setReload((value) => value + 1)}>Retry access settings</Button></>}</Notice>}
    {token && <div className="identity-token" role="status"><div><strong>Provisioning token — shown once</strong><p>Copy this token to your SCIM provider now. It cannot be recovered after you leave this screen.</p><code>{token}</code></div><div className="identity-token-actions"><button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(token)}>Copy</button><button className="text-button" type="button" onClick={() => setToken("")}>Hide</button></div></div>}

    {!embedded && <div className="identity-access-tabs" role="tablist" aria-label="Identity and access areas">
      <AreaTab area="positions" active={area} onSelect={setArea}>Organization</AreaTab>
      <AreaTab area="reporting" active={area} onSelect={setArea}>Reporting lines</AreaTab>
      <AreaTab area="directory" active={area} onSelect={setArea}>Directory groups & access</AreaTab>
      <AreaTab area="escalation" active={area} onSelect={setArea}>Escalation routes</AreaTab>
    </div>}

    {(area === "people" || area === "roles") && <div className="people-access-summary" aria-label="Loaded access records">
      <MetricCard label="People" value={overview.people.length} quality={refreshState === "idle" ? "partial" : "stale"} qualityLabel={refreshState === "idle" ? "Loaded" : "Previous list"}/>
      <MetricCard label="Groups" value={overview.groups.length} quality={refreshState === "idle" ? "partial" : "stale"} qualityLabel={refreshState === "idle" ? "Loaded" : "Previous list"}/>
      <MetricCard label="Sources" value={overview.sources.length} quality={refreshState === "idle" ? "partial" : "stale"} qualityLabel={refreshState === "idle" ? "Loaded" : "Previous list"}/>
      <MetricCard label="Roles" value={overview.roles.length} quality={refreshState === "idle" ? "partial" : "stale"} qualityLabel={refreshState === "idle" ? "Loaded" : "Previous list"}/>
    </div>}
    <div className="identity-access-area" role={embedded ? undefined : "tabpanel"} aria-label={embedded ? undefined : areaLabel(area)}>
      {embedded && (area === "people" || visitedAreas.includes("people")) && <div hidden={area !== "people"}><PeopleDirectory people={overview.people} positions={overview.positions} roles={overview.roles} organizationName={organizationName} legalEntityName={legalEntityName} selectedPersonID={selectedPersonID} onSelectPerson={onSelectPerson} onOpenOrganization={() => setArea("positions")}/></div>}
      {embedded && (area === "roles" || visitedAreas.includes("roles")) && <div hidden={area !== "roles"}><PeopleRoleCatalog roles={overview.roles} bindings={overview.bindings} legalEntityName={legalEntityName} selectedRoleID={selectedRoleID} onSelectRole={onSelectRole} onOpenOrganization={() => setArea("positions")} onOpenMappings={() => setArea("directory")}/></div>}
      {(area === "positions" || area === "reporting") && <OrganizationInventory
        positions={overview.positions}
        people={overview.people}
        scopes={overview.organization_scopes}
        revisions={overview.organization_scope_revisions}
        positionRevisions={overview.organization_position_revisions}
        positionHistory={overview.organization_position_history}
        positionRoleRevisions={overview.organization_position_role_revisions}
        roles={overview.roles}
        dataBoundary={overview.data_boundary}
        dataBoundaryRevisions={overview.data_boundary_revisions}
        actorPrincipalID={overview.actor_principal_id}
        canConfigure={overview.can_configure_organization && canChange}
        isBusy={isBusy}
        scopesTruncated={overview.organization_scopes_truncated === true}
        mode={area}
        onProposeScope={proposeScope}
        onApproveScope={approveScope}
        onRejectScope={rejectScope}
        onProposePosition={proposePosition}
        onApprovePosition={approvePosition}
        onRejectPosition={rejectPosition}
        onSimulatePosition={simulatePosition}
        onRestorePosition={restorePosition}
        onProposePositionRole={proposePositionRole}
        onApprovePositionRole={approvePositionRole}
        onRejectPositionRole={rejectPositionRole}
        onProposeDataBoundary={proposeBoundary}
        onApproveDataBoundary={approveBoundary}
        onRejectDataBoundary={rejectBoundary}
      />}

      {area === "directory" && <div className="identity-access-grid">
      <IdentityAccessInventory
        signIn={overview.sign_in}
        sources={overview.sources}
        bindings={overview.bindings}
        groups={overview.groups}
        canConfigure={overview.can_configure && canChange}
        busy={isBusy}
        onAddSource={() => setComposer("source")}
        onAddMapping={() => setComposer("binding")}
        onRotateSource={(source) => void rotateSource(source)}
        onRevokeSource={(source) => void revokeSource(source)}
        onRetireBinding={(binding) => void retireBinding(binding)}
      />
      </div>}

      {area === "escalation" && <EscalationRoutesWorkspace
        overview={overview}
        isBusy={isBusy}
        onBusy={(value) => setBusy(value ? "escalation" : "")}
        onNotice={setNotice}
        onRefresh={refresh}
      />}
    </div>

    <ProvisioningSourceComposer open={composer === "source"} busy={busy === "source-create"} onClose={() => setComposer(null)} onCreate={createSource}/>
    <GroupRoleBindingComposer open={composer === "binding"} busy={busy === "binding-create"} groups={activeGroups} roles={activeRoles} onClose={() => setComposer(null)} onCreate={createBinding}/>
  </section>;
}

function AreaTab({ area, active, onSelect, children }: { area: WorkspaceArea; active: WorkspaceArea; onSelect: (area: WorkspaceArea) => void; children: string }) {
  return <button type="button" role="tab" aria-selected={active === area} onClick={() => onSelect(area)}>{children}</button>;
}

function areaLabel(area: WorkspaceArea) {
  return ({ people: "People", roles: "Roles", positions: "Organization", reporting: "Reporting lines", directory: "Directory groups and access", escalation: "Escalation routes" } as const)[area];
}
