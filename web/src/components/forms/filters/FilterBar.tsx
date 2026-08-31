import { useMemo, useState } from "react";
import type { FormTemplateQuery } from "../../../formsTypes";
import {
  activeFilterConditions,
  filterExpressionSummary,
  removeFilter,
  setAdvancedFilter,
  setFilter,
} from "./filterModel";
import type { FilterCondition, FilterExpression, FilterField } from "./filterModel";
import { filterDefinition, formatFilterValue } from "./filterRegistry";
import { AdvancedFilterEditor } from "./AdvancedFilterEditor";
import { FilterPicker } from "./FilterPicker";

type Props = {
  query: FormTemplateQuery;
  onChange: (query: FormTemplateQuery) => void;
  resultCount?: number;
  revalidating?: boolean;
};

export function FilterBar({ query, onChange, resultCount, revalidating = false }: Props) {
  const [open, setOpen] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const conditions = activeFilterConditions(query);
  const activeFields = useMemo(() => new Set<FilterField>(conditions.map((condition) => condition.field)), [conditions]);
  const advancedSummary = filterExpressionSummary(query.filter);

  function apply(condition: FilterCondition) {
    onChange(setFilter(query, condition));
    setOpen(false);
  }

  function applyAdvanced(expression?: FilterExpression) {
    onChange(setAdvancedFilter(query, expression));
    setAdvancedOpen(false);
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

      {advancedSummary && <button
        type="button"
        className="forms-filter-chip advanced"
        aria-label="Edit advanced filters"
        onClick={() => { setOpen(false); setAdvancedOpen(true); }}
      >
        <span>Advanced</span>
        <strong>{advancedSummary}</strong>
      </button>}

      <div className="forms-filter-add">
        <button type="button" className="forms-filter-add-button" aria-expanded={open} onClick={() => { setAdvancedOpen(false); setOpen((value) => !value); }}>+ Filter</button>
        {open && <FilterPicker activeFields={activeFields} onApply={apply} onClose={() => setOpen(false)}/>} 
      </div>
      <button
        type="button"
        className="forms-filter-advanced-button"
        aria-expanded={advancedOpen}
        onClick={() => { setOpen(false); setAdvancedOpen(true); }}
      >
        {advancedSummary ? "Edit logic" : "Advanced"}
      </button>
    </div>

    <div className="forms-filter-status" aria-live="polite">
      {revalidating ? <span>Updating…</span> : typeof resultCount === "number" ? <span>{resultCount} {resultCount === 1 ? "result" : "results"}</span> : null}
    </div>

    {advancedOpen && <AdvancedFilterEditor expression={query.filter} onApply={applyAdvanced} onClose={() => setAdvancedOpen(false)}/>} 
  </div>;
}
