import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { APIQuestionWizard } from "./APIQuestionWizard";

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

it("provides a typed developer contract while preserving manual answers", () => {
  render(<APIQuestionWizard formID="form/1" version={2} fieldID="q/1" label="Is access reviewed?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  expect(screen.getByText(/Answers remain manual/)).toBeTruthy();
  expect(screen.getByText(/GET/)).toBeTruthy();
  expect(screen.getByText(/"result": true/)).toBeTruthy();
  expect(screen.getByText(/Is access reviewed\?/)).toBeTruthy();
});

it("validates a sample through the scoped test endpoint and never saves an API binding", async () => {
  const fetch = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ mode: "SAMPLE", outcome: "VALID", result_type: "NUMBER", checked_at: "2026-11-20T10:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  render(<APIQuestionWizard formID="form/1" version={2} fieldID="q/1" label="Score?" resultType="NUMBER"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("Sample response"), { target: { value: '{"result":0,"message":"No incidents"}' } });
  fireEvent.click(screen.getByRole("button", { name: "Test sample" }));
  expect((await screen.findByRole("status")).textContent).toContain("Sample format valid");
  expect(fetch).toHaveBeenCalledTimes(1);
  const [url, request] = fetch.mock.calls[0]!;
  expect(url).toContain("/api/v1/forms/templates/form%2F1/revisions/2/questions/q%2F1/test");
  expect(JSON.parse(String(request?.body))).toEqual({ mode: "SAMPLE", result_type: "NUMBER", sample: { result: 0, message: "No incidents" } });
});

it("distinguishes an unavailable endpoint from a failed control", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ mode: "LIVE", outcome: "UNAVAILABLE", result_type: "BOOLEAN", checked_at: "2026-11-20T10:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("HTTPS endpoint"), { target: { value: "https://example.com/status" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  await waitFor(() => expect(screen.getByRole("alert").textContent).toContain("Endpoint unavailable"));
  expect(screen.getByText(/Answers remain manual/)).toBeTruthy();
});

it("rejects endpoint credentials and parameters without sending them", () => {
  const fetch = vi.spyOn(globalThis, "fetch");
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("HTTPS endpoint"), { target: { value: "https://example.com/status?key=secret" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  expect(screen.getByRole("alert").textContent).toContain("without credentials");
  expect(fetch).not.toHaveBeenCalled();
});

it("shows when an endpoint response was tested", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ mode: "LIVE", outcome: "VALID", result_type: "BOOLEAN", checked_at: "2026-11-20T10:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("HTTPS endpoint"), { target: { value: "https://example.com/status" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  expect((await screen.findByRole("status")).textContent).toContain("2026-11-20T10:00:00Z");
});

it("copies the question and response contract for an engineer", async () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Is access reviewed?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Copy developer instructions" }));
  await waitFor(() => expect(writeText).toHaveBeenCalledOnce());
  expect(writeText.mock.calls[0]![0]).toContain("Is access reviewed?");
  expect(writeText.mock.calls[0]![0]).toContain('"result": true');
  expect(writeText.mock.calls[0]![0]).toContain("GET");
  expect((await screen.findByRole("status")).textContent).toContain("Instructions copied");
});

it("reports clipboard failure without claiming instructions were copied", async () => {
  vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) } });
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Score?" resultType="NUMBER"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Copy developer instructions" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Copy unavailable");
});

it("keeps the tested endpoint unchanged until its result arrives", async () => {
  let resolveRequest!: (response: Response) => void;
  vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise<Response>((resolve) => { resolveRequest = resolve; }));
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  const endpoint = screen.getByLabelText("HTTPS endpoint") as HTMLInputElement;
  fireEvent.change(endpoint, { target: { value: "https://example.com/status" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  expect(endpoint.disabled).toBe(true);
  expect((screen.getByLabelText("Sample response") as HTMLTextAreaElement).disabled).toBe(true);
  resolveRequest(new Response(JSON.stringify({ mode: "LIVE", outcome: "VALID", result_type: "BOOLEAN", checked_at: "2026-11-20T10:00:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  await screen.findByRole("status");
  expect(endpoint.disabled).toBe(false);
});

it.each(["https://example.com/status?", "https://example.com/status#"])("rejects an empty query or fragment before testing %s", (endpoint) => {
  const fetch = vi.spyOn(globalThis, "fetch");
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  fireEvent.change(screen.getByLabelText("HTTPS endpoint"), { target: { value: endpoint } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  expect(screen.getByRole("alert").textContent).toContain("without credentials");
  expect(fetch).not.toHaveBeenCalled();
});

it("does not show a delayed clipboard result on the response test step", async () => {
  let finishCopy!: () => void;
  const writeText = vi.fn().mockReturnValue(new Promise<void>((resolve) => { finishCopy = resolve; }));
  vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Access?" resultType="BOOLEAN"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  fireEvent.click(screen.getByRole("button", { name: "Copy developer instructions" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  finishCopy();
  await waitFor(() => expect(writeText).toHaveBeenCalledOnce());
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(screen.queryByText("Instructions copied.")).toBeNull();
});


it("provides the same complete response limits on screen and in developer instructions", async () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
  render(<APIQuestionWizard formID="f" version={1} fieldID="q" label="Score?" resultType="NUMBER"/>);
  fireEvent.click(screen.getByRole("button", { name: "Prepare API question" }));
  expect(screen.getByText(/65,536 bytes/)).toBeTruthy();
  expect(screen.getByText(/1,000 characters/)).toBeTruthy();
  expect(screen.getByText(/Content-Type/)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Copy developer instructions" }));
  await waitFor(() => expect(writeText).toHaveBeenCalledOnce());
  const instructions = writeText.mock.calls[0]![0] as string;
  expect(instructions).toContain("65,536 bytes");
  expect(instructions).toContain("1,000 characters");
  expect(instructions).toContain("Content-Type");
  expect(instructions).toContain("finite number");
});
