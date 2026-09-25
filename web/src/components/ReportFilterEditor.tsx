import { useMemo, useState } from "react";
import { Button, Notice, SelectField, StatusBadge, TextField } from "./ui";
import type { ReportDataset, ReportFilterExpression, ReportFilterFieldDefinition } from "../reportingTypes";
import "./ropa.css";

const MAX_FILTER_NODES = 12;
const MAX_FILTER_DEPTH = 3;

export type ReportFilterEditorProps = {
  fields: readonly ReportFilterFieldDefinition[];
  dataset: ReportDataset;
  value?: ReportFilterExpression;
  filter?: ReportFilterExpression;
  initialFilter?: ReportFilterExpression;
  onChange: (value: ReportFilterExpression) => void;
  onSave?: (value: ReportFilterExpression) => void;
  disabled?: boolean;
};

export function ReportFilterEditor({ fields, dataset, value, filter, initialFilter, onChange, onSave, disabled = false }: ReportFilterEditorProps) {
  const controlledValue = value ?? filter;
  const [internalValue, setInternalValue] = useState<ReportFilterExpression>(controlledValue ?? initialFilter ?? emptyGroup());
  const expression = controlledValue ?? internalValue;
  const datasetFields = useMemo(() => fieldsForDataset(fields, dataset), [fields, dataset]);
  const validationMessage = validateReportFilter(expression, fields, dataset);

  function update(next: ReportFilterExpression) {
    if (disabled) return;
    if (!controlledValue) setInternalValue(next);
    const nextMessage = validateReportFilter(next, fields, dataset);
    if (nextMessage) return;
    onChange(next);
  }

  function save() {
    if (disabled || validationMessage) return;
    if (onSave) onSave(expression);
    else onChange(expression);
  }

  return <section className="report-filter-editor" aria-labelledby="report-filter-editor-heading">
    <div className="report-filter-editor__heading">
      <div>
        <h3 id="report-filter-editor-heading">Filter the report population</h3>
        <p>Choose only the fields published for {datasetLabel(dataset)}. The saved run will use these conditions against the selected source boundary.</p>
      </div>
      <StatusBadge tone="info">{datasetLabel(dataset)}</StatusBadge>
    </div>

    {validationMessage && <Notice tone="error"><span>{validationMessage}</span></Notice>}

    <FilterGroupEditor
      expression={expression}
      fields={datasetFields}
      allFields={fields}
      dataset={dataset}
      disabled={disabled}
      onChange={update}
    />

    <div className="report-filter-editor__actions">
      <Button variant="primary" onPress={save} isDisabled={disabled || Boolean(validationMessage)}>Save filter</Button>
      <span>{validationMessage ? "Resolve the filter before saving this report." : "The filter is ready to be saved with this report definition."}</span>
    </div>
  </section>;
}

