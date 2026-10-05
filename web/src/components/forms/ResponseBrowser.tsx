import { useEffect, useState, type ReactNode } from "react";
import type { CompletedResponseQuery, CompletedResponseSummary, ResponseSort } from "../../formsDistributionApi";
import { Button, CheckboxField, FilterBar, FilterChip, SearchField, SelectField, TextField, type DataColumn } from "../ui";

export const responseSortOptions = [
  { id: "CONCERN_DESC", label: "Needs attention first" },
  { id: "COMPLETED_DESC", label: "Most recent" },
  { id: "RAW_ASC", label: "Lowest score first" },
  { id: "RAW_DESC", label: "Highest score first" },
] as const;
const concernOptions = [{ id: "CRITICAL", label: "Critical" }, { id: "HIGH", label: "High" }, { id: "MODERATE", label: "Moderate" }, { id: "LOW", label: "Low" }] as const;
const modeOptions = [{ id: "COMPLIANCE", label: "Compliance" }, { id: "RISK", label: "Risk" }] as const;
const stateOptions = [{ id: "FINAL", label: "Final score" }, { id: "PROVISIONAL", label: "Provisional score" }, { id: "FAILED", label: "Score unavailable" }, { id: "NOT_CONFIGURED", label: "Not scored" }] as const;
const subjectOptions = [{ id: "VENDOR_RELATIONSHIP", label: "Vendor services" }, { id: "PROGRAM", label: "Programs" }, { id: "MATTER", label: "Issues and changes" }, { id: "VENDOR", label: "Vendors (legacy)" }];

export function hasResponseFilters(query: CompletedResponseQuery) {
  return Boolean(query.search?.trim() || query.bands?.length || query.modes?.length || query.states?.length || query.completed_from || query.completed_until);
}

export function responseDateError(query: CompletedResponseQuery) {
  return query.completed_from && query.completed_until && query.completed_from > query.completed_until ? "The end date must be on or after the start date." : undefined;
}

export function sortableResponseColumns(columns: readonly DataColumn<CompletedResponseSummary>[], sort: ResponseSort, onSort: (sort: ResponseSort) => void) {
  return columns.map((column) => {
    const active = column.id === "completed" ? sort === "COMPLETED_DESC" : column.id === "concern" ? sort === "CONCERN_DESC" : column.id === "score" ? sort === "RAW_ASC" || sort === "RAW_DESC" : false;
    const next = column.id === "completed" ? "COMPLETED_DESC" : column.id === "concern" ? "CONCERN_DESC" : column.id === "score" ? sort === "RAW_DESC" ? "RAW_ASC" : "RAW_DESC" : undefined;
    return next ? { ...column, onSort: () => onSort(next), sortDirection: active ? sort === "RAW_ASC" ? "ascending" as const : "descending" as const : undefined } : column;
  });
}

export function useResponseColumns(columns: readonly DataColumn<CompletedResponseSummary>[]) {
  const [hidden, setHidden] = useState<string[]>([]);
  const picker = <details className="response-browser__columns"><summary>Columns</summary><div>
    {columns.filter((column) => column.id !== "form" && column.id !== "action").map((column) => <CheckboxField key={column.id} label={column.header} isSelected={!hidden.includes(column.id)} onChange={(selected) => setHidden((current) => selected ? current.filter((id) => id !== column.id) : [...current, column.id])}/>)}
  </div></details>;
  return { columns: columns.filter((column) => !hidden.includes(column.id)), picker };
}

