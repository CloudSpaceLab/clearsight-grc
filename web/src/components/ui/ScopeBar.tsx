import { Button as AriaButton } from "react-aria-components";

export type ScopeItem<T extends string> = {
  id: T;
  label: string;
  count?: number;
};

export type ScopeBarProps<T extends string> = {
  ariaLabel: string;
  items: readonly ScopeItem<T>[];
  selectedKey: T;
  onSelectionChange: (key: T) => void;
};

export function ScopeBar<T extends string>({ ariaLabel, items, selectedKey, onSelectionChange }: ScopeBarProps<T>) {
  return <nav className="cs-scope-bar" aria-label={ariaLabel}>
    {items.map((item) => {
      const selected = item.id === selectedKey;
      const name = item.count === undefined ? item.label : `${item.label} ${item.count}`;
      return <AriaButton
        className="cs-scope-bar__item"
        aria-label={name}
        aria-current={selected ? "page" : undefined}
        data-selected={selected || undefined}
        key={item.id}
        onPress={() => onSelectionChange(item.id)}
      >
        <span>{item.label}</span>
        {item.count !== undefined && <strong>{item.count}</strong>}
      </AriaButton>;
    })}
  </nav>;
}
