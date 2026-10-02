import { useMemo, useState } from "react";
import { Button as AriaButton } from "react-aria-components";
import type { ScopeHierarchy } from "../api";
import { PopoverDialog, SearchField } from "./ui";

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
    label="Change legal entity"
    isOpen={open}
    onOpenChange={handleOpenChange}
    placement="bottom start"
    trigger={<AriaButton
      className="enterprise-scope-trigger"
      aria-label={`Legal entity, ${current.name}`}
      isDisabled={isChanging}
    >
      <span className="enterprise-scope-trigger__name">{current.name}</span>
      <span className="enterprise-scope-trigger__chevron" aria-hidden="true">⌄</span>
    </AriaButton>}
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
            return <li key={entity.id}>
              <button
                type="button"
                className="enterprise-scope-option"
                aria-current={selected ? "true" : undefined}
                onClick={() => chooseScope(entity.id)}
                disabled={isChanging}
              >
                <span className="enterprise-scope-option__branch" aria-hidden="true"/>
                <span className="enterprise-scope-option__content">
                  <strong>{entity.name}</strong>
                  <small>{entity.jurisdiction || "Legal entity"}</small>
                </span>
                {selected && <span className="enterprise-scope-option__current">Current</span>}
              </button>
            </li>;
          })}
        </ul>
      </div>

      {!visibleEntities.length && <p className="enterprise-scope-empty">No legal entities match this search.</p>}
    </div>
  </PopoverDialog>;
}
