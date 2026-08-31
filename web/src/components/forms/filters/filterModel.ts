import type { FormTemplateQuery } from "../../../formsTypes";

export type FilterField = "status" | "owner" | "program" | "use" | "tag";
export type FilterOperator = "is";

export type FilterCondition = {
  kind: "condition";
  field: FilterField;
  operator: FilterOperator;
  value: string;
};

export type FilterGroup = {
  kind: "group";
  operator: "and" | "or";
  children: FilterExpression[];
};

export type FilterExpression = FilterCondition | FilterGroup;

const queryFields: FilterField[] = ["status", "owner", "program", "use", "tag"];

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
  const conditions = flattenServerConditions(expression);
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

export function isServerSerializableExpression(expression: FilterExpression): boolean {
  try {
    flattenServerConditions(expression);
    return true;
  } catch {
    return false;
  }
}

function flattenServerConditions(expression: FilterExpression): FilterCondition[] {
  if (expression.kind === "condition") return [expression];
  if (expression.operator !== "and") throw new Error("The current Forms API supports all-match filters only.");
  if (expression.children.length > 12) throw new Error("Forms filter expressions are limited to 12 conditions.");

  const conditions: FilterCondition[] = [];
  const seen = new Set<FilterField>();
  for (const child of expression.children) {
    if (child.kind !== "condition") throw new Error("Nested Forms filter groups are not yet server-supported.");
    if (seen.has(child.field)) throw new Error(`Forms filter field ${child.field} can only appear once.`);
    if (!child.value.trim()) continue;
    seen.add(child.field);
    conditions.push({ ...child, value: child.value.trim() });
  }
  return conditions;
}
