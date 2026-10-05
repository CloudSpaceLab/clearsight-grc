import { useEffect, useMemo, useState } from "react";
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

type Composer = "source" | "binding" | null;
type WorkspaceArea = "positions" | "reporting" | "directory" | "escalation";

export function IdentityAccessPanel() {
  const [overview, setOverview] = useState<IdentityAccessOverview | null>(null);
  const [state, setState] = useState<"loading" | "live" | "restricted" | "unavailable">("loading");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");
  const [token, setToken] = useState("");
  const [composer, setComposer] = useState<Composer>(null);
  const [area, setArea] = useState<WorkspaceArea>("positions");

  async function load() {
    setState("loading");
    try {
      setOverview(await loadIdentityAccessOverview());
      setState("live");
    } catch (error) {
      setOverview(null);
      setState(error instanceof ApiError && (error.status === 401 || error.status === 403) ? "restricted" : "unavailable");
    }
  }

  async function refresh() {
    try {
      setOverview(await loadIdentityAccessOverview());
    } catch {
      setNotice("The change was recorded, but the latest access inventory could not be refreshed. Reload before making another change.");
    }
  }

  useEffect(() => { void load(); }, []);

  const activeGroups = useMemo(() => overview?.groups.filter((group) => group.source_state === "ACTIVE") ?? [], [overview]);
  const activeRoles = overview?.roles ?? [];

  async function run(label: string, action: () => Promise<void>): Promise<boolean> {
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
    return run("source-create", async () => {
      const result = await createIdentitySource(input);
      setToken(result.token);
      setNotice("Provisioning source created. Copy the token now; it will not be shown again.");
      setOverview((current) => current ? {
        ...current,
        sources: [...current.sources, result.source].sort((left, right) => left.code.localeCompare(right.code)),
      } : current);
    });
  }

  async function createBinding(input: { group_id: string; role_template_id: string; department_path: string[] }) {
    return run("binding-create", async () => {
      const binding = await createGroupRoleBinding(input);
      setNotice("Directory group role mapping added. Material decision authority is unchanged.");
      setOverview((current) => current ? { ...current, bindings: [...current.bindings, binding] } : current);
    });
  }

  async function rotateSource(source: IdentitySource) {
    await run(`rotate-${source.id}`, async () => {
      const next = await rotateIdentitySourceToken(source.id);
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

  if (state === "loading") return <section className="identity-access-panel" aria-busy="true"><div className="section-header"><div><span className="eyebrow">Organization & access</span><h2>Loading…</h2></div></div></section>;
  if (state === "restricted") return <section className="identity-access-panel"><div className="section-header"><div><span className="eyebrow">Organization & access</span><h2>Access restricted</h2></div></div></section>;
  if (state === "unavailable" || !overview) return <section className="identity-access-panel"><div className="section-header"><div><span className="eyebrow">Organization & access</span><h2>Unavailable</h2></div><button className="secondary-button" type="button" onClick={() => void load()}>Retry</button></div></section>;

  const isBusy = busy !== "";

  return <section className="identity-access-panel" aria-label="Identity and access configuration">
    <div className="section-header identity-access-heading"><div><span className="eyebrow">Organization & access</span><h2>Organization & access</h2></div><span className="identity-health">{overview.sources.filter((source) => source.status === "ACTIVE").length} active source{overview.sources.filter((source) => source.status === "ACTIVE").length === 1 ? "" : "s"}</span></div>
    {notice && <div className="inline-notice" role="status">{notice}</div>}
    {token && <div className="identity-token" role="status"><div><strong>Provisioning token — shown once</strong><p>Copy this token to your SCIM provider now. It cannot be recovered after you leave this screen.</p><code>{token}</code></div><div className="identity-token-actions"><button className="secondary-button" type="button" onClick={() => void navigator.clipboard?.writeText(token)}>Copy</button><button className="text-button" type="button" onClick={() => setToken("")}>Hide</button></div></div>}

    <div className="identity-access-tabs" role="tablist" aria-label="Identity and access areas">
      <AreaTab area="positions" active={area} onSelect={setArea}>Organization</AreaTab>
      <AreaTab area="reporting" active={area} onSelect={setArea}>Reporting lines</AreaTab>
      <AreaTab area="directory" active={area} onSelect={setArea}>Directory groups & access</AreaTab>
      <AreaTab area="escalation" active={area} onSelect={setArea}>Escalation routes</AreaTab>
    </div>

    <div className="identity-access-area" role="tabpanel" aria-label={areaLabel(area)}>
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
        canConfigure={overview.can_configure_organization}
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
        people={overview.people}
        groups={overview.groups}
        canConfigure={overview.can_configure}
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
  return ({ positions: "Organization", reporting: "Reporting lines", directory: "Directory groups and access", escalation: "Escalation routes" } as const)[area];
}
