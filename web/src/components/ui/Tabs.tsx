import { type Key, type ReactNode } from "react";
import { Tab, TabList, TabPanel, Tabs as AriaTabs } from "react-aria-components";

export type TabItem<T extends string> = { id: T; label: string };

export type TabsProps<T extends string> = {
  ariaLabel: string;
  items: readonly TabItem<T>[];
  selectedKey: T;
  onSelectionChange: (key: T) => void;
  children: (key: T) => ReactNode;
};

export function Tabs<T extends string>({ ariaLabel, items, selectedKey, onSelectionChange, children }: TabsProps<T>) {
  function select(key: Key) {
    if (typeof key === "string") onSelectionChange(key as T);
  }

  return <AriaTabs className="cs-tabs" selectedKey={selectedKey} onSelectionChange={select} keyboardActivation="automatic">
    <TabList aria-label={ariaLabel} className="cs-tabs__list" items={items}>
      {(item) => <Tab id={item.id} className="cs-tabs__tab">
        {({ isSelected }) => <>
          <span>{item.label}</span>
          {isSelected && <span className="cs-tabs__indicator" aria-hidden="true"/>}
        </>}
      </Tab>}
    </TabList>
    <TabPanel id={selectedKey} className="cs-tabs__panel">{children(selectedKey)}</TabPanel>
  </AriaTabs>;
}
