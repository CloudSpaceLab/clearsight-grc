import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramSection } from "../appRouting";
import { ProgramDetailSections, programSections } from "./ProgramDetailSections";

const panels = Object.fromEntries(programSections.map((section) => [section.id, <p key={section.id}>{section.label} content</p>])) as Record<ProgramSection, ReactNode>;

function Harness({ compact = false, onChange = vi.fn() }: { compact?: boolean; onChange?: (section: ProgramSection) => void }) {
  const [section, setSection] = useState<ProgramSection>("overview");
  return <ProgramDetailSections section={section} compact={compact} panels={panels} onSectionChange={(next) => { setSection(next); onChange(next); }}/>;
}

describe("Program detail sections", () => {
  it("renders six keyboard tabs and one selected panel", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange}/>);
    expect(screen.getAllByRole("tab")).toHaveLength(6);
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1);
    expect(screen.getByRole("tabpanel").textContent).toContain("Overview content");

    const overview = screen.getByRole("tab", { name: "Overview" });
    overview.focus();
    fireEvent.keyDown(overview, { key: "ArrowRight" });
    expect(onChange).toHaveBeenLastCalledWith("requirements-controls");
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Requirements & controls" }));
    fireEvent.keyDown(document.activeElement!, { key: "End" });
    expect(onChange).toHaveBeenLastCalledWith("history");
    fireEvent.keyDown(document.activeElement!, { key: "Home" });
    expect(onChange).toHaveBeenLastCalledWith("overview");
    fireEvent.keyDown(document.activeElement!, { key: "ArrowLeft" });
    expect(onChange).toHaveBeenLastCalledWith("history");
  });

  it("replaces the tablist with a labelled selector in compact layouts", () => {
    const onChange = vi.fn();
    render(<Harness compact onChange={onChange}/>);
    expect(screen.queryByRole("tablist")).toBeNull();
    const selector = screen.getByRole("combobox", { name: "Program section" });
    fireEvent.change(selector, { target: { value: "monitoring" } });
    expect(onChange).toHaveBeenCalledWith("monitoring");
    expect(screen.getByRole("tabpanel").textContent).toContain("Monitoring content");
  });
});
