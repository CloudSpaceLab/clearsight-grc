import { act, cleanup, createEvent, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AppNavigation } from "./AppNavigation";

type NavigationProps = ComponentProps<typeof AppNavigation>;

function shell(props: NavigationProps) {
  return <div className="app-shell">
    <AppNavigation {...props}/>
    <main><button type="button">Review portfolio</button></main>
  </div>;
}

function renderNavigation(overrides: Partial<NavigationProps> = {}) {
  const props: NavigationProps = {
    activeView: "programs",
    target: {},
    configureEnabled: true,
    actorName: "Operations reviewer",
    onNavigate: vi.fn(),
    ...overrides,
  };
  return { ...render(shell(props)), props };
}

function mockCompactViewport() {
  const original = window.matchMedia.bind(window);
  const query = "(max-width: 1024px)";
  const media = Object.assign(new EventTarget(), {
    matches: true,
    media: query,
    onchange: null,
    addListener() {},
    removeListener() {},
  }) satisfies MediaQueryList;
  vi.stubGlobal("matchMedia", (value: string): MediaQueryList => value === query ? media : original(value));
  return {
    change(next: boolean) {
      media.matches = next;
      media.dispatchEvent(Object.assign(new Event("change"), { matches: next, media: query }));
    },
  };
}

async function openNavigation() {
  const opener = screen.getByRole("button", { name: "Open navigation" });
  opener.focus();
  fireEvent.click(opener);
  const dialog = await screen.findByRole("dialog", { name: "Navigation" });
  await waitFor(() => expect(document.activeElement).toBe(within(dialog).getByRole("button", { name: "Close navigation" })));
  return { opener, dialog };
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("AppNavigation interactions", () => {
  it.each([
    { activeView: "programs", target: { programID: "program-1", programSection: "monitoring" }, label: "Programs", destination: "programs", tab: undefined },
    { activeView: "losses", target: { lossID: "loss-1" }, label: "Risk & Losses", destination: "risks", tab: undefined },
    { activeView: "work", target: { matterID: "matter-1" }, label: "Work & Approvals", destination: "work", tab: "assigned" },
  ] satisfies Array<{ activeView: NavigationProps["activeView"]; target: NavigationProps["target"]; label: string; destination: string; tab: string | undefined }>)
  ("opens $label from an active detail instead of preserving the old target", ({ activeView, target, label, destination, tab }) => {
    const { props } = renderNavigation({ activeView, target });
    const link = screen.getByRole("link", { name: label });
    expect(link.getAttribute("aria-current")).toBe("page");
    const click = createEvent.click(link, { button: 0 });

    fireEvent(link, click);

    expect(click.defaultPrevented).toBe(true);
    expect(props.onNavigate).toHaveBeenCalledExactlyOnceWith(destination, {}, tab);
  });

  it.each([
    { modifier: "Control", ctrlKey: true },
    { modifier: "Command", metaKey: true },
    { modifier: "Shift", shiftKey: true },
    { modifier: "Alt", altKey: true },
    { modifier: "middle button", button: 1 },
  ])("keeps native link navigation for a $modifier click", ({ modifier: _modifier, ...options }) => {
    const { props } = renderNavigation();
    const link = screen.getByRole("link", { name: "Frameworks" });
    const click = createEvent.click(link, { button: 0, ...options });

    fireEvent(link, click);

    expect(click.defaultPrevented).toBe(false);
    expect(link.getAttribute("href")).toBe("#programs/library");
    expect(props.onNavigate).not.toHaveBeenCalled();
  });

  it("discloses risk destinations independently of navigation and exposes the selected child", () => {
    const view = renderNavigation({ activeView: "forms" });
    const expand = screen.getByRole("button", { name: "Show risk sections" });
    const childrenID = expand.getAttribute("aria-controls");
    expect(childrenID).toBeTruthy();
    expect(expand.getAttribute("aria-expanded")).toBe("false");
    expect(document.getElementById(childrenID!)?.hidden).toBe(true);
    expect(screen.queryByRole("link", { name: "RCSA" })).toBeNull();

    fireEvent.click(expand);

    const collapse = screen.getByRole("button", { name: "Hide risk sections" });
    expect(collapse.getAttribute("aria-controls")).toBe(childrenID);
    expect(collapse.getAttribute("aria-expanded")).toBe("true");
    expect(document.getElementById(childrenID!)?.hidden).toBe(false);
    expect(view.props.onNavigate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("link", { name: "RCSA" }));
    expect(view.props.onNavigate).toHaveBeenCalledExactlyOnceWith("rcsa", {}, undefined);

    fireEvent.click(collapse);
    expect(screen.queryByRole("link", { name: "RCSA" })).toBeNull();
    view.rerender(shell({ ...view.props, activeView: "losses", target: { lossID: "loss-1" } }));

    expect(screen.getByRole("button", { name: "Hide risk sections" }).getAttribute("aria-expanded")).toBe("true");
    expect(screen.getByRole("link", { name: "Loss events" }).getAttribute("aria-current")).toBe("location");
    expect(screen.getByRole("link", { name: "RCSA" }).getAttribute("aria-current")).toBeNull();
  });

  it("contains keyboard focus, hides the background, and restores the opener after Escape", async () => {
    mockCompactViewport();
    renderNavigation();
    const background = screen.getByRole("button", { name: "Review portfolio" });
    const { opener, dialog } = await openNavigation();

    expect(opener.getAttribute("aria-expanded")).toBe("true");
    expect(screen.queryByRole("complementary", { name: "Primary navigation" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Review portfolio" })).toBeNull();
    expect(background.closest('[aria-hidden="true"]')).not.toBeNull();
    expect(screen.getAllByRole("navigation", { name: "Workspaces" })).toHaveLength(1);
    const close = within(dialog).getByRole("button", { name: "Close navigation" });
    const last = within(dialog).getByRole("link", { name: "Settings" });
    last.focus();
    fireEvent.keyDown(last, { key: "Tab", code: "Tab" });
    await waitFor(() => expect(document.activeElement).toBe(close));
    fireEvent.keyDown(close, { key: "Tab", code: "Tab", shiftKey: true });
    await waitFor(() => expect(document.activeElement).toBe(last));

    fireEvent.keyDown(last, { key: "Escape" });

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(opener));
    expect(opener.getAttribute("aria-expanded")).toBe("false");
    expect(screen.getByRole("button", { name: "Review portfolio" })).toBe(background);
    expect(document.body.style.overflow).toBe("");
  });

  it("closes the compact menu and restores focus when re-entering the active workspace", async () => {
    mockCompactViewport();
    const { props } = renderNavigation({ activeView: "work", target: { matterID: "matter-1" } });
    const { opener, dialog } = await openNavigation();

    fireEvent.click(within(dialog).getByRole("link", { name: "Work & Approvals" }));

    expect(props.onNavigate).toHaveBeenCalledExactlyOnceWith("work", {}, "assigned");
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(opener));
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("keeps the compact menu open for a native modified link click", async () => {
    mockCompactViewport();
    const { props } = renderNavigation();
    const { dialog } = await openNavigation();
    const link = within(dialog).getByRole("link", { name: "Reports" });
    const click = createEvent.click(link, { button: 0, ctrlKey: true });

    fireEvent(link, click);

    expect(click.defaultPrevented).toBe(false);
    expect(props.onNavigate).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog", { name: "Navigation" })).toBe(dialog);
  });

  it("closes the compact menu when history restores another target in the same workspace", async () => {
    mockCompactViewport();
    const view = renderNavigation({ activeView: "programs", target: { programID: "program-1" } });
    const { opener } = await openNavigation();

    view.rerender(shell({ ...view.props, target: { programID: "program-2" } }));

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(opener));
    expect(view.props.onNavigate).not.toHaveBeenCalled();
    expect(screen.getByRole("link", { name: "Programs" }).getAttribute("aria-current")).toBe("page");
  });

  it("closes the compact menu and focuses the current rail link on desktop expansion", async () => {
    const viewport = mockCompactViewport();
    renderNavigation({ activeView: "programs", target: { programsPage: "library" } });
    const railLink = screen.getByRole("link", { name: "Frameworks" });
    await openNavigation();

    act(() => viewport.change(false));

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(railLink));
    expect(screen.getByRole("link", { name: "Frameworks" })).toBe(railLink);
    expect(screen.getByRole("button", { name: "Open navigation" }).getAttribute("aria-expanded")).toBe("false");
    await act(async () => { await new Promise<void>((resolve) => requestAnimationFrame(() => resolve())); });
    expect(document.activeElement).toBe(railLink);
  });

  it("restores a visible rail link on desktop expansion from a contextual route", async () => {
    const viewport = mockCompactViewport();
    renderNavigation({ activeView: "people", target: { personID: "person-1" } });
    const railLinks = screen.getAllByRole("link");
    await openNavigation();

    act(() => viewport.change(false));

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
    await waitFor(() => expect(railLinks).toContain(document.activeElement));
  });
});
