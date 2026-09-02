import { useMemo, useState } from "react";
import type { OrganizationPosition } from "../../identityAccessApi";

type Props = {
  positions: OrganizationPosition[];
  mode: "positions" | "reporting";
};

export function OrganizationInventory({ positions, mode }: Props) {
  const [query, setQuery] = useState("");
  const positionByID = useMemo(() => new Map(positions.map((position) => [position.id, position])), [positions]);
  const normalizedQuery = query.trim().toLowerCase();
  const visible = useMemo(() => {
    if (!normalizedQuery) return positions;
    return positions.filter((position) => [
      position.code,
      position.title,
      position.function_name,
      position.occupant_name,
      position.parent_position_code,
      position.parent_position_title,
      position.department_path.join(" "),
      position.role_codes.join(" "),
    ].some((value) => value?.toLowerCase().includes(normalizedQuery)));
  }, [normalizedQuery, positions]);

  const occupied = positions.filter((position) => position.occupant_principal_id).length;
  const vacancies = positions.length - occupied;
  const unresolvedParents = positions.filter((position) => position.parent_position_id && !positionByID.has(position.parent_position_id)).length;

  return <div className="identity-organization-view">
    <div className="identity-organization-summary" aria-label="Active organization position summary">
      <div><strong>{positions.length}</strong><span>Active positions in this legal entity</span></div>
      <div><strong>{occupied}</strong><span>Positions with an active occupant</span></div>
      <div><strong>{vacancies}</strong><span>Vacant positions requiring coverage</span></div>
      <div><strong>{unresolvedParents}</strong><span>Parent positions outside this bounded view</span></div>
    </div>

    <article className="config-card identity-organization-card">
      <div className="section-header identity-card-header">
        <div>
          <h3>{mode === "positions" ? "Positions and assigned roles" : "Reporting lines"}</h3>
          <p>{mode === "positions"
            ? "Active positions, occupants and workspace roles recorded for this legal entity."
            : "Active reporting relationships used to determine who may hand off assigned work."}</p>
        </div>
        <label className="identity-position-search">
          <span>Search positions</span>
          <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Name, position, role or department"/>
        </label>
      </div>

      {mode === "positions" ? <PositionTable positions={visible}/> : <ReportingList positions={visible} positionByID={positionByID}/>} 

      {!visible.length && <div className="identity-empty-state">
        <strong>{positions.length ? "No positions match this search" : "No active positions were recorded"}</strong>
        <span>{positions.length ? "Clear the search or use a position, person, role or department name." : "Add effective-dated organization positions before assigning reporting lines or roles."}</span>
      </div>}
    </article>

    <div className="identity-authority-note" role="note">
      <strong>{mode === "positions" ? "Position roles do not replace approval routes." : "Reporting lines permit responsibility handoff only."}</strong>
      <span>{mode === "positions"
        ? "Approval, review and signing eligibility continue to use the active authority policy for each record."
        : "A manager does not gain approval, review or signing authority unless the active authority policy grants it."}</span>
    </div>
  </div>;
}

function PositionTable({ positions }: { positions: OrganizationPosition[] }) {
  if (!positions.length) return null;
  return <div className="identity-position-table-wrap">
    <table className="identity-position-table">
      <thead><tr><th>Position</th><th>Current occupant</th><th>Workspace roles</th><th>Reports to</th></tr></thead>
      <tbody>{positions.map((position) => <tr key={position.id}>
        <td data-label="Position"><strong>{position.title}</strong><span>{position.code} · {departmentLabel(position)}</span></td>
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
        <span className={`identity-line-state ${position.occupant_name && (!position.parent_position_id || parent) ? "complete" : "attention"}`}>
          {!position.occupant_name ? "Vacant" : position.parent_position_id && !parent ? "Parent outside view" : "Active"}
        </span>
      </li>;
    })}
  </ol>;
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
