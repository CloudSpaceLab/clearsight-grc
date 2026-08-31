import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { UIComponentGallery } from "./UIComponentGallery";

describe("UIComponentGallery", () => {
  it("renders production component families with explicit sample-data labels", () => {
    const view = render(<UIComponentGallery/>);
    for (const title of ["Actions", "Fields", "Selection", "Navigation", "Feedback", "Surfaces", "Data", "Overlays"]) {
      expect(screen.getByRole("heading", { name: title })).toBeTruthy();
    }

    const contracts = Array.from(view.container.querySelectorAll<HTMLElement>("[data-component-contract]"));
    expect(contracts.length).toBeGreaterThanOrEqual(16);
    for (const contract of contracts) {
      expect(contract.getAttribute("aria-label")).toMatch(/^Sample component data:/);
      expect(within(contract).getByText(/Keyboard:/)).toBeTruthy();
    }
  });
});
