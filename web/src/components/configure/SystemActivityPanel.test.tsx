import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { SystemActivityPanel } from "./SystemActivityPanel";

const activityApi = vi.hoisted(() => ({ loadSystemActivity: vi.fn(), createAuditExport: vi.fn(), downloadAuditExport: vi.fn() }));
const appApi = vi.hoisted(() => ({ loadContext: vi.fn() }));
vi.mock("../../systemActivityApi", () => activityApi);
vi.mock("../../api", () => appApi);

beforeEach(() => {
  activityApi.loadSystemActivity.mockReset();
  activityApi.createAuditExport.mockReset();
  activityApi.downloadAuditExport.mockReset();
  appApi.loadContext.mockReset();
  appApi.loadContext.mockResolvedValue({ capabilities: { audit_export: true } });
  activityApi.loadSystemActivity.mockResolvedValue({
    as_of: "2026-09-03T10:30:00Z",
    next_cursor: "event-1",
    items: [{
      event_id: "event-2",
      occurred_at: "2026-09-03T10:29:00Z",
      category: "VENDOR",
      event_type: "THIRD_PARTY_ASSESSMENT_COMPLETED",
      action: "Third party assessment completed",
      outcome: "SUCCEEDED",
      actor_kind: "EXTERNAL_PARTICIPANT",
      actor_id: "vendor-user",
      actor_display_name: "Acme Payments",
      object_type: "THIRD_PARTY_ASSESSMENT",
      object_id: "assessment-1",
      source: "OUTBOX_EVENT",
    }],
  });
  activityApi.createAuditExport.mockResolvedValue({
    id: "auditexp-1",
    tenant_id: "bank-demo",
    requested_by: "admin",
    format: "CSV",
    filter: { actor_query: "Acme" },
    as_of: "2026-09-03T10:30:00Z",
    status: "READY",
    row_count: 1,
    data_sha256: "checksum",
    manifest_sha256: "manifest-checksum",
    created_at: "2026-09-03T10:30:00Z",
    completed_at: "2026-09-03T10:30:01Z",
    expires_at: "2026-09-10T10:30:00Z",
  });
  activityApi.downloadAuditExport.mockResolvedValue({ blob: new Blob(["event_id\nvisible\n"], { type: "text/csv" }), filename: "audit.csv" });
  Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:audit") });
  Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
});

it("applies audit actor and date filters through the server query", async () => {
  render(<SystemActivityPanel mode="audit" />);
  await screen.findByText("Acme Payments");

  fireEvent.change(screen.getByLabelText("Actor"), { target: { value: "Acme" } });
  fireEvent.click(screen.getByLabelText("Actor type"));
  fireEvent.click(await screen.findByRole("option", { name: "Vendor / external participant" }));
  fireEvent.change(screen.getByLabelText("From"), { target: { value: "2026-09-01T00:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));

  await waitFor(() => expect(activityApi.loadSystemActivity).toHaveBeenLastCalledWith(expect.objectContaining({
    actor: "Acme",
    actorKind: "EXTERNAL_PARTICIPANT",
    from: new Date("2026-09-01T00:00").toISOString(),
    limit: 50,
  })));
});

it("loads older events without replacing the current page", async () => {
  activityApi.loadSystemActivity
    .mockResolvedValueOnce({ as_of: "2026-09-03T10:30:00Z", next_cursor: "event-1", items: [{
      event_id: "event-2", occurred_at: "2026-09-03T10:29:00Z", category: "SYSTEM", event_type: "RUNTIME_RECOVERED", action: "Runtime recovered", outcome: "SUCCEEDED", actor_kind: "SYSTEM", object_type: "RUNTIME", object_id: "worker", source: "OUTBOX_EVENT",
    }] })
    .mockResolvedValueOnce({ as_of: "2026-09-03T10:30:00Z", items: [{
      event_id: "event-1", occurred_at: "2026-09-03T10:20:00Z", category: "GRC_WORK", event_type: "MATTER_CREATED", action: "Matter created", outcome: "SUCCEEDED", actor_kind: "UNKNOWN", object_type: "MATTER", object_id: "matter-1", source: "OUTBOX_EVENT",
    }] });

  render(<SystemActivityPanel mode="audit" />);
  await screen.findByText("Runtime recovered");
  fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));
  await screen.findByText("Matter created");
  expect(screen.getByText("Runtime recovered")).toBeTruthy();
});

it("exports only the applied audit filters through the governed download path", async () => {
  render(<SystemActivityPanel mode="audit" />);
  await screen.findByText("Acme Payments");

  fireEvent.change(screen.getByLabelText("Actor"), { target: { value: "Acme" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
  await waitFor(() => expect(activityApi.loadSystemActivity).toHaveBeenLastCalledWith(expect.objectContaining({ actor: "Acme" })));

  fireEvent.change(screen.getByLabelText("Actor"), { target: { value: "Unapplied draft" } });
  fireEvent.click(await screen.findByRole("button", { name: "Export" }));
  fireEvent.click(screen.getByRole("button", { name: "Create export" }));

  await waitFor(() => expect(activityApi.createAuditExport).toHaveBeenCalledWith("CSV", expect.objectContaining({ actor: "Acme" })));
  const exportCall = activityApi.createAuditExport.mock.calls.at(-1);
  if (!exportCall) throw new Error("expected governed audit export call");
  expect(exportCall[1].actor).not.toBe("Unapplied draft");
  await waitFor(() => expect(activityApi.downloadAuditExport).toHaveBeenCalledWith("auditexp-1"));
  expect(URL.createObjectURL).toHaveBeenCalled();
});
