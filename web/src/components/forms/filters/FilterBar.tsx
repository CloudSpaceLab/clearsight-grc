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
import { Button, FilterChip, PopoverDialog, SearchField } from "../../ui";

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
    <SearchField
      label="Search templates"
      value={query.search ?? ""}
      placeholder="Search forms…"
      isLoading={revalidating}
      onChange={(search) => onChange({ ...query, search: search || undefined, cursor: undefined })}
    />

    <div className="forms-filter-chips" aria-label="Active form filters">
      {conditions.map((condition) => {
        const definition = filterDefinition(condition.field);
        return <FilterChip
          key={condition.field}
          label={definition.label}
          value={formatFilterValue(condition.field, condition.value)}
          onRemove={() => onChange(removeFilter(query, condition.field))}
        />;
      })}

      {advancedSummary && <FilterChip
        label="Advanced"
        value={advancedSummary}
        emphasis="accent"
        actionLabel="Edit advanced filters"
        onPress={() => { setOpen(false); setAdvancedOpen(true); }}
      />}

      <PopoverDialog
        label="Add filter"
        isOpen={open}
        onOpenChange={(next) => { setAdvancedOpen(false); setOpen(next); }}
        trigger={<Button size="compact">+ Filter</Button>}
      >
        <FilterPicker activeFields={activeFields} onApply={apply} onClose={() => setOpen(false)}/>
      </PopoverDialog>
      <Button
        size="compact"
        variant="quiet"
        aria-expanded={advancedOpen}
        onPress={() => { setOpen(false); setAdvancedOpen(true); }}
      >
        {advancedSummary ? "Edit logic" : "Advanced"}
      </Button>
      <Button
        size="compact"
        variant="quiet"
        aria-label={query.sort === "UPDATED_ASC" ? "Sort by newest update" : "Sort by oldest update"}
        onPress={() => onChange({ ...query, sort: query.sort === "UPDATED_ASC" ? "UPDATED_DESC" : "UPDATED_ASC", cursor: undefined })}
      >
        Updated {query.sort === "UPDATED_ASC" ? "↑" : "↓"}
      </Button>
    </div>

    <div className="forms-filter-status" aria-live="polite">
      {revalidating ? <span>Updating…</span> : typeof resultCount === "number" ? <span>{resultCount} {resultCount === 1 ? "result" : "results"}</span> : null}
    </div>

    {advancedOpen && <AdvancedFilterEditor expression={query.filter} onApply={applyAdvanced} onClose={() => setAdvancedOpen(false)}/>} 
  </div>;
}
