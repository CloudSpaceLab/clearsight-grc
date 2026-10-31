import { useEffect, useMemo, useState, type FormEvent } from "react";
import type {
  DetailTransferMode,
  IdentityPerson,
  IdentityRole,
  LegalEntityDataBoundary,
  LegalEntityDataBoundaryRevision,
  OrganizationPosition,
  OrganizationPositionRevision,
  OrganizationPositionRoleRevision,
  OrganizationPositionRouteSimulation,
  OrganizationScope,
  OrganizationScopeRevision,
  ProposeLegalEntityDataBoundaryInput,
  ProposeOrganizationPositionInput,
  ProposeOrganizationPositionRoleInput,
  ProposeOrganizationScopeInput,
} from "../../identityAccessApi";
import { OrganizationPositionManager } from "./OrganizationPositionManager";
import { OrganizationScopeManager } from "./OrganizationScopeManager";
import { enterpriseSetupRoute } from "../configure/enterpriseSetupRouting";
import { peopleAccessHash } from "../configure/peopleAccessRouting";
import { ActionLink, Button, FocusedSheet, Notice, SearchField, SelectField, StatusBadge, Surface } from "../ui";

type Props = {
  positions: OrganizationPosition[];
  people: IdentityPerson[];
  scopes: OrganizationScope[];
  revisions?: OrganizationScopeRevision[];
  positionRevisions?: OrganizationPositionRevision[];
  positionHistory?: OrganizationPositionRevision[];
  positionRoleRevisions?: OrganizationPositionRoleRevision[];
  roles?: IdentityRole[];
  dataBoundary?: LegalEntityDataBoundary;
  dataBoundaryRevisions?: LegalEntityDataBoundaryRevision[];
  actorPrincipalID?: string;
  canConfigure?: boolean;
  isBusy?: boolean;
  scopesTruncated?: boolean;
  mode: "positions" | "reporting";
  legalEntityName?: string;
  legalEntityCode?: string;
  selectedScopeID?: string;
  onSelectScope?: (id: string | undefined) => void;
  importsEnabled?: boolean;
  onProposeScope?: (input: ProposeOrganizationScopeInput) => Promise<boolean>;
  onApproveScope?: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
  onRejectScope?: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
  onProposePosition?: (input: ProposeOrganizationPositionInput) => Promise<boolean>;
  onApprovePosition?: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onRejectPosition?: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onSimulatePosition?: (revision: OrganizationPositionRevision) => Promise<OrganizationPositionRouteSimulation | null>;
  onRestorePosition?: (revision: OrganizationPositionRevision) => Promise<boolean>;
  onProposePositionRole?: (input: ProposeOrganizationPositionRoleInput) => Promise<boolean>;
  onApprovePositionRole?: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
  onRejectPositionRole?: (revision: OrganizationPositionRoleRevision, rationale: string) => Promise<boolean>;
  onProposeDataBoundary?: (input: ProposeLegalEntityDataBoundaryInput) => Promise<boolean>;
  onApproveDataBoundary?: (revision: LegalEntityDataBoundaryRevision, rationale: string) => Promise<boolean>;
  onRejectDataBoundary?: (revision: LegalEntityDataBoundaryRevision, rationale: string) => Promise<boolean>;
};