export function ResponseBrowser({ query, onChange, count, hasMore, loading, showSubject = false, columns, children }: {
  query: CompletedResponseQuery; onChange: (patch: Partial<CompletedResponseQuery>) => void;
  count: number | undefined; hasMore: boolean; loading: boolean; showSubject?: boolean; columns: ReactNode; children: ReactNode;
}) {
  const dateError = responseDateError(query);
  const [filtersOpen, setFiltersOpen] = useState(() => !window.matchMedia("(max-width: 900px)").matches);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 900px)");
    const replace = () => setFiltersOpen(!media.matches);
    media.addEventListener("change", replace);
    return () => media.removeEventListener("change", replace);
  }, []);
  const clear = () => onChange({ search: undefined, bands: undefined, modes: undefined, states: undefined, completed_from: undefined, completed_until: undefined, ...(showSubject ? { subject_type: undefined } : {}) });
  const applied = hasResponseFilters(query) || showSubject && !!query.subject_type;
  return <div className="response-browser">
    <details className="response-browser__filters" open={filtersOpen} onToggle={(event) => setFiltersOpen(event.currentTarget.open)}>
      <summary>Filters{applied ? " · Applied" : ""}</summary>
      <FilterBar label="Response filters" fields={<div className="response-browser__filter-fields">
        <SelectField label="Concern" value={query.bands?.[0]} placeholder="All concern levels" options={concernOptions} onChange={(band) => onChange({ bands: band ? [band] : undefined })}/>
        <SelectField label="Score meaning" value={query.modes?.[0]} placeholder="All score meanings" options={modeOptions} onChange={(mode) => onChange({ modes: mode ? [mode] : undefined })}/>
        <SelectField label="Score state" value={query.states?.[0]} placeholder="All score states" options={stateOptions} onChange={(state) => onChange({ states: state ? [state] : undefined })}/>
        <TextField label="Submitted from" type="date" value={query.completed_from?.slice(0, 10) ?? ""} onChange={(value) => onChange({ completed_from: dateBoundary(value, false) })}/>
        <TextField label="Submitted until" type="date" value={query.completed_until?.slice(0, 10) ?? ""} isInvalid={!!dateError} errorMessage={dateError} onChange={(value) => onChange({ completed_until: dateBoundary(value, true) })}/>
        {showSubject && <SelectField label="Subject type" value={query.subject_type} placeholder="All subjects" options={subjectOptions} onChange={(subject_type) => onChange({ subject_type })}/>}
      </div>}/>
    </details>
    <div className="response-browser__main">
      <div className="response-browser__toolbar">
        <SearchField label="Search response titles" placeholder="Search form titles" value={query.search ?? ""} onChange={(search) => onChange({ search: search || undefined })} isLoading={loading}/>
        <SelectField label="Priority" value={query.sort ?? "CONCERN_DESC"} placeholder="Needs attention first" options={responseSortOptions} allowsEmpty={false} onChange={(sort) => onChange({ sort })}/>
        {columns}
      </div>
      <div className="response-browser__summary">
        <output aria-live="polite">{loading ? "Loading responses…" : count === undefined || dateError ? "Response count unavailable" : `${count} ${count === 1 ? "response" : "responses"}${hasMore ? " · More available" : ""}`}</output>
        <SelectField label="Batch size" value={String(query.limit ?? 25)} placeholder="25" allowsEmpty={false} options={[...new Set([20, 25, 50, 100, query.limit ?? 25])].sort((a, b) => a - b).map((value) => ({ id: String(value), label: String(value) }))} onChange={(value) => onChange({ limit: Number(value) })}/>
      </div>
      {applied && <div className="forms-responses__chips" aria-label="Applied response filters">
        <Button variant="quiet" onPress={clear}>Clear response filters</Button>
        {query.search && <FilterChip label="Title" value={query.search} onRemove={() => onChange({ search: undefined })}/>}
        {query.bands?.[0] && <FilterChip label="Concern" value={optionLabel(concernOptions, query.bands[0])} onRemove={() => onChange({ bands: undefined })}/>}
        {query.modes?.[0] && <FilterChip label="Score meaning" value={optionLabel(modeOptions, query.modes[0])} onRemove={() => onChange({ modes: undefined })}/>}
        {query.states?.[0] && <FilterChip label="Score state" value={optionLabel(stateOptions, query.states[0])} onRemove={() => onChange({ states: undefined })}/>}
        {query.completed_from && <FilterChip label="Submitted from" value={query.completed_from.slice(0, 10)} onRemove={() => onChange({ completed_from: undefined })}/>}
        {query.completed_until && <FilterChip label="Submitted until" value={query.completed_until.slice(0, 10)} onRemove={() => onChange({ completed_until: undefined })}/>}
        {showSubject && query.subject_type && <FilterChip label="Subject type" value={optionLabel(subjectOptions, query.subject_type)} onRemove={() => onChange({ subject_type: undefined })}/>}
      </div>}
      <div aria-busy={loading || undefined}>{dateError ? filtersOpen ? null : <p role="alert">{dateError}</p> : children}</div>
    </div>
  </div>;
}

function optionLabel(options: readonly { id: string; label: string }[], id: string) { return options.find((option) => option.id === id)?.label ?? id; }
function dateBoundary(value: string, end: boolean) {
  if (!value) return undefined;
  const date = new Date(`${value}T${end ? "23:59:59.999" : "00:00:00.000"}Z`);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}
