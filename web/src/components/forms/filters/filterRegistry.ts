import type { LifecycleStatus } from "../../../monitoringTypes";
import type { FilterField, FilterOperator } from "./filterModel";

export type FilterOption = { value: string; label: string };
export type FilterDefinition = {
  field: FilterField;
  label: string;
  category: "Common" | "Form";
  operator: FilterOperator;
  input: "select" | "text";
  placeholder?: string;
  options?: readonly FilterOption[];
  formatValue?: (value: string) => string;
};

const statusOptions: readonly FilterOption[] = [
  ["DRAFT", "Draft"],
  ["PENDING_APPROVAL", "In review"],
  ["ACTIVE", "Active"],
  ["PAUSED", "Paused"],
  ["RETIRED", "Retired"],
  ["REJECTED", "Rejected"],
].map(([value, label]) => ({ value: value as LifecycleStatus, label }));

export const formFilterRegistry: readonly FilterDefinition[] = [
  {
    field: "status",
    label: "Status",
    category: "Common",
    operator: "is",
    input: "select",
    options: statusOptions,
    formatValue: (value) => statusOptions.find((option) => option.value === value)?.label ?? value,
  },
  {
    field: "owner",
    label: "Owner",
    category: "Common",
    operator: "is",
    input: "text",
    placeholder: "Owner or principal ID",
  },
  {
    field: "program",
    label: "Program",
    category: "Common",
    operator: "is",
    input: "text",
    placeholder: "Program ID",
  },
  {
    field: "use",
    label: "Approved use",
    category: "Form",
    operator: "is",
    input: "text",
    placeholder: "e.g. vendor due diligence",
  },
  {
    field: "tag",
    label: "Tag",
    category: "Form",
    operator: "is",
    input: "text",
    placeholder: "e.g. third-party",
  },
] as const;

export function filterDefinition(field: FilterField): FilterDefinition {
  const definition = formFilterRegistry.find((item) => item.field === field);
  if (!definition) throw new Error(`Unsupported Forms filter field: ${field}`);
  return definition;
}

export function formatFilterValue(field: FilterField, value: string): string {
  const definition = filterDefinition(field);
  return definition.formatValue?.(value) ?? value;
}
