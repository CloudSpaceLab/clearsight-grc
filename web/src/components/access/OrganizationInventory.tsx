import { useMemo, useState } from "react";
import type { OrganizationPosition, OrganizationScope } from "../../identityAccessApi";
import { SelectField, StatusBadge } from "../ui";

type Props = {
  positions: OrganizationPosition[];
  scopes: OrganizationScope[];
  scopesTruncated?: boolean;
  mode: "positions" | "reporting";
};

export function OrganizationInventory({ positions, scopes, scopesTruncated = false, mode }: Props) {
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
    <div className="identity-organization-summary" aria-label="Active organization position summary">
      <div><strong>{scopes.length}</strong><span>Active organization scopes in this legal entity</span></div>
      <div><strong>{positions.length}</strong><span>Active positions in this legal entity</span></div>
      <div><strong>{occupied}</strong><span>Positions with an active occupant</span></div>
      <div><strong>{vacancies}</strong><span>Vacant positions requiring coverage</span></div>
    </div>

    <article className="config-card identity-organization-card">
      <div className="section-header identity-card-header">
        <div>
          <h3>{mode === "positions" ? "Organization, positions & roles" : "Reporting lines"}</h3>
          <p>{mode === "positions"
            ? "Browse stable organization scopes, positions, occupants and workspace roles for this legal entity."
            : "Active reporting relationships used to determine who may hand off assigned work."}</p>
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
      {mode === "positions" ? <PositionTable positions={visible} scopeByID={scopeByID}/> : <ReportingList positions={visible} positionByID={positionByID}/>} 

      {!visible.length && <div className="identity-empty-state">
        <strong>{positions.length ? "No positions match these filters" : "No active positions were recorded"}</strong>
        <span>{positions.length ? "Clear the search or choose another organization area." : "Organization positions must be recorded before reporting lines are available."}</span>
      </div>}
    </article>

    <div className="identity-authority-note" role="note">
      <strong>{mode === "positions" ? "Organization scopes have stable server IDs." : "Reporting lines permit responsibility handoff only."}</strong>
      <span>{mode === "positions"
        ? "Legacy department paths are reconciled to these scopes. Dashboard filtering remains disabled until business records carry an authoritative scope reference."
        : "A manager does not gain approval, review or signing authority unless the active authority policy grants it."}</span>
    </div>
  </div>;
}

function PositionTable({ positions, scopeByID }: { positions: OrganizationPosition[]; scopeByID: Map<string, OrganizationScope> }) {
  if (!positions.length) return null;
  return <div className="identity-position-table-wrap">
    <table className="identity-position-table">
      <thead><tr><th>Position</th><th>Current occupant</th><th>Workspace roles</th><th>Reports to</th></tr></thead>
      <tbody>{positions.map((position) => <tr key={position.id}>
        <td data-label="Position"><strong>{position.title}</strong><span>{position.code} · {scopeLabel(position, scopeByID)}</span></td>
        <td data-label="Current occupant">{position.occupant_name
          ? <><strong>{position.occupant_name}</strong><span>{humanize(position.occupant_status || "active")}</span></>
          : <span className="identity-vacancy">Vacant — coverage required</span>}</td>
        <td data-label="Workspace roles"><div className="identity-role-chips">{position.role_codes.length
          ? position.role_codes.map((role) => <span key={role}>{role}</span>)
          : <span className="identity-empty-value">No role assigned</span>}</div></td>
        <td data-label="Reports to"><strong>{position.parent_position_title || "Top-level position"}</strong>{position.parent_position_code && <span>{position.parent_position_code}</span>}</td>
      </tr>)}</tbody>
    </table>
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
