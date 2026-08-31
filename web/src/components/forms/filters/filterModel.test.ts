import { describe, expect, it } from "vitest";
import type { FormTemplateQuery } from "../../../formsTypes";
import {
  activeFilterConditions,
  filterExpressionToQuery,
  isServerSerializableExpression,
  queryToFilterExpression,
  removeFilter,
  setFilter,
} from "./filterModel";

describe("Forms filter model", () => {
  it("round-trips the server-supported query dimensions without duplicating search state", () => {
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

  it("adds and removes one typed filter without disturbing the rest of the query", () => {
    const query: FormTemplateQuery = { search: "vendor", status: "DRAFT", limit: 25 };
    const withOwner = setFilter(query, { kind: "condition", field: "owner", operator: "is", value: "owner-1" });
    expect(withOwner).toMatchObject({ search: "vendor", status: "DRAFT", owner: "owner-1", limit: 25 });
    expect(removeFilter(withOwner, "status")).toMatchObject({ search: "vendor", owner: "owner-1", limit: 25, status: undefined });
  });

  it("refuses OR or nested expressions until the server supports them", () => {
    expect(isServerSerializableExpression({
      kind: "group",
      operator: "or",
      children: [
        { kind: "condition", field: "status", operator: "is", value: "ACTIVE" },
        { kind: "condition", field: "status", operator: "is", value: "DRAFT" },
      ],
    })).toBe(false);
  });
});
