import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { LossPeriodBundle } from "../../metricApi";
import { OrganizationLossSummary } from "./OrganizationLossSummary";
import { LossMovement } from "./LossMovement";
import { lossComparisonLabel, lossMovementPoints, lossPeriodValue } from "./lossOverviewPresentation";

function singleCurrencyBundle(): LossPeriodBundle {
  return {
    generated_at: "2026-10-07T12:00:00Z",
    period_start: "2026-09-08T00:00:00Z",
    period_end: "2026-10-07T12:00:00Z",
    scope_id: "bank-ng",
    scope_kind: "LEGAL_ENTITY",
    source_id: "8f7a0000-0000-4000-8000-000000000001",
    source_revision: "operational-loss-ledger-v1",
    definition_revision: "operational-loss-period-v1",
    event_count: 2,
    contributing_loss_count: 2,
    unattributed_event_count: 0,
    mixed_currencies: false,
    net_loss: { minor_units: "9223372036854775807", currency: "NGN" },
    currencies: [{
      currency: "NGN",
      gross: { minor_units: "9223372036854775807", currency: "NGN" },
      recovery: { minor_units: "0", currency: "NGN" },
      reversal: { minor_units: "0", currency: "NGN" },
      net: { minor_units: "9223372036854775807", currency: "NGN" },
      loss_event_count: 2,
      recovery_event_count: 0,
      reversal_event_count: 0,
    }],
    organization_breakdown: [{
      key: "scope:technology",
      scope_id: "technology",
      label: "Technology",
      kind: "ORGANIZATION_SCOPE",
      loss_event_count: 2,
      contributing_loss_count: 2,
      mixed_currencies: false,
      net_loss: { minor_units: "9223372036854775807", currency: "NGN" },
      currencies: [{
        currency: "NGN",
        gross: { minor_units: "9223372036854775807", currency: "NGN" },
        recovery: { minor_units: "0", currency: "NGN" },
        reversal: { minor_units: "0", currency: "NGN" },
        net: { minor_units: "9223372036854775807", currency: "NGN" },
        loss_event_count: 2,
        recovery_event_count: 0,
        reversal_event_count: 0,
      }],
    }],
    flow_resolution: "DAY",
    flow_points: [
      {
        start: "2026-10-06T00:00:00Z",
        end: "2026-10-06T23:59:59Z",
        loss_event_count: 1,
        contributing_loss_count: 1,
        mixed_currencies: false,
        net_loss: { minor_units: "900719925474099300", currency: "NGN" },
        currencies: [],
      },
      {
        start: "2026-10-07T00:00:00Z",
        end: "2026-10-07T12:00:00Z",
        loss_event_count: 1,
        contributing_loss_count: 1,
        mixed_currencies: false,
        net_loss: { minor_units: "9223372036854775807", currency: "NGN" },
        currencies: [],
      },
    ],
    comparison: {
      period_start: "2026-08-09T00:00:00Z",
      period_end: "2026-09-07T23:59:59Z",
      event_count: 3,
      contributing_loss_count: 3,
      mixed_currencies: false,
      net_loss: { minor_units: "9223372036854775807", currency: "NGN" },
      currencies: [],
      event_delta: -1,
      net_delta: { minor_units: "-450", currency: "NGN" },
      direction: "IMPROVED",
      comparison_quality: "COMPLETE",
    },
  };
}

function mixedCurrencyBundle(): LossPeriodBundle {
  const base = singleCurrencyBundle();
  return {
    ...base,
    mixed_currencies: true,
    net_loss: undefined,
    currencies: [
      base.currencies[0]!,
      {
        currency: "USD",
        gross: { minor_units: "10000", currency: "USD" },
        recovery: { minor_units: "0", currency: "USD" },
        reversal: { minor_units: "0", currency: "USD" },
        net: { minor_units: "10000", currency: "USD" },
        loss_event_count: 1,
        recovery_event_count: 0,
        reversal_event_count: 0,
      },
    ],
    organization_breakdown: [{
      ...base.organization_breakdown[0]!,
      mixed_currencies: true,
      net_loss: undefined,
      currencies: [
        base.currencies[0]!,
        {
          currency: "USD",
          gross: { minor_units: "10000", currency: "USD" },
          recovery: { minor_units: "0", currency: "USD" },
          reversal: { minor_units: "0", currency: "USD" },
          net: { minor_units: "10000", currency: "USD" },
          loss_event_count: 1,
          recovery_event_count: 0,
          reversal_event_count: 0,
        },
      ],
    }],
    flow_points: base.flow_points.map((point, index) => ({
      ...point,
      loss_event_count: index + 1,
      mixed_currencies: true,
      net_loss: undefined,
      currencies: [],
    })),
    comparison: {
      ...base.comparison,
      mixed_currencies: true,
      net_loss: undefined,
      net_delta: undefined,
      direction: "UNKNOWN",
      comparison_quality: "LIMITED",
      event_delta: -1,
    },
  };
}

it("keeps exact large money values through the Loss movement presentation", () => {
  const bundle = singleCurrencyBundle();
  expect(lossPeriodValue(bundle).replace(/\D/g, "")).toBe("9223372036854775807");
  expect(lossComparisonLabel(bundle)).toContain("improved");

  const points = lossMovementPoints(bundle);
  expect(points[0]?.displayValue?.replace(/\D/g, "")).toBe("900719925474099300");
  expect(points[1]?.displayValue?.replace(/\D/g, "")).toBe("9223372036854775807");

  render(<LossMovement bundle={bundle} state="live"/>);
  expect(screen.getByRole("img", { name: "Net operational Loss movement" })).toBeTruthy();
  expect(screen.getByRole("table", { name: "Net operational Loss movement" }).textContent)
    .toContain("9223372036854775807".slice(0, 6));
});

it("falls back to event counts when currencies are mixed", () => {
  const bundle = mixedCurrencyBundle();

  expect(lossPeriodValue(bundle)).toBe("2 events");
  expect(lossComparisonLabel(bundle)).toBe("1 fewer event vs prior period");

  render(<>
    <OrganizationLossSummary bundle={bundle} state="live" onOpenScope={vi.fn()}/>
    <LossMovement bundle={bundle} state="live"/>
  </>);

  expect(screen.getByRole("list", { name: "Operational Loss events by organization area" })).toBeTruthy();
  expect(screen.getByRole("img", { name: "Operational Loss event movement" })).toBeTruthy();
  expect(screen.getByText("Amounts remain separated by currency.")).toBeTruthy();
  expect(screen.queryByText(/NGN.*USD|USD.*NGN/)).toBeNull();
});
