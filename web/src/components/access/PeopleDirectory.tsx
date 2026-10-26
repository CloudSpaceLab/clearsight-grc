import { useId, useMemo, useRef, useState } from "react";
import type { IdentityPerson, IdentityRole, OrganizationPosition } from "../../identityAccessApi";
import { Button, Notice, SearchField, SelectField, SelectableRecord, StatusBadge, Surface } from "../ui";
import { humanAccessLabel, initials, personName, positionsFor } from "./peopleAccessPresentation";

type Props = {
  people: IdentityPerson[];
  positions: OrganizationPosition[];
  roles: IdentityRole[];
  organizationName?: string;
  legalEntityName?: string;
  selectedPersonID?: string;
  onSelectPerson?: (id: string | undefined) => void;
  onOpenOrganization?: () => void;
};

export function PeopleDirectory({ people, positions, roles, organizationName, legalEntityName, selectedPersonID, onSelectPerson, onOpenOrganization }: Props) {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<string>();
  const [source, setSource] = useState<string>();
  const [localSelectedID, setLocalSelectedID] = useState<string>();
  const detailRef = useRef<HTMLElement>(null);
  const detailHeadingID = useId();
  const entity = legalEntityName || "this legal entity";
  const statuses = useMemo(() => [...new Set(people.map((person) => person.status).filter(Boolean))].sort().map((id) => ({ id, label: humanAccessLabel(id) })), [people]);
  const sources = useMemo(() => [...new Set(people.map((person) => person.source_code || "__local"))].sort().map((id) => ({ id, label: id === "__local" ? "Local profile" : id })), [people]);
  const filtered = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    return people.filter((person) => (!status || person.status === status)
      && (!source || (person.source_code || "__local") === source)
      && (!query || [personName(person), person.user_name, person.source_code, ...positionsFor(person.id, positions).map((position) => `${position.title} ${position.department_path.join(" ")}`)].join(" ").toLocaleLowerCase().includes(query)));
  }, [people, positions, search, status, source]);
  const selectedID = selectedPersonID ?? (filtered.some((person) => person.id === localSelectedID) ? localSelectedID : filtered[0]?.id);
  const selected = filtered.find((person) => person.id === selectedID);

  function selectPerson(id: string) {
    setLocalSelectedID(id);
    onSelectPerson?.(id);
    if (window.matchMedia("(max-width: 1024px)").matches) {
      requestAnimationFrame(() => {
        detailRef.current?.focus({ preventScroll: true });
        detailRef.current?.scrollIntoView({ block: "start", behavior: "instant" });
      });
    }
  }

  function resetFilters() {
    setSearch("");
    setStatus(undefined);
    setSource(undefined);
  }

  function clearSelection() {
    setLocalSelectedID(undefined);
    onSelectPerson?.(undefined);
  }

  return <div className="people-access-layout people-directory">
    <section className="people-access-list-panel" aria-label="People directory">
      <Surface>
        <header className="people-access-panel-heading"><div><h2>People <span className="people-access-count">{filtered.length}</span></h2><p>Up to 50 profiles across {organizationName || "the organization"}.</p></div></header>
        <div className="people-directory__toolbar">
          <SearchField label="Search people" value={search} onChange={setSearch} placeholder="Search name, username or position"/>
          <SelectField label="Profile status" isLabelHidden value={status} placeholder="Any status" options={statuses} onChange={setStatus}/>
          <SelectField label="Provisioning source" isLabelHidden value={source} placeholder="All sources" options={sources} onChange={setSource}/>
        </div>
        <div className="people-directory__list-head" aria-hidden="true"><span>Person</span><span>Position & scope</span><span>Profile</span></div>
        <div className="people-directory__records">
          {filtered.map((person) => {
            const recordedPositions = positionsFor(person.id, positions);
            return <SelectableRecord key={person.id} variant="list" title={personName(person)} metadata={person.user_name || "Username not provided"}
              aria-label={`Inspect ${personName(person)}`} isSelected={selectedID === person.id} onPress={() => selectPerson(person.id)}
              startContent={<span className="people-access-avatar">{initials(personName(person))}</span>}
              endContent={<><span className="people-directory__position"><strong>{recordedPositions[0]?.title || "Position not returned"}</strong><small>{recordedPositions.length > 1 ? `${recordedPositions.length} positions in this list` : recordedPositions[0]?.department_path.join(" / ") || person.source_code || "Local profile"}</small></span><StatusBadge>{humanAccessLabel(person.status) || "Not reported"}</StatusBadge></>}/>
          })}
          {!filtered.length && <div className="people-access-empty"><h3>{people.length ? "No people match these filters" : "No people returned"}</h3><p>{people.length ? "Search checks only the profiles in this list." : "The directory returned no profiles for the organization."}</p>{people.length > 0 && <Button variant="quiet" onPress={resetFilters}>Clear filters</Button>}</div>}
        </div>
        <footer className="people-access-list-footer"><span>{filtered.length} of {people.length} loaded profiles</span><span>Search covers this list</span></footer>
      </Surface>
    </section>
    <aside ref={detailRef} tabIndex={-1} className="people-access-detail people-directory__detail" aria-labelledby={detailHeadingID}>
      <Surface>
        {selected ? <>
          <header className="people-access-detail__heading"><span className="people-access-kicker">Person record</span><div className="people-directory__identity"><span className="people-access-avatar" aria-hidden="true">{initials(personName(selected))}</span><div><h2 id={detailHeadingID}>{personName(selected)}</h2><p>{selected.user_name || "Username not provided"}</p></div></div><StatusBadge>Profile {humanAccessLabel(selected.status).toLowerCase() || "not reported"}</StatusBadge></header>
          <div className="people-access-detail__body">
            <section><h3>Profile details</h3><dl className="people-access-facts"><dt>Directory</dt><dd>{organizationName || "Organization directory"}</dd><dt>Username</dt><dd>{selected.user_name || "Not provided"}</dd><dt>Provisioning</dt><dd>{selected.source_code || "Local profile"}</dd>{selected.source_code && <><dt>Source status</dt><dd>{humanAccessLabel(selected.source_state || "") || "Not reported"}</dd></>}</dl></section>
            <section><h3>Recorded positions</h3><p className="people-access-context">{entity} · Up to 50 positions returned.</p>
              {positionsFor(selected.id, positions).length ? <ul className="people-access-assignments">{positionsFor(selected.id, positions).map((position) => <li key={position.id}><strong>{position.title}</strong><span>{position.department_path.join(" / ") || "Legal entity wide"}</span>{position.role_codes.length > 0 && <span>Position roles: {position.role_codes.map((code) => roles.find((role) => role.code === code)?.name || humanAccessLabel(code)).join(", ")}</span>}</li>)}</ul> : <p className="people-access-context">No position for this person was returned in this list.</p>}
              {onOpenOrganization && <Button variant="quiet" onPress={onOpenOrganization}>View organization</Button>}
            </section>
            <Notice>Profile status and position roles do not confirm access to a record or approval authority.</Notice>
          </div>
        </> : <div className="people-access-empty"><h2 id={detailHeadingID}>{selectedPersonID ? "Person not in this list" : "Select a person"}</h2><p>{selectedPersonID ? "This person was not returned or does not match the current filters." : "Choose a profile to view its source and recorded positions."}</p>{selectedPersonID && <Button onPress={clearSelection}>Return to listed people</Button>}</div>}
      </Surface>
    </aside>
  </div>;
}
