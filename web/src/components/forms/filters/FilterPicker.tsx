import { useEffect, useMemo, useRef, useState } from "react";
import type { FilterCondition, FilterField } from "./filterModel";
import { filterDefinition, selectableFormFilterRegistry } from "./filterRegistry";
import { Button, IconButton, SelectField, TextField } from "../../ui";

type Props = {
  activeFields: ReadonlySet<FilterField>;
  onApply: (condition: FilterCondition) => void;
  onClose: () => void;
};

export function FilterPicker({ activeFields, onApply, onClose }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [field, setField] = useState<FilterField>();
  const [value, setValue] = useState("");
  const available = useMemo(() => selectableFormFilterRegistry.filter((item) => !activeFields.has(item.field)), [activeFields]);
  const definition = field ? filterDefinition(field) : undefined;

  useEffect(() => {
    panelRef.current?.querySelector<HTMLElement>("button, input")?.focus();
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

  return <div className="forms-filter-picker" ref={panelRef}>
    <div className="forms-filter-popover-header">
      {field ? <Button size="compact" variant="quiet" onPress={() => setField(undefined)}>← Filters</Button> : <strong>Add filter</strong>}
      <IconButton aria-label="Close filters" variant="quiet" onPress={onClose}>×</IconButton>
    </div>

    {!definition ? <div className="forms-filter-fields">
      {available.length ? available.map((item) => <Button variant="quiet" key={item.field} onPress={() => choose(item.field)}>
        <span>{item.label}</span>
        <small>{item.category}</small>
      </Button>) : <p className="forms-filter-empty">All available filters are already applied.</p>}
    </div> : <div className="forms-filter-editor">
      <div className="forms-filter-editor-title">
        <span>{definition.label}</span>
        <small>is</small>
      </div>
      {definition.input === "select" ? <SelectField
        label={`${definition.label} value`}
        value={value || undefined}
        placeholder={`Choose ${definition.label.toLowerCase()}`}
        options={definition.options?.map((option) => ({ id: option.value, label: option.label })) ?? []}
        onChange={(next) => setValue(next ?? "")}
        isLabelHidden
      /> : <TextField
        label={`${definition.label} value`}
        value={value}
        placeholder={definition.placeholder}
        onChange={setValue}
        isLabelHidden
      />}
      <Button variant="primary" isDisabled={!value.trim()} onPress={apply}>Apply filter</Button>
    </div>}
  </div>;
}
