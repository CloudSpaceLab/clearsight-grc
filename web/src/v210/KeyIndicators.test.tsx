import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { KeyIndicators } from "./KeyIndicators";

const api = vi.hoisted(() => ({ loadIndicatorPopulation: vi.fn() }));
vi.mock("../indicatorApi", () => api);
const indicator = { check_id: "c1", check_code: "KRI-1", check_name: "Failed backups", program_id: "p1", program_name: "Continuity", link: { kind: "KRI" }, state: "BREACH", reason: "Latest native measurement is outside its approved limit.", native_measurement: { value: "2", unit: "COUNT", limits: [{ operator: "LESS_OR_EQUAL", expected: "1" }], condition: "BREACHED" }, evaluated_at: "2026-11-19T10:00:00Z", freshness_minutes: 1440 };
beforeEach(() => { vi.resetAllMocks(); });

it("shows the stored condition, value and approved limit", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator, risks: [{ id: "r1", name: "Backup failure", code: "R1" }], risk_count: 1 }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("Failed backups")).toBeTruthy();
  expect(screen.getByText("Breach")).toBeTruthy();
  expect(screen.getByText("2")).toBeTruthy();
  expect(screen.getByText("≤ 1")).toBeTruthy();
  expect(screen.getByText("Backup failure")).toBeTruthy();
  expect(screen.getByText(/^Evaluated /)).toBeTruthy();
  expect(screen.queryByText(/^Observed /)).toBeNull();
  expect(api.loadIndicatorPopulation).toHaveBeenCalledWith({ limit: 25 }, expect.any(AbortSignal));
});

it("shows stale results as unknown and identifies the reason", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator: { ...indicator, state: "UNKNOWN", reason: "Latest monitoring result is stale." }, risks: [], risk_count: 0 }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("Unknown")).toBeTruthy();
  expect(screen.getByText("Latest monitoring result is stale.")).toBeTruthy();
  expect(screen.queryByText("Breach")).toBeNull();
});

it("labels partial data and loads additional pages without inventing totals", async () => {
  api.loadIndicatorPopulation.mockResolvedValueOnce({ items: [{ indicator, risks: [], risk_count: 0 }], complete: false, next_cursor: "next" }).mockResolvedValueOnce({ items: [], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("Some indicators could not be loaded.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more indicators" }));
  expect(api.loadIndicatorPopulation).toHaveBeenLastCalledWith({ limit: 25, cursor: "next" });
  expect(screen.queryByText(/Total indicators/)).toBeNull();
});

it("distinguishes a denied or unavailable read from an empty register", async () => {
  api.loadIndicatorPopulation.mockRejectedValue(new Error("denied"));
  render(<KeyIndicators/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("No indicators found.")).toBeNull();
});

it("only offers the loss register when access is available", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [], complete: true });
  const view = render(<KeyIndicators/>);
  await screen.findByText("No indicators found.");
  expect(screen.queryByRole("link", { name: "Loss events" })).toBeNull();
  view.rerender(<KeyIndicators canViewLosses/>);
  expect(screen.getByRole("link", { name: "Loss events" }).getAttribute("href")).toBe("#/risks/losses");
});

it("does not invent an observation or limit when they are missing", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator: { ...indicator, state: "UNKNOWN", reason: "No current monitoring result.", native_measurement: undefined, evaluated_at: undefined }, risks: [], risk_count: 0 }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("Native value unavailable")).toBeTruthy();
  expect(screen.getByText("Not recorded")).toBeTruthy();
  expect(screen.getByText("No evaluation date")).toBeTruthy();
});

it("uses the authorized server filter for indicator type", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [], complete: true });
  render(<KeyIndicators/>);
  await screen.findByText("No indicators found.");
  fireEvent.change(screen.getByLabelText("Indicator type"), { target: { value: "KCI" } });
  await screen.findByText("No indicators found.");
  expect(api.loadIndicatorPopulation).toHaveBeenLastCalledWith({ limit: 25, kind: "KCI" }, expect.any(AbortSignal));
});
