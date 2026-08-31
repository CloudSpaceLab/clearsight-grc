import type { FilterCondition, FilterExpression, FilterField, FilterGroup } from "./filterModel";
import { filterExpressionNodeCount } from "./filterModel";
import { filterDefinition, formatFilterValue, selectableFormFilterRegistry } from "./filterRegistry";
import { Button, IconButton, SelectField, TextField } from "../../ui";

const MAX_NODES = 12;
const MAX_DEPTH = 3;
const matchOptions = [
  { id: "and", label: "All conditions" },
  { id: "or", label: "Any condition" },
] as const;

export function FilterExpressionEditor({ expression, onChange }: { expression: FilterGroup; onChange: (expression: FilterGroup) => void }) {
  const nodeCount = filterExpressionNodeCount(expression);
  return <GroupEditor group={expression} depth={1} nodeCount={nodeCount} onChange={onChange}/>;
}

function GroupEditor({ group, depth, nodeCount, onChange, onRemove }: {
  group: FilterGroup;
  depth: number;
  nodeCount: number;
  onChange: (group: FilterGroup) => void;
  onRemove?: () => void;
}) {
  const canAddCondition = nodeCount < MAX_NODES;
  const canAddGroup = depth < MAX_DEPTH && nodeCount <= MAX_NODES - 2;

  function updateChild(index: number, child: FilterExpression) {
    onChange({ ...group, children: group.children.map((current, childIndex) => childIndex === index ? child : current) });
  }

  function removeChild(index: number) {
    onChange({ ...group, children: group.children.filter((_, childIndex) => childIndex !== index) });
  }

  function addCondition() {
    if (canAddCondition) onChange({ ...group, children: [...group.children, emptyCondition()] });
  }

  function addGroup() {
    if (!canAddGroup) return;
    onChange({ ...group, children: [...group.children, { kind: "group", operator: "and", children: [emptyCondition()] }] });
  }

  return <fieldset className={depth === 1 ? "forms-expression-group root" : "forms-expression-group"}>
    <legend className="cs-sr-only">{depth === 1 ? "Advanced filter logic" : "Filter group"}</legend>
    <div className="forms-expression-group-heading">
      <div className="forms-expression-mode">
        <span aria-hidden="true">{depth === 1 ? "Match" : "Group"}</span>
        <div className="forms-expression-mode-control"><SelectField
          label={depth === 1 ? "Advanced filter match mode" : `Filter group level ${depth} match mode`}
          value={group.operator}
          placeholder="Choose match mode"
          options={matchOptions}
          allowsEmpty={false}
          isLabelHidden
          onChange={(operator) => operator && onChange({ ...group, operator })}
        /></div>
      </div>
      {onRemove && <Button size="compact" variant="quiet" onPress={onRemove}>Remove group</Button>}
    </div>

    <div className="forms-expression-children">
      {group.children.map((child, index) => child.kind === "condition"
        ? <ConditionEditor key={`${depth}:condition:${index}`} condition={child} index={index} onChange={(next) => updateChild(index, next)} onRemove={() => removeChild(index)}/>
        : <GroupEditor key={`${depth}:group:${index}`} group={child} depth={depth + 1} nodeCount={nodeCount} onChange={(next) => updateChild(index, next)} onRemove={() => removeChild(index)}/>)}
      {group.children.length === 0 && <p className="forms-expression-empty">Add a condition to define this group.</p>}
    </div>

    <div className="forms-expression-actions">
      <Button size="compact" isDisabled={!canAddCondition} onPress={addCondition}>+ Condition</Button>
      {depth < MAX_DEPTH && <Button size="compact" variant="quiet" isDisabled={!canAddGroup} onPress={addGroup}>+ Group</Button>}
    </div>
  </fieldset>;
}

function ConditionEditor({ condition, index, onChange, onRemove }: {
  condition: FilterCondition;
  index: number;
  onChange: (condition: FilterCondition) => void;
  onRemove: () => void;
}) {
  const definition = filterDefinition(condition.field);
  const definitions = definition.selectable === false ? [definition, ...selectableFormFilterRegistry] : selectableFormFilterRegistry;

  function chooseField(field: FilterField) {
    onChange({ kind: "condition", field, operator: "is", value: "" });
  }

  return <div className="forms-expression-condition">
    <SelectField
      label={`Condition ${index + 1} field`}
      value={condition.field}
      placeholder="Choose field"
      options={definitions.map((item) => ({ id: item.field, label: item.label }))}
      allowsEmpty={false}
      isLabelHidden
      onChange={(field) => field && chooseField(field)}
    />
    <span className="forms-expression-operator">is</span>
    {definition.selectable === false ? <span className="forms-expression-preserved-value" aria-label={`Condition ${index + 1} ${definition.label} value`}>
      {formatFilterValue(condition.field, condition.value)}
    </span> : definition.input === "select" ? <SelectField
      label={`Condition ${index + 1} ${definition.label} value`}
      value={condition.value || undefined}
      placeholder={`Choose ${definition.label.toLowerCase()}`}
      options={definition.options?.map((option) => ({ id: option.value, label: option.label })) ?? []}
      isLabelHidden
      onChange={(value) => onChange({ ...condition, value: value ?? "" })}
    /> : <TextField
      label={`Condition ${index + 1} ${definition.label} value`}
      value={condition.value}
      maxLength={200}
      placeholder={definition.placeholder}
      isLabelHidden
      onChange={(value) => onChange({ ...condition, value })}
    />}
    <IconButton aria-label={`Remove condition ${index + 1}`} variant="quiet" onPress={onRemove}>×</IconButton>
  </div>;
}

function emptyCondition(): FilterCondition {
  return { kind: "condition", field: "status", operator: "is", value: "" };
}
