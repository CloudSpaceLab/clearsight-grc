import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { submitCaptureRequest } from "../api";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

vi.mock("../api", () => ({ submitCaptureRequest: vi.fn() }));

const request: CaptureRequest = {
  id: "request-1",
  title: "Confirm the current control owner",
  purpose: "Resolve the final ownership gap.",
  why_you: "You own the affected process.",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 2,
  deadline: "2026-08-09T12:00:00Z",
  known_facts: { process: "Treasury operations" },
  fields: [{ id: "owner", label: "Current owner", type: "text", required: true }],
  version: 3,
  source: "evidence",
};

describe("CapturePanel", () => {
  it("reviews exact assertions before submitting", async () => {
    vi.mocked(submitCaptureRequest).mockResolvedValue({ request_id: request.id, status: "SUBMITTED", submitted_at: "2026-08-06T19:30:00Z" });
    render(<CapturePanel request={request}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));

    expect(screen.getByRole("heading", { name: "Confirm the assertions you are submitting" })).toBeTruthy();
    expect(screen.getByText("Treasury Technology")).toBeTruthy();
    expect(submitCaptureRequest).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submitCaptureRequest).toHaveBeenCalledWith(request.id, 3, { owner: "Treasury Technology" }, "evidence"));
    expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
  });
});
