import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { KeyIndicators } from "../../v210/KeyIndicators";

const api = vi.hoisted(() => ({ loadIndicatorPopulation: vi.fn() }));
vi.mock("../../indicatorApi", () => api);

const indicator = {
  check_id: "check-1", check_code: "KRI-1", check_name: "Success rate",
  program_name: "Continuity", link: { kind: "KRI" }, state: "BREACH",
  reason: "The recorded result is below the limit.",
};
beforeEach(() => { api.loadIndicatorPopulation.mockReset(); });

it("shows scientific native values and limits without inventing a concern score", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator: { ...indicator,
    native_measurement: { field: "success_rate", unit: "PERCENT", precision: 2, value: "9.87e1", limits: [{ operator: "GREATER_OR_EQUAL", expected: "9.95e1" }], condition: "BREACHED" },
  }, risks: [] }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("98.70%")).toBeTruthy();
  expect(screen.getByText("≥ 99.50%")).toBeTruthy();
  expect(screen.queryByText(/\/ 100 concern/)).toBeNull();
});

it("preserves source precision rather than rounding an adverse value into its limit", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator: { ...indicator,
    native_measurement: { field: "success_rate", unit: "PERCENT", precision: 2, value: "99.4999", limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.5" }], condition: "BREACHED" },
  }, risks: [] }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("99.4999%")).toBeTruthy();
  expect(screen.getByText("≥ 99.50%")).toBeTruthy();
});

it("keeps extreme exponents bounded without changing the reported value", async () => {
  api.loadIndicatorPopulation.mockResolvedValue({ items: [{ indicator: { ...indicator,
    native_measurement: { field: "success_rate", unit: "PERCENT", precision: 2, value: "1e999999999", limits: [], condition: "BREACHED" },
  }, risks: [] }], complete: true });
  render(<KeyIndicators/>);
  expect(await screen.findByText("1e999999999%")).toBeTruthy();
});
