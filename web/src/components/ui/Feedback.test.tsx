import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button, Card, EmptyState, Notice, StatusBadge, Surface } from "./index";

describe("feedback and surfaces", () => {
  it("renders status with text and a non-color marker", () => {
    const view = render(<StatusBadge tone="warning">Awaiting review</StatusBadge>);

    expect(screen.getByText("Awaiting review")).toBeTruthy();
    expect(view.container.querySelector("[aria-hidden='true']")).toBeTruthy();
  });

  it("uses alert semantics only for blocking feedback", () => {
    const view = render(<><Notice tone="error">Sent forms could not be loaded. Retry the request.</Notice><Notice tone="success">The distribution was locked.</Notice></>);

    expect(screen.getByRole("alert").textContent).toContain("Sent forms could not be loaded");
    expect(screen.getByRole("status").textContent).toContain("distribution was locked");
    view.unmount();
  });

  it("states the checked population, result and next action", () => {
    render(<EmptyState
      population="Sent forms in the current filter"
      title="No sent forms match this view"
      description="Change the filters or send a form to create a request."
      action={<Button variant="primary">Send form</Button>}
    />);

    expect(screen.getByText("Sent forms in the current filter")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "No sent forms match this view" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Send form" })).toBeTruthy();
  });

  it("keeps containment non-interactive", () => {
    const view = render(<Surface>Summary</Surface>);
    expect(view.container.firstElementChild?.tagName).toBe("DIV");
    view.rerender(<Card>Quarterly control review</Card>);
    expect(view.container.firstElementChild?.tagName).toBe("ARTICLE");
    expect(screen.queryByRole("button")).toBeNull();
  });
});
