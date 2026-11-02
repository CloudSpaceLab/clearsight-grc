import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NotificationCenter, safeNotificationPath } from "./NotificationCenter";

describe("NotificationCenter", () => {
  it("announces the shared server count and opens the workspace without another inbox owner", () => {
    const onOpen = vi.fn();
    const { rerender } = render(<NotificationCenter inbox={{ state: "live", unreadCount: 107 }} onOpen={onOpen}/>);
    fireEvent.click(screen.getByRole("button", { name: "Notifications, 107 unread" }));
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(screen.getByText("99+").getAttribute("aria-hidden")).toBe("true");
    expect(screen.queryByRole("dialog")).toBeNull();

    rerender(<NotificationCenter inbox={{ state: "live", unreadCount: 106 }} onOpen={onOpen} isCurrent/>);
    expect(screen.getByRole("button", { name: "Notifications, 106 unread" }).getAttribute("aria-current")).toBe("page");
    rerender(<NotificationCenter inbox={{ state: "live", unreadCount: 0 }} onOpen={onOpen}/>);
    expect(screen.getByRole("button", { name: "Notifications" }).getAttribute("aria-current")).toBeNull();
    expect(screen.queryByText("99+")).toBeNull();
  });

  it("keeps loading and unavailable counts unknown while still opening the workspace", () => {
    const onOpen = vi.fn();
    const { rerender } = render(<NotificationCenter inbox={{ state: "loading", unreadCount: undefined }} onOpen={onOpen}/>);
    fireEvent.click(screen.getByRole("button", { name: "Notifications loading" }));
    rerender(<NotificationCenter inbox={{ state: "unavailable", unreadCount: undefined }} onOpen={onOpen}/>);
    fireEvent.click(screen.getByRole("button", { name: "Notifications unavailable" }));
    expect(onOpen).toHaveBeenCalledTimes(2);
    expect(screen.queryByText("0")).toBeNull();
  });
});

describe("safeNotificationPath", () => {
  it.each([
    "#work/matters/22222222-2222-4222-8222-222222222222",
    "#risks/22222222-2222-4222-8222-222222222222",
    "#losses/33333333-3333-4333-8333-333333333333",
    "#forms/responses/response-1?assessment=assessment-1",
    "#reports",
  ])("accepts the existing in-app destination %s", (path) => {
    expect(safeNotificationPath(path)).toBe(true);
  });

  it.each(["https://example.test/phish", "javascript:alert(1)", "//example.test", "#work-other", "#unknown/record", "#work\n/record", "#work\r/record"])("rejects unsafe or unknown destination %s", (path) => {
    expect(safeNotificationPath(path)).toBe(false);
  });
});
