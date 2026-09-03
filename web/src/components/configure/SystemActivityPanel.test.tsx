import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { SystemActivityPanel } from "./SystemActivityPanel";

const activityApi = vi.hoisted(() => ({ loadSystemActivity: vi.fn() }));
vi.mock("../../systemActivityApi", () => activityApi);

beforeEach(() => {
  activityApi.loadSystemActivity.mockReset();
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
});

it("applies audit actor and date filters through the server query", async () => {
  render(<SystemActivityPanel mode="audit" />);
  await screen.findByText("Acme Payments");

  fireEvent.change(screen.getByLabelText("Actor"), { target: { value: "Acme" } });
  fireEvent.change(screen.getByLabelText("Actor type"), { target: { value: "EXTERNAL_PARTICIPANT" } });
  fireEvent.change(screen.getByLabelText("From"), { target: { value: "2026-09-01T00:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));

  await waitFor(() => expect(activityApi.loadSystemActivity).toHaveBeenLastCalledWith(expect.objectContaining({
    actor: "Acme",
    actorKind: "EXTERNAL_PARTICIPANT",
    from: "2026-09-01T00:00:00.000Z",
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
