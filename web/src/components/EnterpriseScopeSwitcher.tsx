import { useMemo, useState } from "react";
import type { ScopeHierarchy } from "../api";
import { PopoverDialog, SearchField, SelectableRecord } from "./ui";

type Props = {
  hierarchy: ScopeHierarchy;
  currentScopeID: string;
  isChanging?: boolean;
  onSelectionChange: (legalEntityID: string) => void;
};

const searchThreshold = 7;

export function EnterpriseScopeSwitcher({
  hierarchy,
  currentScopeID,
  isChanging = false,
  onSelectionChange,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const current = hierarchy.legal_entities.find((entity) => entity.id === currentScopeID) ?? hierarchy.current;
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
    if (id === currentScopeID || isChanging) return;
    setOpen(false);
    setQuery("");
    onSelectionChange(id);
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
                isDisabled={isChanging}
                onPress={() => chooseScope(entity.id)}
              />
            </li>;
          })}
        </ul>
      </div>

      {!visibleEntities.length && <p className="enterprise-scope-empty">No legal entities match this search.</p>}
    </div>
  </PopoverDialog>;
}
