import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import { loadFrameworkEditionPage } from "./programOperationsApi";
import type { FrameworkEditionOption } from "./programOperationsApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

const edition = (number: number): FrameworkEditionOption => ({
  id: "edition-" + number, publisher_code: "PUBLISHER", framework_code: "SECURITY",
  edition_code: String(number), title: "Edition " + number,
  jurisdiction: "GLOBAL", source_url: "https://example.org/source",
  source_published_on: "2026-01-01", license_basis: "CITATIONS_ONLY",
});

describe("Framework edition pagination", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(loadContext).mockResolvedValue({ tenant: { id: "tenant-1" } } as Awaited<ReturnType<typeof loadContext>>);
  });

  it("keeps the final displayed edition as the next bounded cursor", async () => {
    vi.mocked(requestJSON).mockResolvedValueOnce({items: Array.from({length: 26}, (_, index) => edition(index + 1))});
    const page = await loadFrameworkEditionPage("GLOBAL", "", 25);
    expect(page.items).toHaveLength(25);
    expect(page.next_after_id).toBe("edition-25");
    const path = vi.mocked(requestJSON).mock.calls[0]?.[1] ?? "";
    const url = new URL(path, "https://example.org");
    expect(url.pathname).toBe("/api/v1/framework-catalog/editions");
    expect(url.searchParams.get("limit")).toBe("26");
    expect(url.searchParams.get("jurisdiction")).toBe("GLOBAL");
    expect(url.searchParams.get("tenant_id")).toBe("tenant-1");
  });

  it("does not infer further results without the extra server row", async () => {
    vi.mocked(requestJSON).mockResolvedValueOnce({items: [edition(26)]});
    const page = await loadFrameworkEditionPage("GLOBAL", "edition-25", 25);
    expect(page.items.map((item) => item.id)).toEqual(["edition-26"]);
    expect(page.next_after_id).toBe("");
    expect(new URL(vi.mocked(requestJSON).mock.calls[0]![1], "https://example.org").searchParams.get("after_id")).toBe("edition-25");
  });
});
