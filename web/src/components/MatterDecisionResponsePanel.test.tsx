import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { loadResponsePackageHistory } from "../api";
import type { MatterAggregate } from "../types";
import { MatterDecisionResponsePanel } from "./MatterDecisionResponsePanel";

vi.mock("../api", () => ({ loadResponsePackageHistory: vi.fn() }));
vi.mock("../continuityCommands", () => ({ addResponsePackage: vi.fn(), recordMatterDecision: vi.fn(), transitionResponsePackage: vi.fn() }));

it("loads bounded response transitions only when history is expanded", async () => {
  vi.mocked(loadResponsePackageHistory).mockResolvedValue({ items: [{ status: "APPROVED", occurred_at: "2026-08-25T10:00:00Z", actor_label: "Ada Okafor", matter_version: 7 }], has_more: true, generated_at: "2026-08-25T10:01:00Z" });
  const aggregate = { matter: { id: "matter-1", version: 7 }, decisions: [], response_packages: [{ id: "response-1", purpose: "Regulatory response", audience: "Regulator", status: "APPROVED" }] } as unknown as MatterAggregate;
  render(<MatterDecisionResponsePanel aggregate={aggregate} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
  expect(loadResponsePackageHistory).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText("View response status history"));
  expect(await screen.findByText(/Ada Okafor/)).toBeTruthy();
  expect(screen.getByText(/older transitions are not shown/)).toBeTruthy();
  expect(loadResponsePackageHistory).toHaveBeenCalledWith("matter-1", "response-1", 20);
});
