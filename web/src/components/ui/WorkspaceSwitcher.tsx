import { Button as AriaButton } from "react-aria-components";
import { SelectField } from "./SelectField";

export type WorkspaceSwitcherItem<T extends string> = { id: T; label: string };

export type WorkspaceSwitcherProps<T extends string> = {
  ariaLabel: string;
  compactLabel: string;
  items: readonly WorkspaceSwitcherItem<T>[];
  selectedKey: T;
  onSelectionChange: (key: T) => void;
};

export function WorkspaceSwitcher<T extends string>({ ariaLabel, compactLabel, items, selectedKey, onSelectionChange }: WorkspaceSwitcherProps<T>) {
  return <div className="cs-workspace-switcher">
    <nav className="cs-workspace-switcher__list" aria-label={ariaLabel}>
      {items.map((item) => <AriaButton
        key={item.id}
        className="cs-workspace-switcher__item"
        aria-current={item.id === selectedKey ? "page" : undefined}
        onPress={() => { if (item.id !== selectedKey) onSelectionChange(item.id); }}
      >
        <span>{item.label}</span>
        {item.id === selectedKey && <span className="cs-workspace-switcher__indicator" aria-hidden="true"/>}
      </AriaButton>)}
    </nav>
    <div className="cs-workspace-switcher__compact">
      <SelectField label={compactLabel} value={selectedKey} placeholder={compactLabel} options={items} allowsEmpty={false}
        onChange={(key) => { if (key !== undefined && key !== selectedKey) onSelectionChange(key); }}/>
    </div>
  </div>;
}