export function OrganizationInventory({
  positions,
  people,
  scopes,
  revisions = [],
  positionRevisions = [],
  positionHistory = [],
  positionRoleRevisions = [],
  roles = [],
  dataBoundary,
  dataBoundaryRevisions = [],
  actorPrincipalID = "",
  canConfigure = false,
  isBusy = false,
  scopesTruncated = false,
  mode,
  legalEntityName,
  legalEntityCode,
  selectedScopeID,
  onSelectScope,
  importsEnabled = false,
  onProposeScope,
  onApproveScope,
  onRejectScope,
  onProposePosition,
  onApprovePosition,
  onRejectPosition,
  onSimulatePosition,
  onRestorePosition,
  onProposePositionRole,
  onApprovePositionRole,
  onRejectPositionRole,
  onProposeDataBoundary,
  onApproveDataBoundary,
  onRejectDataBoundary,
}: Props) {
  const [query, setQuery] = useState("");
  const [panel, setPanel] = useState<"positions" | "boundary">();
  const [localSelection, setLocalSelection] = useState<string>();
  const selected = onSelectScope ? selectedScopeID : localSelection;
  const directoryHref = peopleAccessHash("organization", enterpriseSetupRoute(window.location.hash)?.selections, "directory");
  useEffect(() => { setPanel(undefined); }, [selected]);
  useEffect(() => { if (!canConfigure) setPanel(undefined); }, [canConfigure]);
  function selectScope(id: string | undefined) { setLocalSelection(id); onSelectScope?.(id); }
  function openPositions() { setArea(selected); setPanel("positions"); }
  const [area, setArea] = useState<string>();
  const [residencyRegion, setResidencyRegion] = useState("");
  const [transferMode, setTransferMode] = useState<DetailTransferMode>("AGGREGATE_ONLY");
  const [destinationRegions, setDestinationRegions] = useState("");
  const [boundaryRationale, setBoundaryRationale] = useState("");
  const positionByID = useMemo(() => new Map(positions.map((position) => [position.id, position])), [positions]);
  const scopeByID = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope])), [scopes]);
  const areaOptions = useMemo(() => scopes.map((scope) => ({
    id: scope.id,
    label: scope.department_path.join(" / "),
    description: scope.kind === "ORGANIZATION_UNIT" ? "Imported organization area" : humanize(scope.kind),
  })), [scopes]);
  const normalizedQuery = query.trim().toLowerCase();
  const pendingBoundary = dataBoundaryRevisions.find((revision) => revision.status === "PENDING");
  const pendingBoundaryFromAnotherMaker = Boolean(pendingBoundary && pendingBoundary.maker_id !== actorPrincipalID);
  const transferModeOptions: ReadonlyArray<{ id: DetailTransferMode; label: string; description: string }> = [
    { id: "AGGREGATE_ONLY", label: "Aggregate only", description: "No cross-entity record detail." },
    { id: "ALLOWLIST", label: "Destination allowlist", description: "Detail transfer only to approved regions." },
  ];

  useEffect(() => {
    if (pendingBoundary) return;
    setResidencyRegion(dataBoundary?.residency_region ?? "");
    setTransferMode(dataBoundary?.detail_transfer_mode ?? "AGGREGATE_ONLY");
    setDestinationRegions((dataBoundary?.allowed_destination_regions ?? []).join(", "));
    setBoundaryRationale("");
  }, [dataBoundary, pendingBoundary]);

  async function submitDataBoundary(event: FormEvent) {
    event.preventDefault();
    if (!onProposeDataBoundary) return;
    const destinations = transferMode === "ALLOWLIST"
      ? destinationRegions.split(",").map((value) => value.trim()).filter(Boolean)
      : [];
    await onProposeDataBoundary({
      residency_region: residencyRegion,
      detail_transfer_mode: transferMode,
      allowed_destination_regions: destinations,
      expected_version: dataBoundary?.version ?? 0,
    });
  }

  const visible = useMemo(() => positions.filter((position) => {
    const selectedScope = area ? scopeByID.get(area) : undefined;
    if (area && !selectedScope) return false;
    const positionScope = position.organization_scope_id ? scopeByID.get(position.organization_scope_id) : undefined;
    const matchesArea = !selectedScope || Boolean(positionScope && isDescendantPath(positionScope.department_path, selectedScope.department_path));
    if (!matchesArea) return false;
    if (!normalizedQuery) return true;
    return [
      position.code,
      position.title,
      position.function_name,
      position.occupant_name,
      position.parent_position_code,
      position.parent_position_title,
      position.department_path.join(" "),
      position.role_codes.join(" "),
    ].some((value) => value?.toLowerCase().includes(normalizedQuery));
  }), [area, normalizedQuery, positions, scopeByID]);

  const occupied = positions.filter((position) => position.occupant_principal_id).length;
  const vacancies = positions.length - occupied;

  const boundaryContent = dataBoundary ? <article className="config-card identity-data-boundary-card">
      <div className="section-header identity-card-header">
        <div>
          <h2>Data boundary</h2>
          <p>{dataBoundary.legal_entity_name || "Current legal entity"}</p>
        </div>
        <StatusBadge tone={dataBoundary.configured ? "success" : "neutral"}>
          {dataBoundary.configured ? "Configured" : "Aggregate only"}
        </StatusBadge>
      </div>

      <div className="identity-guard-summary configured">
        <strong>{dataBoundary.residency_region || "Residency region not set"}</strong>
        <span>{dataBoundary.detail_transfer_mode === "ALLOWLIST"
          ? `Detail destinations: ${dataBoundary.allowed_destination_regions.join(", ") || "None"}`
          : "Group totals only · Cross-entity records restricted."}</span>
      </div>

      {pendingBoundary && <div className="identity-guard-summary open">
        <strong>Proposed · {pendingBoundary.proposed_residency_region}</strong>
        <span>{pendingBoundary.proposed_detail_transfer_mode === "ALLOWLIST"
          ? `Detail destinations: ${pendingBoundary.proposed_destination_regions.join(", ")}`
          : "Aggregate only"}</span>
      </div>}

      {canConfigure && !pendingBoundary && onProposeDataBoundary && <form className="identity-inline-form" onSubmit={(event) => void submitDataBoundary(event)}>
        <h4>Change data boundary</h4>
        <label>Residency region<input required maxLength={32} value={residencyRegion} onChange={(event) => setResidencyRegion(event.target.value)} placeholder="NG or EU-WEST"/></label>
        <SelectField
          label="Group detail transfer"
          value={transferMode}
          placeholder="Choose transfer policy"
          options={transferModeOptions}
          allowsEmpty={false}
          onChange={(value) => value && setTransferMode(value)}
        />
        {transferMode === "ALLOWLIST" && <label>Destination regions<input required value={destinationRegions} onChange={(event) => setDestinationRegions(event.target.value)} placeholder="GH, EU-WEST"/></label>}
        <button className="secondary-button" disabled={isBusy || !residencyRegion.trim() || (transferMode === "ALLOWLIST" && !destinationRegions.trim())} type="submit">Propose change</button>
      </form>}

      {pendingBoundary && pendingBoundary.maker_id === actorPrincipalID && <div className="identity-guard-summary configured">
        <strong>Awaiting independent approval</strong>
        <span>Current boundary remains active.</span>
      </div>}

      {pendingBoundary && pendingBoundaryFromAnotherMaker && canConfigure && onApproveDataBoundary && onRejectDataBoundary && <form className="identity-inline-form" onSubmit={(event) => event.preventDefault()}>
        <h4>Review proposed boundary</h4>
        <label>Decision rationale<input required value={boundaryRationale} onChange={(event) => setBoundaryRationale(event.target.value)} placeholder="Residency and destination regions reviewed"/></label>
        <div className="identity-guard-actions">
          <button className="secondary-button" disabled={isBusy || !boundaryRationale.trim()} type="button" onClick={() => void onApproveDataBoundary(pendingBoundary, boundaryRationale)}>Approve</button>
          <button className="text-button" disabled={isBusy || !boundaryRationale.trim()} type="button" onClick={() => void onRejectDataBoundary(pendingBoundary, boundaryRationale)}>Reject</button>
        </div>
      </form>}
    </article> : <Notice>Data boundary settings were not returned for this legal entity.</Notice>;

  const positionContent = <article className="config-card identity-organization-card">
      <div className="section-header identity-card-header">
        <div>
          <h2>{mode === "positions" ? "Positions & roles" : "Reporting lines"}</h2>
          <p>{mode === "positions" ? "Positions, occupants and roles." : "Active reporting relationships."}</p>
        </div>
        <div className="identity-position-filters">
          <SearchField label="Search positions" value={query} onChange={setQuery} placeholder="Position, person, role or department"/>
          <SelectField
            label="Department / area"
            value={area}
            placeholder="All areas"
            options={areaOptions}
            onChange={setArea}
          />
        </div>
      </div>

      <p className="organization-setup-note">{visible.length} of {positions.length} loaded positions · Up to 50 positions are returned.</p>
      {area && !scopeByID.has(area) && <Notice tone="warning">The selected area is not in the loaded list. Choose an available area or clear the area filter.</Notice>}
      {scopesTruncated && <Notice tone="warning">First 500 areas loaded. Area filters use this list; additional areas are not included.</Notice>}
      {mode === "positions" && onProposePosition && onApprovePosition && onRejectPosition
        ? <OrganizationPositionManager
          positions={visible}
          allPositions={positions}
          people={people}
          scopes={scopes}
          revisions={positionRevisions}
          history={positionHistory}
          roleRevisions={positionRoleRevisions}
          roles={roles}
          actorPrincipalID={actorPrincipalID}
          canConfigure={canConfigure}
          isBusy={isBusy}
          scopeByID={scopeByID}
          onPropose={onProposePosition}
          onApprove={onApprovePosition}
          onReject={onRejectPosition}
          onSimulate={onSimulatePosition}
          onRestore={onRestorePosition}
          onProposeRole={onProposePositionRole}
          onApproveRole={onApprovePositionRole}
          onRejectRole={onRejectPositionRole}
        />
        : mode === "reporting"
          ? <ReportingList positions={visible} positionByID={positionByID}/>
          : null} 

      {!visible.length && <div className="identity-empty-state">
        <strong>{positions.length ? "No positions match these filters" : "No active positions were recorded"}</strong>
        <span>{positions.length ? "Clear the search or choose another organization area." : "Organization positions must be recorded before reporting lines are available."}</span>
      </div>}
    </article>;

  if (mode === "reporting") return <div className="identity-organization-view">
    <div className="identity-organization-summary" aria-label="Loaded organization position summary">
      <div><strong>{scopes.length}</strong><span>Loaded organization areas</span></div>
      <div><strong>{positions.length}</strong><span>Loaded positions</span></div>
      <div><strong>{occupied}</strong><span>Occupied positions in this list</span></div>
      <div><strong>{vacancies}</strong><span>Vacant positions in this list</span></div>
    </div>
    {positionContent}
    <div className="identity-authority-note" role="note"><strong>Reporting lines do not grant approval authority.</strong><span>Authority comes from active policy.</span></div>
  </div>;

  return <div className="identity-organization-view">
    <OrganizationScopeManager
      scopes={scopes} revisions={revisions} actorPrincipalID={actorPrincipalID}
      canConfigure={canConfigure && Boolean(onProposeScope && onApproveScope && onRejectScope)} isBusy={isBusy}
      scopesTruncated={scopesTruncated} legalEntityName={legalEntityName || dataBoundary?.legal_entity_name}
      legalEntityCode={legalEntityCode || dataBoundary?.legal_entity_code}
      selectedScopeID={selected} onSelectScope={selectScope}
      onPropose={onProposeScope || (async () => false)} onApprove={onApproveScope || (async () => false)} onReject={onRejectScope || (async () => false)}
    >
      <div className="organization-setup-lower">
        <Surface><section className="organization-setup-summary"><h2>Positions &amp; roles</h2><p>{positions.length} loaded positions · {occupied} occupied · {vacancies} vacant</p><p>Review occupants, reporting lines and workspace roles.</p><Button onPress={openPositions}>Positions &amp; roles</Button></section></Surface>
        <Surface><section className="organization-setup-summary"><h2>Data &amp; identity</h2><p>{dataBoundary?.residency_region ? `Residency: ${dataBoundary.residency_region}` : "Residency region not set"}</p><div className="organization-setup-next-actions"><Button onPress={() => setPanel("boundary")}>Data boundary</Button><ActionLink href={directoryHref}>Provisioning</ActionLink>{importsEnabled && <ActionLink href="#imports">Review source files</ActionLink>}</div></section></Surface>
      </div>
    </OrganizationScopeManager>
    {panel === "positions" && <FocusedSheet label="Positions & roles" size="wide" isDismissable={!isBusy} onClose={() => setPanel(undefined)}>{positionContent}</FocusedSheet>}
    {panel === "boundary" && <FocusedSheet label="Data boundary" isDismissable={!isBusy} onClose={() => setPanel(undefined)}>{boundaryContent}</FocusedSheet>}
  </div>;
}

