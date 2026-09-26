import { describe, expect, it, vi } from "vitest";
import type { ReportDefinition } from "../../reportingTypes";
import { buildReportSetupInput, reportSetupArea, reportSetupFocus } from "./reportTemplatePresets";

function definition(overrides: Partial<ReportDefinition> = {}): ReportDefinition {
  return {
    id: "definition-1",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "REPORT_1",
    name: "Report",
    description: "",
    dataset: "VENDORS",
    scope_kind: "LEGAL_ENTITY",
    format: "XLSX",
    filter: { kind: "group", operator: "and", children: [] },
    status: "DRAFT",
    current_version: 1,
    effective: false,
    checksum: "a".repeat(64),
    maker_id: "maker-1",
    created_at: "2026-09-25T08:00:00Z",
    updated_at: "2026-09-25T08:00:00Z",
    version: 1,
    ...overrides,
  };
}

describe("report setup presets", () => {
  it("maps business overview choices to the governed low-level report contract", () => {
    vi.spyOn(globalThis.crypto, "getRandomValues").mockImplementation((array) => {
      (array as Uint8Array).set([0x12, 0x34, 0x56, 0x78]);
      return array;
    });

    expect(buildReportSetupInput("Vendor overview", "VENDORS", "OVERVIEW")).toEqual({
      code: "VENDORS_OVERVIEW_12345678",
      name: "Vendor overview",
      description: "Current vendors overview for the legal entity.",
      dataset: "VENDORS",
      scope_kind: "LEGAL_ENTITY",
      format: "XLSX",
      filter: { kind: "group", operator: "and", children: [] },
    });

    vi.restoreAllMocks();
  });

  it("uses exception datasets where the platform already has a canonical outstanding population", () => {
    expect(buildReportSetupInput("Outstanding work", "WORK", "ATTENTION")).toMatchObject({
      dataset: "MATTER_EXCEPTIONS",
      filter: { kind: "group", operator: "and", children: [] },
    });
    expect(buildReportSetupInput("Privacy exceptions", "PROCESSING", "ATTENTION")).toMatchObject({
      dataset: "PROCESSING_ACTIVITY_EXCEPTIONS",
    });
  });

  it("builds attention filters for vendors and Programs instead of exposing query mechanics", () => {
    const vendor = buildReportSetupInput("Vendor attention", "VENDORS", "ATTENTION");
    expect(vendor.filter).toMatchObject({ kind: "group", operator: "or" });
    expect(vendor.filter?.children?.map((condition) => condition.value)).toEqual([
      "UNDER_REVIEW",
      "RESTRICTED",
      "SUSPENDED",
      "EXITING",
    ]);

    const program = buildReportSetupInput("Program attention", "PROGRAMS", "ATTENTION");
    expect(program.filter?.children?.some((condition) => condition.value === "OVERDUE")).toBe(true);
    expect(program.filter?.children?.some((condition) => condition.value === "GAP_IDENTIFIED")).toBe(true);
  });

  it("recognizes simple and legacy custom setups for display", () => {
    expect(reportSetupArea(definition({ dataset: "MATTER_EXCEPTIONS" }))).toBe("WORK");
    expect(reportSetupFocus(definition({ dataset: "MATTER_EXCEPTIONS" }))).toBe("ATTENTION");
    expect(reportSetupFocus(definition({
      dataset: "VENDORS",
      filter: { kind: "condition", field: "criticality", operator: "is", value: "CRITICAL" },
    }))).toBe("CUSTOM");
  });
});
