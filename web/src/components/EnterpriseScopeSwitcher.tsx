import { useMemo, useState } from "react";
import type { ScopeHierarchy } from "../api";
import { Button, PopoverDialog, SearchField, SelectableRecord } from "./ui";

type Props = {
  hierarchy: ScopeHierarchy;
  currentScopeID: string;
  canSwitchLegalEntity?: boolean;
  isChanging?: boolean;
  onSelectionChange: (legalEntityID: string) => void;
  onManageOrganization?: () => void;
};

const searchThreshold = 7;

export function EnterpriseScopeSwitcher({
  hierarchy,
  currentScopeID,
  canSwitchLegalEntity = true,
  isChanging = false,
  onSelectionChange,
  onManageOrganization,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const current = hierarchy.legal_entities.find((entity) => entity.id === currentScopeID) ?? hierarchy.current;
  const areas = useMemo(() => hierarchy.organization_scopes ?? [], [hierarchy.organization_scopes]);
  const normalizedQuery = query.trim().toLowerCase();
  const visibleEntities = useMemo(() => {
    if (!normalizedQuery) return hierarchy.legal_entities;
    return hierarchy.legal_entities.filter((entity) =>
      [entity.name, entity.code, entity.jurisdiction]
        .filter(Boolean)
        .some((value) => value!.toLowerCase().includes(normalizedQuery)));
  }, [hierarchy.legal_entities, normalizedQuery]);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) setQuery("");
  }

  function chooseScope(id: string) {
    if (!canSwitchLegalEntity || id === currentScopeID || isChanging) return;
    setOpen(false);
    setQuery("");
    onSelectionChange(id);
  }

  function manageOrganization() {
    setOpen(false);
    onManageOrganization?.();
  }

  return <PopoverDialog
    label="Change organization scope"
    isOpen={open}
    onOpenChange={handleOpenChange}
    placement="bottom end"
    triggerLabel={`Organization scope, ${current.name}`}
    triggerClassName="enterprise-scope-trigger"
    triggerDisabled={isChanging}
    triggerChildren={<>
      <span className="enterprise-scope-trigger__label">Scope</span>
      <span className="enterprise-scope-trigger__name">{current.name}</span>
      <span className="enterprise-scope-trigger__chevron" aria-hidden="true">⌄</span>
    </>}
  >
    <div className="enterprise-scope-switcher">
      <div className="enterprise-scope-root" aria-label={`Organization ${hierarchy.root.name}`}>
        <span className="enterprise-scope-root__marker" aria-hidden="true"/>
        <span className="enterprise-scope-root__content">
          <strong>{hierarchy.root.name}</strong>
          <small>Organization</small>
        </span>
      </div>

      <section className="enterprise-scope-section" aria-labelledby="enterprise-legal-entities-heading">
        <div className="enterprise-scope-section__heading">
          <strong id="enterprise-legal-entities-heading">Legal entities</strong>
          {!canSwitchLegalEntity && hierarchy.legal_entities.length > 1 && <small>Switching is unavailable in this session.</small>}
        </div>

        {hierarchy.legal_entities.length >= searchThreshold && <SearchField
          label="Search legal entities"
          value={query}
          onChange={setQuery}
          placeholder="Search legal entities"
          isDisabled={isChanging}
        />}

        <div className="enterprise-scope-tree">
          <span className="enterprise-scope-tree__line" aria-hidden="true"/>
          <ul className="enterprise-scope-list" aria-label={`Legal entities in ${hierarchy.root.name}`}>
            {visibleEntities.map((entity) => {
              const selected = entity.id === currentScopeID;
              const metadata = entity.jurisdiction || "Legal entity";
              return <li className="enterprise-scope-option-row" data-current={selected || undefined} key={entity.id}>
                <span className="enterprise-scope-option__branch" aria-hidden="true"/>
                <SelectableRecord
                  title={entity.name}
                  metadata={selected ? `${metadata} · Current` : metadata}
                  isSelected={selected}
                  isDisabled={isChanging || (!selected && !canSwitchLegalEntity)}
                  onPress={() => chooseScope(entity.id)}
                />
              </li>;
            })}
          </ul>
        </div>

        {!visibleEntities.length && <p className="enterprise-scope-empty">No legal entities match this search.</p>}
      </section>

      <section className="enterprise-scope-section enterprise-scope-areas" aria-labelledby="enterprise-areas-heading">
        <div className="enterprise-scope-section__heading">
          <strong id="enterprise-areas-heading">Your organization areas</strong>
          <small>{areas.length ? "Server-authorized organization scopes for this legal entity." : "No subordinate organization scope is available."}</small>
        </div>
        {areas.length > 0 && <ul className="enterprise-scope-area-list">
          {areas.map((area) => <li key={area.id}><span>{area.department_path?.join(" / ") || area.name}</span><small>{humanizeScopeKind(area.kind)}</small></li>)}
        </ul>}
        <p className="enterprise-scope-area-note">These scopes now have stable IDs. Home filtering remains unavailable until business records are explicitly attributed to them.</p>
      </section>

      {onManageOrganization && <div className="enterprise-scope-management">
        <Button size="compact" variant="secondary" onPress={manageOrganization}>Organization & access</Button>
      </div>}
    </div>
  </PopoverDialog>;
}

function humanizeScopeKind(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
