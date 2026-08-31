import type { ReactNode } from "react";
import { Button } from "./Button";

export type FilterBarProps = {
  label: string;
  fields: ReactNode;
  resultCount?: number;
  resultLabel?: (count: number) => string;
  clearLabel?: string;
  onClear?: () => void;
};

export function FilterBar({ label, fields, resultCount, resultLabel, clearLabel = "Clear filters", onClear }: FilterBarProps) {
  return <section className="cs-filter-bar" aria-label={label}>
    <div className="cs-filter-bar__fields">{fields}</div>
    {(resultCount !== undefined || onClear) && <div className="cs-filter-bar__summary">
      {resultCount !== undefined && <output className="cs-filter-bar__count">{resultLabel?.(resultCount) ?? `${resultCount} results`}</output>}
      {onClear && <Button variant="quiet" onPress={onClear}>{clearLabel}</Button>}
    </div>}
  </section>;
}
