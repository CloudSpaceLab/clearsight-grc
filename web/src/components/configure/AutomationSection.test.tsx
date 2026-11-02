import { useEffect, useState } from "react";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadAutomationPolicyPage, type RuntimeContext } from "../../api";
import { ApiError } from "../../http";
import type { AutomationPolicy, AutomationPolicyPage } from "../../types";
import { AutomationSection } from "./AutomationSection";

vi.mock("../../api", () => ({ loadAutomationPolicyPage: vi.fn() }));

const context: RuntimeContext = {
  tenant: { id: "tenant-a", name: "Example Bank Group" },
  legal_entity: { id: "entity-a", name: "Example Bank Nigeria" },
  actor: { id: "actor-a", name: "Administrator", session_id: "session-a" },
  mode: "test", capabilities: { config_read: true },
};
const policy: AutomationPolicy = {
  id: "policy-1", tenant_id: "tenant-a", code: "EVIDENCE-REFRESH", name: "Evidence refresh",
  action_class: "REVERSIBLE_WRITE", status: "ACTIVE", version: 2,
  eligibility: { sensitivity: "INTERNAL" }, blast_radius_limit: { max_records: 25 },
  verification_contract: { method: "source_recheck", required: true }, rollout_mode: "SHADOW",
};
const second: AutomationPolicy = {
  ...policy, id: "policy-2", code: "REVIEW-REMINDER", name: "Review reminder",
  action_class: "NOTIFICATION", status: "DRAFT", version: 3,
};
function page(items: AutomationPolicy[] = [policy, second], extra: Partial<AutomationPolicyPage> = {}): AutomationPolicyPage {
  return { items, tenant_id: "tenant-a", scope: "TENANT", revision_basis: "LATEST_PER_CODE", limit: 200, returned_count: items.length, truncated: false, ...extra };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}
function Host({ initial = "#configure/automation", runtime = context }: { initial?: string; runtime?: RuntimeContext | null }) {
  const [hash, setHash] = useState(initial);
  useEffect(() => {
    const sync = () => setHash(window.location.hash);
    window.addEventListener("popstate", sync);
    window.addEventListener("hashchange", sync);
    return () => { window.removeEventListener("popstate", sync); window.removeEventListener("hashchange", sync); };
  }, []);
  return <AutomationSection hash={hash} runtimeContext={runtime} onNavigate={(next) => {
    window.history.pushState(null, "", next);
    setHash(next);
  }}/>;
}
function selected() { return screen.getByRole("complementary", { name: "Selected automation policy" }); }

beforeEach(() => {
  vi.resetAllMocks();
  window.history.replaceState(null, "", "#configure/automation");
  vi.mocked(loadAutomationPolicyPage).mockResolvedValue(page());
});

