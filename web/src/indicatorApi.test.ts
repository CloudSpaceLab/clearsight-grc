import { beforeEach, expect, it, vi } from "vitest";
import { loadIndicatorPopulation } from "./indicatorApi";
import { requestJSON } from "./http";

vi.mock("./http", () => ({ requestJSON: vi.fn() }));

beforeEach(() => {
  vi.mocked(requestJSON).mockReset();
  vi.mocked(requestJSON).mockResolvedValue({ items: [], complete: true });
});

it("builds scoped paginated Indicator reads without empty parameters", async () => {
  await loadIndicatorPopulation({
    kind: "KRI",
    organizationScopeID: " scope-a ",
    cursor: " next-page ",
    limit: 50,
  });

  expect(requestJSON).toHaveBeenCalledWith(
    "",
    "/api/v1/risk-indicators?kind=KRI&organization_scope_id=scope-a&cursor=next-page&limit=50",
    undefined,
  );
});

it("keeps exact Indicator lookup separate from page cursors", async () => {
  await loadIndicatorPopulation({ kind: "KCI", checkID: " check-1 ", limit: 1 });
  expect(requestJSON).toHaveBeenCalledWith(
    "",
    "/api/v1/risk-indicators?kind=KCI&check_id=check-1&limit=1",
    undefined,
  );
});
