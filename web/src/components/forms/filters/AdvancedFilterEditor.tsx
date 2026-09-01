import { useState } from "react";
import type { FilterExpression, FilterGroup } from "./filterModel";
import { filterExpressionNodeCount, isServerSerializableExpression } from "./filterModel";
import { Button, FocusedSheet } from "../../ui";
import { FilterExpressionEditor } from "./FilterExpressionEditor";

type Props = {
  expression?: FilterExpression;
  onApply: (expression?: FilterExpression) => void;
  onClose: () => void;
};

export function AdvancedFilterEditor({ expression, onApply, onClose }: Props) {
  const [draft, setDraft] = useState<FilterGroup>(() => asEditableRoot(expression));
  const nodes = filterExpressionNodeCount(draft);
  const valid = isServerSerializableExpression(draft);

  return <FocusedSheet
    label="Advanced form filters"
    closeLabel="Close advanced filters"
    size="wide"
    onClose={onClose}
  >
    <div className="forms-advanced-filter-content">
      <header className="forms-advanced-filter-heading">
        <div>
          <span className="eyebrow">Advanced filters</span>
          <h2>Combine exact conditions</h2>
          <p>Use only governed form fields. Groups are bounded to 12 nodes and 3 levels.</p>
        </div>
        <span className="forms-expression-budget" aria-label={`${nodes} of 12 filter nodes used`}>{nodes}/12</span>
      </header>

      <FilterExpressionEditor expression={draft} onChange={setDraft}/>

      <footer className="forms-advanced-filter-footer">
        <Button variant="quiet" isDisabled={!expression} onPress={() => onApply(undefined)}>Clear advanced</Button>
        <div>
          <Button onPress={onClose}>Cancel</Button>
          <Button variant="primary" isDisabled={!valid} onPress={() => onApply(draft)}>Apply filters</Button>
        </div>
      </footer>
    </div>
  </FocusedSheet>;
}

function asEditableRoot(expression?: FilterExpression): FilterGroup {
  if (!expression) {
    return {
      kind: "group",
      operator: "and",
      children: [{ kind: "condition", field: "status", operator: "is", value: "" }],
    };
  }
  if (expression.kind === "group") return cloneGroup(expression);
  return { kind: "group", operator: "and", children: [{ ...expression }] };
}

function cloneGroup(group: FilterGroup): FilterGroup {
  return {
    ...group,
    children: group.children.map((child) => child.kind === "condition" ? { ...child } : cloneGroup(child)),
  };
}