describe("AutomationSection", () => {
  it("uses the four source views and tenant latest-revision records without execution claims", async () => {
    render(<Host/>);
    expect(screen.getByRole("heading", { level: 1, name: "Automation & governance" })).toBeTruthy();
    expect(within(screen.getByRole("tablist", { name: "Automation and governance views" })).getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Automations", "Schedules", "Escalations", "Activity"]);
    expect(await screen.findByRole("button", { name: "Inspect Evidence refresh" })).toBeTruthy();
    expect(within(selected()).getByRole("heading", { name: "Evidence refresh" })).toBeTruthy();
    expect(screen.getByText(/latest policy revisions loaded/i)).toBeTruthy();
    expect(screen.getByText(/Example Bank Group/)).toBeTruthy();
    expect(screen.queryByText(/actions executed|successful runs|all systems operational/i)).toBeNull();
    expect(loadAutomationPolicyPage).toHaveBeenCalledOnce();
    expect(vi.mocked(loadAutomationPolicyPage).mock.calls[0]![0]).toBeInstanceOf(AbortSignal);
  });

  it("keeps policy selection and all four tabs coherent through browser Back and Forward", async () => {
    render(<Host/>);
    fireEvent.click(await screen.findByRole("button", { name: "Inspect Review reminder" }));
    expect(within(selected()).getByRole("heading", { name: "Review reminder" })).toBeTruthy();
    expect(new URLSearchParams(window.location.hash.split("?")[1]).get("policy")).toBe("policy-2");
    fireEvent.click(screen.getByRole("tab", { name: "Schedules" }));
    expect(screen.getByRole("tab", { name: "Schedules", selected: true })).toBeTruthy();
    const schedulesHash = window.location.hash;
    await act(async () => { window.history.back(); });
    await waitFor(() => expect(screen.getByRole("tab", { name: "Automations", selected: true })).toBeTruthy());
    expect(within(selected()).getByRole("heading", { name: "Review reminder" })).toBeTruthy();
    await act(async () => { window.history.forward(); });
    await waitFor(() => expect(window.location.hash).toBe(schedulesHash));
    expect(screen.getByRole("tab", { name: "Schedules", selected: true })).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Escalations" }));
    expect(screen.getByRole("link", { name: /escalation/i })).toHaveProperty("hash", "#configure/access/escalation");
    fireEvent.click(screen.getByRole("tab", { name: "Activity" }));
    expect(screen.getByRole("heading", { name: /activity.*unavailable|history.*unavailable/i })).toBeTruthy();
    expect(loadAutomationPolicyPage).toHaveBeenCalledOnce();
  });

  it("filters loaded records in the URL and clears the prior explicit selection", async () => {
    render(<Host initial="#configure/automation?policy=policy-1"/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    fireEvent.change(screen.getByRole("searchbox", { name: "Search loaded policies" }), { target: { value: "reminder" } });
    const query = new URLSearchParams(window.location.hash.split("?")[1]);
    expect(query.get("q")).toBe("reminder");
    expect(query.has("policy")).toBe(false);
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    expect(screen.getByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    expect(within(selected()).getByRole("heading", { name: "Review reminder" })).toBeTruthy();
    expect(loadAutomationPolicyPage).toHaveBeenCalledOnce();
  });

  it("applies stored action and status filters without fetching or retaining an unrelated policy", async () => {
    render(<Host initial="#configure/automation?action=NOTIFICATION&status=DRAFT"/>);
    expect(await screen.findByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    expect(screen.getByRole("button", { name: /Action class/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Status/i })).toBeTruthy();
    expect(within(selected()).getByRole("heading", { name: "Review reminder" })).toBeTruthy();
    expect(loadAutomationPolicyPage).toHaveBeenCalledOnce();
  });

  it("updates action and status filters through the shared selectors and can clear them", async () => {
    render(<Host initial="#configure/automation?policy=policy-1"/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    fireEvent.click(screen.getByRole("button", { name: /Status/i }));
    fireEvent.click(await screen.findByRole("option", { name: "Draft" }));
    expect(new URLSearchParams(window.location.hash.split("?")[1]).get("status")).toBe("DRAFT");
    expect(new URLSearchParams(window.location.hash.split("?")[1]).has("policy")).toBe(false);
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /Action class/i }));
    fireEvent.click(await screen.findByRole("option", { name: "Notification" }));
    expect(new URLSearchParams(window.location.hash.split("?")[1]).get("action")).toBe("NOTIFICATION");
    expect(screen.getByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(window.location.hash).toBe("#configure/automation");
    expect(screen.getByRole("button", { name: "Inspect Evidence refresh" })).toBeTruthy();
    expect(loadAutomationPolicyPage).toHaveBeenCalledOnce();
  });

  it("does not silently replace an explicitly selected policy missing from the loaded page", async () => {
    render(<Host initial="#configure/automation?policy=not-returned"/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    expect(within(selected()).getByRole("heading", { name: "Selected policy is not in this list" })).toBeTruthy();
    expect(within(selected()).queryByRole("heading", { name: "Evidence refresh" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Inspect Evidence refresh" }));
    expect(within(selected()).getByRole("heading", { name: "Evidence refresh" })).toBeTruthy();
  });

  it("distinguishes a verified empty revision list from an empty loaded-record filter", async () => {
    const { unmount } = render(<Host initial="#configure/automation?q=unmatched"/>);
    expect(await screen.findByRole("heading", { name: /no.*match/i })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "No policy revisions returned" })).toBeNull();
    unmount();
    vi.mocked(loadAutomationPolicyPage).mockResolvedValue(page([]));
    render(<Host/>);
    expect(await screen.findByRole("heading", { name: "No policy revisions returned" })).toBeTruthy();
    expect(screen.queryByText(/no automated actions are approved/i)).toBeNull();
  });

  it("exposes a truncated population before accepting search as complete", async () => {
    const items = Array.from({ length: 200 }, (_, index) => ({ ...policy, id: `policy-${index}`, code: `RULE-${index}`, name: `Rule ${index}` }));
    vi.mocked(loadAutomationPolicyPage).mockResolvedValue(page(items, { truncated: true }));
    render(<Host/>);
    await screen.findByRole("button", { name: "Inspect Rule 0" });
    expect(screen.getByText(/Only the first 200 policy codes are shown/)).toBeTruthy();
    expect(screen.getByRole("searchbox", { name: "Search loaded policies" })).toBeTruthy();
    expect(screen.getAllByRole("button", { name: /^Inspect Rule / })).toHaveLength(200);
  });

  it("clears stale records during refresh and recovers from a failed refresh", async () => {
    const refresh = deferred<AutomationPolicyPage>();
    vi.mocked(loadAutomationPolicyPage).mockResolvedValueOnce(page()).mockReturnValueOnce(refresh.promise).mockResolvedValueOnce(page([second]));
    render(<Host/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    fireEvent.click(screen.getByRole("button", { name: "Refresh policies" }));
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    await act(async () => { refresh.reject(new Error("network unavailable")); });
    expect(screen.getByRole("heading", { name: "Automation policies could not be loaded" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry automation data" }));
    expect(await screen.findByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
  });

  it.each([401, 403])("treats HTTP %i as denied access and discards previously displayed policy data", async (status) => {
    vi.mocked(loadAutomationPolicyPage).mockResolvedValueOnce(page()).mockRejectedValueOnce(new ApiError(status, "Access revoked"));
    render(<Host/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    fireEvent.click(screen.getByRole("button", { name: "Refresh policies" }));
    expect(await screen.findByRole("heading", { name: "Automation policies are restricted" })).toBeTruthy();
    expect(screen.queryByText("Evidence refresh")).toBeNull();
    expect(screen.queryByText("Review reminder")).toBeNull();
  });

  it("requires verified context and explicit configuration read access before loading", () => {
    const { rerender } = render(<Host runtime={null}/>);
    expect(loadAutomationPolicyPage).not.toHaveBeenCalled();
    rerender(<Host runtime={{ ...context, capabilities: {} }}/>);
    expect(screen.getByRole("heading", { name: "Automation policies are restricted" })).toBeTruthy();
    expect(loadAutomationPolicyPage).not.toHaveBeenCalled();
    rerender(<Host runtime={{ ...context, capabilities: { config_read: false } }}/>);
    expect(loadAutomationPolicyPage).not.toHaveBeenCalled();
  });

  it("aborts pending reads and ignores late success after configuration permission is revoked", async () => {
    const pending = deferred<AutomationPolicyPage>();
    vi.mocked(loadAutomationPolicyPage).mockReturnValueOnce(pending.promise);
    const { rerender } = render(<Host/>);
    const signal = vi.mocked(loadAutomationPolicyPage).mock.calls[0]![0]!;
    rerender(<Host runtime={{ ...context, capabilities: { config_read: false } }}/>);
    expect(signal.aborted).toBe(true);
    expect(screen.getByRole("heading", { name: "Automation policies are restricted" })).toBeTruthy();
    await act(async () => { pending.resolve(page()); });
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    expect(screen.getByRole("heading", { name: "Automation policies are restricted" })).toBeTruthy();
    rerender(<Host runtime={context}/>);
    expect(await screen.findByRole("button", { name: "Inspect Evidence refresh" })).toBeTruthy();
    expect(loadAutomationPolicyPage).toHaveBeenCalledTimes(2);
  });

  it("removes already displayed data immediately when a verified identity is replaced", async () => {
    const nextRead = deferred<AutomationPolicyPage>();
    vi.mocked(loadAutomationPolicyPage).mockResolvedValueOnce(page()).mockReturnValueOnce(nextRead.promise);
    const { rerender } = render(<Host/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    rerender(<Host runtime={{ ...context, actor: { ...context.actor, id: "actor-b" } }}/>);
    expect(screen.queryByText("Evidence refresh")).toBeNull();
    expect(screen.queryByText("Review reminder")).toBeNull();
    await act(async () => { nextRead.resolve(page([second])); });
    expect(screen.getByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
  });

  it("does not let a late rejected read erase a newer identity's successful result", async () => {
    const oldRead = deferred<AutomationPolicyPage>();
    vi.mocked(loadAutomationPolicyPage).mockReturnValueOnce(oldRead.promise).mockResolvedValueOnce(page([second]));
    const { rerender } = render(<Host/>);
    rerender(<Host runtime={{ ...context, actor: { ...context.actor, id: "actor-b" } }}/>);
    await screen.findByRole("button", { name: "Inspect Review reminder" });
    await act(async () => { oldRead.reject(new ApiError(403, "Old request denied")); });
    expect(screen.getByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Automation policies are restricted" })).toBeNull();
  });

  it.each([
    ["tenant", { ...context, tenant: { id: "tenant-b", name: "Other Bank" } }],
    ["entity", { ...context, legal_entity: { id: "entity-b", name: "Other Entity" } }],
    ["actor", { ...context, actor: { ...context.actor, id: "actor-b" } }],
    ["session", { ...context, actor: { ...context.actor, session_id: "session-b" } }],
  ] as const)("clears records synchronously and rejects late responses when the %s changes", async (_dimension, nextContext) => {
    const oldRead = deferred<AutomationPolicyPage>(), newRead = deferred<AutomationPolicyPage>();
    vi.mocked(loadAutomationPolicyPage).mockResolvedValueOnce(page()).mockReturnValueOnce(oldRead.promise).mockReturnValueOnce(newRead.promise);
    const { rerender } = render(<Host/>);
    await screen.findByRole("button", { name: "Inspect Evidence refresh" });
    fireEvent.click(screen.getByRole("button", { name: "Refresh policies" }));
    const oldSignal = vi.mocked(loadAutomationPolicyPage).mock.calls[1]![0]!;
    rerender(<Host runtime={nextContext}/>);
    expect(oldSignal.aborted).toBe(true);
    expect(screen.queryByText("Evidence refresh")).toBeNull();
    await act(async () => { oldRead.resolve(page()); });
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
    const nextPolicy = { ...second, tenant_id: nextContext.tenant.id };
    await act(async () => { newRead.resolve(page([nextPolicy], { tenant_id: nextContext.tenant.id })); });
    expect(screen.getByRole("button", { name: "Inspect Review reminder" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Inspect Evidence refresh" })).toBeNull();
  });

  it.each([
    ["wrong tenant", () => page([policy], { tenant_id: "other-tenant" })],
    ["wrong population scope", () => ({ ...page(), scope: "ENTITY" }) as unknown as AutomationPolicyPage],
    ["wrong revision basis", () => ({ ...page(), revision_basis: "ALL_VERSIONS" }) as unknown as AutomationPolicyPage],
    ["incorrect returned count", () => page([policy], { returned_count: 2 })],
    ["incorrect limit", () => page([policy], { limit: 500 })],
  ] as const)("rejects %s rather than displaying unqualified records", async (_case, response) => {
    vi.mocked(loadAutomationPolicyPage).mockResolvedValue(response());
    render(<Host/>);
    expect(await screen.findByRole("heading", { name: "Automation policies could not be loaded" })).toBeTruthy();
    expect(screen.queryByText("Evidence refresh")).toBeNull();
  });

  it("accepts repository tenant aliases only within the matching verified page envelope", async () => {
    vi.mocked(loadAutomationPolicyPage).mockResolvedValue(page([{ ...policy, tenant_id: "example-bank-slug" }]));
    render(<Host/>);
    expect(await screen.findByRole("button", { name: "Inspect Evidence refresh" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Automation policies could not be loaded" })).toBeNull();
  });

  it("aborts the current request on unmount", () => {
    vi.mocked(loadAutomationPolicyPage).mockReturnValue(new Promise(() => {}));
    const { unmount } = render(<Host/>);
    const signal = vi.mocked(loadAutomationPolicyPage).mock.calls[0]![0]!;
    unmount();
    expect(signal.aborted).toBe(true);
  });
});
