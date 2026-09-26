import type { ReportDataset, ReportDefinition, ReportDefinitionInput, ReportFilterExpression } from "../../reportingTypes";

export type ReportSetupArea = "VENDORS" | "PROGRAMS" | "WORK" | "PROCESSING";
export type ReportSetupFocus = "OVERVIEW" | "ATTENTION";
export type ReportSetupFocusLabel = ReportSetupFocus | "CUSTOM";

export const reportSetupAreaOptions: readonly { id: ReportSetupArea; label: string; description: string }[] = [
  { id: "VENDORS", label: "Vendors", description: "Status · services · criticality" },
  { id: "PROGRAMS", label: "Programs", description: "Health · status · open work" },
  { id: "WORK", label: "Work", description: "Issues · changes · actions" },
  { id: "PROCESSING", label: "Processing activities", description: "Inventory · privacy exceptions" },
];

export const reportSetupFocusOptions: readonly { id: ReportSetupFocus; label: string; description: string }[] = [
  { id: "OVERVIEW", label: "Overview", description: "Summary and charts" },
  { id: "ATTENTION", label: "Exceptions & outstanding", description: "Items needing action" },
];

export function reportSetupAreaLabel(area: ReportSetupArea) {
  return reportSetupAreaOptions.find((option) => option.id === area)?.label ?? area;
}

export function reportSetupFocusLabel(focus: ReportSetupFocusLabel) {
  if (focus === "CUSTOM") return "Custom";
  return reportSetupFocusOptions.find((option) => option.id === focus)?.label ?? focus;
}

export function reportSetupArea(definition: ReportDefinition): ReportSetupArea {
  switch (definition.dataset) {
    case "VENDORS": return "VENDORS";
    case "PROGRAMS": return "PROGRAMS";
    case "MATTERS":
    case "MATTER_EXCEPTIONS": return "WORK";
    default: return "PROCESSING";
  }
}

export function reportSetupFocus(definition: ReportDefinition): ReportSetupFocusLabel {
  if (definition.dataset === "MATTER_EXCEPTIONS" || definition.dataset === "PROCESSING_ACTIVITY_EXCEPTIONS") return "ATTENTION";
  if (definition.dataset === "VENDORS" && hasCondition(definition.filter, "status", vendorAttentionStatuses)) return "ATTENTION";
  if (definition.dataset === "PROGRAMS" && hasCondition(definition.filter, "overall_state", programAttentionStates)) return "ATTENTION";
  return hasConditions(definition.filter) ? "CUSTOM" : "OVERVIEW";
}

export function buildReportSetupInput(name: string, area: ReportSetupArea, focus: ReportSetupFocus): ReportDefinitionInput {
  const normalizedName = name.trim();
  return {
    code: reportSetupCode(normalizedName, area, focus),
    name: normalizedName,
    description: setupDescription(area, focus),
    dataset: setupDataset(area, focus),
    scope_kind: "LEGAL_ENTITY",
    format: "XLSX",
    filter: setupFilter(area, focus),
  };
}

const vendorAttentionStatuses = ["UNDER_REVIEW", "RESTRICTED", "SUSPENDED", "EXITING"] as const;
const programAttentionStates = [
  "AT_RISK",
  "GAP_IDENTIFIED",
  "EVIDENCE_INSUFFICIENT",
  "IMPLEMENTATION_PENDING",
  "OVERDUE",
  "UNDER_REVIEW",
  "UNKNOWN",
] as const;

function setupDataset(area: ReportSetupArea, focus: ReportSetupFocus): ReportDataset {
  if (area === "VENDORS") return "VENDORS";
  if (area === "PROGRAMS") return "PROGRAMS";
  if (area === "WORK") return focus === "ATTENTION" ? "MATTER_EXCEPTIONS" : "MATTERS";
  return focus === "ATTENTION" ? "PROCESSING_ACTIVITY_EXCEPTIONS" : "PROCESSING_ACTIVITIES";
}

function setupFilter(area: ReportSetupArea, focus: ReportSetupFocus): ReportFilterExpression {
  if (focus === "OVERVIEW" || area === "WORK" || area === "PROCESSING") return emptyFilter();
  if (area === "VENDORS") {
    return {
      kind: "group",
      operator: "or",
      children: vendorAttentionStatuses.map((value) => ({ kind: "condition", field: "status", operator: "is", value })),
    };
  }
  return {
    kind: "group",
    operator: "or",
    children: programAttentionStates.map((value) => ({ kind: "condition", field: "overall_state", operator: "is", value })),
  };
}

function emptyFilter(): ReportFilterExpression {
  return { kind: "group", operator: "and", children: [] };
}

function setupDescription(area: ReportSetupArea, focus: ReportSetupFocus) {
  const subject = reportSetupAreaLabel(area).toLowerCase();
  return focus === "ATTENTION"
    ? `Exceptions and outstanding ${subject} for the current legal entity.`
    : `Current ${subject} overview for the legal entity.`;
}

function reportSetupCode(name: string, area: ReportSetupArea, focus: ReportSetupFocus) {
  const readable = name
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
  const fallback = `${area === "PROCESSING" ? "PROCESSING" : area}_${focus}`;
  const base = readable.length >= 3 ? readable : fallback;
  const suffix = randomSuffix();
  return `${base.slice(0, 39).replace(/_+$/g, "")}_${suffix}`;
}

function randomSuffix() {
  const cryptoObject = globalThis.crypto;
  if (cryptoObject?.getRandomValues) {
    const bytes = cryptoObject.getRandomValues(new Uint8Array(4));
    return [...bytes].map((value) => value.toString(16).padStart(2, "0")).join("").toUpperCase();
  }
  return Date.now().toString(36).slice(-8).toUpperCase().padStart(8, "0");
}

function hasConditions(filter?: ReportFilterExpression): boolean {
  if (!filter) return false;
  if (filter.kind === "condition") return true;
  return (filter.children ?? []).some(hasConditions);
}

function hasCondition(filter: ReportFilterExpression | undefined, field: string, accepted: readonly string[]): boolean {
  if (!filter) return false;
  if (filter.kind === "condition") return filter.field === field && accepted.includes(filter.value ?? "");
  return (filter.children ?? []).some((child) => hasCondition(child, field, accepted));
}
