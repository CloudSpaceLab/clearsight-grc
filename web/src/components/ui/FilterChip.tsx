import { Button as AriaButton } from "react-aria-components";

export type FilterChipProps = {
  label: string;
  value: string;
  onRemove?: () => void;
  onPress?: () => void;
  actionLabel?: string;
  emphasis?: "default" | "accent";
};

export function FilterChip({ label, value, onRemove, onPress, actionLabel, emphasis = "default" }: FilterChipProps) {
  const press = onRemove ?? onPress;
  const accessibleName = actionLabel ?? (onRemove ? `Remove ${label} filter` : `${label}: ${value}`);
  return <AriaButton
    className={`cs-filter-chip cs-filter-chip--${emphasis}`}
    aria-label={accessibleName}
    onPress={press}
  >
    <span className="cs-filter-chip__label">{label}</span>
    <strong className="cs-filter-chip__value">{value}</strong>
    {onRemove && <span className="cs-filter-chip__remove" aria-hidden="true">×</span>}
  </AriaButton>;
}