function FilterGroupEditor({ expression, fields, allFields, dataset, disabled, onChange, depth = 1, path = "root" }: {
  expression: ReportFilterExpression;
  fields: readonly ReportFilterFieldDefinition[];
  allFields: readonly ReportFilterFieldDefinition[];
  dataset: ReportDataset;
  disabled: boolean;
  onChange: (value: ReportFilterExpression) => void;
  depth?: number;
  path?: string;
}) {
  const group: ReportFilterExpression = expression.kind === "group" ? expression : emptyGroup();
  const childCount = group.children?.length ?? 0;
  const canAdd = !disabled && countNodes(expression) < MAX_FILTER_NODES;

  function updateChild(index: number, child: ReportFilterExpression) {
    const children = [...(group.children ?? [])];
    children[index] = child;
    onChange({ ...group, children });
  }

  function removeChild(index: number) {
    const children = [...(group.children ?? [])];
    children.splice(index, 1);
    onChange({ ...group, children });
  }

  function addCondition() {
    if (!fields[0]) return;
    onChange({ ...group, children: [...(group.children ?? []), emptyCondition(fields[0].field, fields[0].operators[0] ?? "is")] });
  }

  function addGroup() {
    if (!canAdd || depth >= MAX_FILTER_DEPTH) return;
    onChange({ ...group, children: [...(group.children ?? []), emptyGroup()] });
  }

  return <div className={`report-filter-group report-filter-group--depth-${depth}`} data-filter-path={path}>
    {depth === 1 && <div className="report-filter-group__header">
      <SelectField
        label="Combine conditions"
        value={group.operator === "or" ? "or" : "and"}
        placeholder="All conditions"
        options={[{ id: "and", label: "All conditions must match" }, { id: "or", label: "Any condition may match" }]}
        onChange={(next) => onChange({ ...group, operator: next ?? "and" })}
        isDisabled={disabled}
      />
    </div>}
    {childCount === 0 && <Notice tone="warning"><span>Add at least one condition before saving this report. Choose a field from the published vocabulary, then enter the value to match.</span></Notice>}
    <div className="report-filter-group__children">
      {(group.children ?? []).map((child, index) => child.kind === "group"
        ? <FilterGroupEditor key={`${path}-${index}`} expression={child} fields={fields} allFields={allFields} dataset={dataset} disabled={disabled} onChange={(next) => updateChild(index, next)} depth={depth + 1} path={`${path}.${index}`} />
        : <ConditionEditor key={`${path}-${index}`} expression={child} fields={fields} allFields={allFields} dataset={dataset} disabled={disabled} onChange={(next) => updateChild(index, next)} onRemove={() => removeChild(index)} />)}
    </div>
    <div className="report-filter-group__actions">
      <Button variant="secondary" size="compact" onPress={addCondition} isDisabled={!canAdd || fields.length === 0}>Add condition</Button>
      {depth < MAX_FILTER_DEPTH && <Button variant="quiet" size="compact" onPress={addGroup} isDisabled={!canAdd}>Add condition group</Button>}
    </div>
  </div>;
}

function ConditionEditor({ expression, fields, allFields, dataset, disabled, onChange, onRemove }: {
  expression: ReportFilterExpression;
  fields: readonly ReportFilterFieldDefinition[];
  allFields: readonly ReportFilterFieldDefinition[];
  dataset: ReportDataset;
  disabled: boolean;
  onChange: (value: ReportFilterExpression) => void;
  onRemove: () => void;
}) {
  const field = fields.find((candidate) => candidate.field === expression.field);
  const operators = field?.operators ?? [];
  const valueLabel = field?.label ?? "Filter value";

  return <div className="report-filter-condition" data-filter-field={expression.field ?? ""}>
    <div className="report-filter-condition__fields">
      <SelectField
        label="Change filter field"
        value={field?.field}
        placeholder="Choose a published field"
        options={fields.map((item) => ({ id: item.field, label: item.label, description: item.indexed ? "Indexed" : "Not indexed" }))}
        onChange={(next) => {
          const nextField = fields.find((item) => item.field === next);
          onChange({ ...expression, field: next, operator: nextField?.operators[0] ?? expression.operator, value: "" });
        }}
        isDisabled={disabled}
      />
      <SelectField
        label="Filter comparison"
        value={operators.includes(expression.operator) ? expression.operator : operators[0]}
        placeholder="Choose a comparison"
        options={operators.map((operator) => ({ id: operator, label: operatorLabel(operator) }))}
        onChange={(next) => onChange({ ...expression, operator: next ?? operators[0] ?? "is" })}
        isDisabled={disabled || !field}
      />
      <TextField label={valueLabel} value={expression.value ?? ""} onChange={(value) => onChange({ ...expression, value })} isDisabled={disabled || !field} isRequired />
    </div>
    {field && !field.indexed && <p className="report-filter-editor__help">This field is not indexed, so the report may take longer to run. Use it only when the slower read is acceptable.</p>}
    <div className="report-filter-condition__remove"><Button variant="quiet" size="compact" onPress={onRemove} isDisabled={disabled}>Remove condition</Button></div>
  </div>;
}

