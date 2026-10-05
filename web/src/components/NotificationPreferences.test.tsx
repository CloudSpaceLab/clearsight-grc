import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NotificationPreferences } from "./NotificationPreferences";
import type { NotificationPreferences as Value } from "../notificationPreferencesApi";

function value(overrides: Partial<Value> = {}): Value {
  return {
    tenant_id: "bank",
    principal_id: "actor",
    daily_digest_enabled: true,
    digest_minute: 420,
    time_zone: "Africa/Lagos",
    quiet_hours_enabled: false,
    quiet_start_minute: 1320,
    quiet_end_minute: 420,
    critical_email_required: true,
    version: 1,
    ...overrides,
  };
}

describe("NotificationPreferences", () => {
  it("keeps critical email required while allowing the daily digest to be disabled", async () => {
    const load = vi.fn().mockResolvedValue(value());
    const save = vi.fn().mockResolvedValue(value({ daily_digest_enabled: false, version: 2 }));
    render(<NotificationPreferences load={load} save={save}/>);

    expect(await screen.findByText("Critical required")).toBeTruthy();
    const digest = screen.getByRole("checkbox", { name: "Daily digest" });
    expect((digest as HTMLInputElement).checked).toBe(true);

    fireEvent.click(digest);
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));

    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({
      daily_digest_enabled: false,
      version: 1,
    })));
    expect(screen.queryByRole("checkbox", { name: /critical/i })).toBeNull();
  });

  it("saves digest timezone and quiet hours as bounded local minutes", async () => {
    const load = vi.fn().mockResolvedValue(value());
    const save = vi.fn().mockImplementation(async (input) => value({ ...input, version: 2 }));
    render(<NotificationPreferences load={load} save={save}/>);

    await screen.findByDisplayValue("07:00");
    fireEvent.input(screen.getByLabelText("Digest time"), { target: { value: "08:30" } });
    fireEvent.change(screen.getByLabelText("Time zone"), { target: { value: "UTC" } });
    fireEvent.click(screen.getByRole("checkbox", { name: "Quiet hours" }));
    fireEvent.input(screen.getByLabelText("Starts"), { target: { value: "21:30" } });
    fireEvent.input(screen.getByLabelText("Ends"), { target: { value: "06:15" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));

    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({
      digest_minute: 510,
      time_zone: "UTC",
      quiet_hours_enabled: true,
      quiet_start_minute: 1290,
      quiet_end_minute: 375,
      version: 1,
    })));
  });
});
