import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { IndicatorValue } from "./IndicatorValue";

it("formats scientific native values without floating-point rounding", () => {
  render(<IndicatorValue
    denominator={100}
    score={0}
    measurement={{
      field: "success_rate",
      unit: "PERCENT",
      precision: 2,
      value: "9.87e1",
      limits: [{ operator: "GREATER_OR_EQUAL", expected: "9.95e1" }],
      condition: "BREACHED",
    }}
  />);

  expect(screen.getByText("98.70%")).toBeTruthy();
  expect(screen.getByText("Limit ≥ 99.50%")).toBeTruthy();
  expect(screen.getByText("0 / 100 concern")).toBeTruthy();
});

it("preserves source precision instead of rounding an adverse value into its limit", () => {
  render(<IndicatorValue
    denominator={100}
    measurement={{
      field: "success_rate",
      unit: "PERCENT",
      precision: 2,
      value: "99.4999",
      limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.5" }],
      condition: "BREACHED",
    }}
  />);

  expect(screen.getByText("99.4999%")).toBeTruthy();
  expect(screen.getByText("Limit ≥ 99.50%")).toBeTruthy();
});
