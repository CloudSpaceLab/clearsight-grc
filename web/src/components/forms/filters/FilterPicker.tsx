import { useEffect, useMemo, useRef, useState } from "react";
import type { FilterCondition, FilterField } from "./filterModel";
import { filterDefinition, formFilterRegistry } from "./filterRegistry";

type Props = {
  activeFields: ReadonlySet<FilterField>;
  onApply: (condition: FilterCondition) => void;
  onClose: () => void;
};

export function FilterPicker({ activeFields, onApply, onClose }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [field, setField] = useState<FilterField>();
  const [value, setValue] = useState("");
  const available = useMemo(() => formFilterRegistry.filter((item) => !activeFields.has(item.field)), [activeFields]);
  const definition = field ? filterDefinition(field) : undefined;

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  useEffect(() => {
    panelRef.current?.querySelector<HTMLElement>("button, select, input")?.focus();
  }, [field]);

  function choose(nextField: FilterField) {
    setField(nextField);
    setValue("");
  }

  function apply() {
    const normalized = value.trim();
    if (!field || !normalized) return;
    onApply({ kind: "condition", field, operator: "is", value: normalized });
  }

  return <div className="forms-filter-popover" role="dialog" aria-label="Add filter" ref={panelRef}>
    <div className="forms-filter-popover-header">
      {field ? <button type="button" className="text-button" onClick={() => setField(undefined)}>← Filters</button> : <strong>Add filter</strong>}
      <button type="button" className="icon-button" aria-label="Close filters" onClick={onClose}>×</button>
    </div>

    {!definition ? <div className="forms-filter-fields">
      {available.length ? available.map((item) => <button type="button" key={item.field} onClick={() => choose(item.field)}>
        <span>{item.label}</span>
        <small>{item.category}</small>
      </button>) : <p className="forms-filter-empty">All available filters are already applied.</p>}
    </div> : <div className="forms-filter-editor">
      <div className="forms-filter-editor-title">
        <span>{definition.label}</span>
        <small>is</small>
      </div>
      {definition.input === "select" ? <select aria-label={`${definition.label} value`} value={value} onChange={(event) => setValue(event.target.value)}>
        <option value="">Choose {definition.label.toLowerCase()}</option>
        {definition.options?.map((option) => <option value={option.value} key={option.value}>{option.label}</option>)}
      </select> : <input
        aria-label={`${definition.label} value`}
        value={value}
        placeholder={definition.placeholder}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); apply(); } }}
      />}
      <button type="button" className="forms-primary" disabled={!value.trim()} onClick={apply}>Apply filter</button>
    </div>}
  </div>;
}
