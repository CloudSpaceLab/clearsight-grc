import { type Key, type ReactNode } from "react";
import { Tab, TabList, TabPanel, Tabs as AriaTabs } from "react-aria-components";
import { SelectField } from "./SelectField";

export type TabItem<T extends string> = { id: T; label: string };

export type TabsProps<T extends string> = {
  ariaLabel: string;
  compactLabel?: string;
  items: readonly TabItem<T>[];
  selectedKey: T;
  onSelectionChange: (key: T) => void;
  children: (key: T) => ReactNode;
};

export function Tabs<T extends string>({ ariaLabel, compactLabel, items, selectedKey, onSelectionChange, children }: TabsProps<T>) {
  function select(key: Key) {
    if (typeof key === "string" && key !== selectedKey) onSelectionChange(key as T);
  }

  return <AriaTabs className={`cs-tabs${compactLabel ? " cs-tabs--compact-select" : ""}`} selectedKey={selectedKey} onSelectionChange={select} keyboardActivation="automatic">
    {compactLabel && <div className="cs-tabs__compact"><SelectField label={compactLabel} value={selectedKey} placeholder={compactLabel}
      options={items} allowsEmpty={false} onChange={(key) => { if (key !== undefined && key !== selectedKey) onSelectionChange(key); }}/></div>}
    <TabList aria-label={ariaLabel} className="cs-tabs__list" items={items}>
      {(item) => <Tab id={item.id} className="cs-tabs__tab">
        {({ isSelected }) => <>
          <span>{item.label}</span>
          {isSelected && <span className="cs-tabs__indicator" aria-hidden="true"/>}
        </>}
      </Tab>}
    </TabList>
    <TabPanel key={selectedKey} id={selectedKey} className="cs-tabs__panel">{children(selectedKey)}</TabPanel>
  </AriaTabs>;
}
