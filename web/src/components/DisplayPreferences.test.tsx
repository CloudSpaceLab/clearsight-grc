import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DisplayPreferencesMenu, DisplayPreferencesRoot } from "./DisplayPreferences";

const presentationApi = vi.hoisted(() => ({
  loadPresentationPreferences: vi.fn(),
  savePresentationPreferences: vi.fn(),
}));
vi.mock("../presentationPreferencesApi", () => presentationApi);

beforeEach(() => {
  presentationApi.loadPresentationPreferences.mockReset();
  presentationApi.savePresentationPreferences.mockReset();
  presentationApi.loadPresentationPreferences.mockResolvedValue({
    tenant_id: "bank",
    principal_id: "cro-1",
    home_focus: "AUTO",
    portfolio_lens: "AUTO",
    effective_home_focus: "POSTURE",
    effective_portfolio_lens: "RISKS",
    version: 0,
  });
  presentationApi.savePresentationPreferences.mockImplementation(async (value) => ({
    tenant_id: "bank",
    principal_id: "cro-1",
    home_focus: value.home_focus,
    portfolio_lens: value.portfolio_lens,
    effective_home_focus: value.home_focus === "AUTO" ? "POSTURE" : value.home_focus,
    effective_portfolio_lens: value.portfolio_lens === "AUTO" ? "RISKS" : value.portfolio_lens,
    version: value.version + 1,
  }));
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


  it("shows role-derived workspace defaults and persists bounded overrides", async () => {
    renderPreferences();

    fireEvent.click(screen.getByText("Display"));
    const home = await screen.findByLabelText("Home focus");
    const portfolio = screen.getByLabelText("Portfolio start");
    expect((home as HTMLSelectElement).value).toBe("AUTO");
    expect((portfolio as HTMLSelectElement).value).toBe("AUTO");
    expect(screen.getByRole("option", { name: "Role default · Oversight first" })).toBeTruthy();
    expect(screen.getByRole("option", { name: "Role default · Risks" })).toBeTruthy();

    fireEvent.change(home, { target: { value: "MY_WORK" } });
    await waitFor(() => expect(presentationApi.savePresentationPreferences).toHaveBeenCalledWith({
      home_focus: "MY_WORK",
      portfolio_lens: "AUTO",
      version: 0,
    }));
  });
});
