import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplateQuery } from "../../../formsTypes";
import { FilterBar } from "./FilterBar";

describe("Forms FilterBar", () => {
  it("adds a typed status filter from one contextual picker", async () => {
    const onChange = vi.fn();
    render(<FilterBar query={{ limit: 25 }} onChange={onChange} resultCount={8}/>);

    fireEvent.click(screen.getByRole("button", { name: "+ Filter" }));
    const picker = screen.getByRole("dialog", { name: "Add filter" });
    expect(within(picker).queryByRole("button", { name: /Owner/ })).toBeNull();
    expect(within(picker).queryByRole("button", { name: /Program/ })).toBeNull();
    fireEvent.click(within(picker).getByRole("button", { name: /Status/ }));
    fireEvent.click(within(picker).getByRole("button", { name: /Status value/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Active" }));
    fireEvent.click(within(picker).getByRole("button", { name: "Apply filter" }));

    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ status: "ACTIVE", limit: 25 }));
  });

  it("changes updated-time order without disturbing active filters", () => {
    const onChange = vi.fn();
    render(<FilterBar query={{ status: "ACTIVE", limit: 25 }} onChange={onChange}/>);
    fireEvent.click(screen.getByRole("button", { name: "Sort by oldest update" }));
    expect(onChange).toHaveBeenCalledWith({ status: "ACTIVE", limit: 25, sort: "UPDATED_ASC", cursor: undefined });
  });

  it("renders active filters as removable chips without turning search into a filter node", () => {
    const onChange = vi.fn();
    const query: FormTemplateQuery = { search: "vendor", status: "ACTIVE", owner: "principal-1", limit: 25 };
    render(<FilterBar query={query} onChange={onChange}/>);

    expect((screen.getByRole("searchbox", { name: "Search templates" }) as HTMLInputElement).value).toBe("vendor");
    expect(screen.getByRole("button", { name: "Remove Status filter" }).textContent).toContain("Active");
    expect(screen.getByRole("button", { name: "Remove Owner filter" }).textContent).toContain("Selected owner");

    fireEvent.click(screen.getByRole("button", { name: "Remove Status filter" }));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ search: "vendor", status: undefined, owner: "principal-1", limit: 25 }));
  });
});
