import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ActionLink, Button, IconButton } from "./index";

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

    expect((screen.getByRole("link", { name: "Open Forms" }) as HTMLAnchorElement).getAttribute("href")).toBe("/forms");
  });
});
