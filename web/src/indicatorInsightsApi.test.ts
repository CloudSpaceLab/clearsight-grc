import { beforeEach, expect, it, vi } from "vitest";
import { loadIndicatorPortfolio } from "./indicatorInsightsApi";
import { requestJSON } from "./http";

vi.mock("./http", () => ({ requestJSON: vi.fn() }));

beforeEach(() => {
  vi.mocked(requestJSON).mockReset();
  vi.mocked(requestJSON).mockResolvedValue({ generated_at: "2026-10-06T07:00:00Z", items: [] });
});

it("builds bounded indicator filters without leaking empty parameters", async () => {
  await loadIndicatorPortfolio({ kind: "KRI", state: "BREACH", search: " mobile ", cursor: "next", limit: 25 });
  expect(requestJSON).toHaveBeenCalledWith(
    "",
    "/api/v1/metrics/indicators?kind=KRI&state=BREACH&search=mobile&cursor=next&limit=25",
    undefined,
  );
});

it("uses the unfiltered legal-entity endpoint by default", async () => {
  await loadIndicatorPortfolio();
  expect(requestJSON).toHaveBeenCalledWith("", "/api/v1/metrics/indicators", undefined);
});
