import { beforeEach, expect, it, vi } from "vitest";
import { getRCSACycle, listRCSACycles } from "./rcsaApi";
import { requestJSON } from "./http";

vi.mock("./http", () => ({ requestJSON: vi.fn() }));

beforeEach(() => {
  vi.mocked(requestJSON).mockReset();
  vi.mocked(requestJSON).mockResolvedValue({ items: [], complete: true });
});

it("builds bounded RCSA list filters and cursors", async () => {
  await listRCSACycles({ status: "AWAITING_CHALLENGE", cursor: " next-page ", limit: 25 });
  expect(requestJSON).toHaveBeenCalledWith(
    "",
    "/api/v1/rcsa/cycles?status=AWAITING_CHALLENGE&cursor=next-page&limit=25",
    undefined,
  );
});

it("loads an exact encoded RCSA cycle", async () => {
  await getRCSACycle("cycle/1");
  expect(requestJSON).toHaveBeenCalledWith("", "/api/v1/rcsa/cycles/cycle%2F1", undefined);
});
