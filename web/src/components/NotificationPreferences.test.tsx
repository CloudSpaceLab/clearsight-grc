import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NotificationPreferences } from "./NotificationPreferences";
import type { NotificationPreferences as Value } from "../notificationPreferencesApi";
import { ApiError } from "../http";

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

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((yes) => { resolve = yes; });
  return { promise, resolve };
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

  it("saves edited local times as bounded minutes with the current version and timezone", async () => {
    const load = vi.fn().mockResolvedValue(value({
      digest_minute: 510,
      quiet_hours_enabled: true,
      quiet_start_minute: 1290,
      quiet_end_minute: 375,
    }));
    const save = vi.fn().mockImplementation(async (input) => value({ ...input, version: 2 }));
    render(<NotificationPreferences load={load} save={save}/>);

    expect(await screen.findByDisplayValue("08:30")).toBeTruthy();
    expect(screen.getByDisplayValue("21:30")).toBeTruthy();
    expect(screen.getByDisplayValue("06:15")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Digest time"), { target: { value: "09:45" } });
    fireEvent.change(screen.getByLabelText("Starts"), { target: { value: "22:10" } });
    fireEvent.change(screen.getByLabelText("Ends"), { target: { value: "06:20" } });
    fireEvent.change(screen.getByLabelText("Time zone"), { target: { value: "UTC" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));

    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({
      digest_minute: 585,
      time_zone: "UTC",
      quiet_hours_enabled: true,
      quiet_start_minute: 1330,
      quiet_end_minute: 380,
      version: 1,
    })));
    expect(await screen.findByText("Email preferences saved.")).toBeTruthy();
  });

  it("saves midnight and the last minute of the day without shifting the local times", async () => {
    const load = vi.fn().mockResolvedValue(value({ quiet_hours_enabled: true }));
    const save = vi.fn().mockImplementation(async (input) => value({ ...input, version: 2 }));
    render(<NotificationPreferences load={load} save={save}/>);
    fireEvent.change(await screen.findByLabelText("Digest time"), { target: { value: "00:00" } });
    fireEvent.change(screen.getByLabelText("Starts"), { target: { value: "23:59" } });
    fireEvent.change(screen.getByLabelText("Ends"), { target: { value: "00:00" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));
    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({ digest_minute: 0, quiet_start_minute: 1439, quiet_end_minute: 0 })));
  });

  it("retains a conflicted draft without claiming success, then reloads the current version before saving", async () => {
    const load = vi.fn().mockResolvedValueOnce(value({ version: 3 })).mockResolvedValueOnce(value({ version: 4, digest_minute: 600 }));
    const save = vi.fn().mockRejectedValueOnce(new ApiError(409, "Version changed", "version_conflict")).mockImplementationOnce(async (input) => value({ ...input, version: 5 }));
    render(<NotificationPreferences load={load} save={save}/>);
    fireEvent.change(await screen.findByLabelText("Digest time"), { target: { value: "09:45" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));
    expect(await screen.findByText("Email preferences changed. Reload them before saving again.")).toBeTruthy();
    expect((screen.getByLabelText("Digest time") as HTMLInputElement).value).toBe("09:45");
    expect(screen.queryByText("Email preferences saved.")).toBeNull();
    expect(save).toHaveBeenNthCalledWith(1, expect.objectContaining({ version: 3, digest_minute: 585 }));

    fireEvent.click(screen.getByRole("button", { name: "Reload preferences" }));
    expect(await screen.findByDisplayValue("10:00")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Digest time"), { target: { value: "11:00" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));
    expect(await screen.findByText("Email preferences saved.")).toBeTruthy();
    expect(save).toHaveBeenNthCalledWith(2, expect.objectContaining({ version: 4, digest_minute: 660 }));
  });

  it("offers recovery for unavailable preferences without saving default settings", async () => {
    const load = vi.fn().mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce(value());
    const save = vi.fn();
    render(<NotificationPreferences load={load} save={save}/>);
    expect(await screen.findByText("Email preferences are unavailable. Try again to load your settings.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save email preferences" })).toBeNull();
    expect(save).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByLabelText("Digest time")).toBeTruthy();
    expect(load).toHaveBeenCalledTimes(2);
  });

  it("ignores a pending save from a replaced preferences owner", async () => {
    const oldSave = deferred<Value>();
    const load = vi.fn().mockResolvedValue(value());
    const nextLoad = vi.fn().mockResolvedValue(value({ principal_id: "actor-2", time_zone: "Europe/London", digest_minute: 540 }));
    const save = vi.fn().mockReturnValue(oldSave.promise);
    const { rerender } = render(<NotificationPreferences load={load} save={save}/>);
    fireEvent.change(await screen.findByLabelText("Time zone"), { target: { value: "UTC" } });
    fireEvent.click(screen.getByRole("button", { name: "Save email preferences" }));
    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    rerender(<NotificationPreferences load={nextLoad} save={save}/>);
    expect(await screen.findByDisplayValue("Europe/London")).toBeTruthy();
    expect((load.mock.calls[0]![0] as AbortSignal).aborted).toBe(true);
    await act(async () => { oldSave.resolve(value({ time_zone: "UTC", version: 2 })); });
    expect((screen.getByLabelText("Time zone") as HTMLInputElement).value).toBe("Europe/London");
    expect(screen.queryByText("Email preferences saved.")).toBeNull();
  });
});
