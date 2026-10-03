import { useEffect, useMemo, useRef, useState } from "react";
import { searchOrganizationScopes } from "../api";
import type { OrganizationScopeSearchPage, ScopeHierarchy, ScopeNode } from "../api";
import { Button, PopoverDialog, SearchField, SelectableRecord } from "./ui";

type Props = {
  hierarchy: ScopeHierarchy;
  currentScopeID: string;
  canSwitchLegalEntity?: boolean;
  activeOrganizationScope?: ScopeNode;
  activeOrganizationScopeID?: string;
  isChanging?: boolean;
  onSelectionChange: (legalEntityID: string) => void;
  onOrganizationScopeChange?: (organizationScope?: ScopeNode) => void;
  onManageOrganization?: () => void;
  searchOrganizationAreas?: (query: string, limit?: number) => Promise<OrganizationScopeSearchPage>;
};

const searchThreshold = 7;

export function EnterpriseScopeSwitcher({
  hierarchy,
  currentScopeID,
  canSwitchLegalEntity = true,
  activeOrganizationScope,
  activeOrganizationScopeID,
  isChanging = false,
  onSelectionChange,
  onOrganizationScopeChange,
  onManageOrganization,
  searchOrganizationAreas = searchOrganizationScopes,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [remoteAreas, setRemoteAreas] = useState<ScopeNode[]>([]);
  const [remoteState, setRemoteState] = useState<"idle" | "loading" | "live" | "unavailable">("idle");
  const searchRequestID = useRef(0);
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
  const selectedAreaID = activeOrganizationScope?.id ?? activeOrganizationScopeID;
  const activeArea = activeOrganizationScope ?? areas.find((area) => area.id === selectedAreaID && area.filterable);
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
  const remoteSearchActive = hierarchy.organization_scopes_truncated === true && normalizedQuery.length >= 2;
  const showSearch = hierarchy.organization_scopes_truncated === true || hierarchy.legal_entities.length + areas.length >= searchThreshold;

  useEffect(() => {
    const requestID = ++searchRequestID.current;
    if (!open || !remoteSearchActive) {
      setRemoteAreas([]);
      setRemoteState("idle");
      return;
    }
    setRemoteState("loading");
    const timer = window.setTimeout(() => {
      void searchOrganizationAreas(query.trim(), 30).then((page) => {
        if (requestID !== searchRequestID.current) return;
        setRemoteAreas(page.items ?? []);
        setRemoteState("live");
      }).catch(() => {
        if (requestID !== searchRequestID.current) return;
        setRemoteAreas([]);
        setRemoteState("unavailable");
      });
    }, 180);
    return () => window.clearTimeout(timer);
  }, [open, query, remoteSearchActive, searchOrganizationAreas]);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen) setQuery("");
  }

  function chooseScope(id: string) {
    if (isChanging) return;
    if (id === currentScopeID) {
      if (selectedAreaID && onOrganizationScopeChange) {
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

  function chooseOrganizationScope(area: ScopeNode) {
    if (!area.filterable || !onOrganizationScopeChange || isChanging) return;
    setOpen(false);
    setQuery("");
    onOrganizationScopeChange(area);
  }

  function manageOrganization() {
    setOpen(false);
    onManageOrganization?.();
  }

  function renderArea(area: ScopeNode) {
    if (!visibleAreaIDs.has(area.id)) return null;
    const children = (childrenByParent.get(area.id) ?? []).filter((child) => visibleAreaIDs.has(child.id));
    const selected = area.id === selectedAreaID;
    return <li className="enterprise-scope-area-node" key={area.id}>
      {area.filterable && onOrganizationScopeChange
        ? <SelectableRecord
          title={area.name}
          metadata={selected ? `${humanizeScopeKind(area.kind)} · Selected` : humanizeScopeKind(area.kind)}
          isSelected={selected}
          isDisabled={isChanging}
          onPress={() => chooseOrganizationScope(area)}
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
              const selected = entity.id === currentScopeID && !selectedAreaID;
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
          {!areas.length && !hierarchy.organization_scopes_truncated && <small>No areas available.</small>}
          {hierarchy.organization_scopes_truncated && !remoteSearchActive && <small>More areas available. Search to find them.</small>}
        </div>
        {remoteSearchActive ? <>
          {remoteState === "loading" && <p className="enterprise-scope-empty" role="status">Searching areas…</p>}
          {remoteState === "unavailable" && <p className="enterprise-scope-empty" role="alert">Area search unavailable. Try again.</p>}
          {remoteState === "live" && remoteAreas.length > 0 && <ul className="enterprise-scope-list" aria-label="Matching organization areas">
            {remoteAreas.map((area) => <li className="enterprise-scope-option-row" key={area.id}>
              {area.filterable && onOrganizationScopeChange
                ? <SelectableRecord
                  title={area.name}
                  metadata={area.department_path?.join(" / ") || humanizeScopeKind(area.kind)}
                  isSelected={area.id === selectedAreaID}
                  isDisabled={isChanging}
                  onPress={() => chooseOrganizationScope(area)}
                />
                : <div className="enterprise-scope-area-static"><span>{area.name}</span><small>{area.department_path?.join(" / ") || humanizeScopeKind(area.kind)}</small></div>}
            </li>)}
          </ul>}
          {remoteState === "live" && !remoteAreas.length && <p className="enterprise-scope-empty">No area matches.</p>}
        </> : <>
          {rootAreas.length > 0 && <ul className="enterprise-scope-area-tree">{rootAreas.map(renderArea)}</ul>}
          {areas.length > 0 && !rootAreas.length && normalizedQuery && <p className="enterprise-scope-empty">No area matches.</p>}
        </>}
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
