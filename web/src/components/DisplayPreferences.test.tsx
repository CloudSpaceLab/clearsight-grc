import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { DisplayPreferencesMenu, DisplayPreferencesRoot } from "./DisplayPreferences";

beforeEach(() => {
  window.localStorage.clear();
  delete document.documentElement.dataset.theme;
  delete document.documentElement.dataset.themePreference;
  delete document.documentElement.dataset.density;
});

function renderPreferences() {
  return render(<DisplayPreferencesRoot><main>ClearSight</main><DisplayPreferencesMenu/></DisplayPreferencesRoot>);
}

describe("DisplayPreferencesRoot", () => {
  it("applies and persists explicit theme and density preferences", () => {
    renderPreferences();

    fireEvent.click(screen.getByText("Display"));
    fireEvent.click(screen.getByRole("button", { name: "Dark" }));
    fireEvent.click(screen.getByRole("button", { name: "Compact" }));

    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(document.documentElement.dataset.themePreference).toBe("dark");
    expect(document.documentElement.dataset.density).toBe("compact");
    expect(window.localStorage.getItem("clearsight.theme")).toBe("dark");
    expect(window.localStorage.getItem("clearsight.density")).toBe("compact");
  });

  it("keeps system theme as an explicit preference rather than pretending it is fixed", () => {
    renderPreferences();

    fireEvent.click(screen.getByText("Display"));
    fireEvent.click(screen.getByRole("button", { name: "System" }));

    expect(document.documentElement.dataset.themePreference).toBe("system");
    expect(["light", "dark"]).toContain(document.documentElement.dataset.theme);
  });
});
