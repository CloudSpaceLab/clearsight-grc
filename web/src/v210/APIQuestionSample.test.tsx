import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { APIQuestionSample } from "./APIQuestionSample";

it("tests a boolean example without contacting an endpoint", () => {
  const fetch = vi.spyOn(globalThis, "fetch");
  render(<APIQuestionSample/>);
  fireEvent.change(screen.getByLabelText("Sample response"), { target: { value: '{"result":false,"message":"Not enabled"}' } });
  fireEvent.click(screen.getByRole("button", { name: "Test sample" }));
  expect(screen.getByRole("status").textContent).toContain("Sample format valid");
  expect(screen.getByText("Sample only · Endpoint not contacted")).toBeTruthy();
  expect(fetch).not.toHaveBeenCalled();
  fetch.mockRestore();
});

it("rejects contradictory keys and clears a prior result after changes", () => {
  render(<APIQuestionSample/>);
  fireEvent.change(screen.getByLabelText("Sample response"), { target: { value: '{"result":false,"result":true}' } });
  fireEvent.click(screen.getByRole("button", { name: "Test sample" }));
  expect(screen.getByRole("alert").textContent).toContain("duplicate");
  fireEvent.change(screen.getByLabelText("Result type"), { target: { value: "number" } });
  expect(screen.queryByRole("alert")).toBeNull();
});
