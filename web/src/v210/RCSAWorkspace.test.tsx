import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RCSAWorkspace } from "./RCSAWorkspace";
const api = vi.hoisted(() => ({ listRCSACycles: vi.fn(), getRCSACycle: vi.fn() }));
vi.mock("../rcsaApi", () => api);
const cycle = { id: "cycle/1", code: "RC-01", name: "Annual assessment", status: "DRAFT", version: 1 };
beforeEach(() => { vi.resetAllMocks(); api.listRCSACycles.mockResolvedValue({ items: [{ cycle, risk_count: 4, control_count: 6, handoff: { label: "Setup" } }], complete: true }); api.getRCSACycle.mockResolvedValue({ cycle, risks: [], controls: [], phase: { label: "Setup" }, handoff: { label: "Setup" }, complete: true }); });
it("opens the exact assessment cycle", async () => {
  render(<RCSAWorkspace/>);
  expect((await screen.findByRole("link", { name: "Annual assessment" })).getAttribute("href")).toBe("#/risks/rcsa/cycle%2F1");
});
it("does not render a different cycle", async () => {
  api.getRCSACycle.mockResolvedValue({ cycle: { ...cycle, id: "other" }, risks: [], controls: [] });
  render(<RCSAWorkspace id="cycle/1"/>);
  expect(await screen.findByRole("heading", { name: "Assessment unavailable" })).toBeTruthy();
});
it("does not report an empty register when authorization results are incomplete", async () => {
  api.listRCSACycles.mockResolvedValue({ items: [], complete: false });
  render(<RCSAWorkspace/>);
  expect(await screen.findByText("Assessment list unavailable.")).toBeTruthy();
  expect(screen.queryByText("No assessment cycles found.")).toBeNull();
});
it("retains the incomplete visibility warning across pages", async () => {
  api.listRCSACycles.mockResolvedValueOnce({ items: [{ cycle, risk_count: 1, control_count: 0 }], complete: false, next_cursor: "next" }).mockResolvedValueOnce({ items: [], complete: true });
  render(<RCSAWorkspace/>);
  expect(await screen.findByText("Assessment list may be incomplete.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  await waitFor(() => expect(api.listRCSACycles).toHaveBeenCalledTimes(2));
  expect(screen.getByText("Assessment list may be incomplete.")).toBeTruthy();
});
