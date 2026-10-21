import { useCallback, useEffect, useMemo, useState, type Key, type ReactNode } from "react";
import { Tab, TabList, TabPanel, Tabs as AriaTabs } from "react-aria-components";
import { SelectField } from "./SelectField";

export type TabItem<T extends string> = { id: T; label: string; isContextual?: boolean };

export type TabsProps<T extends string> = {
  ariaLabel: string;
  compactLabel?: string;
  retainVisitedPanels?: boolean;
  items: readonly TabItem<T>[];
  selectedKey: T;
  onSelectionChange: (key: T) => void;
  children: (key: T) => ReactNode;
};

export function Tabs<T extends string>({ ariaLabel, compactLabel, retainVisitedPanels = false, items, selectedKey, onSelectionChange, children }: TabsProps<T>) {
  // Contextual destinations are entered by a command elsewhere in the workspace.
  // Keep their panel identity even while their inactive trigger is absent.
  const visibleItems = items.filter((item) => !item.isContextual || item.id === selectedKey);
  const [visited, setVisited] = useState<readonly T[]>([selectedKey]);
  const mounted = useMemo(() => retainVisitedPanels ? visited.includes(selectedKey) ? visited : [...visited, selectedKey] : [selectedKey], [retainVisitedPanels, selectedKey, visited]);
  useEffect(() => {
    if (retainVisitedPanels && !visited.includes(selectedKey)) setVisited((current) => current.includes(selectedKey) ? current : [...current, selectedKey]);
  }, [retainVisitedPanels, selectedKey, visited]);
  const revealSelectedTab = useCallback((selected: HTMLDivElement | null) => {
    if (!selected) return;
    // React Aria may mount a contextual trigger after the parent's layout
    // effects. Reveal it from its own ref without moving the page vertically.
    selected.ownerDocument.defaultView?.requestAnimationFrame(() => {
      const list = selected.closest<HTMLElement>('[role="tablist"]');
      if (!selected.isConnected || !list || selected.getAttribute("aria-selected") !== "true") return;
      const bounds = list.getBoundingClientRect();
      const destination = selected.getBoundingClientRect();
      if (destination.left < bounds.left) list.scrollLeft += destination.left - bounds.left;
      else if (destination.right > bounds.right) list.scrollLeft += destination.right - bounds.right;
    });
  }, []);
  function select(key: Key) {
    if (typeof key === "string" && key !== selectedKey) onSelectionChange(key as T);
  }

  return <AriaTabs className={`cs-tabs${compactLabel ? " cs-tabs--compact-select" : ""}`} selectedKey={selectedKey} onSelectionChange={select} keyboardActivation="automatic">
    {compactLabel && <div className="cs-tabs__compact"><SelectField label={compactLabel} value={selectedKey} placeholder={compactLabel}
      options={visibleItems} allowsEmpty={false} onChange={(key) => { if (key !== undefined && key !== selectedKey) onSelectionChange(key); }}/></div>}
    <TabList aria-label={ariaLabel} className="cs-tabs__list" items={visibleItems}>
      {(item) => <Tab ref={item.id === selectedKey ? revealSelectedTab : undefined} id={item.id} className="cs-tabs__tab">
        {({ isSelected }) => <>
          <span>{item.label}</span>
          {isSelected && <span className="cs-tabs__indicator" aria-hidden="true"/>}
        </>}
      </Tab>}
    </TabList>
    {retainVisitedPanels ? items.filter((item) => mounted.includes(item.id)).map((item) => <TabPanel key={item.id} id={item.id} shouldForceMount className="cs-tabs__panel" style={item.id === selectedKey ? undefined : { display: "none" }}>{children(item.id)}</TabPanel>) : <TabPanel key={selectedKey} id={selectedKey} className="cs-tabs__panel">{children(selectedKey)}</TabPanel>}
  </AriaTabs>;
}
