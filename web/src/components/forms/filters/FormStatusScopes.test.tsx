import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplateQuery } from "../../../formsTypes";
import { FormStatusScopes } from "./FormStatusScopes";

describe("FormStatusScopes", () => {
  it("uses authoritative facet counts and moves status into the exact scope", () => {
    const onChange = vi.fn();
    const query: FormTemplateQuery = {
      filter: {
        kind: "group",
        operator: "and",
        children: [
          { kind: "condition", field: "status", operator: "is", value: "ACTIVE" },
          { kind: "condition", field: "tag", operator: "is", value: "third-party" },
        ],
      },
      limit: 25,
    };

    render(<FormStatusScopes
      query={query}
      facets={{ status: { DRAFT: 3, ACTIVE: 5, PAUSED: 1 } }}
      onChange={onChange}
    />);

    expect(screen.getByRole("button", { name: "All 9" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Draft 3" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Rejected/ })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Draft 3" }));
    expect(onChange).toHaveBeenCalledWith({
      status: "DRAFT",
      filter: { kind: "condition", field: "tag", operator: "is", value: "third-party" },
      limit: 25,
      cursor: undefined,
    });
  });
});
