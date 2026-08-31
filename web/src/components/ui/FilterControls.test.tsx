import axe from "axe-core";
import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { Button, FilterChip, PopoverDialog, ScopeBar, SearchField } from "./index";

describe("shared filter controls", () => {
  it("keeps search labelled while presenting the compact search treatment", () => {
    const onChange = vi.fn();
    render(<SearchField label="Search templates" value="vendor" placeholder="Search forms…" onChange={onChange}/>);

    const search = screen.getByRole("searchbox", { name: "Search templates" });
    expect((search as HTMLInputElement).value).toBe("vendor");
    fireEvent.change(search, { target: { value: "policy" } });
    expect(onChange).toHaveBeenCalledWith("policy");
  });

  it("names removable filters from their business field and value", () => {
    const onRemove = vi.fn();
    render(<FilterChip label="Status" value="Active" onRemove={onRemove}/>);

    const chip = screen.getByRole("button", { name: "Remove Status filter" });
    expect(chip.textContent).toContain("Active");
    fireEvent.click(chip);
    expect(onRemove).toHaveBeenCalledOnce();
  });

  it("exposes one selected lifecycle scope and authoritative counts", () => {
    const onSelectionChange = vi.fn();
    render(<ScopeBar
      ariaLabel="Form status scopes"
      selectedKey="ACTIVE"
      items={[{ id: "ALL", label: "All", count: 8 }, { id: "ACTIVE", label: "Active", count: 5 }]}
      onSelectionChange={onSelectionChange}
    />);

    expect(screen.getByRole("button", { name: "Active 5" }).getAttribute("aria-current")).toBe("page");
    fireEvent.click(screen.getByRole("button", { name: "All 8" }));
    expect(onSelectionChange).toHaveBeenCalledWith("ALL");
  });

  it("contains focusable filter work in a dismissable anchored dialog", async () => {
    function Harness() {
      const [open, setOpen] = useState(false);
      return <PopoverDialog
        label="Add filter"
        isOpen={open}
        onOpenChange={setOpen}
        trigger={<Button>Filter</Button>}
      >
        <Button>Choose status</Button>
      </PopoverDialog>;
    }
    render(<Harness/>);

    fireEvent.click(screen.getByRole("button", { name: "Filter" }));
    expect(await screen.findByRole("dialog", { name: "Add filter" })).toBeTruthy();
    const results = await axe.run(document.body, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);

    fireEvent.keyDown(screen.getByRole("dialog", { name: "Add filter" }), { key: "Escape" });
    expect(screen.queryByRole("dialog", { name: "Add filter" })).toBeNull();
  });
});
