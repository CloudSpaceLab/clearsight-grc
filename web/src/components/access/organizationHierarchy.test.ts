import { describe, expect, it } from "vitest";
import type { OrganizationScope } from "../../identityAccessApi";
import { filterOrganizationHierarchy, organizationHierarchy, scopeKindLabel } from "./organizationHierarchy";

function scope(id: string, overrides: Partial<OrganizationScope> = {}): OrganizationScope {
  return {
    id,
    legal_entity_id: "entity-a",
    code: id.toUpperCase(),
    name: id,
    kind: "DEPARTMENT",
    department_path: [id.toUpperCase()],
    origin: "MANAGED",
    status: "ACTIVE",
    valid_from: "2026-01-01T00:00:00Z",
    version: 1,
    ...overrides,
  };
}

describe("organization hierarchy", () => {
  it("uses exact parent IDs despite duplicate names, codes and paths, and preserves a missing parent", () => {
    const scopes = [
      scope("org-1", { name: "Operations", code: "OPS", department_path: ["BANK", "OPS"] }),
      scope("org-10", { name: "Operations", code: "OPS", department_path: ["BANK", "OPS"] }),
      scope("child", { parent_scope_id: "org-10", department_path: ["BANK", "OPS", "CHILD"] }),
      scope("orphan", { parent_scope_id: "org", department_path: ["BANK", "OPS", "ORPHAN"] }),
      scope("orphan-child", { parent_scope_id: "orphan" }),
    ];
    const rows = organizationHierarchy(scopes);
    const byID = new Map(rows.map((row) => [row.scope.id, row]));

    expect(rows).toHaveLength(scopes.length);
    expect(new Set(rows.map((row) => row.scope.id)).size).toBe(scopes.length);
    expect(byID.get("child")).toMatchObject({ depth: 1, ancestors: ["org-10"], issue: undefined });
    expect(byID.get("org-1")?.childCount).toBe(0);
    expect(byID.get("org-10")?.childCount).toBe(1);
    expect(byID.get("orphan")).toMatchObject({ depth: 0, ancestors: [], childCount: 1, issue: "missing-parent" });
    expect(byID.get("orphan")?.scope.parent_scope_id).toBe("org");
    expect(byID.get("orphan-child")).toMatchObject({ depth: 1, ancestors: ["orphan"], issue: undefined });
    expect(organizationHierarchy([...scopes].reverse()).map((row) => row.scope.id)).toEqual(rows.map((row) => row.scope.id));
  });

  it("exposes self and multi-node cycles exactly once, preserves real edges, and marks only cycle members", () => {
    const scopes = [
      scope("tail", { parent_scope_id: "b" }),
      scope("b", { parent_scope_id: "c" }),
      scope("self", { parent_scope_id: "self" }),
      scope("a", { parent_scope_id: "b" }),
      scope("c", { parent_scope_id: "a" }),
    ];
    const before = structuredClone(scopes);
    const rows = organizationHierarchy(scopes);
    const byID = new Map(rows.map((row) => [row.scope.id, row]));

    expect(rows.map((row) => row.scope.id).sort()).toEqual(["a", "b", "c", "self", "tail"]);
    expect(byID.get("a")).toMatchObject({ depth: 0, ancestors: [], issue: "cycle", childCount: 1 });
    expect(byID.get("c")).toMatchObject({ ancestors: ["a"], issue: "cycle" });
    expect(byID.get("b")).toMatchObject({ ancestors: ["a", "c"], issue: "cycle" });
    expect(byID.get("tail")).toMatchObject({ ancestors: ["a", "c", "b"], issue: undefined });
    expect(byID.get("self")).toMatchObject({ depth: 0, ancestors: [], issue: "cycle", childCount: 0 });
    for (const row of rows) {
      expect(row.ancestors).not.toContain(row.scope.id);
      if (row.depth > 0) expect(row.ancestors.at(-1)).toBe(row.scope.parent_scope_id);
    }
    expect(scopes).toEqual(before);
    expect(organizationHierarchy([...scopes].reverse())).toEqual(rows);
  });

  const searchScopes = [
    scope("root", { name: "Head Office", code: "BANK", department_path: ["BANK"], kind: "BUSINESS_UNIT" }),
    scope("tech", { name: "Technology", code: "TECH", parent_scope_id: "root", department_path: ["BANK", "TECH"] }),
    scope("security", { name: "Information Security", code: "SEC", parent_scope_id: "tech", department_path: ["BANK", "TECH", "SEC"], kind: "FUNCTION" }),
    scope("infra", { name: "Infrastructure", code: "INFRA", parent_scope_id: "tech", department_path: ["BANK", "TECH", "INFRA"], kind: "FUNCTION" }),
    scope("finance", { name: "Finance", code: "FIN", parent_scope_id: "root", department_path: ["BANK", "FIN"] }),
  ];

  it("collapses descendants while keeping the loaded population count and restores matching ancestors during search", () => {
    const rows = organizationHierarchy(searchScopes);
    const collapsed = new Set(["tech"]);
    const hidden = filterOrganizationHierarchy(rows, "  ", collapsed);
    expect(hidden.rows.map((row) => row.scope.id)).toEqual(["root", "finance", "tech"]);
    expect(hidden.matches).toBe(5);

    const search = filterOrganizationHierarchy(rows, "  sEcUrItY  ", new Set(["root", "tech"]));
    expect(search.rows.map((row) => row.scope.id)).toEqual(["root", "tech", "security"]);
    expect(search.matches).toBe(1);
    expect(search.rows.find((row) => row.scope.id === "tech")?.childCount).toBe(2);
    expect(filterOrganizationHierarchy(rows, "", collapsed)).toEqual(hidden);
    expect(filterOrganizationHierarchy(rows, "no matching area", collapsed)).toEqual({ rows: [], matches: 0 });
  });

  it("counts name, code, path and human-readable kind matches without counting retained ancestors", () => {
    const rows = organizationHierarchy(searchScopes);
    const pathMatches = filterOrganizationHierarchy(rows, "tech", new Set());
    expect(pathMatches.matches).toBe(3);
    expect(pathMatches.rows.map((row) => row.scope.id)).toEqual(["root", "tech", "infra", "security"]);
    const kindMatches = filterOrganizationHierarchy(rows, "function", new Set());
    expect(kindMatches.matches).toBe(2);
    expect(kindMatches.rows.map((row) => row.scope.id)).toEqual(["root", "tech", "infra", "security"]);
    expect(filterOrganizationHierarchy(rows, "business unit", new Set()).matches).toBe(1);
    expect(filterOrganizationHierarchy(rows, "FIN", new Set()).rows.map((row) => row.scope.id)).toEqual(["root", "finance"]);
  });

  it.each([false, true])("keeps a 500-area deep chain complete and searchable with cyclic=%s", (cyclic) => {
    const ids = Array.from({ length: 500 }, (_, index) => `area-${String(index).padStart(3, "0")}`);
    const scopes = ids.map((id, index) => scope(id, {
      parent_scope_id: index ? ids[index - 1] : cyclic ? ids.at(-1) : undefined,
      name: index === 499 ? "Deep review leaf" : `Area ${index}`,
    })).reverse();
    for (const item of scopes) { Object.freeze(item.department_path); Object.freeze(item); }
    Object.freeze(scopes);
    const rows = organizationHierarchy(scopes);

    expect(rows.map((row) => row.scope.id)).toEqual(ids);
    expect(rows.at(-1)).toMatchObject({ depth: 499, ancestors: ids.slice(0, -1), childCount: 0 });
    expect(rows.filter((row) => row.issue === "cycle")).toHaveLength(cyclic ? 500 : 0);
    expect(rows.slice(0, -1).every((row) => row.childCount === 1)).toBe(true);
    const collapsed = new Set([ids[250]!]);
    expect(filterOrganizationHierarchy(rows, "", collapsed).rows).toHaveLength(251);
    const search = filterOrganizationHierarchy(rows, "deep review leaf", collapsed);
    expect(search.matches).toBe(1);
    expect(search.rows.map((row) => row.scope.id)).toEqual(ids);
  });

  it("handles an empty returned population and humanizes imported and managed types", () => {
    expect(organizationHierarchy([])).toEqual([]);
    expect(filterOrganizationHierarchy([], "risk", new Set())).toEqual({ rows: [], matches: 0 });
    expect(scopeKindLabel("ORGANIZATION_UNIT")).toBe("Organization Unit");
    expect(scopeKindLabel("CRITICAL_SERVICE")).toBe("Critical Service");
  });
});
