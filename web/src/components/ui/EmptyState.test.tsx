import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { EmptyState } from "./EmptyState";

it("renders a compact accessible empty state without redundant population text", () => {
  const { container } = render(<EmptyState
    compact
    population="No risks outside appetite"
    title="No risks outside appetite"
    description="No current risks exceed approved appetite."
  />);

  expect(screen.getByRole("heading", { name: "No risks outside appetite" })).toBeTruthy();
  expect(screen.getAllByText("No risks outside appetite")).toHaveLength(1);
  expect(container.querySelector(".cs-empty-state--compact")).toBeTruthy();
});

it("keeps the standard empty state when compact is omitted", () => {
  const { container } = render(<EmptyState
    population="Current assessments"
    title="No assessments"
    description="No matching assessments."
  />);

  expect(container.querySelector(".cs-empty-state--compact")).toBeNull();
  expect(screen.getByText("Current assessments")).toBeTruthy();
});
