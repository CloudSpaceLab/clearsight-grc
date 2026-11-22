import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Reports } from "./Reports";
const api = vi.hoisted(() => ({ listReportDefinitions: vi.fn(), listReportRunPage: vi.fn(), createIdempotentReportRun: vi.fn(), downloadReportRun: vi.fn() }));
const authority = vi.hoisted(() => ({ resolveAuthority: vi.fn() }));
vi.mock("../reportingApi", () => api);
vi.mock("../api", () => authority);
const context = { tenant: { id: "bank" }, legal_entity: { id: "entity" }, actor: { id: "person" } } as never;
const definition = { id: "definition", legal_entity_id: "entity", name: "Operational risk summary", description: "Risk posture", dataset: "MATTERS", status: "ACTIVE", effective: true, current_version: 2, format: "PDF", scope_kind: "LEGAL_ENTITY" };
const run = { id: "run", legal_entity_id: "entity", definition_id: "definition", definition_code: "RISK", definition_version: 2, dataset: "MATTERS", status: "READY", format: "PDF", expires_at: "2099-01-01T00:00:00Z", as_of: "2026-11-20T10:00:00Z", created_at: "2026-11-20T10:00:00Z", source_boundary: { population: 4, population_complete: true, captured_at: "2026-11-20T10:00:00Z" } };
beforeEach(() => { vi.clearAllMocks(); localStorage.clear(); Object.defineProperty(navigator, "locks", { configurable: true, value: { request: vi.fn(async (_key: string, _options: unknown, callback: (lock: unknown) => Promise<unknown>) => callback({ name: "report" })) } }); api.listReportDefinitions.mockResolvedValue([definition]); api.listReportRunPage.mockResolvedValue({ items: [run] }); authority.resolveAuthority.mockResolvedValue({ principal: { id: "person" }, rule_id: "rule", policy_version: "v1" }); });
for (const dataset of ["PROGRAMS", "MATTER_BOARD_BRIEF"]) it(`does not offer contained ${dataset} reports`, async () => {
  api.listReportDefinitions.mockResolvedValue([{ ...definition, dataset }]);
  api.listReportRunPage.mockResolvedValue({ items: [{ ...run, dataset }] });
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  expect(screen.getAllByText("Report unavailable. Contact your administrator.").length).toBeGreaterThan(0);
  expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Download RISK" })).toBeNull();
});
it("retains an uncertain report request without retrying after its dataset is contained", async () => {
  api.createIdempotentReportRun.mockRejectedValueOnce(new Error("connection lost"));
  const first = render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await screen.findByRole("button", { name: "Generate report" }));
  await screen.findByText("Report request status unknown. Check recent reports before requesting another.");
  const key = api.createIdempotentReportRun.mock.calls[0]![2];
  first.unmount();
  api.listReportDefinitions.mockResolvedValue([{ ...definition, dataset: "PROGRAMS" }]);
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  expect(await screen.findByText("Report unavailable. Contact your administrator.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Retry report request" })).toBeNull();
  expect(api.createIdempotentReportRun).toHaveBeenCalledTimes(1);
  expect(Array.from({ length: localStorage.length }, (_, index) => localStorage.getItem(localStorage.key(index)!)).some((value) => value?.includes(key))).toBe(true);
});
it("shows scoped report definitions and recent report status", async () => {
  render(<Reports context={context}/>);
  expect(await screen.findByText("Operational risk summary")).toBeTruthy();
  expect(await screen.findByText("RISK")).toBeTruthy();
  expect(screen.getAllByText("Ready").length).toBeGreaterThan(0);
  fireEvent.click(screen.getByRole("button", { name: "View Operational risk summary" }));
  expect(screen.getByText("Risk posture")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
});
it("does not display records from another legal entity", async () => {
  api.listReportDefinitions.mockResolvedValue([{ ...definition, legal_entity_id: "other", name: "Restricted report" }]);
  api.listReportRunPage.mockResolvedValue({ items: [{ ...run, legal_entity_id: "other", definition_code: "SECRET" }] });
  render(<Reports context={context}/>);
  expect(await screen.findByText("Report definitions unavailable.")).toBeTruthy();
  expect(await screen.findByText("Recent reports unavailable.")).toBeTruthy();
  expect(screen.queryByText("Restricted report")).toBeNull();
  expect(screen.queryByText("SECRET")).toBeNull();
});
it("loads the next authorized page of report runs", async () => {
  api.listReportRunPage.mockResolvedValueOnce({ items: [run], next_cursor: "next" }).mockResolvedValueOnce({ items: [{ ...run, id: "run-2", definition_code: "PRIVACY" }] });
  render(<Reports context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more reports" }));
  expect(await screen.findByText("PRIVACY")).toBeTruthy();
  expect(api.listReportRunPage).toHaveBeenLastCalledWith({ limit: 25, cursor: "next" }, expect.any(AbortSignal));
});
it("generates with the authorized definition version and does not repeat an uncertain command", async () => {
  api.createIdempotentReportRun.mockRejectedValue(new Error("connection lost"));
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await screen.findByRole("button", { name: "Generate report" }));
  expect(await screen.findByText("Report request status unknown. Check recent reports before requesting another." )).toBeTruthy();
  expect(api.createIdempotentReportRun).toHaveBeenCalledWith("definition", 2, expect.any(String));
  expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
});
it("retries an uncertain report request with its original key after reload", async () => {
  api.createIdempotentReportRun.mockRejectedValueOnce(new Error("connection lost"));
  const first = render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await screen.findByRole("button", { name: "Generate report" }));
  await screen.findByText("Report request status unknown. Check recent reports before requesting another.");
  const key = api.createIdempotentReportRun.mock.calls[0]![2];
  first.unmount();
  api.createIdempotentReportRun.mockResolvedValue({ ...run, requested_by_ref: "person" });
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "Retry report request" }));
  await waitFor(() => expect(api.createIdempotentReportRun).toHaveBeenLastCalledWith("definition", 2, key));
  expect(await screen.findByText("Report requested.")).toBeTruthy();
});
it("does not offer generation without the current performer route", async () => {
  authority.resolveAuthority.mockResolvedValue({ principal: { id: "other" }, rule_id: "rule", policy_version: "v1" });
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  expect(await screen.findByText("Report generation unavailable for your role.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
});
it("does not send a report request when its recovery key cannot be saved", async () => {
  const failStorage = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage unavailable"); });
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await screen.findByRole("button", { name: "Generate report" }));
  expect(await screen.findByText("Report requests unavailable. Check browser storage.")).toBeTruthy();
  expect(api.createIdempotentReportRun).not.toHaveBeenCalled();
  failStorage.mockRestore();
});
it("does not send a report request when cross-tab coordination is unavailable", async () => {
  Object.defineProperty(navigator, "locks", { configurable: true, value: undefined });
  render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await screen.findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await screen.findByRole("button", { name: "Generate report" }));
  expect(await screen.findByText("Report requests unavailable. Check browser support.")).toBeTruthy();
  expect(api.createIdempotentReportRun).not.toHaveBeenCalled();
});
it("prevents another tab from sending the same in-flight request", async () => {
  let held = false;
  Object.defineProperty(navigator, "locks", { configurable: true, value: { request: async (_key: string, _options: unknown, callback: (lock: unknown) => Promise<unknown>) => {
    if (held) return callback(null); held = true; try { return await callback({ name: "report" }); } finally { held = false; }
  } } });
  let finish!: (value: unknown) => void;
  api.createIdempotentReportRun.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
  const first = render(<Reports context={context} enableGeneration/>);
  const second = render(<Reports context={context} enableGeneration/>);
  fireEvent.click(await within(first.container).findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await within(second.container).findByRole("button", { name: "View Operational risk summary" }));
  fireEvent.click(await within(first.container).findByRole("button", { name: "Generate report" }));
  await waitFor(() => expect(api.createIdempotentReportRun).toHaveBeenCalledTimes(1));
  fireEvent.click(await within(second.container).findByRole("button", { name: "Generate report" }));
  expect(await within(second.container).findByText("Another report request is in progress.")).toBeTruthy();
  finish({ ...run, requested_by_ref: "person" });
  expect(await within(first.container).findByText("Report requested.")).toBeTruthy();
  fireEvent.click(await within(second.container).findByRole("button", { name: "Generate report" }));
  expect(await within(second.container).findByText("Report requested.")).toBeTruthy();
  expect(api.createIdempotentReportRun).toHaveBeenCalledTimes(1);
});
it("downloads a ready, unexpired report", async () => {
  const createURL = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:report");
  const revokeURL = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  api.downloadReportRun.mockResolvedValue({ blob: new Blob(["report"]), filename: "report.pdf" });
  render(<Reports context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Download RISK" }));
  expect(await screen.findByText("Report downloaded.")).toBeTruthy();
  expect(api.downloadReportRun).toHaveBeenCalledWith("run");
  createURL.mockRestore(); revokeURL.mockRestore(); click.mockRestore();
});
