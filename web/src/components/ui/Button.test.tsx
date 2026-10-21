import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ActionCard, ActionLink, Button, IconButton } from "./index";

describe("Button", () => {
  it("runs a named primary action", () => {
    const run = vi.fn();
    render(<Button variant="primary" onPress={run}>Send form</Button>);

    fireEvent.click(screen.getByRole("button", { name: "Send form" }));

    expect(run).toHaveBeenCalledTimes(1);
  });

  it("keeps the action name and prevents repeat activation while loading", () => {
    const run = vi.fn();
    render(<Button variant="primary" isLoading onPress={run}>Send form</Button>);

    const button = screen.getByRole("button", { name: "Send form" });
    fireEvent.click(button);

    expect(run).not.toHaveBeenCalled();
    expect((button as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByRole("status", { name: "Send form in progress" })).toBeTruthy();
  });

  it("requires an accessible business name for an icon action", () => {
    render(<IconButton aria-label="Close distribution detail"><span aria-hidden="true">×</span></IconButton>);

    expect(screen.getByRole("button", { name: "Close distribution detail" })).toBeTruthy();
  });

  it("uses a real link for navigation styling", () => {
    render(<ActionLink href="/forms">Open Forms</ActionLink>);

    const link = screen.getByRole("link", { name: "Open Forms" });
    expect(link.getAttribute("href")).toBe("/forms");
    expect(link.className).toBe("cs-action-link");
  });

  it("preserves link destinations and distinct page/location semantics in navigation rows", () => {
    const follow = vi.fn();
    render(<nav aria-label="Workspaces">
      <ActionLink variant="navigation" href="#programs" aria-current="page" title="Programs" onPress={follow}>
        <span aria-hidden="true">◫</span>Programs
      </ActionLink>
      <ActionLink variant="navigation" href="#programs/program-1/requirements-controls" aria-current="location">Requirements</ActionLink>
      <ActionLink variant="navigation" href="#forms">Forms</ActionLink>
    </nav>);

    const programs = screen.getByRole("link", { name: "Programs" });
    expect(programs.getAttribute("href")).toBe("#programs");
    expect(programs.getAttribute("title")).toBe("Programs");
    expect(programs.getAttribute("aria-current")).toBe("page");
    expect(programs.classList.contains("cs-action-link--navigation")).toBe(true);
    expect(screen.getByRole("link", { name: "Requirements" }).getAttribute("aria-current")).toBe("location");
    expect(screen.getByRole("link", { name: "Forms" }).hasAttribute("aria-current")).toBe(false);
    fireEvent.click(programs);
    expect(follow).toHaveBeenCalledTimes(1);
  });

  it("keeps unavailable navigation links outside activation and announces the state", () => {
    const follow = vi.fn();
    render(<ActionLink variant="navigation" href="#configure" title="Settings unavailable" isDisabled onPress={follow}>Settings</ActionLink>);

    const settings = screen.getByRole("link", { name: "Settings" });
    fireEvent.click(settings);
    fireEvent.keyDown(settings, { key: "Enter", code: "Enter" });
    fireEvent.keyUp(settings, { key: "Enter", code: "Enter" });
    expect(follow).not.toHaveBeenCalled();
    expect(settings.getAttribute("aria-disabled")).toBe("true");
    expect(settings.getAttribute("title")).toBe("Settings unavailable");
    expect(settings.tabIndex).toBe(-1);
  });

  it("presents a named creation choice with supporting context", () => {
    const choose = vi.fn();
    render(<ActionCard title="Blank form" description="Start with an empty governed draft" onPress={choose} icon={<span aria-hidden="true">+</span>}/>);

    fireEvent.click(screen.getByRole("button", { name: /Blank form/ }));

    expect(choose).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Start with an empty governed draft")).toBeTruthy();
  });
});
