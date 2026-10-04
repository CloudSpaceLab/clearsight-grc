import { describe, expect, it } from "vitest";
import { homeMetricDetail, homeMetricFilter, homeMetricMeta, homeMetricQuality, homeMetricTone, headlineMetricDefinitions } from "./homeMetricPresentation";

describe("home metric presentation", () => {
  it("keeps stable headline positions", () => {
    expect(headlineMetricDefinitions.map((item) => item.id)).toEqual([
      "critical_high_open",
      "overdue_open",
      "routing_gaps",
      "outcome_failures",
    ]);
  });

  it("lets incomplete data override reassuring clear-state styling", () => {
    const metric = {
      id: "overdue_open",
      condition: "CLEAR" as const,
      freshness: "CURRENT" as const,
      completeness: "PARTIAL" as const,
      population: 20,
      excluded: 1,
      unknown: 2,
      basis: "CURRENT_POSTURE" as const,
    };
    expect(homeMetricTone(metric)).toBe("success");
    expect(homeMetricQuality(metric)).toBe("partial");
    expect(homeMetricMeta(metric)).toBe("Current posture · 20 checked · 1 excluded · 2 unknown");
  });

  it("uses the server drill filter and rejects unknown filters", () => {
    expect(homeMetricFilter("overdue")).toBe("overdue");
    expect(homeMetricFilter("routing-gaps")).toBe("routing-gaps");
    expect(homeMetricFilter("made-up")).toBe("all");
  });

  it("keeps human-readable detail stable by metric id", () => {
    expect(homeMetricDetail("critical_high_open")).toBe("Open priority 4–5 issues");
    expect(homeMetricDetail("unknown")).toBe("Current governed metric");
  });


  it("reserves critical tone for the two material adverse metric families", () => {
    const attention = (id: string) => ({ id, condition: "ATTENTION" as const });
    expect(homeMetricTone(attention("critical_high_open"))).toBe("error");
    expect(homeMetricTone(attention("outcome_failures"))).toBe("error");
    expect(homeMetricTone(attention("overdue_open"))).toBe("warning");
    expect(homeMetricTone(attention("routing_gaps"))).toBe("warning");
  });
});