export function validateReportFilter(expression: ReportFilterExpression, fields: readonly ReportFilterFieldDefinition[], dataset: ReportDataset): string | undefined {
  let nodes = 0;
  return validateNode(expression, fields, dataset, 1, { count: () => ++nodes });
}

function validateNode(expression: ReportFilterExpression, fields: readonly ReportFilterFieldDefinition[], dataset: ReportDataset, depth: number, counter: { count: () => number }): string | undefined {
  if (counter.count() > MAX_FILTER_NODES) return `Report filters are limited to ${MAX_FILTER_NODES} conditions. Remove a condition before saving.`;
  if (depth > MAX_FILTER_DEPTH) return `Report filters are limited to ${MAX_FILTER_DEPTH} levels. Flatten the condition groups before saving.`;
  const kind = expression.kind.trim().toLowerCase();
  if (kind === "group") {
    if (expression.field || expression.value) return "A condition group cannot carry a field or value. Choose a field on each condition.";
    if (expression.operator !== "and" && expression.operator !== "or") return "Choose whether all conditions or any condition must match.";
    if (!expression.children || expression.children.length === 0) return "Add at least one condition before saving this report.";
    for (const child of expression.children) {
      const message = validateNode(child, fields, dataset, depth + 1, counter);
      if (message) return message;
    }
    return undefined;
  }
  if (kind !== "condition") return "A report filter can contain only conditions and condition groups.";
  const published = fields.find((field) => field.field === expression.field && fieldMatchesDataset(field.dataset, dataset));
  if (!published) {
    const other = fields.filter((field) => field.field === expression.field && !fieldMatchesDataset(field.dataset, dataset));
    if (other.length > 0) return `The field "${other[0]?.label ?? expression.field ?? "unknown"}" is not available for this report dataset. It belongs to ${other.map((field) => datasetLabel(field.dataset)).join(" and ")}. Choose a field published for ${datasetLabel(dataset)}.`;
    return `The field "${expression.field ?? "unknown"}" is not published for this report dataset. Choose a field from the published vocabulary.`;
  }
  if (!published.operators.includes(expression.operator)) return `Choose a comparison published for ${published.label}.`;
  if (!expression.value || !expression.value.trim()) return `Enter a value for ${published.label} before saving this report.`;
  if (expression.value.length > 200) return "Filter values are limited to 200 characters. Shorten the value before saving.";
  return undefined;
}

function fieldsForDataset(fields: readonly ReportFilterFieldDefinition[], dataset: ReportDataset) {
  const seen = new Set<string>();
  return fields.filter((field) => {
    if (!fieldMatchesDataset(field.dataset, dataset) || seen.has(field.field)) return false;
    seen.add(field.field);
    return true;
  });
}

export function fieldMatchesDataset(fieldDataset: ReportDataset, selectedDataset: ReportDataset) {
  return fieldDataset === selectedDataset || (selectedDataset === "PROCESSING_ACTIVITY_EXCEPTIONS" && fieldDataset === "PROCESSING_ACTIVITIES");
}

function countNodes(expression: ReportFilterExpression): number {
  return 1 + (expression.children ?? []).reduce((total, child) => total + countNodes(child), 0);
}

function emptyGroup(): ReportFilterExpression {
  return { kind: "group", operator: "and", children: [] };
}

function emptyCondition(field: string, operator: string): ReportFilterExpression {
  return { kind: "condition", field, operator, value: "" };
}

function operatorLabel(operator: string) {
  if (operator === "is") return "is";
  if (operator === "is_not") return "is not";
  if (operator === "contains") return "contains";
  return operator;
}

function datasetLabel(dataset: ReportDataset) {
  if (dataset === "PROCESSING_ACTIVITIES") return "Processing activities";
  if (dataset === "PROCESSING_ACTIVITY_EXCEPTIONS") return "Processing activities with open exceptions";
  if (dataset === "PROGRAMS") return "Programs";
  return "Issues and changes with open exceptions or overdue obligations";
}
