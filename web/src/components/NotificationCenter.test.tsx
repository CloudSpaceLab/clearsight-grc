import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NotificationCenter } from "./NotificationCenter";
import type { InAppNotification, NotificationPage } from "../notificationApi";

function item(overrides: Partial<InAppNotification> = {}): InAppNotification {
  return {
    id: "11111111-1111-4111-8111-111111111111",
    kind: "MATTER_OWNER_ASSIGNED",
    title: "New work assigned",
    summary: "A governed work item was assigned to you.",
    subject_type: "MATTER",
    subject_id: "22222222-2222-4222-8222-222222222222",
    action_path: "#work/matters/22222222-2222-4222-8222-222222222222",
    occurred_at: "2026-10-02T09:00:00Z",
    ...overrides,
  };
}

function page(overrides: Partial<NotificationPage> = {}): NotificationPage {
  return {
    items: [item()],
    unread_count: 1,
    as_of: "2026-10-02T09:05:00Z",
    ...overrides,
  };
}

describe("NotificationCenter", () => {
  it("shows an accessible unread count and opens the canonical work path", async () => {
    const loadPage = vi.fn().mockResolvedValue(page());
    const markRead = vi.fn().mockResolvedValue(item({ read_at: "2026-10-02T09:06:00Z" }));
    const onOpenPath = vi.fn();
    render(<NotificationCenter loadPage={loadPage} markRead={markRead} onOpenPath={onOpenPath}/>);

    const launcher = await screen.findByRole("button", { name: "Notifications, 1 unread" });
    fireEvent.click(launcher);
    expect(await screen.findByRole("dialog", { name: "Notifications" })).toBeTruthy();
    expect(screen.getByText("Unread")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(markRead).toHaveBeenCalledWith("11111111-1111-4111-8111-111111111111");
    expect(onOpenPath).toHaveBeenCalledWith("#work/matters/22222222-2222-4222-8222-222222222222");
    await waitFor(() => expect(screen.getByRole("button", { name: "Notifications" })).toBeTruthy());
  });

  it("rolls unread state back when read metadata cannot be persisted", async () => {
    const loadPage = vi.fn().mockResolvedValue(page());
    const markRead = vi.fn().mockRejectedValue(new Error("unavailable"));
    const onOpenPath = vi.fn();
    render(<NotificationCenter loadPage={loadPage} markRead={markRead} onOpenPath={onOpenPath}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Notifications, 1 unread" }));
    fireEvent.click(screen.getByRole("button", { name: "Open" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Notifications, 1 unread" })).toBeTruthy());
    expect(onOpenPath).toHaveBeenCalledTimes(1);
  });

  it("keeps the current page visible when loading more fails and permits retry", async () => {
    const loadPage = vi.fn()
      .mockResolvedValueOnce(page({ next_cursor: "cursor-2" }))
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce(page({
        items: [item({
          id: "33333333-3333-4333-8333-333333333333",
          title: "You were mentioned",
          kind: "MATTER_COMMENT_MENTIONED",
          occurred_at: "2026-10-02T08:00:00Z",
        })],
        unread_count: 1,
      }));
    render(<NotificationCenter loadPage={loadPage} markRead={vi.fn()} onOpenPath={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Notifications, 1 unread" }));
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByText("More notifications could not be loaded. The current list remains available.")).toBeTruthy();
    expect(screen.getByText("New work assigned")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByText("You were mentioned")).toBeTruthy();
    expect(screen.queryByText("More notifications could not be loaded. The current list remains available.")).toBeNull();
  });

  it("does not offer an unsafe notification path", async () => {
    const loadPage = vi.fn().mockResolvedValue(page({ items: [item({ action_path: "https://example.test/phish" })] }));
    render(<NotificationCenter loadPage={loadPage} markRead={vi.fn()} onOpenPath={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Notifications, 1 unread" }));
    expect(screen.getByRole("button", { name: "Open" }).getAttribute("data-disabled")).not.toBeNull();
  });

  it("keeps unavailable delivery state separate from assigned work", async () => {
    const loadPage = vi.fn().mockRejectedValue(new Error("unavailable"));
    render(<NotificationCenter loadPage={loadPage} markRead={vi.fn()} onOpenPath={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Notifications unavailable" }));
    expect(await screen.findByRole("alert")).toBeTruthy();
    expect(screen.getByText("Assigned work is unchanged. Retry this delivery view.")).toBeTruthy();
  });
});
