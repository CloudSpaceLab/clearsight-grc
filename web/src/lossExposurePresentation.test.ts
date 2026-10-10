import { expect, it } from "vitest";
import type { LossSummary } from "./lossTypes";
import { summarizeLoadedLossExposure } from "./lossExposurePresentation";

function loss(id: string, currency: string, gross: number, recovered: number, net: number, status = "ACTIVE"): LossSummary {
  return {
    loss: { id, status, currency, title: id },
    totals: {
      gross_amount_minor: gross, recovered_amount_minor: recovered, net_loss_minor: net,
      currency, recovery_status: recovered > 0 ? "PARTIAL" : "NONE",
    },
  } as LossSummary;
}

it("keeps each currency separate and balances gross against net and recovered amounts", () => {
  const view = summarizeLoadedLossExposure([
    loss("ngn-a", "NGN", 50000, 20000, 30000),
    loss("usd-a", "USD", 10000, 1000, 9000),
    loss("ngn-b", "NGN", 10000, 0, 10000),
  ]);
  expect(view.groups.map((group) => [group.currency, group.count])).toEqual([["NGN", 2], ["USD", 1]]);
  const ngn = view.groups[0]!;
  expect(ngn.rows.map((row) => [row.label, row.value])).toEqual([
    ["Gross", 1000], ["Net outstanding", 666], ["Recovered", 333],
  ]);
  expect(ngn.rows[0]?.displayValue).toContain("600.00");
  expect(view.excluded).toBe(0);
});

it("omits voided, unsafe, incompatible or inconsistent records without silently treating them as zero", () => {
  const view = summarizeLoadedLossExposure([
    loss("good", "NGN", 500, 50, 450),
    loss("voided", "NGN", 300, 100, 200, "VOIDED"),
    loss("negative", "NGN", 40, 50, -10),
    loss("mismatch", "NGN", 100, 10, 20),
    loss("unsafe", "NGN", Number.MAX_SAFE_INTEGER + 1, 0, Number.MAX_SAFE_INTEGER + 1),
    { ...loss("currency-mismatch", "NGN", 10, 0, 10), totals: { ...loss("x", "NGN", 10, 0, 10).totals, currency: "USD" } },
  ]);
  expect(view.groups).toHaveLength(1);
  expect(view.groups[0]?.count).toBe(1);
  expect(view.voided).toBe(1);
  expect(view.excluded).toBe(4);
});

it("limits presented currencies and never ranks unlike currencies by monetary value", () => {
  const view = summarizeLoadedLossExposure([
    loss("eur", "EUR", 100000000, 0, 100000000),
    loss("usd", "USD", 10, 0, 10),
    loss("ngn-a", "NGN", 20, 0, 20),
    loss("ngn-b", "NGN", 40, 0, 40),
  ], 2);
  expect(view.groups.map((group) => group.currency)).toEqual(["NGN", "EUR"]);
  expect(view.moreCurrencies).toBe(1);
});
