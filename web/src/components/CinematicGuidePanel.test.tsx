import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
// @ts-ignore Vitest executes this CSS source regression in Node.
import { readFileSync } from "node:fs";
import { CinematicGuidePanel, type CinematicGuideVariant } from "./CinematicGuidePanel";

const cinematicGuideCSS = readFileSync("src/cinematic-guide.css", "utf8");

const variants: Array<{
  variant: CinematicGuideVariant;
  panelName: RegExp;
  illustrationName: RegExp;
  stages: string[];
}> = [
  {
    variant: "today",
    panelName: /Today guide/i,
    illustrationName: /Today work path/i,
    stages: ["Source context", "Assigned work", "Review and authority", "Confirmed outcome"],
  },
  {
    variant: "vendors",
    panelName: /Vendor guide/i,
    illustrationName: /Vendor relationship path/i,
    stages: ["Vendor register", "Collect missing facts", "Review exceptions", "Request vendor action", "Confirm the outcome"],
  },
];

describe("CinematicGuidePanel", () => {
  it.each(variants)("renders the $variant introduction with accessible SVG and HTML meaning", ({ variant, panelName, illustrationName, stages }) => {
    render(<CinematicGuidePanel
      variant={variant}
      role={variant === "today" ? "Executive risk or compliance leader" : "Vendor relationship owner"}
      title={variant === "today" ? "Review assigned work" : "Manage vendor relationships"}
      description={variant === "today"
        ? "Use current source context to review assigned work and confirm the outcome."
        : "Record the relationship, collect missing facts and confirm the outcome after review."}
      onStart={vi.fn()}
      onSkip={vi.fn()}
    />);

    const panel = screen.getByRole("complementary", { name: panelName });
    expect(panel.getAttribute("aria-modal")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(within(panel).getByRole("heading", { name: variant === "today" ? "Review assigned work" : "Manage vendor relationships" })).toBeTruthy();
    expect(within(panel).getAllByText(/confirm(?:ed| the) outcome/i).length).toBeGreaterThan(0);
    expect(within(panel).getByText(variant === "today" ? "Guide for Executive risk or compliance leader" : "Guide for Vendor relationship owner")).toBeTruthy();

    const illustration = within(panel).getByRole("img", { name: illustrationName });
    expect(illustration.querySelector("title")?.textContent).toMatch(illustrationName);
    expect(illustration.querySelector("desc")?.textContent?.length).toBeGreaterThan(20);
    for (const stage of stages) {
      expect(within(panel).getByText(stage)).toBeTruthy();
      expect(within(illustration).queryByText(stage)).toBeNull();
    }
  });

  it("offers one dominant start action and an immediate skip action", () => {
    const onStart = vi.fn();
    const onSkip = vi.fn();
    render(<CinematicGuidePanel
      variant="vendors"
      role="Vendor relationship owner"
      title="Manage vendor relationships"
      description="Record the service and collect missing information for review."
      onStart={onStart}
      onSkip={onSkip}
    />);

    const start = screen.getByRole("button", { name: "Start guide" });
    expect(start.classList.contains("primary-button")).toBe(true);
    fireEvent.click(start);
    fireEvent.click(screen.getByRole("button", { name: "Skip for now" }));
    expect(onStart).toHaveBeenCalledOnce();
    expect(onSkip).toHaveBeenCalledOnce();
  });

  it("limits the entrance stagger to opacity and transform and removes it for reduced motion", () => {
    expect(cinematicGuideCSS).toMatch(/@media\s*\(prefers-reduced-motion:\s*no-preference\)/);
    expect(cinematicGuideCSS).toMatch(/@media\s*\(prefers-reduced-motion:\s*reduce\)[\s\S]*animation:\s*none\s*!important/);

    const durations = [...cinematicGuideCSS.matchAll(/(?:animation|transition)[^;]*?(\d+)ms/g)].map((match) => Number(match[1]));
    expect(durations.length).toBeGreaterThan(0);
    expect(Math.max(...durations)).toBeLessThanOrEqual(400);

    const keyframes = cinematicGuideCSS.match(/@keyframes[^@]+/g) ?? [];
    expect(keyframes.length).toBeGreaterThan(0);
    for (const rule of keyframes) {
      expect(rule).not.toMatch(/\b(?:width|height|inset|top|right|bottom|left|filter|clip-path)\s*:/);
      const properties = [...rule.matchAll(/^\s*([a-z-]+)\s*:/gm)].map((match) => match[1]);
      expect(properties.every((property) => property === "opacity" || property === "transform")).toBe(true);
    }
  });
});
