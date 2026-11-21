import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { DataSourceBuilder } from "./DataSourceBuilder";
import { createRESTBinding, prepareRESTSource } from "../sourceConfigApi";

vi.mock("../sourceConfigApi", () => ({ prepareRESTSource: vi.fn(), createRESTBinding: vi.fn() }));
beforeEach(() => vi.clearAllMocks());

it("checks a sample without contacting or creating a source", () => {
  render(<DataSourceBuilder onSaved={vi.fn()} onCancel={vi.fn()}/>);
  fireEvent.change(screen.getByLabelText("Sample result type"), { target: { value: "number" } });
  fireEvent.change(screen.getByLabelText("Sample response"), { target: { value: '{"result":0,"message":"Available"}' } });
  fireEvent.click(screen.getByRole("button", { name: "Test sample" }));
  expect(screen.getByText("Sample valid. Endpoint not contacted.")).toBeTruthy();
  expect(prepareRESTSource).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText("Sample response"), { target: { value: '{"result":false}' } });
  fireEvent.click(screen.getByRole("button", { name: "Test sample" }));
  expect(screen.getByText(/result must be a finite number/i)).toBeTruthy();
});

it("uses the selected standard contract for endpoint inspection and draft binding", async () => {
  const prepared = { resultType: "NUMBER", source: { code: "CHECK", name: "Check", expected_freshness_minutes: 1440 }, connection: { connection_id: "connection", version: 1 }, view: { view_id: "view", version: 2, native_schema: [{ name: "result", native_type: "json:number", nullable: false }, { name: "message", native_type: "json:string", nullable: true }] } } as Awaited<ReturnType<typeof prepareRESTSource>>;
  vi.mocked(prepareRESTSource).mockResolvedValue(prepared);
  vi.mocked(createRESTBinding).mockResolvedValue({ binding_id: "binding", status: "DRAFT" } as Awaited<ReturnType<typeof createRESTBinding>>);
  const onSaved = vi.fn();
  render(<DataSourceBuilder onSaved={onSaved} onCancel={vi.fn()}/>);
  fireEvent.change(screen.getByLabelText("Sample result type"), { target: { value: "number" } });
  fireEvent.change(screen.getByLabelText("Source name"), { target: { value: "Check" } });
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "CHECK" } });
  fireEvent.change(screen.getByLabelText("Status endpoint"), { target: { value: "https://bank.example/check" } });
  fireEvent.change(screen.getByLabelText("Maximum age (minutes)"), { target: { value: "1440" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  await waitFor(() => expect(prepareRESTSource).toHaveBeenCalledWith(expect.objectContaining({ resultType: "NUMBER", freshnessMinutes: 1440 })));
  expect(await screen.findByText(/Source approval is unavailable. The check condition cannot be saved or run yet/)).toBeTruthy();
  expect(screen.queryByLabelText("Observed field")).toBeNull();
  expect(screen.queryByLabelText("Condition")).toBeNull();
  expect(screen.queryByLabelText("Expected value")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await waitFor(() => expect(createRESTBinding).toHaveBeenCalledWith(prepared, "result"));
  expect(await screen.findByText("Source draft saved. The check condition has not been saved. Source approval is unavailable.")).toBeTruthy();
  expect(onSaved).not.toHaveBeenCalled();
});

it("uses yes or no for a boolean result and rejects a mismatched inspected contract", async () => {
  const prepared = { resultType: "BOOLEAN", source: { code: "CHECK", name: "Check", expected_freshness_minutes: 60 }, connection: { connection_id: "connection", version: 1, status: "DRAFT" }, view: { view_id: "view", version: 2, native_schema: [{ name: "result", native_type: "json:boolean", nullable: false }] } } as Awaited<ReturnType<typeof prepareRESTSource>>;
  vi.mocked(prepareRESTSource).mockResolvedValueOnce({ ...prepared, view: { ...prepared.view, native_schema: [{ name: "result", native_type: "json:number", nullable: false }] } }).mockResolvedValueOnce(prepared);
  vi.mocked(createRESTBinding).mockResolvedValue({ binding_id: "binding", status: "DRAFT" } as Awaited<ReturnType<typeof createRESTBinding>>);
  const onSaved = vi.fn();
  render(<DataSourceBuilder onSaved={onSaved} onCancel={vi.fn()}/>);
  fireEvent.change(screen.getByLabelText("Source name"), { target: { value: "Check" } });
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "CHECK" } });
  fireEvent.change(screen.getByLabelText("Status endpoint"), { target: { value: "https://bank.example/check" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  expect(await screen.findByText("The endpoint response does not match the selected result type.")).toBeTruthy();
  expect(createRESTBinding).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  await screen.findByText(/Source approval is unavailable/);
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByText("Source draft saved. The check condition has not been saved. Source approval is unavailable.")).toBeTruthy();
  expect(onSaved).not.toHaveBeenCalled();
});

it("rejects non-numeric thresholds before creating a binding", async () => {
  const prepared = { resultType: "NUMBER", source: { code: "CHECK", name: "Check", expected_freshness_minutes: 60 }, connection: { connection_id: "connection", version: 1, status: "ACTIVE" }, view: { view_id: "view", version: 2, native_schema: [{ name: "result", native_type: "json:number", nullable: false }] } } as Awaited<ReturnType<typeof prepareRESTSource>>;
  vi.mocked(prepareRESTSource).mockResolvedValue(prepared);
  render(<DataSourceBuilder onSaved={vi.fn()} onCancel={vi.fn()}/>);
  fireEvent.change(screen.getByLabelText("Sample result type"), { target: { value: "number" } });
  fireEvent.change(screen.getByLabelText("Source name"), { target: { value: "Check" } });
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "CHECK" } });
  fireEvent.change(screen.getByLabelText("Status endpoint"), { target: { value: "https://bank.example/check" } });
  fireEvent.click(screen.getByRole("button", { name: "Test endpoint" }));
  fireEvent.change(await screen.findByLabelText("Expected value"), { target: { value: "invalid" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByText("Enter a valid numeric value.")).toBeTruthy();
  expect(createRESTBinding).not.toHaveBeenCalled();
});
