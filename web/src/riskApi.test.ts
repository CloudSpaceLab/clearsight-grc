import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getRisk, listRisks } from "./riskApi";

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Risk API", () => {
  it("sends bounded register filters and preserves the opaque cursor", async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ items: [], next_cursor: "cursor/2" }), { status: 200 }));

    await expect(listRisks({
      status: "ACTIVE",
      category: "Operational resilience",
      ownerPrincipalID: "owner-1",
      search: "network",
      appetitePosition: "BREACHED",
      cursor: "cursor/1",
      limit: 25,
    })).resolves.toEqual({ items: [], next_cursor: "cursor/2" });

    const url = new URL(String(fetchMock.mock.calls[0]?.[0]), "https://example.test");
    expect(url.pathname).toBe("/api/v1/risks");
    expect(Object.fromEntries(url.searchParams)).toEqual({
      status: "ACTIVE",
      category: "Operational resilience",
      owner_principal_id: "owner-1",
      search: "network",
      appetite_position: "BREACHED",
      cursor: "cursor/1",
      limit: "25",
    });
  });

  it("encodes exact Risk identity on detail reads and forwards AbortSignal", async () => {
    const controller = new AbortController();
    const aggregate = { risk: { id: "risk/1" }, assessments: [], appetite: [] };
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(aggregate), { status: 200 }));

    await expect(getRisk("risk/1", controller.signal)).resolves.toEqual(aggregate);
    expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/risks/risk%2F1");
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit | undefined)?.signal).toBe(controller.signal);
  });
});
