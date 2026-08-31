import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../../../http";
import { SentFormsView } from "../SentFormsView";

const api = vi.hoisted(() => ({ loadDistributionPage: vi.fn(), loadDistribution: vi.fn(), transitionDistribution: vi.fn() }));
vi.mock("../../../formsDistributionApi", async (original) => ({ ...await original(), ...api }));

const distribution = {
  id: "dist-a", form_template_id: "form-a", form_template_version: 4, subject_type: "CONTROL", subject_id: "control-a",
  title: "Quarterly control review", purpose: "Collect operating evidence", access_policy: "DIRECT_LINK_EMAIL_OTP", status: "OPEN",
  deadline: "2027-09-01T12:00:00Z", route_expires_at: "2027-09-01T11:00:00Z", version: 2,
  created_at: "2026-08-28T10:00:00Z", updated_at: "2026-08-28T10:00:00Z",
} as const;
const detail = { distribution, recipients: [{ id: "r1", role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "jane", state: "PENDING", version: 1 }], workspace: { id: "workspace-a", status: "OPEN", version: 3, updated_at: "2026-08-28T10:00:00Z" } } as const;

beforeEach(() => {
  setMediaQuery("(min-width: 1180px)", true);
  window.history.replaceState(null, "", "/?safe=1#forms");
  Object.values(api).forEach((mock) => mock.mockReset());
  api.loadDistributionPage.mockResolvedValue({ items: [distribution] });
  api.loadDistribution.mockResolvedValue(detail);
  api.transitionDistribution.mockResolvedValue({ ...detail, distribution: { ...distribution, status: "LOCKED", version: 3 } });
});

describe("SentFormsView", () => {
  it("reserves and names the result region while the current population loads", () => {
    api.loadDistributionPage.mockReturnValue(new Promise(() => undefined));
    render(<SentFormsView/>);
    expect(screen.getByRole("status", { name: "Loading sent forms matching the current filters" })).toBeTruthy();
  });

  it("offers sign-in recovery for an expired session", async () => {
    api.loadDistributionPage.mockRejectedValue(new ApiError(401, "Session expired"));
    render(<SentFormsView/>);
    expect(await screen.findByText("Sign in to review sent forms")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Sign in again" })).toBeTruthy();
  });

  it("names an ordinary load failure and retries the same query", async () => {
    api.loadDistributionPage.mockRejectedValueOnce(new ApiError(503, "Temporarily unavailable")).mockResolvedValueOnce({ items: [] });
    render(<SentFormsView/>);
    expect(await screen.findByText("Sent forms could not be loaded")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(api.loadDistributionPage).toHaveBeenCalledTimes(2));
  });

  it("replaces an empty result with one complete empty state", async () => {
    api.loadDistributionPage.mockResolvedValue({ items: [] });
    render(<SentFormsView/>);
    expect(await screen.findByText("No sent forms match these filters")).toBeTruthy();
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryByRole("complementary", { name: "Selected distribution" })).toBeNull();
  });

  it("preserves unrelated route state and existing distribution filters", async () => {
    window.history.replaceState(null, "", "/?safe=1&dist_due=OVERDUE&dist_owner=owner-a#forms");
    render(<SentFormsView/>);
    const status = await screen.findByRole("button", { name: /Status/ });
    fireEvent.click(status);
    fireEvent.click(await screen.findByRole("option", { name: "Responses open" }));
    await waitFor(() => expect(window.location.search).toContain("dist_status=OPEN"));
    expect(window.location.href).toContain("safe=1");
    expect(window.location.href).toContain("dist_due=OVERDUE");
    expect(window.location.href).toContain("dist_owner=owner-a");
    expect(window.location.hash).toBe("#forms");
  });

  it("shows partial-page handling only when the server returns a cursor", async () => {
    api.loadDistributionPage.mockResolvedValueOnce({ items: [distribution], next_cursor: "cursor-2" }).mockResolvedValueOnce({ items: [] });
    render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Load more sent forms" }));
    await waitFor(() => expect(api.loadDistributionPage).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "cursor-2" })));
    expect(screen.queryByRole("button", { name: "Load more sent forms" })).toBeNull();
  });

  it("loads only the distribution selected by its named row action", async () => {
    render(<SentFormsView/>);
    expect(await screen.findByRole("button", { name: "Open Quarterly control review" })).toBeTruthy();
    expect(api.loadDistribution).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Open Quarterly control review" }));
    await waitFor(() => expect(api.loadDistribution).toHaveBeenCalledWith("dist-a"));
  });

  it("keeps lifecycle feedback after the confirmed command", async () => {
    render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
    fireEvent.click(await screen.findByRole("button", { name: "Lock responses" }));
    expect(await screen.findByText("Responses locked. The sent-form change was confirmed.")).toBeTruthy();
    expect(api.transitionDistribution).toHaveBeenCalledWith("dist-a", 2, "lock");
  });

  it("keeps selected details inline when both regions retain useful width", async () => {
    setMediaQuery("(min-width: 1180px)", true);
    render(<SentFormsView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Quarterly control review" }));
    expect(await screen.findByRole("complementary", { name: "Quarterly control review details" })).toBeTruthy();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("replaces inline detail with a focused sheet below the useful-width threshold", async () => {
    setMediaQuery("(min-width: 1180px)", false);
    render(<SentFormsView/>);
    const open = await screen.findByRole("button", { name: "Open Quarterly control review" });
    open.focus();
    fireEvent.click(open);
    expect(await screen.findByRole("dialog", { name: "Quarterly control review details" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(open));
  });
});

function setMediaQuery(query: string, matches: boolean) {
  vi.stubGlobal("matchMedia", vi.fn((value: string) => ({ matches: value === query ? matches : false, media: value, onchange: null, addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn() })));
}
