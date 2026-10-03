import { useMemo, useState } from "react";
import type {
  IdentityPerson,
  OrganizationPosition,
  OrganizationPositionRevision,
  OrganizationScope,
  OrganizationScopeRevision,
  ProposeOrganizationPositionInput,
  ProposeOrganizationScopeInput,
} from "../../identityAccessApi";
import { OrganizationPositionManager } from "./OrganizationPositionManager";
import { OrganizationScopeManager } from "./OrganizationScopeManager";
import { SelectField, StatusBadge } from "../ui";

type Props = {
  positions: OrganizationPosition[];
  people: IdentityPerson[];
  scopes: OrganizationScope[];
  revisions?: OrganizationScopeRevision[];
  positionRevisions?: OrganizationPositionRevision[];
  actorPrincipalID?: string;
  canConfigure?: boolean;
  isBusy?: boolean;
  scopesTruncated?: boolean;
  mode: "positions" | "reporting";
  onProposeScope?: (input: ProposeOrganizationScopeInput) => Promise<boolean>;
  onApproveScope?: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
  onRejectScope?: (revision: OrganizationScopeRevision, rationale: string) => Promise<boolean>;
  onProposePosition?: (input: ProposeOrganizationPositionInput) => Promise<boolean>;
  onApprovePosition?: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
  onRejectPosition?: (revision: OrganizationPositionRevision, rationale: string) => Promise<boolean>;
};

export function OrganizationInventory({
  positions,
  people,
  scopes,
  revisions = [],
  positionRevisions = [],
  actorPrincipalID = "",
  canConfigure = false,
  isBusy = false,
  scopesTruncated = false,
  mode,
  onProposeScope,
  onApproveScope,
  onRejectScope,
  onProposePosition,
  onApprovePosition,
  onRejectPosition,
}: Props) {
  const [query, setQuery] = useState("");
  const [area, setArea] = useState<string>();
  const positionByID = useMemo(() => new Map(positions.map((position) => [position.id, position])), [positions]);
  const scopeByID = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope])), [scopes]);
  const areaOptions = useMemo(() => scopes.map((scope) => ({
    id: scope.id,
    label: scope.department_path.join(" / "),
    description: scope.kind === "ORGANIZATION_UNIT" ? "Imported organization area" : humanize(scope.kind),
  })), [scopes]);
  const normalizedQuery = query.trim().toLowerCase();
  const visible = useMemo(() => positions.filter((position) => {
    const selectedScope = area ? scopeByID.get(area) : undefined;
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

  return <div className="identity-organization-view">
    {mode === "positions" && onProposeScope && onApproveScope && onRejectScope && <OrganizationScopeManager
      scopes={scopes}
      revisions={revisions}
      actorPrincipalID={actorPrincipalID}
      canConfigure={canConfigure}
      isBusy={isBusy}
      onPropose={onProposeScope}
      onApprove={onApproveScope}
      onReject={onRejectScope}
    />}
    <div className="identity-organization-summary" aria-label="Active organization position summary">
      <div><strong>{scopes.length}</strong><span>Active organization scopes in this legal entity</span></div>
      <div><strong>{positions.length}</strong><span>Active positions in this legal entity</span></div>
      <div><strong>{occupied}</strong><span>Positions with an active occupant</span></div>
      <div><strong>{vacancies}</strong><span>Vacant positions requiring coverage</span></div>
    </div>

    <article className="config-card identity-organization-card">
      <div className="section-header identity-card-header">
        <div>
          <h3>{mode === "positions" ? "Positions & roles" : "Reporting lines"}</h3>
          <p>{mode === "positions" ? "Positions, occupants and roles." : "Active reporting relationships."}</p>
        </div>
        <div className="identity-position-filters">
          <label className="identity-position-search">
            <span>Search</span>
            <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Position, person, role or department"/>
          </label>
          <SelectField
            label="Department / area"
            value={area}
            placeholder="All areas"
            options={areaOptions}
            onChange={setArea}
          />
        </div>
      </div>

      {scopesTruncated && <div className="inline-notice" role="status">Only the first 500 organization scopes are shown. Narrow the hierarchy before editing or reviewing a larger structure.</div>}
      {mode === "positions" && onProposePosition && onApprovePosition && onRejectPosition
        ? <OrganizationPositionManager
          positions={visible}
          allPositions={positions}
          people={people}
          scopes={scopes}
          revisions={positionRevisions}
          actorPrincipalID={actorPrincipalID}
          canConfigure={canConfigure}
          isBusy={isBusy}
          scopeByID={scopeByID}
          onPropose={onProposePosition}
          onApprove={onApprovePosition}
          onReject={onRejectPosition}
        />
        : mode === "reporting"
          ? <ReportingList positions={visible} positionByID={positionByID}/>
          : null} 

      {!visible.length && <div className="identity-empty-state">
        <strong>{positions.length ? "No positions match these filters" : "No active positions were recorded"}</strong>
        <span>{positions.length ? "Clear the search or choose another organization area." : "Organization positions must be recorded before reporting lines are available."}</span>
      </div>}
    </article>

    <div className="identity-authority-note" role="note">
      <strong>{mode === "positions" ? "Area changes require approval." : "Reporting lines do not grant approval authority."}</strong>
      <span>{mode === "positions" ? "Home uses explicit Matter area scope." : "Authority comes from active policy."}</span>
    </div>
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
