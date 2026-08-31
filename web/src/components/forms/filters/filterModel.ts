import type {
  FormFilterCondition,
  FormFilterExpression,
  FormFilterField,
  FormFilterGroup,
  FormTemplateQuery,
} from "../../../formsTypes";
import type { LifecycleStatus } from "../../../monitoringTypes";

export type FilterField = FormFilterField;
export type FilterOperator = "is";
export type FilterCondition = FormFilterCondition;
export type FilterGroup = FormFilterGroup;
export type FilterExpression = FormFilterExpression;

const queryFields: FilterField[] = ["status", "owner", "program", "use", "tag"];
const fieldSet = new Set<FilterField>(queryFields);
const lifecycleStatuses = new Set<LifecycleStatus>(["DRAFT", "PENDING_APPROVAL", "ACTIVE", "PAUSED", "REJECTED", "RETIRED"]);
const maxNodes = 12;
const maxDepth = 3;

export function queryToFilterExpression(query: FormTemplateQuery): FilterGroup {
  return {
    kind: "group",
    operator: "and",
    children: queryFields.flatMap((field) => {
      const value = query[field]?.trim();
      return value ? [{ kind: "condition", field, operator: "is", value } satisfies FilterCondition] : [];
    }),
  };
}

export function filterExpressionToQuery(expression: FilterExpression, base: FormTemplateQuery = {}): FormTemplateQuery {
  const next: FormTemplateQuery = { limit: base.limit };
  const conditions = flattenQuickConditions(expression);
  for (const condition of conditions) next[condition.field] = condition.value as never;
  return next;
}

export function activeFilterConditions(query: FormTemplateQuery): FilterCondition[] {
  return queryToFilterExpression(query).children as FilterCondition[];
}

export function removeFilter(query: FormTemplateQuery, field: FilterField): FormTemplateQuery {
  return { ...query, [field]: undefined, cursor: undefined };
}

export function setFilter(query: FormTemplateQuery, condition: FilterCondition): FormTemplateQuery {
  return { ...query, [condition.field]: condition.value, cursor: undefined };
}

export function setAdvancedFilter(query: FormTemplateQuery, expression?: FilterExpression): FormTemplateQuery {
  return { ...query, filter: expression, cursor: undefined };
}

export function isServerSerializableExpression(expression: FilterExpression): boolean {
  return validateExpression(expression).valid;
}

export function filterExpressionNodeCount(expression?: FilterExpression): number {
  if (!expression) return 0;
  return expression.kind === "condition"
    ? 1
    : 1 + expression.children.reduce((total, child) => total + filterExpressionNodeCount(child), 0);
}

export function serializeFilterExpression(expression?: FilterExpression): string | undefined {
  if (!expression || !isServerSerializableExpression(expression)) return undefined;
  return JSON.stringify(expression);
}

export function parseFilterExpression(raw?: string | null): FilterExpression | undefined {
  if (!raw?.trim()) return undefined;
  try {
    const value = JSON.parse(raw) as FilterExpression;
    return isServerSerializableExpression(value) ? value : undefined;
  } catch {
    return undefined;
  }
}

export function stripFilterField(expression: FilterExpression | undefined, field: FilterField): FilterExpression | undefined {
  if (!expression) return undefined;
  if (expression.kind === "condition") return expression.field === field ? undefined : { ...expression };

  const children = expression.children.flatMap((child) => {
    const stripped = stripFilterField(child, field);
    return stripped ? [stripped] : [];
  });
  if (children.length === 0) return undefined;
  if (children.length === 1) return children[0];
  return { ...expression, children };
}

export function withStatusScope(query: FormTemplateQuery, status?: LifecycleStatus): FormTemplateQuery {
  return {
    ...query,
    status,
    filter: stripFilterField(query.filter, "status"),
    cursor: undefined,
  };
}

export function filterExpressionSummary(expression?: FilterExpression): string | undefined {
  if (!expression) return undefined;
  const conditions = collectConditions(expression);
  if (conditions.length === 0) return undefined;
  const join = expression.kind === "group" && expression.operator === "or" ? "any" : "all";
  return `${conditions.length} advanced ${conditions.length === 1 ? "condition" : "conditions"} · ${join}`;
}

function validateExpression(expression: FilterExpression, depth = 1, state = { nodes: 0 }): { valid: boolean } {
  if (!expression || typeof expression !== "object") return { valid: false };
  state.nodes += 1;
  if (state.nodes > maxNodes || depth > maxDepth) return { valid: false };

  if (expression.kind === "condition") {
    if (!fieldSet.has(expression.field) || expression.operator !== "is" || typeof expression.value !== "string") return { valid: false };
    const value = expression.value.trim();
    if (!value || value.length > 200) return { valid: false };
    if (expression.field === "status" && !lifecycleStatuses.has(value.toUpperCase() as LifecycleStatus)) return { valid: false };
    return { valid: true };
  }

  if (expression.kind !== "group" || (expression.operator !== "and" && expression.operator !== "or") || !Array.isArray(expression.children) || expression.children.length === 0) {
    return { valid: false };
  }
  for (const child of expression.children) {
    if (!validateExpression(child, depth + 1, state).valid) return { valid: false };
  }
  return { valid: true };
}

function collectConditions(expression: FilterExpression): FilterCondition[] {
  if (expression.kind === "condition") return [expression];
  return expression.children.flatMap(collectConditions);
}

function flattenQuickConditions(expression: FilterExpression): FilterCondition[] {
  if (expression.kind === "condition") return [expression];
  if (expression.operator !== "and") throw new Error("Quick Forms filters support all-match conditions only.");
  if (expression.children.length > 5) throw new Error("Quick Forms filters are limited to the five supported fields.");

  const conditions: FilterCondition[] = [];
  const seen = new Set<FilterField>();
  for (const child of expression.children) {
    if (child.kind !== "condition") throw new Error("Grouped logic belongs in Advanced filters.");
    if (seen.has(child.field)) throw new Error(`Quick Forms filter field ${child.field} can only appear once.`);
    if (!child.value.trim()) continue;
    seen.add(child.field);
    conditions.push({ ...child, value: child.value.trim() });
  }
  return conditions;
}
