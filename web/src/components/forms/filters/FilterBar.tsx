import { useMemo, useState } from "react";
import type { FormTemplateQuery } from "../../../formsTypes";
import { activeFilterConditions, removeFilter, setFilter } from "./filterModel";
import type { FilterCondition, FilterField } from "./filterModel";
import { filterDefinition, formatFilterValue } from "./filterRegistry";
import { FilterPicker } from "./FilterPicker";

type Props = {
  query: FormTemplateQuery;
  onChange: (query: FormTemplateQuery) => void;
  resultCount?: number;
  revalidating?: boolean;
};

export function FilterBar({ query, onChange, resultCount, revalidating = false }: Props) {
  const [open, setOpen] = useState(false);
  const conditions = activeFilterConditions(query);
  const activeFields = useMemo(() => new Set<FilterField>(conditions.map((condition) => condition.field)), [conditions]);

  function apply(condition: FilterCondition) {
    onChange(setFilter(query, condition));
    setOpen(false);
  }

  return <div className="forms-filter-bar">
    <label className="forms-filter-search">
      <span className="sr-only">Search templates</span>
      <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/></svg>
      <input
        type="search"
        aria-label="Search templates"
        value={query.search ?? ""}
        placeholder="Search forms…"
        onChange={(event) => onChange({ ...query, search: event.target.value || undefined, cursor: undefined })}
      />
    </label>

    <div className="forms-filter-chips" aria-label="Active form filters">
      {conditions.map((condition) => {
        const definition = filterDefinition(condition.field);
        return <button
          type="button"
          className="forms-filter-chip"
          key={condition.field}
          aria-label={`Remove ${definition.label} filter`}
          title={`Remove ${definition.label} filter`}
          onClick={() => onChange(removeFilter(query, condition.field))}
        >
          <span>{definition.label}</span>
          <strong>{formatFilterValue(condition.field, condition.value)}</strong>
          <span aria-hidden="true">×</span>
        </button>;
      })}

      <div className="forms-filter-add">
        <button type="button" className="forms-filter-add-button" aria-expanded={open} onClick={() => setOpen((value) => !value)}>+ Filter</button>
        {open && <FilterPicker activeFields={activeFields} onApply={apply} onClose={() => setOpen(false)}/>} 
      </div>
    </div>

    <div className="forms-filter-status" aria-live="polite">
      {revalidating ? <span>Updating…</span> : typeof resultCount === "number" ? <span>{resultCount} {resultCount === 1 ? "result" : "results"}</span> : null}
    </div>
  </div>;
}
