import type { FilterCondition, FilterExpression, FilterField, FilterGroup } from "./filterModel";
import { filterExpressionNodeCount } from "./filterModel";
import { filterDefinition, formFilterRegistry } from "./filterRegistry";

const MAX_NODES = 12;
const MAX_DEPTH = 3;

export function FilterExpressionEditor({ expression, onChange }: { expression: FilterGroup; onChange: (expression: FilterGroup) => void }) {
  const nodeCount = filterExpressionNodeCount(expression);
  return <GroupEditor
    group={expression}
    depth={1}
    nodeCount={nodeCount}
    onChange={onChange}
  />;
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
    if (!canAddCondition) return;
    onChange({ ...group, children: [...group.children, emptyCondition()] });
  }

  function addGroup() {
    if (!canAddGroup) return;
    onChange({
      ...group,
      children: [...group.children, { kind: "group", operator: "and", children: [emptyCondition()] }],
    });
  }

  return <fieldset className={depth === 1 ? "forms-expression-group root" : "forms-expression-group"}>
    <legend className="sr-only">{depth === 1 ? "Advanced filter logic" : "Filter group"}</legend>
    <div className="forms-expression-group-heading">
      <label>
        <span>{depth === 1 ? "Match" : "Group"}</span>
        <select
          aria-label={depth === 1 ? "Advanced filter match mode" : `Filter group level ${depth} match mode`}
          value={group.operator}
          onChange={(event) => onChange({ ...group, operator: event.target.value as "and" | "or" })}
        >
          <option value="and">All conditions</option>
          <option value="or">Any condition</option>
        </select>
      </label>
      {onRemove && <button type="button" className="text-button" onClick={onRemove}>Remove group</button>}
    </div>

    <div className="forms-expression-children">
      {group.children.map((child, index) => child.kind === "condition"
        ? <ConditionEditor
            key={`${depth}:condition:${index}`}
            condition={child}
            index={index}
            onChange={(next) => updateChild(index, next)}
            onRemove={() => removeChild(index)}
          />
        : <GroupEditor
            key={`${depth}:group:${index}`}
            group={child}
            depth={depth + 1}
            nodeCount={nodeCount}
            onChange={(next) => updateChild(index, next)}
            onRemove={() => removeChild(index)}
          />)}
      {group.children.length === 0 && <p className="forms-expression-empty">Add a condition to define this group.</p>}
    </div>

    <div className="forms-expression-actions">
      <button type="button" className="secondary-button" disabled={!canAddCondition} onClick={addCondition}>+ Condition</button>
      {depth < MAX_DEPTH && <button type="button" className="text-button" disabled={!canAddGroup} onClick={addGroup}>+ Group</button>}
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

  function chooseField(field: FilterField) {
    onChange({ kind: "condition", field, operator: "is", value: "" });
  }

  return <div className="forms-expression-condition">
    <select aria-label={`Condition ${index + 1} field`} value={condition.field} onChange={(event) => chooseField(event.target.value as FilterField)}>
      {formFilterRegistry.map((item) => <option value={item.field} key={item.field}>{item.label}</option>)}
    </select>
    <span className="forms-expression-operator">is</span>
    {definition.input === "select" ? <select
      aria-label={`Condition ${index + 1} ${definition.label} value`}
      value={condition.value}
      onChange={(event) => onChange({ ...condition, value: event.target.value })}
    >
      <option value="">Choose {definition.label.toLowerCase()}</option>
      {definition.options?.map((option) => <option value={option.value} key={option.value}>{option.label}</option>)}
    </select> : <input
      aria-label={`Condition ${index + 1} ${definition.label} value`}
      value={condition.value}
      maxLength={200}
      placeholder={definition.placeholder}
      onChange={(event) => onChange({ ...condition, value: event.target.value })}
    />}
    <button type="button" className="icon-button" aria-label={`Remove condition ${index + 1}`} onClick={onRemove}>×</button>
  </div>;
}

function emptyCondition(): FilterCondition {
  return { kind: "condition", field: "status", operator: "is", value: "" };
}
