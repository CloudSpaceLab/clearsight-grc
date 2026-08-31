import { describe, expect, it } from "vitest";
import type { FormTemplateQuery } from "../../../formsTypes";
import {
  activeFilterConditions,
  filterExpressionNodeCount,
  filterExpressionToQuery,
  isServerSerializableExpression,
  parseFilterExpression,
  queryToFilterExpression,
  removeFilter,
  serializeFilterExpression,
  setFilter,
  stripFilterField,
  withStatusScope,
} from "./filterModel";

describe("Forms filter model", () => {
  it("round-trips the server-supported quick query dimensions without duplicating search state", () => {
    const query: FormTemplateQuery = {
      search: "vendor",
      status: "ACTIVE",
      owner: "principal-1",
      tag: "third-party",
      limit: 50,
    };

    const expression = queryToFilterExpression(query);
    expect(expression.operator).toBe("and");
    expect(activeFilterConditions(query)).toEqual([
      { kind: "condition", field: "status", operator: "is", value: "ACTIVE" },
      { kind: "condition", field: "owner", operator: "is", value: "principal-1" },
      { kind: "condition", field: "tag", operator: "is", value: "third-party" },
    ]);
    expect(filterExpressionToQuery(expression, query)).toEqual({
      status: "ACTIVE",
      owner: "principal-1",
      tag: "third-party",
      limit: 50,
    });
  });

  it("adds and removes one typed quick filter without disturbing the rest of the query", () => {
    const query: FormTemplateQuery = { search: "vendor", status: "DRAFT", limit: 25 };
    const withOwner = setFilter(query, { kind: "condition", field: "owner", operator: "is", value: "owner-1" });
    expect(withOwner).toMatchObject({ search: "vendor", status: "DRAFT", owner: "owner-1", limit: 25 });
    expect(removeFilter(withOwner, "status")).toMatchObject({ search: "vendor", owner: "owner-1", limit: 25, status: undefined });
  });

  it("accepts bounded ALL/ANY groups and rejects unsupported or oversized expressions", () => {
    const expression = {
      kind: "group" as const,
      operator: "or" as const,
      children: [
        { kind: "condition" as const, field: "status" as const, operator: "is" as const, value: "ACTIVE" },
        { kind: "condition" as const, field: "status" as const, operator: "is" as const, value: "DRAFT" },
      ],
    };
    expect(isServerSerializableExpression(expression)).toBe(true);
    expect(filterExpressionNodeCount(expression)).toBe(3);
    expect(parseFilterExpression(serializeFilterExpression(expression))).toEqual(expression);

    const oversized = {
      kind: "group" as const,
      operator: "or" as const,
      children: Array.from({ length: 12 }, () => ({ kind: "condition" as const, field: "tag" as const, operator: "is" as const, value: "priority" })),
    };
    expect(isServerSerializableExpression(oversized)).toBe(false);
    expect(isServerSerializableExpression({ kind: "condition", field: "tag", operator: "contains", value: "priority" } as never)).toBe(false);
  });

  it("moves status scope out of advanced logic so authoritative facet counts stay exact", () => {
    const filter = {
      kind: "group" as const,
      operator: "and" as const,
      children: [
        { kind: "condition" as const, field: "status" as const, operator: "is" as const, value: "ACTIVE" },
        { kind: "condition" as const, field: "tag" as const, operator: "is" as const, value: "third-party" },
      ],
    };
    expect(stripFilterField(filter, "status")).toEqual({ kind: "condition", field: "tag", operator: "is", value: "third-party" });
    expect(withStatusScope({ filter, limit: 25 }, "DRAFT")).toEqual({
      status: "DRAFT",
      filter: { kind: "condition", field: "tag", operator: "is", value: "third-party" },
      limit: 25,
      cursor: undefined,
    });
  });
});
