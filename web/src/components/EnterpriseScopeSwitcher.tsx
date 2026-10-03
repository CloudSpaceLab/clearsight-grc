import { useMemo, useState } from "react";
import type { ScopeHierarchy, ScopeNode } from "../api";
import { Button, PopoverDialog, SearchField, SelectableRecord } from "./ui";

type Props = {
  hierarchy: ScopeHierarchy;
  currentScopeID: string;
  canSwitchLegalEntity?: boolean;
  activeOrganizationScopeID?: string;
  isChanging?: boolean;
  onSelectionChange: (legalEntityID: string) => void;
  onOrganizationScopeChange?: (organizationScopeID?: string) => void;
  onManageOrganization?: () => void;
};

const searchThreshold = 7;

export function EnterpriseScopeSwitcher({
  hierarchy,
  currentScopeID,
  canSwitchLegalEntity = true,
  activeOrganizationScopeID,
  isChanging = false,
  onSelectionChange,
  onOrganizationScopeChange,
  onManageOrganization,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const current = hierarchy.legal_entities.find((entity) => entity.id === currentScopeID) ?? hierarchy.current;
  const areas = useMemo(() => hierarchy.organization_scopes ?? [], [hierarchy.organization_scopes]);
  const areaByID = useMemo(() => new Map(areas.map((area) => [area.id, area])), [areas]);
  const childrenByParent = useMemo(() => {
    const values = new Map<string, ScopeNode[]>();
    for (const area of areas) {
      const key = area.parent_id && areaByID.has(area.parent_id) ? area.parent_id : "";
      const children = values.get(key) ?? [];
      children.push(area);
      values.set(key, children);
    }
    return values;
  }, [areaByID, areas]);
  const activeArea = areas.find((area) => area.id === activeOrganizationScopeID && area.filterable);
  const triggerName = activeArea ? `${current.name} · ${activeArea.name}` : current.name;
  const normalizedQuery = query.trim().toLowerCase();
  const visibleEntities = useMemo(() => {
    if (!normalizedQuery) return hierarchy.legal_entities;
    return hierarchy.legal_entities.filter((entity) => scopeMatches(entity, normalizedQuery));
  }, [hierarchy.legal_entities, normalizedQuery]);
  const visibleAreaIDs = useMemo(() => {
    if (!normalizedQuery) return new Set(areas.map((area) => area.id));
    const visible = new Set<string>();
    for (const area of areas) {
      if (!scopeMatches(area, normalizedQuery)) continue;
      let currentArea: ScopeNode | undefined = area;
      while (currentArea && !visible.has(currentArea.id)) {
        visible.add(currentArea.id);
        currentArea = currentArea.parent_id ? areaByID.get(currentArea.parent_id) : undefined;
      }
    }
    return visible;
  }, [areaByID, areas, normalizedQuery]);
  const showSearch = hierarchy.legal_entities.length + areas.length >= searchThreshold;

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) setQuery("");
  }

  function chooseScope(id: string) {
    if (isChanging) return;
    if (id === currentScopeID) {
      if (activeOrganizationScopeID && onOrganizationScopeChange) {
        setOpen(false);
        setQuery("");
        onOrganizationScopeChange(undefined);
      }
      return;
    }
    if (!canSwitchLegalEntity) return;
    setOpen(false);
    setQuery("");
    onSelectionChange(id);
  }

  function chooseOrganizationScope(id: string) {
    const area = areas.find((item) => item.id === id);
    if (!area?.filterable || !onOrganizationScopeChange || isChanging) return;
    setOpen(false);
    setQuery("");
    onOrganizationScopeChange(id);
  }

  function manageOrganization() {
    setOpen(false);
    onManageOrganization?.();
  }

  function renderArea(area: ScopeNode) {
    if (!visibleAreaIDs.has(area.id)) return null;
    const children = (childrenByParent.get(area.id) ?? []).filter((child) => visibleAreaIDs.has(child.id));
    const selected = area.id === activeOrganizationScopeID;
    return <li className="enterprise-scope-area-node" key={area.id}>
      {area.filterable && onOrganizationScopeChange
        ? <SelectableRecord
          title={area.name}
          metadata={selected ? `${humanizeScopeKind(area.kind)} · Selected` : humanizeScopeKind(area.kind)}
          isSelected={selected}
          isDisabled={isChanging}
          onPress={() => chooseOrganizationScope(area.id)}
        />
        : <div className="enterprise-scope-area-static"><span>{area.name}</span><small>{humanizeScopeKind(area.kind)}</small></div>}
      {children.length > 0 && <ul>{children.map(renderArea)}</ul>}
    </li>;
  }

  const rootAreas = (childrenByParent.get("") ?? []).filter((area) => visibleAreaIDs.has(area.id));

  return <PopoverDialog
    label="Change organization scope"
    isOpen={open}
    onOpenChange={handleOpenChange}
    placement="bottom end"
    triggerLabel={`Organization scope, ${triggerName}`}
    triggerClassName="enterprise-scope-trigger"
    triggerDisabled={isChanging}
    triggerChildren={<>
      <span className="enterprise-scope-trigger__label">Scope</span>
      <span className="enterprise-scope-trigger__name">{triggerName}</span>
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

      {showSearch && <SearchField
        label="Search scope"
        value={query}
        onChange={setQuery}
        placeholder="Search scope"
        isDisabled={isChanging}
      />}

      <section className="enterprise-scope-section" aria-labelledby="enterprise-legal-entities-heading">
        <div className="enterprise-scope-section__heading">
          <strong id="enterprise-legal-entities-heading">Legal entities</strong>
          {!canSwitchLegalEntity && hierarchy.legal_entities.length > 1 && <small>Switching unavailable.</small>}
        </div>

        <div className="enterprise-scope-tree">
          <span className="enterprise-scope-tree__line" aria-hidden="true"/>
          <ul className="enterprise-scope-list" aria-label={`Legal entities in ${hierarchy.root.name}`}>
            {visibleEntities.map((entity) => {
              const selected = entity.id === currentScopeID && !activeOrganizationScopeID;
              const metadata = entity.jurisdiction || "Legal entity";
              return <li className="enterprise-scope-option-row" data-current={selected || undefined} key={entity.id}>
                <span className="enterprise-scope-option__branch" aria-hidden="true"/>
                <SelectableRecord
                  title={entity.name}
                  metadata={selected ? `${metadata} · Current` : metadata}
                  isSelected={selected}
                  isDisabled={isChanging || (entity.id !== currentScopeID && !canSwitchLegalEntity)}
                  onPress={() => chooseScope(entity.id)}
                />
              </li>;
            })}
          </ul>
        </div>

        {!visibleEntities.length && normalizedQuery && <p className="enterprise-scope-empty">No legal entity matches.</p>}
      </section>

      <section className="enterprise-scope-section enterprise-scope-areas" aria-labelledby="enterprise-areas-heading">
        <div className="enterprise-scope-section__heading">
          <strong id="enterprise-areas-heading">Areas</strong>
          {!areas.length && <small>No areas available.</small>}
        </div>
        {rootAreas.length > 0 && <ul className="enterprise-scope-area-tree">{rootAreas.map(renderArea)}</ul>}
        {areas.length > 0 && !rootAreas.length && normalizedQuery && <p className="enterprise-scope-empty">No area matches.</p>}
      </section>

      {onManageOrganization && <div className="enterprise-scope-management">
        <Button size="compact" variant="secondary" onPress={manageOrganization}>Organization & access</Button>
      </div>}
    </div>
  </PopoverDialog>;
}

function scopeMatches(scope: ScopeNode, query: string) {
  return [scope.name, scope.code, scope.kind, scope.jurisdiction, ...(scope.department_path ?? [])]
    .filter(Boolean)
    .some((value) => value!.toLowerCase().includes(query));
}

function humanizeScopeKind(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}
