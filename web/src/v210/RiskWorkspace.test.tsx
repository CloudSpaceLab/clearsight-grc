import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RiskWorkspace } from "./RiskWorkspace";
import { ApiError } from "../http";
const api = vi.hoisted(() => ({ listRisks: vi.fn(), getRisk: vi.fn() }));
vi.mock("../riskApi", () => api);
const risk = { id: "risk/1", code: "R-01", name: "Service disruption", category: "Operational", statement: "Service may be unavailable", impact: "Delayed payments", status: "ACTIVE", version: 2 };
beforeEach(() => { vi.resetAllMocks(); api.listRisks.mockResolvedValue({ items: [{ risk }] }); api.getRisk.mockResolvedValue({ risk, assessments: [], appetite: [] }); });
it("clears the risk register if access is denied during pagination", async () => {
  api.listRisks.mockResolvedValueOnce({ items: [{ risk }], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Denied"));
  render(<RiskWorkspace/>);
  expect(await screen.findByText("Service disruption")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Risks unavailable.");
  expect(screen.queryByText("Service disruption")).toBeNull();
});
it("does not append a risk page that repeats its cursor", async () => {
  api.listRisks.mockResolvedValueOnce({ items: [{ risk }], next_cursor: "next" }).mockResolvedValueOnce({ items: [{ risk }], next_cursor: "next" });
  render(<RiskWorkspace/>);
  expect(await screen.findByText("Service disruption")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("More risks unavailable.");
  expect(screen.getAllByText("Service disruption")).toHaveLength(1);
});
it("offers loss navigation only with global oversight access", async () => {
  const view = render(<RiskWorkspace/>);
  expect(await screen.findByRole("heading", { name: "Risk register" })).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Loss events" })).toBeNull();
  view.rerender(<RiskWorkspace canViewLosses/>);
  expect(screen.getByRole("link", { name: "Loss events" }).getAttribute("href")).toBe("#/risks/losses");
});
it("searches the authorized register and opens the selected risk", async () => {
  render(<RiskWorkspace/>);
  expect((await screen.findByRole("link", { name: "Service disruption" })).getAttribute("href")).toBe("#/risks/risk%2F1");
  fireEvent.change(screen.getByLabelText("Find a risk"), { target: { value: "service" } });
  fireEvent.click(screen.getByRole("button", { name: "Search" }));
  await waitFor(() => expect(api.listRisks).toHaveBeenCalledWith(expect.objectContaining({ search: "service", limit: 25 }), expect.any(AbortSignal)));
});
it("shows only recorded appetite on the exact risk", async () => {
  render(<RiskWorkspace id="risk/1"/>);
  expect(await screen.findByRole("heading", { name: "Service disruption" })).toBeTruthy();
  expect(screen.getByText("No assessment recorded.")).toBeTruthy();
  expect(api.getRisk).toHaveBeenCalledWith("risk/1", expect.any(AbortSignal));
});
it("does not show a different risk returned for the requested identifier", async () => {
  api.getRisk.mockResolvedValue({ risk: { ...risk, id: "other" }, assessments: [], appetite: [] });
  render(<RiskWorkspace id="risk/1"/>);
  expect(await screen.findByRole("heading", { name: "Risk unavailable" })).toBeTruthy();
  expect(screen.queryByText("Service disruption")).toBeNull();
});
it("does not present an assessment from an earlier risk version as current", async () => {
  api.listRisks.mockResolvedValue({ items: [{ risk, latest_assessment: { risk_version: 1, appetite_position: "BREACHED" } }] });
  render(<RiskWorkspace/>);
  expect(await screen.findByText("Appetite unknown · Earlier version")).toBeTruthy();
});
it("does not report an appetite position when the assessment uses an outdated appetite statement", async () => {
  api.listRisks.mockResolvedValue({ items: [{ risk, latest_assessment: { risk_version: 2, appetite_position: "WITHIN", appetite_statement_id: "old" }, active_appetite: { id: "current" } }] });
  render(<RiskWorkspace/>);
  expect(await screen.findByText("Appetite unknown")).toBeTruthy();
  expect(screen.queryByText("Within appetite")).toBeNull();
});
it("shows linked control and indicator details with their recorded states", async () => {
  api.getRisk.mockResolvedValue({ risk, assessments: [], appetite: [], control_details_complete: true, indicator_details_complete: true, control_details: [{ link: { id: "control-1" }, definition: { name: "Review access" }, implementation_status: "IMPLEMENTED", program_name: "Access assurance" }], indicator_details: [{ link: { id: "indicator-1" }, check_name: "Privileged accounts reviewed", state: "BREACH" }] });
  render(<RiskWorkspace id="risk/1"/>);
  expect(await screen.findByText("Review access")).toBeTruthy();
  expect(screen.getByText("Privileged accounts reviewed")).toBeTruthy();
  expect(screen.getByText("Breach")).toBeTruthy();
});
