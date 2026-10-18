import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramSection } from "../appRouting";
import { ProgramDetailSections, programSections } from "./ProgramDetailSections";

const panels = Object.fromEntries(programSections.map((section) => [section.id, <p key={section.id}>{section.label} content</p>])) as Record<ProgramSection, ReactNode>;

function Harness({ onChange = vi.fn() }: { onChange?: (section: ProgramSection) => void }) {
  const [section, setSection] = useState<ProgramSection>("overview");
  return <ProgramDetailSections section={section} panels={panels} onSectionChange={(next) => { setSection(next); onChange(next); }}/>;
}

describe("Program detail sections", () => {
  it("uses the shared keyboard tab contract with one selected panel and indicator", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange}/>);
    expect(screen.getAllByRole("tab")).toHaveLength(6);
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1);
    expect(screen.getByRole("tabpanel", { name: "Overview" }).textContent).toContain("Overview content");
    expect(screen.queryByRole("heading", { name: "Overview" })).toBeNull();

    const overview = screen.getByRole("tab", { name: "Overview" });
    overview.focus();
    fireEvent.keyDown(overview, { key: "ArrowRight" });
    expect(onChange).toHaveBeenLastCalledWith("requirements-controls");
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Requirements" }));
    fireEvent.keyDown(document.activeElement!, { key: "End" });
    expect(onChange).toHaveBeenLastCalledWith("history");
    fireEvent.keyDown(document.activeElement!, { key: "Home" });
    expect(onChange).toHaveBeenLastCalledWith("overview");
    expect(document.querySelectorAll(".cs-tabs__indicator")).toHaveLength(1);
  });

  it("uses the shared compact selector without mounting duplicate program panels", async () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange}/>);
    // CSS hides the desktop tablist at compact widths; the same controlled
    // selector remains present and accessible at every width.
    const trigger = screen.getByRole("button", { name: "Overview Program section" });
    fireEvent.click(trigger);
    fireEvent.click(await screen.findByRole("option", { name: "Assessments" }));
    expect(onChange).toHaveBeenCalledExactlyOnceWith("assessments");
    expect(screen.getByRole("tabpanel", { name: "Assessments" }).textContent).toContain("Assessments content");
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1);
    expect(screen.queryByText("Overview content")).toBeNull();
  });

  it("keeps panel identity aligned with the selected navigation tab", () => {
    render(<Harness/>);
    fireEvent.click(screen.getByRole("tab", { name: "Evidence" }));
    const tab = screen.getByRole("tab", { name: "Evidence", selected: true });
    const panel = screen.getByRole("tabpanel", { name: "Evidence" });
    expect(document.getElementById(tab.getAttribute("aria-controls") ?? "")).toBe(panel);
    expect(panel.getAttribute("aria-labelledby")).toBe(tab.id);
  });
  it("keeps Data collection contextual and leaves exactly six approved primary tabs", () => {
    const onChange = vi.fn();
    render(<ProgramDetailSections section="monitoring"
      panels={{...panels, monitoring: <p>Scoped collection checks</p>}}
      onSectionChange={onChange}/>);
    expect(screen.queryByRole("tab", {name: "Data collection"})).toBeNull();
    expect(screen.getByRole("region", {name: "Data collection workspace"})).toBeTruthy();
    expect(screen.getByText("Scoped collection checks")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Back to Overview"}));
    expect(onChange).toHaveBeenCalledWith("overview");
  });

});
