import { useId, useRef, useState } from "react";
import type { GroupRoleBinding, IdentityRole } from "../../identityAccessApi";
import { Button, Notice, SelectableRecord, Surface } from "../ui";
import { humanAccessLabel } from "./peopleAccessPresentation";

type Props = {
  roles: IdentityRole[];
  bindings: GroupRoleBinding[];
  legalEntityName?: string;
  selectedRoleID?: string;
  onSelectRole?: (id: string | undefined) => void;
  onOpenOrganization?: () => void;
  onOpenMappings?: () => void;
};

export function PeopleRoleCatalog({ roles, bindings, legalEntityName, selectedRoleID, onSelectRole, onOpenOrganization, onOpenMappings }: Props) {
  const [localSelectedID, setLocalSelectedID] = useState<string>();
  const headingID = useId();
  const detailRef = useRef<HTMLElement>(null);
  const selectedID = selectedRoleID ?? (roles.some((role) => role.id === localSelectedID) ? localSelectedID : roles[0]?.id);
  const selected = roles.find((role) => role.id === selectedID);
  const mappings = selected ? bindings.filter((binding) => binding.role_template_id === selected.id) : [];

  function selectRole(id: string) {
    setLocalSelectedID(id);
    onSelectRole?.(id);
    if (window.matchMedia("(max-width: 1024px)").matches) requestAnimationFrame(() => {
      detailRef.current?.focus({ preventScroll: true });
      detailRef.current?.scrollIntoView({ block: "start", behavior: "instant" });
    });
  }

  return <div className="people-access-layout people-role-catalog">
    <section className="people-access-list-panel" aria-label="Role catalog"><Surface>
      <header className="people-access-panel-heading"><div><h2>Role catalog <span className="people-access-count">{roles.length}</span></h2><p>Up to 100 role definitions. Approval authority is checked separately.</p></div></header>
      <div className="people-role-catalog__records">{roles.map((role) => <SelectableRecord key={role.id} variant="list" title={role.name} metadata={`${role.capabilities.length} capabilities · Version ${role.version}`} isSelected={selectedID === role.id} aria-label={`Inspect role ${role.name}`} onPress={() => selectRole(role.id)}
        startContent={<span className="people-access-avatar people-access-avatar--role" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m12 3 8 3v6c0 4-4 7-8 9-4-2-8-5-8-9V6zM8 12l3 3 5-6"/></svg></span>}
        endContent={<span aria-hidden="true">›</span>}/>)}</div>
      {!roles.length && <div className="people-access-empty"><h3>No role definitions returned</h3><p>No role definitions were returned for this organization.</p></div>}
      <footer className="people-access-list-footer"><span>{roles.length} role definitions loaded</span></footer>
    </Surface></section>
    <aside ref={detailRef} tabIndex={-1} className="people-access-detail people-role-catalog__detail" aria-labelledby={headingID}><Surface>
      {selected ? <>
        <header className="people-access-detail__heading"><span className="people-access-kicker">Role definition</span><h2 id={headingID}>{selected.name}</h2><p>Version {selected.version}</p></header>
        <div className="people-access-detail__body">
          <section><h3>Capabilities</h3>{selected.capabilities.length ? <ul className="people-access-capabilities">{selected.capabilities.map((capability) => <li key={capability}>{humanAccessLabel(capability)}</li>)}</ul> : <p className="people-access-context">No workspace capabilities recorded.</p>}</section>
          <section><h3>Directory group mappings</h3><p className="people-access-context">{legalEntityName || "This legal entity"} · Mappings in the loaded list.</p>{mappings.length ? <ul className="people-access-assignments">{mappings.map((binding) => <li key={binding.id}><strong>{binding.group_name}</strong><span>{binding.department_path.join(" / ") || "Legal entity wide"}</span></li>)}</ul> : <p className="people-access-context">No mapping for this role was returned.</p>}{onOpenMappings && <Button variant="quiet" onPress={onOpenMappings}>View group mappings</Button>}</section>
          <section><h3>Authorization boundary</h3><Notice>Roles grant capabilities within their assigned scope. Approval requires the current route, eligible identity and conflict checks.</Notice></section>
          {onOpenOrganization && <Button onPress={onOpenOrganization}>View position roles</Button>}
        </div>
      </> : <div className="people-access-empty"><h2 id={headingID}>{selectedRoleID ? "Role not in this list" : "Select a role"}</h2><p>{selectedRoleID ? "This role definition was not returned." : "Choose a role to view its capabilities and recorded group mappings."}</p>{selectedRoleID && <Button onPress={() => { setLocalSelectedID(undefined); onSelectRole?.(undefined); }}>Return to listed roles</Button>}</div>}
    </Surface></aside>
  </div>;
}