function ReportingList({ positions, positionByID }: { positions: OrganizationPosition[]; positionByID: Map<string, OrganizationPosition> }) {
  if (!positions.length) return null;
  return <ol className="identity-reporting-list">
    {positions.map((position) => {
      const parent = position.parent_position_id ? positionByID.get(position.parent_position_id) : undefined;
      const subject = position.occupant_name || `Vacant ${position.title}`;
      const manager = parent?.occupant_name || position.parent_position_title || (position.parent_position_id ? "a position outside this view" : "no parent position");
      return <li key={position.id}>
        <div className="identity-reporting-marker" aria-hidden="true"><span>{initials(subject)}</span></div>
        <div>
          <p><strong>{subject}</strong> {position.parent_position_id ? `reports to ${manager}` : "is recorded as a top-level position"}</p>
          <span>{position.title} · {departmentLabel(position)}</span>
        </div>
        <StatusBadge tone={position.occupant_name && (!position.parent_position_id || parent) ? "success" : "warning"}>
          {!position.occupant_name ? "Vacant" : position.parent_position_id && !parent ? "Parent outside view" : "Active"}
        </StatusBadge>
      </li>;
    })}
  </ol>;
}

function isDescendantPath(candidate: string[], ancestor: string[]) {
  return ancestor.length <= candidate.length && ancestor.every((part, index) => part.toUpperCase() === candidate[index]?.toUpperCase());
}

function scopeLabel(position: OrganizationPosition, scopeByID: Map<string, OrganizationScope>) {
  const scope = position.organization_scope_id ? scopeByID.get(position.organization_scope_id) : undefined;
  return scope ? scope.department_path.join(" / ") : departmentLabel(position);
}

function departmentLabel(position: OrganizationPosition) {
  return position.department_path.length ? position.department_path.join(" / ") : position.function_name || "Legal entity wide";
}

function humanize(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function initials(value: string) {
  return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join("");
}
